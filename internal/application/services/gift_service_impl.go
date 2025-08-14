package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/domain/services"
	"ai-api-gateway/internal/infrastructure/logger"
)

// giftServiceImpl 赠送服务实现
type giftServiceImpl struct {
	giftRepo       repositories.GiftRecordRepository
	giftRuleRepo   repositories.GiftRuleRepository
	userRepo       repositories.UserRepository
	transactionSvc services.TransactionService
	logger         logger.Logger
}

// NewGiftService 创建赠送服务实例
func NewGiftService(
	giftRepo repositories.GiftRecordRepository,
	giftRuleRepo repositories.GiftRuleRepository,
	userRepo repositories.UserRepository,
	transactionSvc services.TransactionService,
	logger logger.Logger,
) services.GiftService {
	return &giftServiceImpl{
		giftRepo:       giftRepo,
		giftRuleRepo:   giftRuleRepo,
		userRepo:       userRepo,
		transactionSvc: transactionSvc,
		logger:         logger,
	}
}

// CreateGift 创建赠送记录
func (s *giftServiceImpl) CreateGift(ctx context.Context, req *dto.CreateGiftRequest) (*dto.GiftResponse, error) {
	// 验证用户存在
	_, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 创建赠送记录
	var triggerEvent string
	if req.TriggerEvent != nil {
		triggerEvent = *req.TriggerEvent
	}

	record := &entities.GiftRecord{
		UserID:       req.UserID,
		Amount:       req.Amount,
		GiftType:     req.GiftType,
		TriggerEvent: triggerEvent,
		RelatedID:    req.RelatedID,
		RuleID:       req.RuleID,
		Reason:       req.Reason,
		Status:       entities.GiftStatusPending,
	}

	if err := s.giftRepo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create gift record: %w", err)
	}

	// 如果是手动赠送，立即处理
	if req.GiftType == entities.GiftTypeManual {
		if err := s.ProcessGift(ctx, record.ID); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"gift_id": record.ID,
				"user_id": req.UserID,
				"error":   err.Error(),
			}).Error("Failed to process manual gift")
		}
	}

	return s.toGiftResponse(record), nil
}

// ProcessGift 处理赠送（发放到用户余额）
func (s *giftServiceImpl) ProcessGift(ctx context.Context, giftID int64) error {
	// 获取赠送记录
	record, err := s.giftRepo.GetByID(ctx, giftID)
	if err != nil {
		return fmt.Errorf("gift record not found: %w", err)
	}

	// 检查状态
	if !record.IsPending() {
		s.logger.WithField("gift_id", giftID).Warn("Gift is not pending")
		return nil // 已处理过，直接返回成功
	}

	// 更新用户余额
	err = s.transactionSvc.UpdateUserBalance(
		ctx,
		record.UserID,
		record.Amount,
		entities.TransactionTypeGift,
		stringPtr("gift_record"),
		&record.ID,
		fmt.Sprintf("赠送到账：%s", record.Reason),
	)

	if err != nil {
		// 更新赠送记录状态为失败
		record.Status = entities.GiftStatusFailed
		if updateErr := s.giftRepo.Update(ctx, record); updateErr != nil {
			s.logger.WithField("gift_id", giftID).Error("Failed to update gift status to failed")
		}
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// 更新赠送记录状态为成功
	record.Status = entities.GiftStatusSuccess
	now := time.Now()
	record.ProcessedAt = &now

	if err := s.giftRepo.Update(ctx, record); err != nil {
		s.logger.WithField("gift_id", giftID).Error("Failed to update gift status to success")
		return fmt.Errorf("failed to update gift record: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"gift_id": giftID,
		"user_id": record.UserID,
		"amount":  record.Amount,
		"type":    record.GiftType,
	}).Info("Gift processed successfully")

	return nil
}

// ProcessGiftsByTrigger 根据触发事件处理赠送
func (s *giftServiceImpl) ProcessGiftsByTrigger(ctx context.Context, userID int64, triggerEvent string, relatedID *int64, baseAmount *float64) error {
	// 获取匹配的赠送规则
	rules, err := s.giftRuleRepo.GetByTriggerEvent(ctx, triggerEvent)
	if err != nil {
		return fmt.Errorf("failed to get gift rules: %w", err)
	}

	for _, rule := range rules {
		// 检查规则是否有效
		if !rule.IsActive() {
			continue
		}

		// 检查条件是否满足
		if !s.checkRuleConditions(ctx, rule, userID, baseAmount) {
			continue
		}

		// 计算赠送金额
		var amount float64
		if baseAmount != nil {
			amount = *baseAmount
		}
		giftAmount := rule.CalculateGiftAmount(amount)
		if giftAmount <= 0 {
			continue
		}

		// 创建赠送记录
		giftReq := &dto.CreateGiftRequest{
			UserID:       userID,
			Amount:       giftAmount,
			GiftType:     rule.Type,
			TriggerEvent: &triggerEvent,
			RelatedID:    relatedID,
			RuleID:       &rule.ID,
			Reason:       fmt.Sprintf("规则赠送：%s", rule.Name),
		}

		giftResp, err := s.CreateGift(ctx, giftReq)
		if err != nil {
			s.logger.WithFields(map[string]interface{}{
				"rule_id":       rule.ID,
				"user_id":       userID,
				"trigger_event": triggerEvent,
				"error":         err.Error(),
			}).Error("Failed to create gift from rule")
			continue
		}

		// 处理赠送
		if err := s.ProcessGift(ctx, giftResp.ID); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"gift_id": giftResp.ID,
				"rule_id": rule.ID,
				"user_id": userID,
				"error":   err.Error(),
			}).Error("Failed to process gift from rule")
		}
	}

	return nil
}

// QueryGiftRecords 查询赠送记录
func (s *giftServiceImpl) QueryGiftRecords(ctx context.Context, req *dto.QueryGiftRecordsRequest) (*dto.PaginatedGiftResponse, error) {
	offset := (req.Page - 1) * req.PageSize

	var records []*entities.GiftRecord
	var total int64
	var err error

	if req.StartTime != nil && req.EndTime != nil {
		records, total, err = s.giftRepo.GetByDateRange(ctx, req.UserID, *req.StartTime, *req.EndTime, req.PageSize, offset)
	} else if req.GiftType != nil {
		records, total, err = s.giftRepo.GetByType(ctx, req.UserID, *req.GiftType, req.PageSize, offset)
	} else if req.UserID != nil {
		records, total, err = s.giftRepo.GetByUserID(ctx, *req.UserID, req.PageSize, offset)
	} else {
		return nil, fmt.Errorf("invalid query parameters")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query gift records: %w", err)
	}

	// 转换为响应格式
	responses := make([]*dto.GiftResponse, len(records))
	for i, record := range records {
		responses[i] = s.toGiftResponse(record)
	}

	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &dto.PaginatedGiftResponse{
		Data:       responses,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetGiftRecord 获取赠送记录详情
func (s *giftServiceImpl) GetGiftRecord(ctx context.Context, userID int64, giftID int64) (*dto.GiftResponse, error) {
	record, err := s.giftRepo.GetByID(ctx, giftID)
	if err != nil {
		return nil, fmt.Errorf("gift record not found: %w", err)
	}

	if record.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return s.toGiftResponse(record), nil
}

// GetGiftRules 获取赠送规则列表
func (s *giftServiceImpl) GetGiftRules(ctx context.Context, activeOnly bool) ([]*dto.GiftRuleResponse, error) {
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
func (s *giftServiceImpl) GetGiftRule(ctx context.Context, id int64) (*dto.GiftRuleResponse, error) {
	rule, err := s.giftRuleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gift rule not found: %w", err)
	}

	return s.toGiftRuleResponse(rule), nil
}

// 辅助方法

// checkRuleConditions 检查规则条件是否满足
func (s *giftServiceImpl) checkRuleConditions(ctx context.Context, rule *entities.GiftRule, userID int64, baseAmount *float64) bool {
	if rule.Conditions == "" {
		return true // 无条件限制
	}

	// 解析条件（JSON格式）
	var conditions map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Conditions), &conditions); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"rule_id":    rule.ID,
			"conditions": rule.Conditions,
			"error":      err.Error(),
		}).Error("Failed to parse rule conditions")
		return false
	}

	// 检查最小金额条件
	if minAmount, ok := conditions["min_amount"].(float64); ok && baseAmount != nil {
		if *baseAmount < minAmount {
			return false
		}
	}

	// 检查最大金额条件
	if maxAmount, ok := conditions["max_amount"].(float64); ok && baseAmount != nil {
		if *baseAmount > maxAmount {
			return false
		}
	}

	// 可以添加更多条件检查逻辑
	// 例如：用户等级、注册时间、历史充值金额等

	return true
}

// toGiftResponse 转换为赠送响应
func (s *giftServiceImpl) toGiftResponse(record *entities.GiftRecord) *dto.GiftResponse {
	return &dto.GiftResponse{
		ID:           record.ID,
		UserID:       record.UserID,
		Amount:       record.Amount,
		GiftType:     record.GiftType,
		TriggerEvent: record.TriggerEvent,
		RelatedID:    record.RelatedID,
		RuleID:       record.RuleID,
		Reason:       record.Reason,
		Status:       record.Status,
		ProcessedAt:  record.ProcessedAt,
		CreatedAt:    record.CreatedAt,
	}
}

// toGiftRuleResponse 转换为赠送规则响应
func (s *giftServiceImpl) toGiftRuleResponse(rule *entities.GiftRule) *dto.GiftRuleResponse {
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
