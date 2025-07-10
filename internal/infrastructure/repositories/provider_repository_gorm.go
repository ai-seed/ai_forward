package repositories

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/redis"

	"gorm.io/gorm"
)

// providerRepositoryGorm GORM提供商仓储实现
type providerRepositoryGorm struct {
	db    *gorm.DB
	cache *redis.CacheService
}

// NewProviderRepositoryGorm 创建GORM提供商仓储
func NewProviderRepositoryGorm(db *gorm.DB, cache *redis.CacheService) repositories.ProviderRepository {
	return &providerRepositoryGorm{
		db:    db,
		cache: cache,
	}
}

// Create 创建服务提供商
func (r *providerRepositoryGorm) Create(ctx context.Context, provider *entities.Provider) error {
	if err := r.db.WithContext(ctx).Create(provider).Error; err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	return nil
}

// GetByID 根据ID获取服务提供商
func (r *providerRepositoryGorm) GetByID(ctx context.Context, id int64) (*entities.Provider, error) {
	// 尝试从缓存获取提供商
	if r.cache != nil {
		cacheKey := GetProviderCacheKey(id)
		var cachedProvider entities.Provider
		if err := r.cache.Get(ctx, cacheKey, &cachedProvider); err == nil {
			return &cachedProvider, nil
		}
	}

	// 从数据库获取提供商
	var provider entities.Provider
	if err := r.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrProviderNotFound
		}
		return nil, fmt.Errorf("failed to get provider by id: %w", err)
	}

	// 缓存提供商信息（提供商配置基本不变，缓存30分钟）
	if r.cache != nil {
		cacheKey := GetProviderCacheKey(id)
		ttl := 30 * time.Minute
		r.cache.Set(ctx, cacheKey, &provider, ttl)

		// 同时缓存slug索引
		slugCacheKey := GetProviderBySlugCacheKey(provider.Slug)
		r.cache.Set(ctx, slugCacheKey, &provider, ttl)
	}

	return &provider, nil
}

// GetBySlug 根据slug获取服务提供商
func (r *providerRepositoryGorm) GetBySlug(ctx context.Context, slug string) (*entities.Provider, error) {
	// 尝试从缓存获取提供商
	if r.cache != nil {
		cacheKey := GetProviderBySlugCacheKey(slug)
		var cachedProvider entities.Provider
		if err := r.cache.Get(ctx, cacheKey, &cachedProvider); err == nil {
			return &cachedProvider, nil
		}
	}

	// 从数据库获取提供商
	var provider entities.Provider
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrProviderNotFound
		}
		return nil, fmt.Errorf("failed to get provider by slug: %w", err)
	}

	// 缓存提供商信息
	if r.cache != nil {
		ttl := 30 * time.Minute

		// 缓存slug索引
		slugCacheKey := GetProviderBySlugCacheKey(slug)
		r.cache.Set(ctx, slugCacheKey, &provider, ttl)

		// 同时缓存ID索引
		idCacheKey := GetProviderCacheKey(provider.ID)
		r.cache.Set(ctx, idCacheKey, &provider, ttl)
	}

	return &provider, nil
}

// Update 更新服务提供商
func (r *providerRepositoryGorm) Update(ctx context.Context, provider *entities.Provider) error {
	provider.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).Save(provider)
	if result.Error != nil {
		return fmt.Errorf("failed to update provider: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrProviderNotFound
	}

	// 清除提供商相关缓存
	if r.cache != nil {
		// 清除ID索引缓存
		idCacheKey := GetProviderCacheKey(provider.ID)
		r.cache.Delete(ctx, idCacheKey)

		// 清除slug索引缓存
		slugCacheKey := GetProviderBySlugCacheKey(provider.Slug)
		r.cache.Delete(ctx, slugCacheKey)

		// 清除提供商列表缓存
		r.cache.Delete(ctx, CacheKeyActiveProviders)
		r.cache.Delete(ctx, CacheKeyAvailableProviders)
		r.cache.Delete(ctx, CacheKeyProvidersByPriority)
	}

	return nil
}

// UpdateHealthStatus 更新健康状态
func (r *providerRepositoryGorm) UpdateHealthStatus(ctx context.Context, id int64, status entities.HealthStatus) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&entities.Provider{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"health_status":     status,
			"last_health_check": &now,
			"updated_at":        now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update provider health status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrProviderNotFound
	}

	return nil
}

// Delete 删除服务提供商
func (r *providerRepositoryGorm) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&entities.Provider{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete provider: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrProviderNotFound
	}

	return nil
}

// List 获取服务提供商列表
func (r *providerRepositoryGorm) List(ctx context.Context, offset, limit int) ([]*entities.Provider, error) {
	var providers []*entities.Provider
	if err := r.db.WithContext(ctx).
		Order("priority ASC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	return providers, nil
}

// Count 获取服务提供商总数
func (r *providerRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entities.Provider{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count providers: %w", err)
	}
	return count, nil
}

// GetActiveProviders 获取活跃的服务提供商列表
func (r *providerRepositoryGorm) GetActiveProviders(ctx context.Context) ([]*entities.Provider, error) {
	// 尝试从缓存获取活跃提供商列表
	if r.cache != nil {
		cacheKey := CacheKeyActiveProviders
		var cachedProviders []*entities.Provider
		if err := r.cache.Get(ctx, cacheKey, &cachedProviders); err == nil {
			return cachedProviders, nil
		}
	}

	// 从数据库获取活跃提供商列表
	var providers []*entities.Provider
	if err := r.db.WithContext(ctx).
		Where("status = ?", entities.ProviderStatusActive).
		Order("priority ASC").
		Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to get active providers: %w", err)
	}

	// 缓存活跃提供商列表（提供商列表变化不频繁，缓存15分钟）
	if r.cache != nil {
		cacheKey := CacheKeyActiveProviders
		ttl := 15 * time.Minute
		r.cache.Set(ctx, cacheKey, providers, ttl)
	}

	return providers, nil
}

// GetAvailableProviders 获取可用的服务提供商列表（活跃且健康）
func (r *providerRepositoryGorm) GetAvailableProviders(ctx context.Context) ([]*entities.Provider, error) {
	// 尝试从缓存获取可用提供商列表
	if r.cache != nil {
		cacheKey := CacheKeyAvailableProviders
		var cachedProviders []*entities.Provider
		if err := r.cache.Get(ctx, cacheKey, &cachedProviders); err == nil {
			return cachedProviders, nil
		}
	}

	// 从数据库获取可用提供商列表
	var providers []*entities.Provider
	if err := r.db.WithContext(ctx).
		Where("status = ? AND health_status = ?",
			entities.ProviderStatusActive,
			entities.HealthStatusHealthy).
		Order("priority ASC").
		Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to get available providers: %w", err)
	}

	// 缓存可用提供商列表
	if r.cache != nil {
		cacheKey := CacheKeyAvailableProviders
		ttl := 5 * time.Minute // 可用性状态变化较快，缓存5分钟
		r.cache.Set(ctx, cacheKey, providers, ttl)
	}

	return providers, nil
}

// GetProvidersByPriority 按优先级获取服务提供商列表
func (r *providerRepositoryGorm) GetProvidersByPriority(ctx context.Context) ([]*entities.Provider, error) {
	var providers []*entities.Provider
	if err := r.db.WithContext(ctx).
		Where("status = ?", entities.ProviderStatusActive).
		Order("priority ASC, id ASC").
		Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to get providers by priority: %w", err)
	}
	return providers, nil
}

// GetProvidersNeedingHealthCheck 获取需要健康检查的服务提供商列表
func (r *providerRepositoryGorm) GetProvidersNeedingHealthCheck(ctx context.Context) ([]*entities.Provider, error) {
	var providers []*entities.Provider
	now := time.Now()

	// 查询需要健康检查的提供商：
	// 1. 从未进行过健康检查的（last_health_check IS NULL）
	// 2. 距离上次检查时间超过检查间隔的
	if err := r.db.WithContext(ctx).
		Where("status = ? AND (last_health_check IS NULL OR last_health_check + INTERVAL health_check_interval SECOND < ?)",
			entities.ProviderStatusActive, now).
		Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to get providers needing health check: %w", err)
	}

	return providers, nil
}
