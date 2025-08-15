package repositories

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/cache"
	"ai-api-gateway/internal/infrastructure/redis"

	"gorm.io/gorm"
)

// modelRepositoryGorm GORM模型仓储实现
type modelRepositoryGorm struct {
	db    *gorm.DB
	cache *redis.CacheService
}

// NewModelRepositoryGorm 创建GORM模型仓储
func NewModelRepositoryGorm(db *gorm.DB, cache *redis.CacheService) repositories.ModelRepository {
	return &modelRepositoryGorm{
		db:    db,
		cache: cache,
	}
}

// Create 创建模型
func (r *modelRepositoryGorm) Create(ctx context.Context, model *entities.Model) error {
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}
	return nil
}

// GetByID 根据ID获取模型
func (r *modelRepositoryGorm) GetByID(ctx context.Context, id int64) (*entities.Model, error) {
	// 尝试从缓存获取模型
	if r.cache != nil {
		cacheKey := GetModelCacheKey(id)
		var cachedModel entities.Model
		if err := r.cache.Get(ctx, cacheKey, &cachedModel); err == nil {
			return &cachedModel, nil
		}
	}

	// 从数据库获取模型
	var model entities.Model
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrModelNotFound
		}
		return nil, fmt.Errorf("failed to get model by id: %w", err)
	}

	// 缓存模型信息（模型配置基本不变，缓存30分钟）
	if r.cache != nil {
		cacheKey := GetModelCacheKey(id)
		cacheManager := cache.GetCacheTTLManager()
		ttl := cacheManager.GetModelTTL()
		r.cache.Set(ctx, cacheKey, &model, ttl)

		// 同时缓存slug索引
		slugCacheKey := GetModelBySlugCacheKey(model.Slug)
		r.cache.Set(ctx, slugCacheKey, &model, ttl)
	}

	return &model, nil
}

// GetBySlug 根据slug获取模型
func (r *modelRepositoryGorm) GetBySlug(ctx context.Context, slug string) (*entities.Model, error) {
	// 尝试从缓存获取模型
	if r.cache != nil {
		cacheKey := GetModelBySlugCacheKey(slug)
		var cachedModel entities.Model
		if err := r.cache.Get(ctx, cacheKey, &cachedModel); err == nil {
			return &cachedModel, nil
		}
	}

	// 从数据库获取模型
	var model entities.Model
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrModelNotFound
		}
		return nil, fmt.Errorf("failed to get model by slug: %w", err)
	}

	// 缓存模型信息
	if r.cache != nil {
		cacheManager := cache.GetCacheTTLManager()
		ttl := cacheManager.GetModelTTL()

		// 缓存slug索引
		slugCacheKey := GetModelBySlugCacheKey(slug)
		r.cache.Set(ctx, slugCacheKey, &model, ttl)

		// 同时缓存ID索引
		idCacheKey := GetModelCacheKey(model.ID)
		r.cache.Set(ctx, idCacheKey, &model, ttl)
	}

	return &model, nil
}

// Update 更新模型
func (r *modelRepositoryGorm) Update(ctx context.Context, model *entities.Model) error {
	model.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update model: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrModelNotFound
	}

	// 清除模型相关缓存
	if r.cache != nil {
		// 清除ID索引缓存
		idCacheKey := GetModelCacheKey(model.ID)
		r.cache.Delete(ctx, idCacheKey)

		// 清除slug索引缓存
		slugCacheKey := GetModelBySlugCacheKey(model.Slug)
		r.cache.Delete(ctx, slugCacheKey)

		// 清除模型列表缓存
		r.cache.Delete(ctx, CacheKeyActiveModels)
		r.cache.Delete(ctx, CacheKeyAvailableModels)

		// 清除按类型分组的模型列表缓存
		typeCacheKey := GetModelsByTypeCacheKey(string(model.ModelType))
		r.cache.Delete(ctx, typeCacheKey)
	}

	return nil
}

// Delete 删除模型
func (r *modelRepositoryGorm) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&entities.Model{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete model: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrModelNotFound
	}

	return nil
}

// List 获取模型列表
func (r *modelRepositoryGorm) List(ctx context.Context, offset, limit int) ([]*entities.Model, error) {
	var models []*entities.Model
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	return models, nil
}

// Count 获取模型总数
func (r *modelRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entities.Model{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count models: %w", err)
	}
	return count, nil
}

// GetActiveModels 获取活跃的模型列表
func (r *modelRepositoryGorm) GetActiveModels(ctx context.Context) ([]*entities.Model, error) {
	// 尝试从缓存获取活跃模型列表
	if r.cache != nil {
		cacheKey := CacheKeyActiveModels
		var cachedModels []*entities.Model
		if err := r.cache.Get(ctx, cacheKey, &cachedModels); err == nil {
			return cachedModels, nil
		}
	}

	// 从数据库获取活跃模型列表
	var models []*entities.Model
	if err := r.db.WithContext(ctx).
		Where("status = ?", entities.ModelStatusActive).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get active models: %w", err)
	}

	// 手动加载厂商信息
	if err := r.loadProvidersForModels(ctx, models); err != nil {
		return nil, fmt.Errorf("failed to load providers for models: %w", err)
	}

	// 缓存活跃模型列表（模型列表变化不频繁，缓存15分钟）
	if r.cache != nil {
		cacheKey := CacheKeyActiveModels
		cacheManager := cache.GetCacheTTLManager()
		ttl := cacheManager.GetModelListTTL()
		r.cache.Set(ctx, cacheKey, models, ttl)
	}

	return models, nil
}

// GetActiveModelsWithPagination 获取活跃的模型列表（分页）
func (r *modelRepositoryGorm) GetActiveModelsWithPagination(ctx context.Context, offset, limit int) ([]*entities.Model, error) {
	var models []*entities.Model
	if err := r.db.WithContext(ctx).
		Where("status = ?", entities.ModelStatusActive).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get active models with pagination: %w", err)
	}

	// 手动加载厂商信息
	if err := r.loadProvidersForModels(ctx, models); err != nil {
		return nil, fmt.Errorf("failed to load providers for models: %w", err)
	}

	return models, nil
}

// CountActiveModels 获取活跃模型总数
func (r *modelRepositoryGorm) CountActiveModels(ctx context.Context) (int64, error) {
	// 尝试从缓存获取活跃模型总数
	if r.cache != nil {
		cacheKey := CacheKeyActiveModelsCount
		var cachedCount int64
		if err := r.cache.Get(ctx, cacheKey, &cachedCount); err == nil {
			return cachedCount, nil
		}
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&entities.Model{}).
		Where("status = ?", entities.ModelStatusActive).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count active models: %w", err)
	}

	// 缓存活跃模型总数（模型数量变化不频繁，缓存15分钟）
	if r.cache != nil {
		cacheKey := CacheKeyActiveModelsCount
		cacheManager := cache.GetCacheTTLManager()
		ttl := cacheManager.GetModelListTTL()
		r.cache.Set(ctx, cacheKey, count, ttl)
	}

	return count, nil
}

// GetActiveModelsWithPaginationAndFilters 获取活跃的模型列表（分页+筛选）
func (r *modelRepositoryGorm) GetActiveModelsWithPaginationAndFilters(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*entities.Model, error) {
	query := r.db.WithContext(ctx).Where("models.status = ?", entities.ModelStatusActive)

	// 应用筛选条件
	query = r.applyFilters(query, filters)

	var models []*entities.Model
	if err := query.Order("models.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get active models with pagination and filters: %w", err)
	}

	// 手动加载厂商信息
	if err := r.loadProvidersForModels(ctx, models); err != nil {
		return nil, fmt.Errorf("failed to load providers for models: %w", err)
	}

	return models, nil
}

// CountActiveModelsWithFilters 获取活跃模型总数（带筛选）
func (r *modelRepositoryGorm) CountActiveModelsWithFilters(ctx context.Context, filters map[string]interface{}) (int64, error) {
	query := r.db.WithContext(ctx).Model(&entities.Model{}).Where("models.status = ?", entities.ModelStatusActive)

	// 应用筛选条件
	query = r.applyFilters(query, filters)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count active models with filters: %w", err)
	}

	return count, nil
}

// applyFilters 应用筛选条件
func (r *modelRepositoryGorm) applyFilters(query *gorm.DB, filters map[string]interface{}) *gorm.DB {
	// 按厂商筛选
	if provider, ok := filters["provider"].(string); ok && provider != "" {
		// 需要join model_providers表
		query = query.Joins("JOIN model_providers ON models.model_provider_id = model_providers.id").
			Where("model_providers.display_name = ?", provider)
	}

	// 按类型筛选
	if modelType, ok := filters["type"].(string); ok && modelType != "" {
		query = query.Where("models.model_type = ?", modelType)
	}

	return query
}

// loadProvidersForModels 手动加载模型的厂商信息
func (r *modelRepositoryGorm) loadProvidersForModels(ctx context.Context, models []*entities.Model) error {
	if len(models) == 0 {
		return nil
	}

	// 收集所有需要的厂商ID
	providerIDs := make([]int64, 0)
	providerIDSet := make(map[int64]bool)
	for _, model := range models {
		if !providerIDSet[model.ModelProviderID] {
			providerIDs = append(providerIDs, model.ModelProviderID)
			providerIDSet[model.ModelProviderID] = true
		}
	}

	// 批量查询厂商信息
	var providers []entities.ModelProvider
	if err := r.db.WithContext(ctx).
		Where("id IN ?", providerIDs).
		Find(&providers).Error; err != nil {
		return fmt.Errorf("failed to load providers: %w", err)
	}

	// 创建厂商ID到厂商对象的映射
	providerMap := make(map[int64]*entities.ModelProvider)
	for i := range providers {
		providerMap[providers[i].ID] = &providers[i]
	}

	// 为每个模型设置厂商信息
	for _, model := range models {
		if provider, exists := providerMap[model.ModelProviderID]; exists {
			model.ModelProvider = provider
		}
	}

	return nil
}

// GetModelsByType 根据类型获取模型列表
func (r *modelRepositoryGorm) GetModelsByType(ctx context.Context, modelType entities.ModelType) ([]*entities.Model, error) {
	// 尝试从缓存获取按类型分组的模型列表
	if r.cache != nil {
		cacheKey := GetModelsByTypeCacheKey(string(modelType))
		var cachedModels []*entities.Model
		if err := r.cache.Get(ctx, cacheKey, &cachedModels); err == nil {
			return cachedModels, nil
		}
	}

	// 从数据库获取按类型分组的模型列表
	var models []*entities.Model
	if err := r.db.WithContext(ctx).
		Where("model_type = ? AND status = ?", modelType, entities.ModelStatusActive).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get models by type: %w", err)
	}

	// 缓存按类型分组的模型列表
	if r.cache != nil {
		cacheKey := GetModelsByTypeCacheKey(string(modelType))
		cacheManager := cache.GetCacheTTLManager()
		ttl := cacheManager.GetModelListTTL()
		r.cache.Set(ctx, cacheKey, models, ttl)
	}

	return models, nil
}

// GetAvailableModels 获取可用的模型列表
func (r *modelRepositoryGorm) GetAvailableModels(ctx context.Context) ([]*entities.Model, error) {
	// 尝试从缓存获取可用模型列表
	if r.cache != nil {
		cacheKey := CacheKeyAvailableModels
		var cachedModels []*entities.Model
		if err := r.cache.Get(ctx, cacheKey, &cachedModels); err == nil {
			return cachedModels, nil
		}
	}

	// 从数据库获取可用模型列表（包含所有类型的活跃模型）
	var models []*entities.Model
	if err := r.db.WithContext(ctx).
		Where("status = ?", entities.ModelStatusActive).
		Order("model_type ASC, created_at DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get available models: %w", err)
	}

	// 缓存可用模型列表
	if r.cache != nil {
		cacheKey := CacheKeyAvailableModels
		cacheManager := cache.GetCacheTTLManager()
		ttl := cacheManager.GetModelListTTL()
		r.cache.Set(ctx, cacheKey, models, ttl)
	}

	return models, nil
}

// modelPricingRepositoryGorm GORM模型定价仓储实现
type modelPricingRepositoryGorm struct {
	db    *gorm.DB
	cache *redis.CacheService
}

// NewModelPricingRepositoryGorm 创建GORM模型定价仓储
func NewModelPricingRepositoryGorm(db *gorm.DB, cache *redis.CacheService) repositories.ModelPricingRepository {
	return &modelPricingRepositoryGorm{
		db:    db,
		cache: cache,
	}
}

// Create 创建模型定价
func (r *modelPricingRepositoryGorm) Create(ctx context.Context, pricing *entities.ModelPricing) error {
	if err := r.db.WithContext(ctx).Create(pricing).Error; err != nil {
		return fmt.Errorf("failed to create model pricing: %w", err)
	}
	return nil
}

// GetByID 根据ID获取模型定价
func (r *modelPricingRepositoryGorm) GetByID(ctx context.Context, id int64) (*entities.ModelPricing, error) {
	var pricing entities.ModelPricing
	if err := r.db.WithContext(ctx).First(&pricing, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrModelPricingNotFound
		}
		return nil, fmt.Errorf("failed to get model pricing by id: %w", err)
	}
	return &pricing, nil
}

// GetByModelID 根据模型ID获取定价列表
func (r *modelPricingRepositoryGorm) GetByModelID(ctx context.Context, modelID int64) ([]*entities.ModelPricing, error) {
	var pricings []*entities.ModelPricing
	if err := r.db.WithContext(ctx).
		Where("model_id = ?", modelID).
		Order("effective_from DESC").
		Find(&pricings).Error; err != nil {
		return nil, fmt.Errorf("failed to get model pricing by model id: %w", err)
	}
	return pricings, nil
}

// GetCurrentPricing 获取模型当前有效定价
func (r *modelPricingRepositoryGorm) GetCurrentPricing(ctx context.Context, modelID int64) ([]*entities.ModelPricing, error) {
	var pricings []*entities.ModelPricing
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("model_id = ? AND effective_from <= ? AND (effective_until IS NULL OR effective_until > ?)",
			modelID, now, now).
		Order("pricing_type ASC").
		Find(&pricings).Error; err != nil {
		return nil, fmt.Errorf("failed to get current model pricing: %w", err)
	}
	return pricings, nil
}

// Update 更新模型定价
func (r *modelPricingRepositoryGorm) Update(ctx context.Context, pricing *entities.ModelPricing) error {
	result := r.db.WithContext(ctx).Save(pricing)
	if result.Error != nil {
		return fmt.Errorf("failed to update model pricing: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrModelPricingNotFound
	}

	return nil
}

// Delete 删除模型定价
func (r *modelPricingRepositoryGorm) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&entities.ModelPricing{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete model pricing: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrModelPricingNotFound
	}

	return nil
}

// List 获取模型定价列表
func (r *modelPricingRepositoryGorm) List(ctx context.Context, offset, limit int) ([]*entities.ModelPricing, error) {
	var pricings []*entities.ModelPricing
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&pricings).Error; err != nil {
		return nil, fmt.Errorf("failed to list model pricing: %w", err)
	}
	return pricings, nil
}

// Count 获取模型定价总数
func (r *modelPricingRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entities.ModelPricing{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count model pricing: %w", err)
	}
	return count, nil
}

// GetPricingByType 根据定价类型获取定价
func (r *modelPricingRepositoryGorm) GetPricingByType(ctx context.Context, modelID int64, pricingType entities.PricingType) (*entities.ModelPricing, error) {
	var pricing entities.ModelPricing

	if err := r.db.WithContext(ctx).
		Where("model_id = ? AND pricing_type = ?",
			modelID, pricingType).
		Order("effective_from DESC").
		First(&pricing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrModelPricingNotFound
		}
		return nil, fmt.Errorf("failed to get model pricing by type: %w", err)
	}
	return &pricing, nil
}

// GetCurrentPricingBatch 批量获取多个模型的当前有效定价
func (r *modelPricingRepositoryGorm) GetCurrentPricingBatch(ctx context.Context, modelIDs []int64) (map[int64][]*entities.ModelPricing, error) {
	if len(modelIDs) == 0 {
		return make(map[int64][]*entities.ModelPricing), nil
	}

	var pricings []*entities.ModelPricing
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("model_id IN ? AND effective_from <= ? AND (effective_until IS NULL OR effective_until > ?)",
			modelIDs, now, now).
		Order("model_id ASC, pricing_type ASC").
		Find(&pricings).Error; err != nil {
		return nil, fmt.Errorf("failed to get current model pricing batch: %w", err)
	}

	// 按模型ID分组
	result := make(map[int64][]*entities.ModelPricing)
	for _, pricing := range pricings {
		result[pricing.ModelID] = append(result[pricing.ModelID], pricing)
	}

	return result, nil
}
