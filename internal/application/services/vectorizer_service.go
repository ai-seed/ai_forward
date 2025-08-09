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

// VectorizerService Vectorizer服务接口
type VectorizerService interface {
	// Vectorize 矢量化图片
	Vectorize(ctx context.Context, userID, apiKeyID int64, request *clients.VectorizerRequest) (*clients.VectorizerResponse, error)
}

// vectorizerServiceImpl Vectorizer服务实现
type vectorizerServiceImpl struct {
	vectorizerClient         clients.VectorizerClient
	providerRepo             repositories.ProviderRepository
	modelRepo                repositories.ModelRepository
	apiKeyRepo               repositories.APIKeyRepository
	userRepo                 repositories.UserRepository
	providerModelSupportRepo repositories.ProviderModelSupportRepository
	billingService           BillingService
	usageLogService          UsageLogService
	logger                   logger.Logger
}

// NewVectorizerService 创建Vectorizer服务
func NewVectorizerService(
	vectorizerClient clients.VectorizerClient,
	providerRepo repositories.ProviderRepository,
	modelRepo repositories.ModelRepository,
	apiKeyRepo repositories.APIKeyRepository,
	userRepo repositories.UserRepository,
	providerModelSupportRepo repositories.ProviderModelSupportRepository,
	billingService BillingService,
	usageLogService UsageLogService,
	logger logger.Logger,
) VectorizerService {
	return &vectorizerServiceImpl{
		vectorizerClient:         vectorizerClient,
		providerRepo:             providerRepo,
		modelRepo:                modelRepo,
		apiKeyRepo:               apiKeyRepo,
		userRepo:                 userRepo,
		providerModelSupportRepo: providerModelSupportRepo,
		billingService:           billingService,
		usageLogService:          usageLogService,
		logger:                   logger,
	}
}

// Vectorize 矢量化图片
func (s *vectorizerServiceImpl) Vectorize(ctx context.Context, userID, apiKeyID int64, request *clients.VectorizerRequest) (*clients.VectorizerResponse, error) {
	return s.processVectorizerRequestWithModel(ctx, userID, apiKeyID, "vectorizer", "/vectorizer/api/v1/vectorize", func(provider *entities.Provider) (*clients.VectorizerResponse, error) {
		return s.vectorizerClient.Vectorize(ctx, provider, request)
	})
}

// processVectorizerRequestWithModel 处理Vectorizer请求的通用方法
func (s *vectorizerServiceImpl) processVectorizerRequestWithModel(
	ctx context.Context,
	userID, apiKeyID int64,
	modelSlug, endpoint string,
	clientFunc func(*entities.Provider) (*clients.VectorizerResponse, error),
) (*clients.VectorizerResponse, error) {
	startTime := time.Now()

	// 查询模型信息
	model, err := s.modelRepo.GetBySlug(ctx, modelSlug)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"model_slug": modelSlug,
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("Failed to get model")
		return nil, fmt.Errorf("model not found: %s", modelSlug)
	}

	// 通过ProviderModelSupport查询实际的服务提供商信息
	// 注意：model.ModelProviderID指向的是模型厂商，我们需要通过ProviderModelSupport来找到实际的服务提供商
	supportInfos, err := s.providerModelSupportRepo.GetSupportingProviders(ctx, modelSlug)
	if err != nil || len(supportInfos) == 0 {
		s.logger.WithFields(map[string]interface{}{
			"error":      err,
			"model_slug": modelSlug,
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("Failed to get supported providers for model")
		return nil, fmt.Errorf("no available providers for model: %s", modelSlug)
	}

	// 选择第一个可用的提供商（按优先级排序）
	var provider *entities.Provider
	for _, supportInfo := range supportInfos {
		if supportInfo.IsAvailable() {
			provider = supportInfo.Provider
			break
		}
	}

	if provider == nil {
		s.logger.WithFields(map[string]interface{}{
			"model_slug": modelSlug,
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("No available providers for model")
		return nil, fmt.Errorf("no available providers for model: %s", modelSlug)
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":       userID,
		"api_key_id":    apiKeyID,
		"model_slug":    modelSlug,
		"provider_name": provider.Name,
		"endpoint":      endpoint,
	}).Info("Processing vectorizer request")

	// 调用客户端
	response, err := clientFunc(provider)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":         err.Error(),
			"user_id":       userID,
			"api_key_id":    apiKeyID,
			"model_slug":    modelSlug,
			"provider_name": provider.Name,
			"endpoint":      endpoint,
		}).Error("Failed to call vectorizer client")
		return nil, fmt.Errorf("vectorizer request failed: %w", err)
	}

	// 计算处理时间
	duration := time.Since(startTime)

	// 计算费用 - 矢量化按次计费
	cost, err := s.billingService.CalculateCost(ctx, model.ID, 0, 0) // 0 input tokens, 0 output tokens
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"model_id":   model.ID,
		}).Error("Failed to calculate cost")
		return nil, fmt.Errorf("failed to calculate cost: %w", err)
	}

	// 设置成本信息到响应，供billing中间件使用
	response.Cost = &clients.VectorizerCost{
		TotalCost: cost,
		Currency:  "USD",
	}
	response.ProviderID = provider.ID
	
	s.logger.WithFields(map[string]interface{}{
		"cost":        cost,
		"provider_id": provider.ID,
		"user_id":     userID,
		"api_key_id":  apiKeyID,
	}).Info("Vectorizer request completed, billing will be handled by middleware")

	// 更新API密钥最后使用时间
	err = s.apiKeyRepo.UpdateLastUsed(ctx, apiKeyID)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"api_key_id": apiKeyID,
		}).Error("Failed to update API key last used time")
		// 不返回错误，因为主要功能已完成
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":       userID,
		"api_key_id":    apiKeyID,
		"model_slug":    modelSlug,
		"provider_name": provider.Name,
		"cost":          cost,
		"duration":      duration,
		"svg_length":    len(response.SVGData),
	}).Info("Vectorizer request completed successfully")

	return response, nil
}
