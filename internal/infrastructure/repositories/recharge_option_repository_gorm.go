package repositories

import (
	"context"

	"gorm.io/gorm"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
)

// rechargeOptionRepositoryGorm RechargeOption仓储GORM实现
type rechargeOptionRepositoryGorm struct {
	db *gorm.DB
}

// NewRechargeOptionRepository 创建RechargeOption仓储实例
func NewRechargeOptionRepository(db *gorm.DB) repositories.RechargeOptionRepository {
	return &rechargeOptionRepositoryGorm{db: db}
}

// Create 创建充值选项
func (r *rechargeOptionRepositoryGorm) Create(ctx context.Context, option *entities.RechargeOption) error {
	return r.db.WithContext(ctx).Create(option).Error
}

// GetByID 根据ID获取充值选项
func (r *rechargeOptionRepositoryGorm) GetByID(ctx context.Context, id int64) (*entities.RechargeOption, error) {
	var option entities.RechargeOption
	err := r.db.WithContext(ctx).First(&option, id).Error
	if err != nil {
		return nil, err
	}
	return &option, nil
}

// GetEnabled 获取启用的充值选项列表
func (r *rechargeOptionRepositoryGorm) GetEnabled(ctx context.Context) ([]*entities.RechargeOption, error) {
	var options []*entities.RechargeOption
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("sort_order ASC, amount ASC").
		Find(&options).Error
	return options, err
}

// GetAll 获取所有充值选项
func (r *rechargeOptionRepositoryGorm) GetAll(ctx context.Context) ([]*entities.RechargeOption, error) {
	var options []*entities.RechargeOption
	err := r.db.WithContext(ctx).
		Order("sort_order ASC, amount ASC").
		Find(&options).Error
	return options, err
}

// Update 更新充值选项
func (r *rechargeOptionRepositoryGorm) Update(ctx context.Context, option *entities.RechargeOption) error {
	return r.db.WithContext(ctx).Save(option).Error
}

// Delete 删除充值选项
func (r *rechargeOptionRepositoryGorm) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entities.RechargeOption{}, id).Error
}

// UpdateStatus 更新充值选项状态
func (r *rechargeOptionRepositoryGorm) UpdateStatus(ctx context.Context, id int64, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&entities.RechargeOption{}).
		Where("id = ?", id).
		Update("enabled", enabled).Error
}
