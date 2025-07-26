package repositories

import (
	"context"

	"ai-api-gateway/internal/domain/entities"
)

// ModelProviderRepository 模型厂商仓库接口
type ModelProviderRepository interface {
	// Create 创建厂商
	Create(ctx context.Context, provider *entities.ModelProvider) error

	// GetByID 根据ID获取厂商
	GetByID(ctx context.Context, id int64) (*entities.ModelProvider, error)

	// GetByName 根据名称获取厂商
	GetByName(ctx context.Context, name string) (*entities.ModelProvider, error)

	// GetAll 获取所有厂商
	GetAll(ctx context.Context) ([]entities.ModelProvider, error)

	// GetActive 获取所有活跃厂商
	GetActive(ctx context.Context) ([]entities.ModelProvider, error)

	// GetSorted 获取排序后的厂商列表
	GetSorted(ctx context.Context) ([]entities.ModelProvider, error)

	// Update 更新厂商
	Update(ctx context.Context, provider *entities.ModelProvider) error

	// Delete 删除厂商
	Delete(ctx context.Context, id int64) error

	// UpdateSortOrder 更新排序顺序
	UpdateSortOrder(ctx context.Context, id int64, sortOrder int) error
}
