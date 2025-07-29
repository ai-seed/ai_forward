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

// StabilityService Stability.ai服务接口
type StabilityService interface {
	// TextToImage 文本生成图像 (V1)
	TextToImage(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityTextToImageRequest) (*clients.StabilityImageResponse, error)

	// 图片生成接口
	GenerateSD2(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error)
	GenerateSD3(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error)
	GenerateSD3Ultra(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error)
	GenerateSD35Large(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error)
	GenerateSD35Medium(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error)

	// 图生图接口
	ImageToImageSD3(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityImageToImageRequest) (*clients.StabilityImageResponse, error)
	ImageToImageSD35Large(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityImageToImageRequest) (*clients.StabilityImageResponse, error)
	ImageToImageSD35Medium(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityImageToImageRequest) (*clients.StabilityImageResponse, error)

	// 图片处理接口
	FastUpscale(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityUpscaleRequest) (*clients.StabilityImageResponse, error)
	CreativeUpscale(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityUpscaleRequest) (*clients.StabilityImageResponse, error)
	ConservativeUpscale(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityUpscaleRequest) (*clients.StabilityImageResponse, error)
	FetchCreativeUpscale(ctx context.Context, userID, apiKeyID int64, requestID string) (*clients.StabilityImageResponse, error)

	// 图片编辑接口
	Erase(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityEraseRequest) (*clients.StabilityImageResponse, error)
	Inpaint(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityInpaintRequest) (*clients.StabilityImageResponse, error)
	Outpaint(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityOutpaintRequest) (*clients.StabilityImageResponse, error)
	SearchAndReplace(ctx context.Context, userID, apiKeyID int64, request *clients.StabilitySearchReplaceRequest) (*clients.StabilityImageResponse, error)
	SearchAndRecolor(ctx context.Context, userID, apiKeyID int64, request *clients.StabilitySearchRecolorRequest) (*clients.StabilityImageResponse, error)
	RemoveBackground(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityRemoveBgRequest) (*clients.StabilityImageResponse, error)

	// 风格和结构接口
	Sketch(ctx context.Context, userID, apiKeyID int64, request *clients.StabilitySketchRequest) (*clients.StabilityImageResponse, error)
	Structure(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityStructureRequest) (*clients.StabilityImageResponse, error)
	Style(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityStyleRequest) (*clients.StabilityImageResponse, error)
	StyleTransfer(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityStyleTransferRequest) (*clients.StabilityImageResponse, error)
	ReplaceBackground(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityReplaceBgRequest) (*clients.StabilityImageResponse, error)
}

// stabilityServiceImpl Stability.ai服务实现
type stabilityServiceImpl struct {
	stabilityClient clients.StabilityClient
	providerRepo    repositories.ProviderRepository
	modelRepo       repositories.ModelRepository
	apiKeyRepo      repositories.APIKeyRepository
	userRepo        repositories.UserRepository
	billingService  BillingService
	usageLogService UsageLogService
	logger          logger.Logger
}

// NewStabilityService 创建Stability.ai服务
func NewStabilityService(
	stabilityClient clients.StabilityClient,
	providerRepo repositories.ProviderRepository,
	modelRepo repositories.ModelRepository,
	apiKeyRepo repositories.APIKeyRepository,
	userRepo repositories.UserRepository,
	billingService BillingService,
	usageLogService UsageLogService,
	logger logger.Logger,
) StabilityService {
	return &stabilityServiceImpl{
		stabilityClient: stabilityClient,
		providerRepo:    providerRepo,
		modelRepo:       modelRepo,
		apiKeyRepo:      apiKeyRepo,
		userRepo:        userRepo,
		billingService:  billingService,
		usageLogService: usageLogService,
		logger:          logger,
	}
}

// TextToImage 文本生成图像
func (s *stabilityServiceImpl) TextToImage(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityTextToImageRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-xl-1024-v1-0", "/v1/generation/stable-diffusion-xl-1024-v1-0/text-to-image", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.TextToImage(ctx, provider, request)
	})
}

// processStabilityRequestWithModel 通用的Stability.ai请求处理方法（使用模型slug）
func (s *stabilityServiceImpl) processStabilityRequestWithModel(ctx context.Context, userID, apiKeyID int64, modelSlug, endpoint string, clientFunc func(*entities.Provider) (*clients.StabilityImageResponse, error)) (*clients.StabilityImageResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"api_key_id": apiKeyID,
		"model_slug": modelSlug,
		"endpoint":   endpoint,
	}).Info("开始处理Stability.ai请求")

	// 查找Stability.ai提供商
	providers, err := s.providerRepo.GetActiveProviders(ctx)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("获取提供商列表失败")
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"provider_count": len(providers),
	}).Info("获取到提供商列表")

	var stabilityProvider *entities.Provider
	for _, provider := range providers {
		s.logger.WithFields(map[string]interface{}{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
			"provider_slug": provider.Slug,
			"provider_url":  provider.BaseURL,
		}).Debug("检查提供商")

		if provider.ID == 1 || provider.Name == "Stability.ai" {
			stabilityProvider = provider
			break
		}
	}

	if stabilityProvider == nil {
		s.logger.Error("未找到Stability.ai提供商")
		return nil, fmt.Errorf("Stability.ai provider not found or not active")
	}

	s.logger.WithFields(map[string]interface{}{
		"provider_id":   stabilityProvider.ID,
		"provider_name": stabilityProvider.Name,
		"provider_url":  stabilityProvider.BaseURL,
	}).Info("找到Stability.ai提供商")

	// 根据模型slug获取模型信息
	model, err := s.modelRepo.GetBySlug(ctx, modelSlug)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"model_slug": modelSlug,
			"error":      err.Error(),
		}).Error("获取模型信息失败")
		return nil, fmt.Errorf("failed to get model %s: %w", modelSlug, err)
	}

	s.logger.WithFields(map[string]interface{}{
		"model_id":          model.ID,
		"model_slug":        model.Slug,
		"model_name":        model.Name,
		"model_provider_id": model.ProviderID,
		"model_available":   model.IsAvailable(),
	}).Info("获取到模型信息")

	if !model.IsAvailable() {
		s.logger.WithFields(map[string]interface{}{
			"model_slug": modelSlug,
		}).Error("模型不可用")
		return nil, fmt.Errorf("model %s is not available", modelSlug)
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":      userID,
		"api_key_id":   apiKeyID,
		"provider_id":  stabilityProvider.ID,
		"provider_url": stabilityProvider.BaseURL,
		"model_id":     model.ID,
		"model_slug":   model.Slug,
		"endpoint":     endpoint,
	}).Info("准备发送请求到Stability.ai")

	// 调用客户端方法
	response, err := clientFunc(stabilityProvider)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":        err.Error(),
			"user_id":      userID,
			"api_key_id":   apiKeyID,
			"provider_id":  stabilityProvider.ID,
			"provider_url": stabilityProvider.BaseURL,
			"model_slug":   model.Slug,
			"endpoint":     endpoint,
		}).Error("调用Stability.ai API失败")
		return nil, fmt.Errorf("failed to process request: %w", err)
	}

	// 记录使用日志和计费
	go func() {
		logCtx := context.Background()

		// 计算实际成本
		cost, err := s.billingService.CalculateCost(logCtx, model.ID, 0, 0)
		if err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":    err.Error(),
				"model_id": model.ID,
			}).Error("Failed to calculate cost")
			cost = 0.01 // 默认成本
		}

		usageLog := &entities.UsageLog{
			UserID:       userID,
			APIKeyID:     apiKeyID,
			ProviderID:   stabilityProvider.ID,
			ModelID:      model.ID,
			RequestID:    fmt.Sprintf("stability-%d", time.Now().UnixNano()),
			RequestType:  entities.RequestTypeAPI,
			Method:       "POST",
			Endpoint:     endpoint,
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
			DurationMs:   100,
			StatusCode:   200,
			Cost:         cost,
		}

		if err := s.usageLogService.CreateUsageLog(logCtx, usageLog); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":      err.Error(),
				"user_id":    userID,
				"api_key_id": apiKeyID,
			}).Error("Failed to create usage log")
			return
		}

		if err := s.billingService.ProcessBilling(logCtx, usageLog); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":   err.Error(),
				"user_id": userID,
				"cost":    cost,
			}).Error("Failed to process billing")
		}
	}()

	s.logger.WithFields(map[string]interface{}{
		"user_id":         userID,
		"api_key_id":      apiKeyID,
		"provider_id":     stabilityProvider.ID,
		"artifacts_count": len(response.Artifacts),
		"endpoint":        endpoint,
	}).Info("Successfully processed Stability.ai request")

	return response, nil
}

// GenerateSD2 SD2图片生成
func (s *stabilityServiceImpl) GenerateSD2(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-2", "/v2beta/stable-image/generate/sd", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD2(ctx, provider, request)
	})
}

// GenerateSD3 SD3图片生成
func (s *stabilityServiceImpl) GenerateSD3(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-3", "/v2beta/stable-image/generate/sd3", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD3(ctx, provider, request)
	})
}

// GenerateSD3Ultra SD3 Ultra图片生成
func (s *stabilityServiceImpl) GenerateSD3Ultra(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-3-ultra", "/v2beta/stable-image/generate/ultra", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD3Ultra(ctx, provider, request)
	})
}

// GenerateSD35Large SD3.5 Large图片生成
func (s *stabilityServiceImpl) GenerateSD35Large(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-3.5-large", "/v2beta/stable-image/generate/sd3-large", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD35Large(ctx, provider, request)
	})
}

// GenerateSD35Medium SD3.5 Medium图片生成
func (s *stabilityServiceImpl) GenerateSD35Medium(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-3.5-medium", "/v2beta/stable-image/generate/sd3-medium", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD35Medium(ctx, provider, request)
	})
}

// ImageToImageSD3 SD3图生图
func (s *stabilityServiceImpl) ImageToImageSD3(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityImageToImageRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-3-img2img", "/v2beta/stable-image/control/sd3", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ImageToImageSD3(ctx, provider, request)
	})
}

// ImageToImageSD35Large SD3.5 Large图生图
func (s *stabilityServiceImpl) ImageToImageSD35Large(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityImageToImageRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-3.5-large-img2img", "/v2beta/stable-image/control/sd3-large", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ImageToImageSD35Large(ctx, provider, request)
	})
}

// ImageToImageSD35Medium SD3.5 Medium图生图
func (s *stabilityServiceImpl) ImageToImageSD35Medium(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityImageToImageRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-3.5-medium-img2img", "/v2beta/stable-image/control/sd3-medium", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ImageToImageSD35Medium(ctx, provider, request)
	})
}

// FastUpscale 快速图片放大
func (s *stabilityServiceImpl) FastUpscale(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityUpscaleRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-upscale-fast", "/v2beta/stable-image/upscale/fast", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.FastUpscale(ctx, provider, request)
	})
}

// CreativeUpscale 创意图片放大
func (s *stabilityServiceImpl) CreativeUpscale(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityUpscaleRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-upscale-creative", "/v2beta/stable-image/upscale/creative", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.CreativeUpscale(ctx, provider, request)
	})
}

// ConservativeUpscale 保守图片放大
func (s *stabilityServiceImpl) ConservativeUpscale(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityUpscaleRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-upscale-conservative", "/v2beta/stable-image/upscale/conservative", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ConservativeUpscale(ctx, provider, request)
	})
}

// FetchCreativeUpscale 获取创意放大结果
func (s *stabilityServiceImpl) FetchCreativeUpscale(ctx context.Context, userID, apiKeyID int64, requestID string) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-upscale-creative-fetch", "/v2beta/stable-image/upscale/creative/result", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.FetchCreativeUpscale(ctx, provider, requestID)
	})
}

// Erase 物体消除
func (s *stabilityServiceImpl) Erase(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityEraseRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-erase", "/v2beta/stable-image/edit/erase", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Erase(ctx, provider, request)
	})
}

// Inpaint 图片修改
func (s *stabilityServiceImpl) Inpaint(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityInpaintRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-inpaint", "/v2beta/stable-image/edit/inpaint", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Inpaint(ctx, provider, request)
	})
}

// Outpaint 图片扩展
func (s *stabilityServiceImpl) Outpaint(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityOutpaintRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-outpaint", "/v2beta/stable-image/edit/outpaint", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Outpaint(ctx, provider, request)
	})
}

// SearchAndReplace 内容替换
func (s *stabilityServiceImpl) SearchAndReplace(ctx context.Context, userID, apiKeyID int64, request *clients.StabilitySearchReplaceRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-search-replace", "/v2beta/stable-image/edit/search-and-replace", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.SearchAndReplace(ctx, provider, request)
	})
}

// SearchAndRecolor 内容重着色
func (s *stabilityServiceImpl) SearchAndRecolor(ctx context.Context, userID, apiKeyID int64, request *clients.StabilitySearchRecolorRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-search-recolor", "/v2beta/stable-image/edit/search-and-recolor", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.SearchAndRecolor(ctx, provider, request)
	})
}

// RemoveBackground 背景消除
func (s *stabilityServiceImpl) RemoveBackground(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityRemoveBgRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-remove-background", "/v2beta/stable-image/edit/remove-background", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.RemoveBackground(ctx, provider, request)
	})
}

// Sketch 草图转图片
func (s *stabilityServiceImpl) Sketch(ctx context.Context, userID, apiKeyID int64, request *clients.StabilitySketchRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-sketch", "/v2beta/stable-image/control/sketch", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Sketch(ctx, provider, request)
	})
}

// Structure 以图生图
func (s *stabilityServiceImpl) Structure(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityStructureRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-structure", "/v2beta/stable-image/control/structure", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Structure(ctx, provider, request)
	})
}

// Style 风格一致性
func (s *stabilityServiceImpl) Style(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityStyleRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-style", "/v2beta/stable-image/control/style", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Style(ctx, provider, request)
	})
}

// StyleTransfer 风格迁移
func (s *stabilityServiceImpl) StyleTransfer(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityStyleTransferRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-style-transfer", "/v2beta/stable-image/edit/style-transfer", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.StyleTransfer(ctx, provider, request)
	})
}

// ReplaceBackground 更换背景
func (s *stabilityServiceImpl) ReplaceBackground(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityReplaceBgRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequestWithModel(ctx, userID, apiKeyID, "stable-diffusion-replace-background", "/v2beta/stable-image/edit/replace-background", func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ReplaceBackground(ctx, provider, request)
	})
}
