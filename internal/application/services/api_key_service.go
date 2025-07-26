package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/domain/values"
	redisInfra "ai-api-gateway/internal/infrastructure/redis"
)

// generateRandomAPIKeyName 生成随机API密钥名称
func generateRandomAPIKeyName() string {
	adjectives := []string{"Swift", "Bright", "Smart", "Quick", "Fast", "Cool", "Sharp", "Bold"}
	nouns := []string{"Key", "Token", "Access", "Gate", "Bridge", "Link", "Path", "Code"}

	rand.Seed(time.Now().UnixNano())
	randomAdjective := adjectives[rand.Intn(len(adjectives))]
	randomNoun := nouns[rand.Intn(len(nouns))]
	randomNumber := rand.Intn(1000)

	return fmt.Sprintf("%s%s%d", randomAdjective, randomNoun, randomNumber)
}

// APIKeyService API密钥服务接口
type APIKeyService interface {
	// CreateAPIKey 创建API密钥
	CreateAPIKey(ctx context.Context, req *dto.CreateAPIKeyRequest) (*dto.APIKeyCreateResponse, error)

	// GetAPIKey 获取API密钥
	GetAPIKey(ctx context.Context, id int64) (*dto.APIKeyResponse, error)

	// GetAPIKeyByKey 根据密钥获取API密钥
	GetAPIKeyByKey(ctx context.Context, key string) (*dto.APIKeyResponse, error)

	// GetUserAPIKeys 获取用户的API密钥列表
	GetUserAPIKeys(ctx context.Context, userID int64) ([]*dto.APIKeyResponse, error)

	// UpdateAPIKey 更新API密钥
	UpdateAPIKey(ctx context.Context, id int64, req *dto.UpdateAPIKeyRequest) (*dto.APIKeyResponse, error)

	// DeleteAPIKey 删除API密钥
	DeleteAPIKey(ctx context.Context, id int64) error

	// RevokeAPIKey 撤销API密钥
	RevokeAPIKey(ctx context.Context, id int64) error

	// ListAPIKeys 获取API密钥列表
	ListAPIKeys(ctx context.Context, pagination *dto.PaginationRequest) (*dto.APIKeyListResponse, error)

	// ValidateAPIKey 验证API密钥
	ValidateAPIKey(ctx context.Context, keyString string) (*entities.APIKey, *entities.User, error)
}

// apiKeyServiceImpl API密钥服务实现
type apiKeyServiceImpl struct {
	apiKeyRepo   repositories.APIKeyRepository
	userRepo     repositories.UserRepository
	usageLogRepo repositories.UsageLogRepository
	keyGen       *values.APIKeyGenerator
	cache        *redisInfra.CacheService
}

// NewAPIKeyService 创建API密钥服务
func NewAPIKeyService(apiKeyRepo repositories.APIKeyRepository, userRepo repositories.UserRepository, usageLogRepo repositories.UsageLogRepository) APIKeyService {
	return &apiKeyServiceImpl{
		apiKeyRepo:   apiKeyRepo,
		userRepo:     userRepo,
		usageLogRepo: usageLogRepo,
		keyGen:       values.NewAPIKeyGenerator(),
		cache:        nil, // 暂时不使用缓存
	}
}

// CreateAPIKey 创建API密钥
func (s *apiKeyServiceImpl) CreateAPIKey(ctx context.Context, req *dto.CreateAPIKeyRequest) (*dto.APIKeyCreateResponse, error) {
	// 验证用户是否存在
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 检查用户状态
	if !user.IsActive() {
		return nil, fmt.Errorf("user is not active")
	}

	// 生成API密钥
	key, _, prefix, err := s.keyGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate api key: %w", err)
	}

	// 处理API密钥名称：如果用户没有提供名称，则生成随机名称
	var apiKeyName string
	if req.Name != nil && *req.Name != "" {
		apiKeyName = *req.Name
	} else {
		apiKeyName = generateRandomAPIKeyName()
	}

	// 创建API密钥实体
	apiKey := &entities.APIKey{
		UserID:      req.UserID,
		Key:         key,
		KeyPrefix:   prefix,
		Name:        &apiKeyName,
		Status:      entities.APIKeyStatusActive,
		Permissions: req.Permissions,
		ExpiresAt:   req.ExpiresAt,
	}

	// 保存到数据库
	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	// 构造响应
	response := &dto.APIKeyCreateResponse{
		APIKeyResponse: (&dto.APIKeyResponse{}).FromEntity(apiKey),
	}

	return response, nil
}

// GetAPIKey 获取API密钥
func (s *apiKeyServiceImpl) GetAPIKey(ctx context.Context, id int64) (*dto.APIKeyResponse, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return (&dto.APIKeyResponse{}).FromEntity(apiKey), nil
}

// GetAPIKeyByKey 根据密钥获取API密钥
func (s *apiKeyServiceImpl) GetAPIKeyByKey(ctx context.Context, key string) (*dto.APIKeyResponse, error) {
	apiKey, err := s.apiKeyRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	return (&dto.APIKeyResponse{}).FromEntity(apiKey), nil
}

// GetUserAPIKeys 获取用户的API密钥列表
func (s *apiKeyServiceImpl) GetUserAPIKeys(ctx context.Context, userID int64) ([]*dto.APIKeyResponse, error) {
	apiKeys, err := s.apiKeyRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user api keys: %w", err)
	}

	// 转换为DTO并添加成本信息
	apiKeyDTOs := dto.FromAPIKeyEntities(apiKeys)

	// 为每个API密钥添加总成本信息
	for i, apiKey := range apiKeys {
		totalCost, err := s.usageLogRepo.GetAPIKeyTotalCost(ctx, apiKey.ID)
		if err != nil {
			// 如果获取成本失败，记录错误但不中断流程，设置成本为0
			totalCost = 0
		}
		apiKeyDTOs[i].TotalCost = totalCost
	}

	return apiKeyDTOs, nil
}

// UpdateAPIKey 更新API密钥
func (s *apiKeyServiceImpl) UpdateAPIKey(ctx context.Context, id int64, req *dto.UpdateAPIKeyRequest) (*dto.APIKeyResponse, error) {
	// 获取现有API密钥
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Name != nil {
		apiKey.Name = req.Name
	}

	if req.Status != nil {
		apiKey.Status = *req.Status
	}

	if req.Permissions != nil {
		apiKey.Permissions = req.Permissions
	}

	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}

	// 保存更新
	if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to update api key: %w", err)
	}

	return (&dto.APIKeyResponse{}).FromEntity(apiKey), nil
}

// DeleteAPIKey 删除API密钥
func (s *apiKeyServiceImpl) DeleteAPIKey(ctx context.Context, id int64) error {
	return s.apiKeyRepo.Delete(ctx, id)
}

// RevokeAPIKey 撤销API密钥
func (s *apiKeyServiceImpl) RevokeAPIKey(ctx context.Context, id int64) error {
	return s.apiKeyRepo.UpdateStatus(ctx, id, entities.APIKeyStatusRevoked)
}

// ListAPIKeys 获取API密钥列表
func (s *apiKeyServiceImpl) ListAPIKeys(ctx context.Context, pagination *dto.PaginationRequest) (*dto.APIKeyListResponse, error) {
	pagination.SetDefaults()

	// 获取API密钥列表
	apiKeys, err := s.apiKeyRepo.List(ctx, pagination.GetOffset(), pagination.GetLimit())
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}

	// 获取总数
	total, err := s.apiKeyRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count api keys: %w", err)
	}

	// 转换为DTO并添加成本信息
	apiKeyDTOs := dto.FromAPIKeyEntities(apiKeys)

	// 为每个API密钥添加总成本信息
	for i, apiKey := range apiKeys {
		totalCost, err := s.usageLogRepo.GetAPIKeyTotalCost(ctx, apiKey.ID)
		if err != nil {
			// 如果获取成本失败，记录错误但不中断流程，设置成本为0
			totalCost = 0
		}
		apiKeyDTOs[i].TotalCost = totalCost
	}

	// 构造响应
	response := &dto.APIKeyListResponse{
		APIKeys:  apiKeyDTOs,
		Total:    total,
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	// 计算总页数
	paginationResp := &dto.PaginationResponse{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		Total:    total,
	}
	paginationResp.CalculateTotalPages()
	response.TotalPages = paginationResp.TotalPages

	return response, nil
}

// ValidateAPIKey 验证API密钥
func (s *apiKeyServiceImpl) ValidateAPIKey(ctx context.Context, keyString string) (*entities.APIKey, *entities.User, error) {
	// 验证密钥格式
	if !s.keyGen.ValidateFormat(keyString) {
		return nil, nil, entities.ErrAPIKeyInvalid
	}

	// 从数据库获取API密钥（缓存逻辑已移到repository层）
	apiKey, err := s.apiKeyRepo.GetByKey(ctx, keyString)
	if err != nil {
		if err == entities.ErrAPIKeyNotFound {
			return nil, nil, entities.ErrAPIKeyInvalid
		}
		return nil, nil, err
	}

	// 检查API密钥状态
	if !apiKey.IsActive() {
		if apiKey.IsExpired() {
			return nil, nil, entities.ErrAPIKeyExpired
		}
		return nil, nil, entities.ErrAPIKeyInactive
	}

	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, nil, err
	}

	// 检查用户状态
	if !user.IsActive() {
		return nil, nil, entities.ErrUserInactive
	}

	// 更新最后使用时间
	if err := s.apiKeyRepo.UpdateLastUsed(ctx, apiKey.ID); err != nil {
		// 记录错误但不影响验证结果
		// TODO: 添加日志记录
	}

	return apiKey, user, nil
}
