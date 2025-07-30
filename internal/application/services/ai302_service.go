package services

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"
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
	logger logger.Logger,
) AI302Service {
	return &ai302ServiceImpl{
		ai302Client:              ai302Client,
		providerRepo:             providerRepo,
		modelRepo:                modelRepo,
		modelPricingRepo:         modelPricingRepo,
		providerModelSupportRepo: providerModelSupportRepo,
		usageLogRepo:             usageLogRepo,
		logger:                   logger,
	}
}

// Upscale 图片放大
func (s *ai302ServiceImpl) Upscale(ctx context.Context, userID, apiKeyID int64, request *clients.AI302UpscaleRequest) (*clients.AI302UpscaleResponse, error) {
	return s.processAI302RequestWithModel(ctx, userID, apiKeyID, "upscale", "/302/submit/upscale", func(provider *entities.Provider) (*clients.AI302UpscaleResponse, error) {
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

	// 记录使用情况
	duration := time.Since(startTime)
	usageLog := &entities.UsageLog{
		UserID:       userID,
		APIKeyID:     apiKeyID,
		ModelID:      model.ID,
		ProviderID:   provider.ID,
		RequestID:    response.ID,
		RequestType:  entities.RequestTypeAPI,
		Method:       "POST",
		Endpoint:     endpoint,
		InputTokens:  0, // 图片处理不计算token
		OutputTokens: 0,
		TotalTokens:  0,
		DurationMs:   int(duration.Milliseconds()),
		StatusCode:   200, // 成功状态
		Cost:         cost,
		CreatedAt:    time.Now(),
	}

	if err := s.usageLogRepo.Create(ctx, usageLog); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		}).Warn("Failed to record usage")
		// 不返回错误，因为主要功能已完成
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"api_key_id": apiKeyID,
		"model":      modelSlug,
		"duration":   duration.String(),
		"status":     response.Status,
	}).Info("Successfully processed 302.AI request")

	return response, nil
}
