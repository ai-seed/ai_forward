package repositories

import (
	"context"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"

	"gorm.io/gorm"
)

// giftRuleRepository 赠送规则仓储实现
type giftRuleRepository struct {
	db *gorm.DB
}

// NewGiftRuleRepository 创建赠送规则仓储
func NewGiftRuleRepository(db *gorm.DB) repositories.GiftRuleRepository {
	return &giftRuleRepository{db: db}
}

// Create 创建赠送规则
func (r *giftRuleRepository) Create(ctx context.Context, rule *entities.GiftRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID 根据ID获取赠送规则
func (r *giftRuleRepository) GetByID(ctx context.Context, id int64) (*entities.GiftRule, error) {
	var rule entities.GiftRule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// GetAll 获取所有赠送规则
func (r *giftRuleRepository) GetAll(ctx context.Context) ([]*entities.GiftRule, error) {
	var rules []*entities.GiftRule
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&rules).Error
	return rules, err
}

// GetActive 获取启用的赠送规则
func (r *giftRuleRepository) GetActive(ctx context.Context) ([]*entities.GiftRule, error) {
	var rules []*entities.GiftRule
	err := r.db.WithContext(ctx).
		Where("status = ?", entities.GiftRuleStatusActive).
		Order("created_at DESC").
		Find(&rules).Error
	return rules, err
}

// GetByType 根据类型获取赠送规则
func (r *giftRuleRepository) GetByType(ctx context.Context, giftType entities.GiftType) ([]*entities.GiftRule, error) {
	var rules []*entities.GiftRule
	err := r.db.WithContext(ctx).
		Where("type = ? AND status = ?", giftType, entities.GiftRuleStatusActive).
		Order("created_at DESC").
		Find(&rules).Error
	return rules, err
}

// GetByTriggerEvent 根据触发事件获取赠送规则
func (r *giftRuleRepository) GetByTriggerEvent(ctx context.Context, triggerEvent string) ([]*entities.GiftRule, error) {
	var rules []*entities.GiftRule
	err := r.db.WithContext(ctx).
		Where("trigger_event = ? AND status = ?", triggerEvent, entities.GiftRuleStatusActive).
		Order("created_at DESC").
		Find(&rules).Error
	return rules, err
}

// Update 更新赠送规则
func (r *giftRuleRepository) Update(ctx context.Context, rule *entities.GiftRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// UpdateStatus 更新赠送规则状态
func (r *giftRuleRepository) UpdateStatus(ctx context.Context, id int64, status entities.GiftRuleStatus) error {
	return r.db.WithContext(ctx).
		Model(&entities.GiftRule{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// Delete 删除赠送规则
func (r *giftRuleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entities.GiftRule{}, id).Error
}
