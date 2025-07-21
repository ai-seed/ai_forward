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

// apiKeyRepositoryGorm GORM API密钥仓储实现
type apiKeyRepositoryGorm struct {
	db    *gorm.DB
	cache *redis.CacheService
}

// NewAPIKeyRepositoryGorm 创建GORM API密钥仓储
func NewAPIKeyRepositoryGorm(db *gorm.DB, cache *redis.CacheService) repositories.APIKeyRepository {
	return &apiKeyRepositoryGorm{
		db:    db,
		cache: cache,
	}
}

// Create 创建API密钥
func (r *apiKeyRepositoryGorm) Create(ctx context.Context, apiKey *entities.APIKey) error {
	if err := r.db.WithContext(ctx).Create(apiKey).Error; err != nil {
		return fmt.Errorf("failed to create api key: %w", err)
	}
	return nil
}

// GetByID 根据ID获取API密钥
func (r *apiKeyRepositoryGorm) GetByID(ctx context.Context, id int64) (*entities.APIKey, error) {
	// 尝试从缓存获取API密钥
	if r.cache != nil {
		cacheKey := GetAPIKeyByIDCacheKey(id)
		var cachedAPIKey entities.APIKey
		if err := r.cache.Get(ctx, cacheKey, &cachedAPIKey); err == nil {
			return &cachedAPIKey, nil
		}
	}

	// 从数据库获取API密钥
	var apiKey entities.APIKey
	if err := r.db.WithContext(ctx).First(&apiKey, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key by id: %w", err)
	}

	// 缓存API密钥信息
	if r.cache != nil {
		// 缓存ID索引
		idCacheKey := GetAPIKeyByIDCacheKey(id)
		ttl := 10 * time.Minute
		r.cache.Set(ctx, idCacheKey, &apiKey, ttl)

		// 同时缓存Key索引
		keyCacheKey := GetAPIKeyCacheKey(apiKey.Key)
		r.cache.Set(ctx, keyCacheKey, &apiKey, ttl)
	}

	return &apiKey, nil
}

// GetByKey 根据密钥获取API密钥
func (r *apiKeyRepositoryGorm) GetByKey(ctx context.Context, key string) (*entities.APIKey, error) {
	// 尝试从缓存获取API密钥
	if r.cache != nil {
		cacheKey := GetAPIKeyCacheKey(key)
		var cachedAPIKey entities.APIKey
		if err := r.cache.Get(ctx, cacheKey, &cachedAPIKey); err == nil {
			return &cachedAPIKey, nil
		}
	}

	// 从数据库获取API密钥
	var apiKey entities.APIKey
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key by key: %w", err)
	}

	// 缓存API密钥信息
	if r.cache != nil {
		cacheKey := GetAPIKeyCacheKey(key)
		ttl := 10 * time.Minute // API密钥缓存10分钟
		r.cache.Set(ctx, cacheKey, &apiKey, ttl)
	}

	return &apiKey, nil
}

// GetByUserID 根据用户ID获取API密钥列表
func (r *apiKeyRepositoryGorm) GetByUserID(ctx context.Context, userID int64) ([]*entities.APIKey, error) {
	// 直接从数据库获取API密钥列表，不使用缓存
	var apiKeys []*entities.APIKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get api keys by user id: %w", err)
	}

	return apiKeys, nil
}

// Update 更新API密钥
func (r *apiKeyRepositoryGorm) Update(ctx context.Context, apiKey *entities.APIKey) error {
	apiKey.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).Save(apiKey)
	if result.Error != nil {
		return fmt.Errorf("failed to update api key: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrAPIKeyNotFound
	}

	// 清除API密钥缓存
	if r.cache != nil {
		// 清除Key索引缓存
		keyCacheKey := GetAPIKeyCacheKey(apiKey.Key)
		r.cache.Delete(ctx, keyCacheKey)

		// 清除ID索引缓存
		idCacheKey := GetAPIKeyByIDCacheKey(apiKey.ID)
		r.cache.Delete(ctx, idCacheKey)

		// 清除活跃API密钥列表缓存
		activeCacheKey := GetActiveAPIKeysCacheKey(apiKey.UserID)
		r.cache.Delete(ctx, activeCacheKey)
	}

	return nil
}

// UpdateLastUsed 更新最后使用时间
func (r *apiKeyRepositoryGorm) UpdateLastUsed(ctx context.Context, id int64) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&entities.APIKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_used_at": &now,
			"updated_at":   now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update api key last used: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrAPIKeyNotFound
	}

	return nil
}

// UpdateStatus 更新状态
func (r *apiKeyRepositoryGorm) UpdateStatus(ctx context.Context, id int64, status entities.APIKeyStatus) error {
	// 先获取API密钥信息以便清除缓存
	var apiKey entities.APIKey
	if err := r.db.WithContext(ctx).Select("key, user_id").First(&apiKey, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return entities.ErrAPIKeyNotFound
		}
		return fmt.Errorf("failed to get api key for cache invalidation: %w", err)
	}

	result := r.db.WithContext(ctx).Model(&entities.APIKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update api key status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrAPIKeyNotFound
	}

	// 清除API密钥缓存
	if r.cache != nil {
		// 清除Key索引缓存
		keyCacheKey := GetAPIKeyCacheKey(apiKey.Key)
		r.cache.Delete(ctx, keyCacheKey)

		// 清除ID索引缓存
		idCacheKey := GetAPIKeyByIDCacheKey(id)
		r.cache.Delete(ctx, idCacheKey)

		// 清除活跃API密钥列表缓存
		activeCacheKey := GetActiveAPIKeysCacheKey(apiKey.UserID)
		r.cache.Delete(ctx, activeCacheKey)
	}

	return nil
}

// Delete 删除API密钥
func (r *apiKeyRepositoryGorm) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&entities.APIKey{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete api key: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return entities.ErrAPIKeyNotFound
	}

	return nil
}

// List 获取API密钥列表
func (r *apiKeyRepositoryGorm) List(ctx context.Context, offset, limit int) ([]*entities.APIKey, error) {
	var apiKeys []*entities.APIKey
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	return apiKeys, nil
}

// Count 获取API密钥总数
func (r *apiKeyRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entities.APIKey{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count api keys: %w", err)
	}
	return count, nil
}

// GetActiveAPIKeys 获取活跃的API密钥列表
func (r *apiKeyRepositoryGorm) GetActiveAPIKeys(ctx context.Context, offset, limit int) ([]*entities.APIKey, error) {
	var apiKeys []*entities.APIKey
	if err := r.db.WithContext(ctx).
		Where("status = ?", entities.APIKeyStatusActive).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get active api keys: %w", err)
	}
	return apiKeys, nil
}

// GetExpiredAPIKeys 获取过期的API密钥列表
func (r *apiKeyRepositoryGorm) GetExpiredAPIKeys(ctx context.Context) ([]*entities.APIKey, error) {
	var apiKeys []*entities.APIKey
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get expired api keys: %w", err)
	}
	return apiKeys, nil
}

// GetAPIKeysByStatus 根据状态获取API密钥列表
func (r *apiKeyRepositoryGorm) GetAPIKeysByStatus(ctx context.Context, status entities.APIKeyStatus, offset, limit int) ([]*entities.APIKey, error) {
	var apiKeys []*entities.APIKey
	if err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get api keys by status: %w", err)
	}
	return apiKeys, nil
}

// CountByUserID 根据用户ID获取API密钥总数
func (r *apiKeyRepositoryGorm) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entities.APIKey{}).
		Where("user_id = ?", userID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count api keys by user id: %w", err)
	}
	return count, nil
}

// GetByKeyPrefix 根据密钥前缀获取API密钥
func (r *apiKeyRepositoryGorm) GetByKeyPrefix(ctx context.Context, keyPrefix string) (*entities.APIKey, error) {
	var apiKey entities.APIKey
	if err := r.db.WithContext(ctx).Where("key_prefix = ?", keyPrefix).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key by key prefix: %w", err)
	}
	return &apiKey, nil
}

// GetActiveKeys 获取活跃的API密钥列表
func (r *apiKeyRepositoryGorm) GetActiveKeys(ctx context.Context, userID int64) ([]*entities.APIKey, error) {
	var apiKeys []*entities.APIKey
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, entities.APIKeyStatusActive).
		Order("created_at DESC").
		Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get active api keys: %w", err)
	}
	return apiKeys, nil
}

// GetExpiredKeys 获取过期的API密钥列表
func (r *apiKeyRepositoryGorm) GetExpiredKeys(ctx context.Context, limit int) ([]*entities.APIKey, error) {
	var apiKeys []*entities.APIKey
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Limit(limit).
		Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get expired api keys: %w", err)
	}
	return apiKeys, nil
}

// BatchUpdateStatus 批量更新状态
func (r *apiKeyRepositoryGorm) BatchUpdateStatus(ctx context.Context, ids []int64, status entities.APIKeyStatus) error {
	if len(ids) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).Model(&entities.APIKey{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to batch update api key status: %w", result.Error)
	}

	return nil
}
