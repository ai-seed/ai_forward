package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
)

// ModelProviderRepositoryImpl 模型厂商仓库实现
type ModelProviderRepositoryImpl struct {
	db *gorm.DB
}

// NewModelProviderRepository 创建模型厂商仓库实例
func NewModelProviderRepository(db *gorm.DB) repositories.ModelProviderRepository {
	return &ModelProviderRepositoryImpl{db: db}
}

// Create 创建厂商
func (r *ModelProviderRepositoryImpl) Create(ctx context.Context, provider *entities.ModelProvider) error {
	if err := r.db.WithContext(ctx).Create(provider).Error; err != nil {
		return fmt.Errorf("failed to create model provider: %w", err)
	}
	return nil
}

// GetByID 根据ID获取厂商
func (r *ModelProviderRepositoryImpl) GetByID(ctx context.Context, id int64) (*entities.ModelProvider, error) {
	var provider entities.ModelProvider
	if err := r.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get model provider by id: %w", err)
	}
	return &provider, nil
}

// GetByName 根据名称获取厂商
func (r *ModelProviderRepositoryImpl) GetByName(ctx context.Context, name string) (*entities.ModelProvider, error) {
	var provider entities.ModelProvider
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get model provider by name: %w", err)
	}
	return &provider, nil
}

// GetAll 获取所有厂商
func (r *ModelProviderRepositoryImpl) GetAll(ctx context.Context) ([]entities.ModelProvider, error) {
	var providers []entities.ModelProvider
	if err := r.db.WithContext(ctx).Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to get all model providers: %w", err)
	}
	return providers, nil
}

// GetActive 获取所有活跃厂商
func (r *ModelProviderRepositoryImpl) GetActive(ctx context.Context) ([]entities.ModelProvider, error) {
	var providers []entities.ModelProvider
	if err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to get active model providers: %w", err)
	}
	return providers, nil
}

// GetSorted 获取排序后的厂商列表
func (r *ModelProviderRepositoryImpl) GetSorted(ctx context.Context) ([]entities.ModelProvider, error) {
	var providers []entities.ModelProvider
	if err := r.db.WithContext(ctx).Where("status = ?", "active").Order("sort_order ASC, display_name ASC").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to get sorted model providers: %w", err)
	}
	return providers, nil
}

// Update 更新厂商
func (r *ModelProviderRepositoryImpl) Update(ctx context.Context, provider *entities.ModelProvider) error {
	if err := r.db.WithContext(ctx).Save(provider).Error; err != nil {
		return fmt.Errorf("failed to update model provider: %w", err)
	}
	return nil
}

// Delete 删除厂商
func (r *ModelProviderRepositoryImpl) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).Delete(&entities.ModelProvider{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete model provider: %w", err)
	}
	return nil
}

// UpdateSortOrder 更新排序顺序
func (r *ModelProviderRepositoryImpl) UpdateSortOrder(ctx context.Context, id int64, sortOrder int) error {
	if err := r.db.WithContext(ctx).Model(&entities.ModelProvider{}).Where("id = ?", id).Update("sort_order", sortOrder).Error; err != nil {
		return fmt.Errorf("failed to update sort order: %w", err)
	}
	return nil
}
