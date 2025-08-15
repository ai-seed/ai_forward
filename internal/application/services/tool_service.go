package services

import (
	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/utils"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ToolService 工具服务
type ToolService struct {
	toolRepo          repositories.ToolRepository
	apiKeyRepo        repositories.APIKeyRepository
	modelRepo         repositories.ModelRepository
	modelProviderRepo repositories.ModelProviderRepository
	modelPricingRepo  repositories.ModelPricingRepository
}

// NewToolService 创建工具服务
func NewToolService(toolRepo repositories.ToolRepository, apiKeyRepo repositories.APIKeyRepository, modelRepo repositories.ModelRepository, modelProviderRepo repositories.ModelProviderRepository, modelPricingRepo repositories.ModelPricingRepository) *ToolService {
	return &ToolService{
		toolRepo:          toolRepo,
		apiKeyRepo:        apiKeyRepo,
		modelRepo:         modelRepo,
		modelProviderRepo: modelProviderRepo,
		modelPricingRepo:  modelPricingRepo,
	}
}

// GetTools 获取所有工具模板
func (s *ToolService) GetTools(ctx context.Context) ([]*entities.Tool, error) {
	return s.toolRepo.GetTools(ctx)
}

// GetToolByID 根据ID获取工具模板
func (s *ToolService) GetToolByID(ctx context.Context, id string) (*entities.Tool, error) {
	return s.toolRepo.GetToolByID(ctx, id)
}

// CreateUserToolInstance 创建用户工具实例
func (s *ToolService) CreateUserToolInstance(ctx context.Context, userID int64, req *entities.CreateUserToolInstanceRequest) (*entities.UserToolInstance, error) {
	// 验证工具模板是否存在
	tool, err := s.toolRepo.GetToolByID(ctx, req.ToolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool: %w", err)
	}
	if tool == nil {
		return nil, fmt.Errorf("tool not found")
	}
	fmt.Printf("DEBUG: Tool found: %+v\n", tool)

	// 验证模型是否被工具支持
	modelSupported := false
	for _, model := range tool.SupportedModels {
		if model.ID == req.ModelID {
			modelSupported = true
			break
		}
	}
	if !modelSupported {
		return nil, fmt.Errorf("model %d is not supported by tool %s", req.ModelID, req.ToolID)
	}

	// 验证API Key是否属于用户且有效
	apiKey, err := s.apiKeyRepo.GetByID(ctx, req.APIKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	if apiKey == nil || apiKey.UserID != userID {
		return nil, fmt.Errorf("API key not found or not owned by user")
	}
	if apiKey.Status != "active" {
		return nil, fmt.Errorf("API key is not active")
	}

	// 生成唯一的code
	code, err := s.generateCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	// 创建工具实例
	instance := &entities.UserToolInstance{
		ID:          uuid.New().String(),
		UserID:      userID,
		ToolID:      req.ToolID,
		Name:        req.Name,
		Description: req.Description,
		ModelID:     req.ModelID,
		APIKeyID:    req.APIKeyID,
		IsPublic:    req.IsPublic,
		Code:        code,
		UsageCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 处理配置
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		instance.Config = configJSON
	} else {
		instance.Config = json.RawMessage("{}")
	}

	// 如果是公开工具，生成分享token
	if req.IsPublic {
		shareToken, err := s.generateShareToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate share token: %w", err)
		}
		instance.ShareToken = &shareToken
	}

	// 保存到数据库
	if err := s.toolRepo.CreateUserToolInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to create user tool instance: %w", err)
	}

	// 获取完整的工具实例信息（包含关联数据）
	return s.toolRepo.GetUserToolInstanceByID(ctx, instance.ID)
}

// GetUserToolInstances 获取用户的工具实例列表
func (s *ToolService) GetUserToolInstances(ctx context.Context, userID int64, category string) ([]*entities.UserToolInstance, error) {
	switch category {
	case "my_tools":
		return s.toolRepo.GetUserToolInstancesByUserID(ctx, userID, true)
	case "public":
		return s.toolRepo.GetPublicUserToolInstances(ctx, 100, 0)
	case "shared":
		// 获取用户的公开工具（有分享链接的）
		instances, err := s.toolRepo.GetUserToolInstancesByUserID(ctx, userID, true)
		if err != nil {
			return nil, err
		}
		var sharedInstances []*entities.UserToolInstance
		for _, instance := range instances {
			if instance.ShareToken != nil {
				sharedInstances = append(sharedInstances, instance)
			}
		}
		return sharedInstances, nil
	default:
		// 默认返回用户的所有工具实例
		return s.toolRepo.GetUserToolInstancesByUserID(ctx, userID, true)
	}
}

// GetUserToolInstanceByID 获取用户工具实例详情
func (s *ToolService) GetUserToolInstanceByID(ctx context.Context, id string, userID int64) (*entities.UserToolInstance, error) {
	instance, err := s.toolRepo.GetUserToolInstanceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tool instance: %w", err)
	}
	if instance == nil {
		return nil, fmt.Errorf("tool instance not found")
	}

	// 检查权限：工具所有者或公开工具
	if instance.UserID != userID && !instance.IsPublic {
		return nil, fmt.Errorf("access denied")
	}

	return instance, nil
}

// UpdateUserToolInstance 更新用户工具实例
func (s *ToolService) UpdateUserToolInstance(ctx context.Context, id string, userID int64, req *entities.UpdateUserToolInstanceRequest) (*entities.UserToolInstance, error) {
	// 获取现有工具实例
	instance, err := s.toolRepo.GetUserToolInstanceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tool instance: %w", err)
	}
	if instance == nil {
		return nil, fmt.Errorf("tool instance not found")
	}
	if instance.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// 更新字段
	if req.Name != nil {
		instance.Name = *req.Name
	}
	if req.Description != nil {
		instance.Description = *req.Description
	}
	if req.ModelID != nil {
		// 验证新模型是否被工具支持
		tool, err := s.toolRepo.GetToolByID(ctx, instance.ToolID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tool: %w", err)
		}

		modelSupported := false
		for _, model := range tool.SupportedModels {
			if model.ID == *req.ModelID {
				modelSupported = true
				break
			}
		}
		if !modelSupported {
			return nil, fmt.Errorf("model %d is not supported by tool %s", *req.ModelID, instance.ToolID)
		}

		instance.ModelID = *req.ModelID
	}
	if req.APIKeyID != nil {
		// 验证新API Key是否属于用户且有效
		apiKey, err := s.apiKeyRepo.GetByID(ctx, *req.APIKeyID)
		if err != nil {
			return nil, fmt.Errorf("failed to get API key: %w", err)
		}
		if apiKey == nil || apiKey.UserID != userID {
			return nil, fmt.Errorf("API key not found or not owned by user")
		}
		if apiKey.Status != "active" {
			return nil, fmt.Errorf("API key is not active")
		}

		instance.APIKeyID = *req.APIKeyID
	}
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		instance.Config = configJSON
	}
	if req.IsPublic != nil {
		instance.IsPublic = *req.IsPublic

		// 如果变为公开且没有分享token，生成一个
		if *req.IsPublic && instance.ShareToken == nil {
			shareToken, err := s.generateShareToken()
			if err != nil {
				return nil, fmt.Errorf("failed to generate share token: %w", err)
			}
			instance.ShareToken = &shareToken
		}
		// 如果变为私有，清除分享token
		if !*req.IsPublic {
			instance.ShareToken = nil
		}
	}

	instance.UpdatedAt = time.Now()

	// 保存更新
	if err := s.toolRepo.UpdateUserToolInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to update user tool instance: %w", err)
	}

	// 返回更新后的工具实例信息
	return s.toolRepo.GetUserToolInstanceByID(ctx, id)
}

// DeleteUserToolInstance 删除用户工具实例
func (s *ToolService) DeleteUserToolInstance(ctx context.Context, id string, userID int64) error {
	return s.toolRepo.DeleteUserToolInstance(ctx, id, userID)
}

// GetSharedToolInstance 获取分享的工具实例
func (s *ToolService) GetSharedToolInstance(ctx context.Context, shareToken string) (*entities.UserToolInstance, error) {
	return s.toolRepo.GetUserToolInstanceByShareToken(ctx, shareToken)
}

// IncrementUsageCount 增加工具实例使用次数
func (s *ToolService) IncrementUsageCount(ctx context.Context, instanceID string) error {
	return s.toolRepo.IncrementUsageCount(ctx, instanceID)
}

// GetToolInstanceByCode 通过code获取工具实例信息（用于第三方鉴权）
func (s *ToolService) GetToolInstanceByCode(ctx context.Context, code string) (*entities.ToolInstanceByCodeResponse, error) {
	// 获取工具实例
	instance, err := s.toolRepo.GetUserToolInstanceByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool instance by code: %w", err)
	}
	if instance == nil {
		return nil, fmt.Errorf("tool instance not found")
	}

	// 构建响应
	response := &entities.ToolInstanceByCodeResponse{
		ID:          instance.ID,
		Name:        instance.Name,
		Description: instance.Description,
		UsageCount:  instance.UsageCount,
		CreatedAt:   instance.CreatedAt,
		UpdatedAt:   instance.UpdatedAt,
	}

	// 获取工具配置
	if config, err := instance.GetConfig(); err == nil {
		response.Config = config
	} else {
		response.Config = make(map[string]interface{})
	}

	// 填充API Key信息（不包含敏感数据）
	if instance.APIKey != nil {
		response.APIKeyInfo.ID = instance.APIKey.ID
		if instance.APIKey.Name != nil {
			response.APIKeyInfo.Name = *instance.APIKey.Name
		}
		response.APIKeyInfo.Value = instance.APIKey.Key
		response.APIKeyInfo.Status = string(instance.APIKey.Status)
		// 这里需要获取Provider信息，暂时留空
		response.APIKeyInfo.ProviderName = "" // TODO: 从Provider获取
	}

	// 填充模型信息
	response.ModelInfo.ID = instance.ModelID
	response.ModelInfo.Name = instance.ModelName
	// TODO: 从Model实体获取更多信息
	response.ModelInfo.ProviderName = ""
	response.ModelInfo.Type = ""

	// 填充工具模板信息
	if instance.Tool != nil {
		response.ToolInfo.ID = instance.Tool.ID
		response.ToolInfo.Name = instance.Tool.Name
		response.ToolInfo.Description = instance.Tool.Description
		response.ToolInfo.Category = instance.Tool.Category
		response.ToolInfo.Icon = instance.Tool.Icon
		response.ToolInfo.Color = instance.Tool.Color
		response.ToolInfo.Path = instance.Tool.Path
		response.Type = instance.Tool.Category
	}

	return response, nil
}

// GetAvailableModels 获取可用模型列表（按厂商分组）
func (s *ToolService) GetAvailableModels(ctx context.Context) ([]map[string]interface{}, error) {
	// 获取所有活跃模型（已包含厂商信息）
	models, err := s.modelRepo.GetActiveModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}

	// 收集所有模型ID
	modelIDs := make([]int64, len(models))
	for i, model := range models {
		modelIDs[i] = model.ID
	}

	// 批量获取所有模型的价格信息
	pricingMap, err := s.modelPricingRepo.GetCurrentPricingBatch(ctx, modelIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing batch: %w", err)
	}

	var result []map[string]interface{}
	for _, model := range models {
		displayName := model.Name
		if model.DisplayName != nil {
			displayName = *model.DisplayName
		}

		description := ""
		if model.Description != nil {
			description = *model.Description
		}

		// 构建厂商信息（如果厂商信息存在）
		var providerInfo map[string]interface{}
		if model.ModelProvider != nil {
			providerInfo = map[string]interface{}{
				"id":           model.ModelProvider.ID,
				"name":         model.ModelProvider.Name,
				"display_name": model.ModelProvider.DisplayName,
				"color":        model.ModelProvider.Color,
				"sort_order":   model.ModelProvider.SortOrder,
			}
		} else {
			// 如果厂商信息不存在，提供默认值
			providerInfo = map[string]interface{}{
				"id":           0,
				"name":         "unknown",
				"display_name": "Unknown",
				"color":        "#6B7280",
				"sort_order":   999,
			}
		}

		// 从批量获取的数据中获取模型定价信息
		pricingInfo := s.buildModelPricingInfo(pricingMap[model.ID])

		// 添加调试日志
		fmt.Printf("[DEBUG] GetAvailableModels: model %d (%s) - pricingInfo: %+v\n", model.ID, model.Name, pricingInfo)
		fmt.Printf("[DEBUG] GetAvailableModels: model %d - rate_multiplier value: %v (type: %T)\n",
			model.ID, pricingInfo["rate_multiplier"], pricingInfo["rate_multiplier"])

		result = append(result, map[string]interface{}{
			"id":                 model.ID,
			"name":               model.Name,
			"slug":               model.Slug,
			"display_name":       displayName,
			"description":        description,
			"description_en":     model.DescriptionEN,
			"description_zh":     model.DescriptionZH,
			"description_jp":     model.DescriptionJP,
			"model_type":         model.ModelType,
			"model_type_en":      model.ModelTypeEN,
			"model_type_jp":      model.ModelTypeJP,
			"model_type_zh":      model.ModelTypeZH,
			"provider":           providerInfo,
			"context_length":     model.ContextLength,
			"max_tokens":         model.MaxTokens,
			"supports_streaming": model.SupportsStreaming,
			"supports_functions": model.SupportsFunctions,
			"status":             model.Status,
			"pricing":            pricingInfo["pricing"],
			"rate_multiplier":    pricingInfo["rate_multiplier"],
			"created_at":         model.CreatedAt,
			"updated_at":         model.UpdatedAt,
		})
	}

	return result, nil
}

// GetAvailableModelsWithPagination 获取可用模型列表（分页）
func (s *ToolService) GetAvailableModelsWithPagination(ctx context.Context, pagination *dto.PaginationRequest) (*dto.ListResponseBase, error) {
	// 获取总数
	total, err := s.modelRepo.CountActiveModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active models: %w", err)
	}

	// 计算偏移量
	offset := pagination.GetOffset()
	limit := pagination.GetLimit()

	// 获取分页的模型列表
	models, err := s.modelRepo.GetActiveModelsWithPagination(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get models with pagination: %w", err)
	}

	// 收集所有模型ID
	modelIDs := make([]int64, len(models))
	for i, model := range models {
		modelIDs[i] = model.ID
	}

	// 批量获取所有模型的价格信息
	pricingMap, err := s.modelPricingRepo.GetCurrentPricingBatch(ctx, modelIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing batch: %w", err)
	}

	var result []map[string]interface{}
	for _, model := range models {
		displayName := model.Name
		if model.DisplayName != nil {
			displayName = *model.DisplayName
		}

		description := ""
		if model.Description != nil {
			description = *model.Description
		}

		// 构建厂商信息（如果厂商信息存在）
		var providerInfo map[string]interface{}
		if model.ModelProvider != nil {
			providerInfo = map[string]interface{}{
				"id":           model.ModelProvider.ID,
				"name":         model.ModelProvider.Name,
				"display_name": model.ModelProvider.DisplayName,
				"color":        model.ModelProvider.Color,
				"sort_order":   model.ModelProvider.SortOrder,
			}
		} else {
			// 如果厂商信息不存在，提供默认值
			providerInfo = map[string]interface{}{
				"id":           0,
				"name":         "unknown",
				"display_name": "Unknown",
				"color":        "#6B7280",
				"sort_order":   999,
			}
		}

		// 从批量获取的数据中获取模型定价信息
		pricingInfo := s.buildModelPricingInfo(pricingMap[model.ID])

		result = append(result, map[string]interface{}{
			"id":                 model.ID,
			"name":               model.Name,
			"slug":               model.Slug,
			"display_name":       displayName,
			"description":        description,
			"description_en":     model.DescriptionEN,
			"description_zh":     model.DescriptionZH,
			"description_jp":     model.DescriptionJP,
			"model_type":         model.ModelType,
			"model_type_en":      model.ModelTypeEN,
			"model_type_jp":      model.ModelTypeJP,
			"model_type_zh":      model.ModelTypeZH,
			"provider":           providerInfo,
			"context_length":     model.ContextLength,
			"max_tokens":         model.MaxTokens,
			"supports_streaming": model.SupportsStreaming,
			"supports_functions": model.SupportsFunctions,
			"status":             model.Status,
			"pricing":            pricingInfo["pricing"],
			"rate_multiplier":    pricingInfo["rate_multiplier"],
			"created_at":         model.CreatedAt,
			"updated_at":         model.UpdatedAt,
		})
	}

	// 使用分页助手构建响应
	paginationHelper := utils.NewPaginationHelper()
	return paginationHelper.BuildListResponse(result, total, pagination), nil
}

// GetAvailableModelsWithPaginationAndFilters 获取可用模型列表（分页+筛选）
func (s *ToolService) GetAvailableModelsWithPaginationAndFilters(ctx context.Context, pagination *dto.PaginationRequest, filters map[string]interface{}) (*dto.ListResponseBase, error) {
	// 获取总数（带筛选）
	total, err := s.modelRepo.CountActiveModelsWithFilters(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to count active models with filters: %w", err)
	}

	// 计算偏移量
	offset := pagination.GetOffset()
	limit := pagination.GetLimit()

	// 获取分页的模型列表（带筛选）
	models, err := s.modelRepo.GetActiveModelsWithPaginationAndFilters(ctx, offset, limit, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get models with pagination and filters: %w", err)
	}

	// 收集所有模型ID
	modelIDs := make([]int64, len(models))
	for i, model := range models {
		modelIDs[i] = model.ID
	}

	// 批量获取所有模型的价格信息
	pricingMap, err := s.modelPricingRepo.GetCurrentPricingBatch(ctx, modelIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing batch: %w", err)
	}

	var result []map[string]interface{}
	for _, model := range models {
		displayName := model.Name
		if model.DisplayName != nil {
			displayName = *model.DisplayName
		}

		description := ""
		if model.Description != nil {
			description = *model.Description
		}

		// 构建厂商信息（如果厂商信息存在）
		var providerInfo map[string]interface{}
		if model.ModelProvider != nil {
			providerInfo = map[string]interface{}{
				"id":           model.ModelProvider.ID,
				"name":         model.ModelProvider.Name,
				"display_name": model.ModelProvider.DisplayName,
				"color":        model.ModelProvider.Color,
				"sort_order":   model.ModelProvider.SortOrder,
			}
		} else {
			// 如果厂商信息不存在，提供默认值
			providerInfo = map[string]interface{}{
				"id":           0,
				"name":         "unknown",
				"display_name": "Unknown",
				"color":        "#6B7280",
				"sort_order":   999,
			}
		}

		// 从批量获取的数据中获取模型定价信息
		pricingInfo := s.buildModelPricingInfo(pricingMap[model.ID])

		result = append(result, map[string]interface{}{
			"id":                 model.ID,
			"name":               model.Name,
			"slug":               model.Slug,
			"display_name":       displayName,
			"description":        description,
			"description_en":     model.DescriptionEN,
			"description_zh":     model.DescriptionZH,
			"description_jp":     model.DescriptionJP,
			"model_type":         model.ModelType,
			"model_type_en":      model.ModelTypeEN,
			"model_type_jp":      model.ModelTypeJP,
			"model_type_zh":      model.ModelTypeZH,
			"provider":           providerInfo,
			"context_length":     model.ContextLength,
			"max_tokens":         model.MaxTokens,
			"supports_streaming": model.SupportsStreaming,
			"supports_functions": model.SupportsFunctions,
			"status":             model.Status,
			"pricing":            pricingInfo["pricing"],
			"rate_multiplier":    pricingInfo["rate_multiplier"],
			"created_at":         model.CreatedAt,
			"updated_at":         model.UpdatedAt,
		})
	}

	// 使用分页助手构建响应
	paginationHelper := utils.NewPaginationHelper()
	return paginationHelper.BuildListResponse(result, total, pagination), nil
}

// GetModelCategories 获取模型分类信息
func (s *ToolService) GetModelCategories(ctx context.Context) (map[string]interface{}, error) {
	// 获取所有活跃模型（用于提取厂商信息）
	models, err := s.modelRepo.GetActiveModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active models: %w", err)
	}

	// 提取厂商信息
	providerMap := make(map[int64]map[string]interface{})
	typeSet := make(map[string]bool)

	for _, model := range models {
		// 收集厂商信息
		if model.ModelProvider != nil {
			providerMap[model.ModelProvider.ID] = map[string]interface{}{
				"id":           model.ModelProvider.ID,
				"name":         model.ModelProvider.Name,
				"display_name": model.ModelProvider.DisplayName,
				"color":        model.ModelProvider.Color,
				"sort_order":   model.ModelProvider.SortOrder,
			}
		}

		// 收集模型类型
		if model.ModelType != "" {
			typeSet[string(model.ModelType)] = true
		}
	}

	// 转换厂商信息为数组并按sort_order排序
	var providers []map[string]interface{}
	for _, provider := range providerMap {
		providers = append(providers, provider)
	}

	// 按sort_order排序厂商
	for i := 0; i < len(providers)-1; i++ {
		for j := i + 1; j < len(providers); j++ {
			sortOrderI, okI := providers[i]["sort_order"].(int)
			sortOrderJ, okJ := providers[j]["sort_order"].(int)
			if okI && okJ && sortOrderI > sortOrderJ {
				providers[i], providers[j] = providers[j], providers[i]
			}
		}
	}

	// 转换类型信息为数组
	var types []string
	for modelType := range typeSet {
		types = append(types, modelType)
	}

	return map[string]interface{}{
		"providers": providers,
		"types":     types,
	}, nil
}

// buildModelPricingInfo 构建模型定价信息（从已获取的定价数据）
func (s *ToolService) buildModelPricingInfo(pricings []*entities.ModelPricing) map[string]interface{} {
	// 如果没有定价信息，返回默认值
	if len(pricings) == 0 {
		return map[string]interface{}{
			"pricing": map[string]interface{}{
				"input":  0.003,
				"output": 0.015,
				"unit":   "1K tokens",
			},
			"rate_multiplier": 1.0,
		}
	}

	// 构建定价信息
	pricingData := map[string]interface{}{
		"input":  0.003, // 默认输入价格
		"output": 0.015, // 默认输出价格
		"unit":   "1K tokens",
	}

	var multiplier float64 = 1.0 // 默认倍率

	// 解析定价信息
	for _, pricing := range pricings {
		// 获取倍率（所有定价类型都应该有相同的倍率）
		if pricing.Multiplier > 0 {
			multiplier = pricing.Multiplier
		}

		switch pricing.PricingType {
		case entities.PricingTypeInput:
			pricingData["input"] = pricing.PricePerUnit
		case entities.PricingTypeOutput:
			pricingData["output"] = pricing.PricePerUnit
		case entities.PricingTypeRequest:
			// 对于按请求计费的模型（如图像生成）
			pricingData["input"] = pricing.PricePerUnit
			pricingData["output"] = 0.0
			pricingData["unit"] = string(pricing.Unit)
		}
	}

	return map[string]interface{}{
		"pricing":         pricingData,
		"rate_multiplier": multiplier,
	}
}

// GetUserAPIKeys 获取用户API密钥列表
func (s *ToolService) GetUserAPIKeys(ctx context.Context, userID int64) ([]map[string]interface{}, error) {
	// 获取用户的活跃API密钥
	apiKeys, err := s.apiKeyRepo.GetActiveKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	var result []map[string]interface{}
	for _, apiKey := range apiKeys {
		result = append(result, map[string]interface{}{
			"id":         apiKey.ID,
			"name":       apiKey.Name,
			"key_prefix": apiKey.KeyPrefix,
			"status":     apiKey.Status,
		})
	}

	return result, nil
}

// generateShareToken 生成分享token
func (s *ToolService) generateShareToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateCode 生成唯一的鉴权代码
func (s *ToolService) generateCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
