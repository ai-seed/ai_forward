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

// paymentMethodRepositoryGorm GORM支付方式仓储实现
type paymentMethodRepositoryGorm struct {
	db    *gorm.DB
	cache *redis.CacheService
}

// NewPaymentMethodRepositoryGorm 创建GORM支付方式仓储
func NewPaymentMethodRepositoryGorm(db *gorm.DB, cache *redis.CacheService) repositories.PaymentMethodRepository {
	return &paymentMethodRepositoryGorm{
		db:    db,
		cache: cache,
	}
}

// 缓存键常量
const (
	CacheKeyPaymentMethods       = "payment_methods:all"
	CacheKeyActivePaymentMethods = "payment_methods:active"
	CacheKeyPaymentMethodByID    = "payment_method:id:%d"
	CacheKeyPaymentMethodByCode  = "payment_method:code:%s"
	CacheKeyPaymentMethodsTTL    = 30 * time.Minute
)

// Create 创建支付方式
func (r *paymentMethodRepositoryGorm) Create(ctx context.Context, method *entities.PaymentMethod) error {
	if err := r.db.WithContext(ctx).Create(method).Error; err != nil {
		return fmt.Errorf("failed to create payment method: %w", err)
	}

	// 清除相关缓存
	r.clearCache(ctx)
	return nil
}

// GetByID 根据ID获取支付方式
func (r *paymentMethodRepositoryGorm) GetByID(ctx context.Context, id int64) (*entities.PaymentMethod, error) {
	// 尝试从缓存获取
	if r.cache != nil {
		cacheKey := fmt.Sprintf(CacheKeyPaymentMethodByID, id)
		var cachedMethod entities.PaymentMethod
		if err := r.cache.Get(ctx, cacheKey, &cachedMethod); err == nil {
			return &cachedMethod, nil
		}
	}

	var method entities.PaymentMethod
	if err := r.db.WithContext(ctx).Preload("Provider").Where("id = ?", id).First(&method).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment method not found")
		}
		return nil, fmt.Errorf("failed to get payment method by id: %w", err)
	}

	// 缓存结果
	if r.cache != nil {
		cacheKey := fmt.Sprintf(CacheKeyPaymentMethodByID, id)
		r.cache.Set(ctx, cacheKey, &method, CacheKeyPaymentMethodsTTL)
	}

	return &method, nil
}

// GetByCode 根据代码获取支付方式
func (r *paymentMethodRepositoryGorm) GetByCode(ctx context.Context, code string) (*entities.PaymentMethod, error) {
	// 尝试从缓存获取
	if r.cache != nil {
		cacheKey := fmt.Sprintf(CacheKeyPaymentMethodByCode, code)
		var cachedMethod entities.PaymentMethod
		if err := r.cache.Get(ctx, cacheKey, &cachedMethod); err == nil {
			return &cachedMethod, nil
		}
	}

	var method entities.PaymentMethod
	if err := r.db.WithContext(ctx).Preload("Provider").Where("code = ?", code).First(&method).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment method not found")
		}
		return nil, fmt.Errorf("failed to get payment method by code: %w", err)
	}

	// 缓存结果
	if r.cache != nil {
		cacheKey := fmt.Sprintf(CacheKeyPaymentMethodByCode, code)
		r.cache.Set(ctx, cacheKey, &method, CacheKeyPaymentMethodsTTL)
	}

	return &method, nil
}

// GetActive 获取启用的支付方式列表
func (r *paymentMethodRepositoryGorm) GetActive(ctx context.Context) ([]*entities.PaymentMethod, error) {
	// 尝试从缓存获取
	if r.cache != nil {
		var cachedMethods []*entities.PaymentMethod
		if err := r.cache.Get(ctx, CacheKeyActivePaymentMethods, &cachedMethods); err == nil {
			return cachedMethods, nil
		}
	}

	var methods []*entities.PaymentMethod
	if err := r.db.WithContext(ctx).
		Where("status = ?", entities.PaymentMethodStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&methods).Error; err != nil {
		return nil, fmt.Errorf("failed to get active payment methods: %w", err)
	}

	// 缓存结果
	if r.cache != nil {
		r.cache.Set(ctx, CacheKeyActivePaymentMethods, methods, CacheKeyPaymentMethodsTTL)
	}

	return methods, nil
}

// GetAll 获取所有支付方式
func (r *paymentMethodRepositoryGorm) GetAll(ctx context.Context) ([]*entities.PaymentMethod, error) {
	// 尝试从缓存获取
	if r.cache != nil {
		var cachedMethods []*entities.PaymentMethod
		if err := r.cache.Get(ctx, CacheKeyPaymentMethods, &cachedMethods); err == nil {
			return cachedMethods, nil
		}
	}

	var methods []*entities.PaymentMethod
	if err := r.db.WithContext(ctx).
		Order("sort_order ASC, id ASC").
		Find(&methods).Error; err != nil {
		return nil, fmt.Errorf("failed to get all payment methods: %w", err)
	}

	// 缓存结果
	if r.cache != nil {
		r.cache.Set(ctx, CacheKeyPaymentMethods, methods, CacheKeyPaymentMethodsTTL)
	}

	return methods, nil
}

// Update 更新支付方式
func (r *paymentMethodRepositoryGorm) Update(ctx context.Context, method *entities.PaymentMethod) error {
	method.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).Save(method)
	if result.Error != nil {
		return fmt.Errorf("failed to update payment method: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("payment method not found")
	}

	// 清除相关缓存
	r.clearCache(ctx)
	return nil
}

// UpdateStatus 更新支付方式状态
func (r *paymentMethodRepositoryGorm) UpdateStatus(ctx context.Context, id int64, status entities.PaymentMethodStatus) error {
	result := r.db.WithContext(ctx).Model(&entities.PaymentMethod{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update payment method status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("payment method not found")
	}

	// 清除相关缓存
	r.clearCache(ctx)
	return nil
}

// Delete 删除支付方式
func (r *paymentMethodRepositoryGorm) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&entities.PaymentMethod{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete payment method: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("payment method not found")
	}

	// 清除相关缓存
	r.clearCache(ctx)
	return nil
}

// GetWithProvider 获取支付方式列表（不再加载服务商信息）
func (r *paymentMethodRepositoryGorm) GetWithProvider(ctx context.Context, activeOnly bool) ([]*entities.PaymentMethod, error) {
	if activeOnly {
		return r.GetActive(ctx)
	}
	return r.GetAll(ctx)
}

// clearCache 清除相关缓存
func (r *paymentMethodRepositoryGorm) clearCache(ctx context.Context) {
	if r.cache != nil {
		r.cache.Delete(ctx, CacheKeyPaymentMethods)
		r.cache.Delete(ctx, CacheKeyActivePaymentMethods)
		// 注意：这里没有清除具体ID和Code的缓存，因为我们不知道具体的键
		// 在实际应用中，可以考虑使用缓存标签或者其他策略来批量清除
	}
}
