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
	// 查找Stability.ai提供商
	providers, err := s.providerRepo.GetActiveProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}

	var stabilityProvider *entities.Provider
	for _, provider := range providers {
		if provider.Slug == "stability" || provider.Name == "Stability.ai" {
			stabilityProvider = provider
			break
		}
	}

	if stabilityProvider == nil {
		return nil, fmt.Errorf("Stability.ai provider not found or not active")
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":     userID,
		"api_key_id":  apiKeyID,
		"provider_id": stabilityProvider.ID,
		"provider":    stabilityProvider.Name,
	}).Info("Sending request to Stability.ai")

	// 调用Stability.ai API
	response, err := s.stabilityClient.TextToImage(ctx, stabilityProvider, request)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":       err.Error(),
			"user_id":     userID,
			"api_key_id":  apiKeyID,
			"provider_id": stabilityProvider.ID,
		}).Error("Failed to call Stability.ai API")
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	// 记录使用日志
	go func() {
		logCtx := context.Background()

		// 计算成本（假设每次生成0.01 PTC）
		cost := 0.01 * float64(request.Samples)

		usageLog := &entities.UsageLog{
			UserID:       userID,
			APIKeyID:     apiKeyID,
			ProviderID:   stabilityProvider.ID,
			ModelID:      1, // 需要从数据库获取实际的模型ID
			RequestID:    fmt.Sprintf("stability-%d", time.Now().UnixNano()),
			RequestType:  entities.RequestTypeAPI,
			Method:       "POST",
			Endpoint:     "/sd/v1/generation/stable-diffusion-xl-1024-v1-0/text-to-image",
			InputTokens:  0, // 图像生成不计算token
			OutputTokens: 0,
			TotalTokens:  0,
			DurationMs:   100, // 假设值
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

		// 处理计费
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
	}).Info("Successfully generated image with Stability.ai")

	return response, nil
}

// processStabilityRequest 通用的Stability.ai请求处理方法
func (s *stabilityServiceImpl) processStabilityRequest(ctx context.Context, userID, apiKeyID int64, modelName, endpoint string, cost float64, clientFunc func(*entities.Provider) (*clients.StabilityImageResponse, error)) (*clients.StabilityImageResponse, error) {
	// 查找Stability.ai提供商
	providers, err := s.providerRepo.GetActiveProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}

	var stabilityProvider *entities.Provider
	for _, provider := range providers {
		if provider.Slug == "stability" || provider.Name == "Stability.ai" {
			stabilityProvider = provider
			break
		}
	}

	if stabilityProvider == nil {
		return nil, fmt.Errorf("Stability.ai provider not found or not active")
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":     userID,
		"api_key_id":  apiKeyID,
		"provider_id": stabilityProvider.ID,
		"model":       modelName,
		"endpoint":    endpoint,
	}).Info("Sending request to Stability.ai")

	// 调用客户端方法
	response, err := clientFunc(stabilityProvider)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":       err.Error(),
			"user_id":     userID,
			"api_key_id":  apiKeyID,
			"provider_id": stabilityProvider.ID,
			"endpoint":    endpoint,
		}).Error("Failed to call Stability.ai API")
		return nil, fmt.Errorf("failed to process request: %w", err)
	}

	// 记录使用日志和计费
	go func() {
		logCtx := context.Background()

		usageLog := &entities.UsageLog{
			UserID:       userID,
			APIKeyID:     apiKeyID,
			ProviderID:   stabilityProvider.ID,
			ModelID:      1, // 需要从数据库获取实际的模型ID
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
	return s.processStabilityRequest(ctx, userID, apiKeyID, "sd2", "/v2beta/stable-image/generate/sd", 0.02, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD2(ctx, provider, request)
	})
}

// GenerateSD3 SD3图片生成
func (s *stabilityServiceImpl) GenerateSD3(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "sd3", "/v2beta/stable-image/generate/sd3", 0.03, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD3(ctx, provider, request)
	})
}

// GenerateSD3Ultra SD3 Ultra图片生成
func (s *stabilityServiceImpl) GenerateSD3Ultra(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "sd3-ultra", "/v2beta/stable-image/generate/ultra", 0.08, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD3Ultra(ctx, provider, request)
	})
}

// GenerateSD35Large SD3.5 Large图片生成
func (s *stabilityServiceImpl) GenerateSD35Large(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "sd3.5-large", "/v2beta/stable-image/generate/sd3-large", 0.04, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD35Large(ctx, provider, request)
	})
}

// GenerateSD35Medium SD3.5 Medium图片生成
func (s *stabilityServiceImpl) GenerateSD35Medium(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "sd3.5-medium", "/v2beta/stable-image/generate/sd3-medium", 0.035, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.GenerateSD35Medium(ctx, provider, request)
	})
}

// ImageToImageSD3 SD3图生图
func (s *stabilityServiceImpl) ImageToImageSD3(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityImageToImageRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "sd3-i2i", "/v2beta/stable-image/control/sd3", 0.03, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ImageToImageSD3(ctx, provider, request)
	})
}

// ImageToImageSD35Large SD3.5 Large图生图
func (s *stabilityServiceImpl) ImageToImageSD35Large(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityImageToImageRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "sd3.5-large-i2i", "/v2beta/stable-image/control/sd3-large", 0.04, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ImageToImageSD35Large(ctx, provider, request)
	})
}

// ImageToImageSD35Medium SD3.5 Medium图生图
func (s *stabilityServiceImpl) ImageToImageSD35Medium(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityImageToImageRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "sd3.5-medium-i2i", "/v2beta/stable-image/control/sd3-medium", 0.035, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ImageToImageSD35Medium(ctx, provider, request)
	})
}

// FastUpscale 快速图片放大
func (s *stabilityServiceImpl) FastUpscale(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityUpscaleRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "fast-upscale", "/v2beta/stable-image/upscale/fast", 0.01, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.FastUpscale(ctx, provider, request)
	})
}

// CreativeUpscale 创意图片放大
func (s *stabilityServiceImpl) CreativeUpscale(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityUpscaleRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "creative-upscale", "/v2beta/stable-image/upscale/creative", 0.025, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.CreativeUpscale(ctx, provider, request)
	})
}

// ConservativeUpscale 保守图片放大
func (s *stabilityServiceImpl) ConservativeUpscale(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityUpscaleRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "conservative-upscale", "/v2beta/stable-image/upscale/conservative", 0.02, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ConservativeUpscale(ctx, provider, request)
	})
}

// FetchCreativeUpscale 获取创意放大结果
func (s *stabilityServiceImpl) FetchCreativeUpscale(ctx context.Context, userID, apiKeyID int64, requestID string) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "fetch-creative-upscale", "/v2beta/stable-image/upscale/creative/result", 0.0, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.FetchCreativeUpscale(ctx, provider, requestID)
	})
}

// Erase 物体消除
func (s *stabilityServiceImpl) Erase(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityEraseRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "erase", "/v2beta/stable-image/edit/erase", 0.01, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Erase(ctx, provider, request)
	})
}

// Inpaint 图片修改
func (s *stabilityServiceImpl) Inpaint(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityInpaintRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "inpaint", "/v2beta/stable-image/edit/inpaint", 0.015, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Inpaint(ctx, provider, request)
	})
}

// Outpaint 图片扩展
func (s *stabilityServiceImpl) Outpaint(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityOutpaintRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "outpaint", "/v2beta/stable-image/edit/outpaint", 0.015, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Outpaint(ctx, provider, request)
	})
}

// SearchAndReplace 内容替换
func (s *stabilityServiceImpl) SearchAndReplace(ctx context.Context, userID, apiKeyID int64, request *clients.StabilitySearchReplaceRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "search-replace", "/v2beta/stable-image/edit/search-and-replace", 0.02, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.SearchAndReplace(ctx, provider, request)
	})
}

// SearchAndRecolor 内容重着色
func (s *stabilityServiceImpl) SearchAndRecolor(ctx context.Context, userID, apiKeyID int64, request *clients.StabilitySearchRecolorRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "search-recolor", "/v2beta/stable-image/edit/search-and-recolor", 0.02, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.SearchAndRecolor(ctx, provider, request)
	})
}

// RemoveBackground 背景消除
func (s *stabilityServiceImpl) RemoveBackground(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityRemoveBgRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "remove-bg", "/v2beta/stable-image/edit/remove-background", 0.01, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.RemoveBackground(ctx, provider, request)
	})
}

// Sketch 草图转图片
func (s *stabilityServiceImpl) Sketch(ctx context.Context, userID, apiKeyID int64, request *clients.StabilitySketchRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "sketch", "/v2beta/stable-image/control/sketch", 0.02, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Sketch(ctx, provider, request)
	})
}

// Structure 以图生图
func (s *stabilityServiceImpl) Structure(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityStructureRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "structure", "/v2beta/stable-image/control/structure", 0.02, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Structure(ctx, provider, request)
	})
}

// Style 风格一致性
func (s *stabilityServiceImpl) Style(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityStyleRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "style", "/v2beta/stable-image/control/style", 0.02, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.Style(ctx, provider, request)
	})
}

// StyleTransfer 风格迁移
func (s *stabilityServiceImpl) StyleTransfer(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityStyleTransferRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "style-transfer", "/v2beta/stable-image/edit/style-transfer", 0.025, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.StyleTransfer(ctx, provider, request)
	})
}

// ReplaceBackground 更换背景
func (s *stabilityServiceImpl) ReplaceBackground(ctx context.Context, userID, apiKeyID int64, request *clients.StabilityReplaceBgRequest) (*clients.StabilityImageResponse, error) {
	return s.processStabilityRequest(ctx, userID, apiKeyID, "replace-bg", "/v2beta/stable-image/edit/replace-background", 0.02, func(provider *entities.Provider) (*clients.StabilityImageResponse, error) {
		return s.stabilityClient.ReplaceBackground(ctx, provider, request)
	})
}
