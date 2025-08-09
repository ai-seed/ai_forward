package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/infrastructure/storage"
)

// AI302Service 302.AI服务接口
type AI302Service interface {
	// Upscale 图片放大
	Upscale(ctx context.Context, userID, apiKeyID int64, request *clients.AI302UpscaleRequest) (*clients.AI302UpscaleResponse, error)
}

// ai302ServiceImpl 302.AI服务实现
type ai302ServiceImpl struct {
	ai302Client              clients.AI302Client
	providerRepo             repositories.ProviderRepository
	modelRepo                repositories.ModelRepository
	modelPricingRepo         repositories.ModelPricingRepository
	providerModelSupportRepo repositories.ProviderModelSupportRepository
	usageLogRepo             repositories.UsageLogRepository
	s3Service                storage.S3Service
	logger                   logger.Logger
}

// NewAI302Service 创建302.AI服务
func NewAI302Service(
	ai302Client clients.AI302Client,
	providerRepo repositories.ProviderRepository,
	modelRepo repositories.ModelRepository,
	modelPricingRepo repositories.ModelPricingRepository,
	providerModelSupportRepo repositories.ProviderModelSupportRepository,
	usageLogRepo repositories.UsageLogRepository,
	s3Service storage.S3Service,
	logger logger.Logger,
) AI302Service {
	return &ai302ServiceImpl{
		ai302Client:              ai302Client,
		providerRepo:             providerRepo,
		modelRepo:                modelRepo,
		modelPricingRepo:         modelPricingRepo,
		providerModelSupportRepo: providerModelSupportRepo,
		usageLogRepo:             usageLogRepo,
		s3Service:                s3Service,
		logger:                   logger,
	}
}

// Upscale 图片放大
func (s *ai302ServiceImpl) Upscale(ctx context.Context, userID, apiKeyID int64, request *clients.AI302UpscaleRequest) (*clients.AI302UpscaleResponse, error) {
	return s.processAI302RequestWithModel(ctx, userID, apiKeyID, "upscale", "/ai/upscale", func(provider *entities.Provider) (*clients.AI302UpscaleResponse, error) {
		return s.ai302Client.Upscale(ctx, provider, request)
	})
}

// processAI302RequestWithModel 处理302.AI请求的通用方法
func (s *ai302ServiceImpl) processAI302RequestWithModel(
	ctx context.Context,
	userID, apiKeyID int64,
	modelSlug, endpoint string,
	requestFunc func(provider *entities.Provider) (*clients.AI302UpscaleResponse, error),
) (*clients.AI302UpscaleResponse, error) {
	startTime := time.Now()

	s.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"api_key_id": apiKeyID,
		"model":      modelSlug,
		"endpoint":   endpoint,
	}).Info("Processing 302.AI request")

	// 获取模型信息
	model, err := s.modelRepo.GetBySlug(ctx, modelSlug)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"model_slug": modelSlug,
		}).Error("Failed to get model")
		return nil, fmt.Errorf("model not found: %s", modelSlug)
	}

	if !model.IsAvailable() {
		return nil, fmt.Errorf("model %s is not available", modelSlug)
	}

	// 获取支持该模型的提供商列表
	supportInfos, err := s.providerModelSupportRepo.GetSupportingProviders(ctx, modelSlug)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"model_slug": modelSlug,
		}).Error("Failed to get supporting providers")
		return nil, fmt.Errorf("no supporting providers found for model %s", modelSlug)
	}

	if len(supportInfos) == 0 {
		return nil, fmt.Errorf("no supporting providers found for model %s", modelSlug)
	}

	// 选择第一个可用的提供商（已按优先级排序）
	var provider *entities.Provider
	for _, supportInfo := range supportInfos {
		if supportInfo.IsAvailable() {
			provider = supportInfo.Provider
			break
		}
	}

	if provider == nil {
		return nil, fmt.Errorf("no available providers found for model %s", modelSlug)
	}

	// 获取模型定价信息
	pricingList, err := s.modelPricingRepo.GetByModelID(ctx, model.ID)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"model_id": model.ID,
		}).Error("Failed to get model pricing")
		return nil, fmt.Errorf("model pricing not found")
	}

	// 查找按次计费的定价
	var pricing *entities.ModelPricing
	for _, p := range pricingList {
		if p.PricingType == entities.PricingTypeRequest && p.IsEffective(time.Now()) {
			pricing = p
			break
		}
	}

	if pricing == nil {
		return nil, fmt.Errorf("no effective per-request pricing found for model %s", modelSlug)
	}

	// 执行请求
	response, err := requestFunc(provider)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"provider": provider.Name,
			"model":    modelSlug,
		}).Error("Failed to process 302.AI request")
		return nil, fmt.Errorf("failed to process request: %w", err)
	}

	// 计算成本（按次计费，1次请求）
	cost := pricing.CalculateCost(1) // 使用定价信息计算成本，1次请求

	// 记录计费相关信息（供计费中间件使用）
	duration := time.Since(startTime)
	
	// 注意：不在这里创建UsageLog，让计费中间件统一处理
	// 设置响应级别的成本信息
	response.Cost = &clients.CostInfo{
		TotalCost: cost,
	}
	
	s.logger.WithFields(map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
		"model_id":    model.ID,
		"provider_id": provider.ID,
		"cost":        cost,
	}).Debug("Request processing completed, billing info prepared")

	s.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"api_key_id": apiKeyID,
		"model":      modelSlug,
		"duration":   duration.String(),
		"status":     response.Status,
	}).Info("Successfully processed 302.AI request")

	// 如果响应成功且有Output URL，下载并上传到OSS
	if response != nil && response.Output != "" && (response.Status == "completed" || response.Status == "succeeded") {
		s.logger.WithFields(map[string]interface{}{
			"response_status": response.Status,
			"output_url":      response.Output,
			"s3_enabled":      s.s3Service.IsEnabled(),
		}).Info("Checking if should upload to OSS")

		if s.s3Service.IsEnabled() {
			s.logger.WithFields(map[string]interface{}{
				"original_url": response.Output,
			}).Info("Starting upload to OSS")

			newURL, err := s.downloadAndUploadToOSS(ctx, response.Output)
			if err != nil {
				s.logger.WithFields(map[string]interface{}{
					"error":        err.Error(),
					"original_url": response.Output,
				}).Warn("Failed to upload output to OSS, using original URL")
			} else {
				s.logger.WithFields(map[string]interface{}{
					"original_url": response.Output,
					"new_url":      newURL,
				}).Info("Successfully uploaded output to OSS")
				response.Output = newURL
			}
		} else {
			s.logger.Info("S3 service is not enabled, skipping OSS upload")
		}
	} else {
		var status string
		if response != nil {
			status = response.Status
		}
		s.logger.WithFields(map[string]interface{}{
			"response_nil": response == nil,
			"output_empty": response != nil && response.Output == "",
			"status":       status,
		}).Info("Skipping OSS upload - conditions not met")
	}

	return response, nil
}

// downloadAndUploadToOSS 下载URL内容并上传到OSS
func (s *ai302ServiceImpl) downloadAndUploadToOSS(ctx context.Context, url string) (string, error) {
	s.logger.WithFields(map[string]interface{}{
		"url": url,
	}).Info("Starting download from URL")

	// 下载文件
	resp, err := http.Get(url)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"url":   url,
			"error": err.Error(),
		}).Error("Failed to download file from URL")
		return "", fmt.Errorf("failed to download file from URL: %w", err)
	}
	defer resp.Body.Close()

	s.logger.WithFields(map[string]interface{}{
		"url":            url,
		"status_code":    resp.StatusCode,
		"content_type":   resp.Header.Get("Content-Type"),
		"content_length": resp.Header.Get("Content-Length"),
	}).Info("Download response received")

	if resp.StatusCode != http.StatusOK {
		s.logger.WithFields(map[string]interface{}{
			"url":         url,
			"status_code": resp.StatusCode,
		}).Error("Download failed with non-200 status code")
		return "", fmt.Errorf("failed to download file, status code: %d", resp.StatusCode)
	}

	// 获取文件扩展名
	filename := s.extractFilenameFromURL(url)
	if filename == "" {
		filename = "ai302_output.png" // 默认文件名
	}

	// 获取Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png" // 默认为PNG图片
	}

	s.logger.WithFields(map[string]interface{}{
		"filename":     filename,
		"content_type": contentType,
	}).Info("Preparing to upload to OSS")

	// 读取响应体到内存中，避免流被重复读取的问题
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to read response body")
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"actual_size":    len(imageData),
		"content_length": resp.Header.Get("Content-Length"),
	}).Info("Read image data from response")

	// 创建新的读取器用于上传
	imageReader := bytes.NewReader(imageData)

	// 上传到S3
	result, err := s.s3Service.UploadFile(ctx, filename, contentType, imageReader)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"filename":     filename,
			"content_type": contentType,
			"error":        err.Error(),
		}).Error("Failed to upload file to OSS")
		return "", fmt.Errorf("failed to upload file to OSS: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"filename":  filename,
		"oss_url":   result.URL,
		"file_size": result.Size,
	}).Info("Successfully uploaded file to OSS")

	return result.URL, nil
}

// extractFilenameFromURL 从URL中提取文件名
func (s *ai302ServiceImpl) extractFilenameFromURL(url string) string {
	// 从URL中提取文件名
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		// 如果文件名包含查询参数，去掉它们
		if idx := strings.Index(filename, "?"); idx != -1 {
			filename = filename[:idx]
		}
		// 如果没有扩展名，添加.png
		if filepath.Ext(filename) == "" {
			filename += ".png"
		}
		return filename
	}
	return ""
}
