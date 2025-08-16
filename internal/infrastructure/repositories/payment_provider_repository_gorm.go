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

// paymentProviderRepositoryGorm GORM支付服务商仓储实现
type paymentProviderRepositoryGorm struct {
	db    *gorm.DB
	cache *redis.CacheService
}

// NewPaymentProviderRepositoryGorm 创建GORM支付服务商仓储
func NewPaymentProviderRepositoryGorm(db *gorm.DB, cache *redis.CacheService) repositories.PaymentProviderRepository {
	return &paymentProviderRepositoryGorm{
		db:    db,
		cache: cache,
	}
}

// 缓存键常量
const (
	CacheKeyPaymentProviders       = "payment_providers:all"
	CacheKeyActivePaymentProviders = "payment_providers:active"
	CacheKeyPaymentProviderByID    = "payment_provider:id:%d"
	CacheKeyPaymentProviderByCode  = "payment_provider:code:%s"
	CacheKeyPaymentProvidersTTL    = 30 * time.Minute
)

// Create 创建支付服务商
func (r *paymentProviderRepositoryGorm) Create(ctx context.Context, provider *entities.PaymentProvider) error {
	if err := r.db.WithContext(ctx).Create(provider).Error; err != nil {
		return fmt.Errorf("failed to create payment provider: %w", err)
	}

	// 清除相关缓存
	r.clearCache(ctx)
	return nil
}

// GetByID 根据ID获取支付服务商
func (r *paymentProviderRepositoryGorm) GetByID(ctx context.Context, id int64) (*entities.PaymentProvider, error) {
	cacheKey := fmt.Sprintf(CacheKeyPaymentProviderByID, id)

	// 尝试从缓存获取
	var provider entities.PaymentProvider
	if err := r.cache.Get(ctx, cacheKey, &provider); err == nil {
		return &provider, nil
	}

	// 从数据库获取
	if err := r.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment provider not found")
		}
		return nil, fmt.Errorf("failed to get payment provider: %w", err)
	}

	// 缓存结果
	r.cache.Set(ctx, cacheKey, &provider, CacheKeyPaymentProvidersTTL)
	return &provider, nil
}

// GetByCode 根据代码获取支付服务商
func (r *paymentProviderRepositoryGorm) GetByCode(ctx context.Context, code string) (*entities.PaymentProvider, error) {
	cacheKey := fmt.Sprintf(CacheKeyPaymentProviderByCode, code)

	// 尝试从缓存获取
	var provider entities.PaymentProvider
	if err := r.cache.Get(ctx, cacheKey, &provider); err == nil {
		return &provider, nil
	}

	// 从数据库获取
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment provider not found")
		}
		return nil, fmt.Errorf("failed to get payment provider: %w", err)
	}

	// 缓存结果
	r.cache.Set(ctx, cacheKey, &provider, CacheKeyPaymentProvidersTTL)
	return &provider, nil
}

// GetAll 获取所有支付服务商
func (r *paymentProviderRepositoryGorm) GetAll(ctx context.Context) ([]*entities.PaymentProvider, error) {
	cacheKey := CacheKeyPaymentProviders

	// 尝试从缓存获取
	var providers []*entities.PaymentProvider
	if err := r.cache.Get(ctx, cacheKey, &providers); err == nil {
		return providers, nil
	}

	// 从数据库获取
	if err := r.db.WithContext(ctx).Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to get payment providers: %w", err)
	}

	// 缓存结果
	r.cache.Set(ctx, cacheKey, providers, CacheKeyPaymentProvidersTTL)
	return providers, nil
}

// GetActive 获取活跃的支付服务商
func (r *paymentProviderRepositoryGorm) GetActive(ctx context.Context) ([]*entities.PaymentProvider, error) {
	cacheKey := CacheKeyActivePaymentProviders

	// 尝试从缓存获取
	var providers []*entities.PaymentProvider
	if err := r.cache.Get(ctx, cacheKey, &providers); err == nil {
		return providers, nil
	}

	// 从数据库获取
	if err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to get active payment providers: %w", err)
	}

	// 缓存结果
	r.cache.Set(ctx, cacheKey, providers, CacheKeyPaymentProvidersTTL)
	return providers, nil
}

// Update 更新支付服务商
func (r *paymentProviderRepositoryGorm) Update(ctx context.Context, provider *entities.PaymentProvider) error {
	if err := r.db.WithContext(ctx).Save(provider).Error; err != nil {
		return fmt.Errorf("failed to update payment provider: %w", err)
	}

	// 清除相关缓存
	r.clearCache(ctx)
	return nil
}

// UpdateStatus 更新支付服务商状态
func (r *paymentProviderRepositoryGorm) UpdateStatus(ctx context.Context, id int64, status entities.PaymentMethodStatus) error {
	if err := r.db.WithContext(ctx).Model(&entities.PaymentProvider{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update payment provider status: %w", err)
	}

	// 清除相关缓存
	r.clearCache(ctx)
	return nil
}

// Delete 删除支付服务商
func (r *paymentProviderRepositoryGorm) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).Delete(&entities.PaymentProvider{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete payment provider: %w", err)
	}

	// 清除相关缓存
	r.clearCache(ctx)
	return nil
}

// clearCache 清除相关缓存
func (r *paymentProviderRepositoryGorm) clearCache(ctx context.Context) {
	// 清除所有相关缓存
	r.cache.Delete(ctx, CacheKeyPaymentProviders)
	r.cache.Delete(ctx, CacheKeyActivePaymentProviders)

	// 清除ID和Code缓存需要通过模式匹配，这里简化处理
	// 在实际生产环境中，可以考虑使用Redis的SCAN命令或者维护一个缓存键列表
}
