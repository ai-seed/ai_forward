package services

import (
	"context"
	"fmt"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/domain/services"
	"ai-api-gateway/internal/infrastructure/logger"
)

// giftRuleServiceImpl 赠送规则管理服务实现
type giftRuleServiceImpl struct {
	giftRuleRepo repositories.GiftRuleRepository
	logger       logger.Logger
}

// NewGiftRuleService 创建赠送规则管理服务实例
func NewGiftRuleService(
	giftRuleRepo repositories.GiftRuleRepository,
	logger logger.Logger,
) services.GiftRuleService {
	return &giftRuleServiceImpl{
		giftRuleRepo: giftRuleRepo,
		logger:       logger,
	}
}

// CreateGiftRule 创建赠送规则
func (s *giftRuleServiceImpl) CreateGiftRule(ctx context.Context, req *dto.CreateGiftRuleRequest) (*dto.GiftRuleResponse, error) {
	// 验证规则参数
	if err := s.validateGiftRule(req); err != nil {
		return nil, err
	}

	// 创建赠送规则实体
	var conditions string
	if req.Conditions != nil {
		conditions = *req.Conditions
	}

	rule := &entities.GiftRule{
		Name:         req.Name,
		Type:         req.Type,
		TriggerEvent: req.TriggerEvent,
		Conditions:   conditions,
		GiftAmount:   req.GiftAmount,
		GiftRate:     req.GiftRate,
		MaxGift:      req.MaxGift,
		Status:       req.Status,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
	}

	if err := s.giftRuleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create gift rule: %w", err)
	}

	return s.toGiftRuleResponse(rule), nil
}

// UpdateGiftRule 更新赠送规则
func (s *giftRuleServiceImpl) UpdateGiftRule(ctx context.Context, id int64, req *dto.UpdateGiftRuleRequest) (*dto.GiftRuleResponse, error) {
	// 获取现有规则
	rule, err := s.giftRuleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gift rule not found: %w", err)
	}

	// 更新字段
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.TriggerEvent != nil {
		rule.TriggerEvent = *req.TriggerEvent
	}
	if req.Conditions != nil {
		rule.Conditions = *req.Conditions
	}
	if req.GiftAmount != nil {
		rule.GiftAmount = req.GiftAmount
	}
	if req.GiftRate != nil {
		rule.GiftRate = req.GiftRate
	}
	if req.MaxGift != nil {
		rule.MaxGift = req.MaxGift
	}
	if req.Status != nil {
		rule.Status = *req.Status
	}
	if req.StartTime != nil {
		rule.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		rule.EndTime = req.EndTime
	}

	// 验证更新后的规则
	if err := s.validateGiftRuleEntity(rule); err != nil {
		return nil, err
	}

	if err := s.giftRuleRepo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update gift rule: %w", err)
	}

	return s.toGiftRuleResponse(rule), nil
}

// DeleteGiftRule 删除赠送规则
func (s *giftRuleServiceImpl) DeleteGiftRule(ctx context.Context, id int64) error {
	// 检查规则是否存在
	_, err := s.giftRuleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("gift rule not found: %w", err)
	}

	// TODO: 检查是否有关联的赠送记录，如果有则不允许删除

	return s.giftRuleRepo.Delete(ctx, id)
}

// GetGiftRules 获取赠送规则列表
func (s *giftRuleServiceImpl) GetGiftRules(ctx context.Context, activeOnly bool) ([]*dto.GiftRuleResponse, error) {
	var rules []*entities.GiftRule
	var err error

	if activeOnly {
		rules, err = s.giftRuleRepo.GetActive(ctx)
	} else {
		rules, err = s.giftRuleRepo.GetAll(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get gift rules: %w", err)
	}

	responses := make([]*dto.GiftRuleResponse, len(rules))
	for i, rule := range rules {
		responses[i] = s.toGiftRuleResponse(rule)
	}

	return responses, nil
}

// GetGiftRule 获取赠送规则详情
func (s *giftRuleServiceImpl) GetGiftRule(ctx context.Context, id int64) (*dto.GiftRuleResponse, error) {
	rule, err := s.giftRuleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gift rule not found: %w", err)
	}

	return s.toGiftRuleResponse(rule), nil
}

// UpdateGiftRuleStatus 更新赠送规则状态
func (s *giftRuleServiceImpl) UpdateGiftRuleStatus(ctx context.Context, id int64, status entities.GiftRuleStatus) error {
	// 检查规则是否存在
	_, err := s.giftRuleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("gift rule not found: %w", err)
	}

	return s.giftRuleRepo.UpdateStatus(ctx, id, status)
}

// GetActiveRulesByTrigger 根据触发事件获取活跃规则
func (s *giftRuleServiceImpl) GetActiveRulesByTrigger(ctx context.Context, triggerEvent string) ([]*dto.GiftRuleResponse, error) {
	rules, err := s.giftRuleRepo.GetByTriggerEvent(ctx, triggerEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to get gift rules by trigger: %w", err)
	}

	responses := make([]*dto.GiftRuleResponse, len(rules))
	for i, rule := range rules {
		responses[i] = s.toGiftRuleResponse(rule)
	}

	return responses, nil
}

// 辅助方法

// validateGiftRule 验证赠送规则请求
func (s *giftRuleServiceImpl) validateGiftRule(req *dto.CreateGiftRuleRequest) error {
	// 检查赠送金额和比例至少有一个
	if req.GiftAmount == nil && req.GiftRate == nil {
		return fmt.Errorf("either gift_amount or gift_rate must be specified")
	}

	// 检查赠送比例范围
	if req.GiftRate != nil && (*req.GiftRate < 0 || *req.GiftRate > 1) {
		return fmt.Errorf("gift_rate must be between 0 and 1")
	}

	// 检查时间范围
	if req.StartTime != nil && req.EndTime != nil && req.EndTime.Before(*req.StartTime) {
		return fmt.Errorf("end_time must be after start_time")
	}

	return nil
}

// validateGiftRuleEntity 验证赠送规则实体
func (s *giftRuleServiceImpl) validateGiftRuleEntity(rule *entities.GiftRule) error {
	// 检查赠送金额和比例至少有一个
	if rule.GiftAmount == nil && rule.GiftRate == nil {
		return fmt.Errorf("either gift_amount or gift_rate must be specified")
	}

	// 检查赠送比例范围
	if rule.GiftRate != nil && (*rule.GiftRate < 0 || *rule.GiftRate > 1) {
		return fmt.Errorf("gift_rate must be between 0 and 1")
	}

	// 检查时间范围
	if rule.StartTime != nil && rule.EndTime != nil && rule.EndTime.Before(*rule.StartTime) {
		return fmt.Errorf("end_time must be after start_time")
	}

	return nil
}

// toGiftRuleResponse 转换为赠送规则响应
func (s *giftRuleServiceImpl) toGiftRuleResponse(rule *entities.GiftRule) *dto.GiftRuleResponse {
	return &dto.GiftRuleResponse{
		ID:           rule.ID,
		Name:         rule.Name,
		Type:         rule.Type,
		TriggerEvent: rule.TriggerEvent,
		Conditions:   rule.Conditions,
		GiftAmount:   rule.GiftAmount,
		GiftRate:     rule.GiftRate,
		MaxGift:      rule.MaxGift,
		Status:       rule.Status,
		StartTime:    rule.StartTime,
		EndTime:      rule.EndTime,
		CreatedAt:    rule.CreatedAt,
	}
}
