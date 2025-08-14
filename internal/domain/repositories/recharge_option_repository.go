package repositories

import (
	"context"

	"ai-api-gateway/internal/domain/entities"
)

// RechargeOptionRepository 充值选项仓储接口
type RechargeOptionRepository interface {
	// Create 创建充值选项
	Create(ctx context.Context, option *entities.RechargeOption) error

	// GetByID 根据ID获取充值选项
	GetByID(ctx context.Context, id int64) (*entities.RechargeOption, error)

	// GetEnabled 获取启用的充值选项列表
	GetEnabled(ctx context.Context) ([]*entities.RechargeOption, error)

	// GetAll 获取所有充值选项
	GetAll(ctx context.Context) ([]*entities.RechargeOption, error)

	// Update 更新充值选项
	Update(ctx context.Context, option *entities.RechargeOption) error

	// Delete 删除充值选项
	Delete(ctx context.Context, id int64) error

	// UpdateStatus 更新充值选项状态
	UpdateStatus(ctx context.Context, id int64, enabled bool) error
}
