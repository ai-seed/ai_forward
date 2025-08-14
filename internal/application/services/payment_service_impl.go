package services

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/domain/services"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/google/uuid"
)

// paymentServiceImpl 支付服务实现
type paymentServiceImpl struct {
	rechargeRepo       repositories.RechargeRecordRepository
	rechargeOptionRepo repositories.RechargeOptionRepository
	paymentMethodRepo  repositories.PaymentMethodRepository
	transactionRepo    repositories.TransactionRepository
	userRepo           repositories.UserRepository
	transactionSvc     services.TransactionService
	logger             logger.Logger
}

// NewPaymentService 创建支付服务实例
func NewPaymentService(
	rechargeRepo repositories.RechargeRecordRepository,
	rechargeOptionRepo repositories.RechargeOptionRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	transactionRepo repositories.TransactionRepository,
	userRepo repositories.UserRepository,
	transactionSvc services.TransactionService,
	logger logger.Logger,
) services.PaymentService {
	return &paymentServiceImpl{
		rechargeRepo:       rechargeRepo,
		rechargeOptionRepo: rechargeOptionRepo,
		paymentMethodRepo:  paymentMethodRepo,
		transactionRepo:    transactionRepo,
		userRepo:           userRepo,
		transactionSvc:     transactionSvc,
		logger:             logger,
	}
}

// CreateRechargeOrder 创建充值订单
func (s *paymentServiceImpl) CreateRechargeOrder(ctx context.Context, userID int64, req *dto.CreateRechargeRequest) (*dto.RechargeResponse, error) {
	// 验证用户存在
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 根据支付方式ID查询支付方式信息
	paymentMethod, err := s.paymentMethodRepo.GetByID(ctx, req.PaymentMethodID)
	if err != nil {
		return nil, fmt.Errorf("payment method not found: %w", err)
	}

	// 验证支付方式状态
	if paymentMethod.Status != entities.PaymentMethodStatusActive {
		return nil, fmt.Errorf("payment method is not active")
	}

	// 验证充值金额（根据支付方式限制）
	if req.Amount < paymentMethod.MinAmount {
		return nil, fmt.Errorf("amount must be at least %.2f", paymentMethod.MinAmount)
	}
	if req.Amount > paymentMethod.MaxAmount {
		return nil, fmt.Errorf("amount cannot exceed %.2f", paymentMethod.MaxAmount)
	}

	// 计算手续费和实际到账金额
	fee := req.Amount*paymentMethod.FeeRate + paymentMethod.FixedFee
	actualAmount := req.Amount - fee

	// 生成订单号
	orderNo := s.generateOrderNo()

	// 创建充值记录，记录完整的支付方式和服务商信息快照
	record := &entities.RechargeRecord{
		UserID:            userID,
		OrderNo:           orderNo,
		Amount:            req.Amount,
		ActualAmount:      actualAmount,
		PaymentMethodID:   paymentMethod.ID,
		PaymentMethodCode: paymentMethod.Code,
		ProviderID:        paymentMethod.ProviderID,
		PaymentMethod:     paymentMethod.Code, // 兼容字段
		PaymentProvider:   paymentMethod.Name, // 兼容字段，使用支付方式名称
		Status:            entities.RechargeStatusPending,
		ExpiredAt:         s.calculateExpiredTime(),
	}

	if err := s.rechargeRepo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create recharge record: %w", err)
	}

	// 生成支付链接（这里可以集成第三方支付）
	paymentURL := s.generatePaymentURL(record, req)
	if paymentURL != "" {
		record.PaymentURL = &paymentURL
		if err := s.rechargeRepo.Update(ctx, record); err != nil {
			s.logger.WithField("order_no", orderNo).Error("Failed to update payment URL")
		}
	}

	return s.toRechargeResponse(record), nil
}

// GetRechargeOrder 获取充值订单
func (s *paymentServiceImpl) GetRechargeOrder(ctx context.Context, userID int64, orderID int64) (*dto.RechargeResponse, error) {
	record, err := s.rechargeRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("recharge order not found: %w", err)
	}

	if record.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return s.toRechargeResponse(record), nil
}

// GetRechargeOrderByNo 根据订单号获取充值订单
func (s *paymentServiceImpl) GetRechargeOrderByNo(ctx context.Context, orderNo string) (*dto.RechargeResponse, error) {
	record, err := s.rechargeRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, fmt.Errorf("recharge order not found: %w", err)
	}

	return s.toRechargeResponse(record), nil
}

// ProcessPaymentCallback 处理支付回调
func (s *paymentServiceImpl) ProcessPaymentCallback(ctx context.Context, req *dto.PaymentCallbackRequest) error {
	// 获取充值记录
	record, err := s.rechargeRepo.GetByOrderNo(ctx, req.OrderNo)
	if err != nil {
		return fmt.Errorf("recharge order not found: %w", err)
	}

	// 检查订单状态
	if !record.IsPending() {
		s.logger.WithField("order_no", req.OrderNo).Warn("Order is not pending")
		return nil // 已处理过，直接返回成功
	}

	// 验证签名（这里需要根据具体的支付提供商实现）
	if !s.verifySignature(req) {
		return fmt.Errorf("invalid signature")
	}

	// 验证金额
	if req.Amount != record.Amount {
		return fmt.Errorf("amount mismatch: expected=%.2f, actual=%.2f", record.Amount, req.Amount)
	}

	// 更新充值记录状态
	if req.Status == "success" {
		record.Status = entities.RechargeStatusSuccess
		record.PaymentID = &req.PaymentID
		record.PaidAt = req.PaidAt
		if record.PaidAt == nil {
			now := time.Now()
			record.PaidAt = &now
		}

		// 更新用户余额
		if err := s.transactionSvc.UpdateUserBalance(
			ctx,
			record.UserID,
			record.ActualAmount,
			entities.TransactionTypeRecharge,
			stringPtr("recharge_record"),
			&record.ID,
			fmt.Sprintf("充值到账，订单号：%s", record.OrderNo),
		); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"order_no": req.OrderNo,
				"user_id":  record.UserID,
				"amount":   record.ActualAmount,
				"error":    err.Error(),
			}).Error("Failed to update user balance")
			return fmt.Errorf("failed to update user balance: %w", err)
		}

		s.logger.WithFields(map[string]interface{}{
			"order_no": req.OrderNo,
			"user_id":  record.UserID,
			"amount":   record.ActualAmount,
		}).Info("Recharge completed successfully")

	} else {
		record.Status = entities.RechargeStatusFailed
	}

	if err := s.rechargeRepo.Update(ctx, record); err != nil {
		return fmt.Errorf("failed to update recharge record: %w", err)
	}

	return nil
}

// CancelRechargeOrder 取消充值订单
func (s *paymentServiceImpl) CancelRechargeOrder(ctx context.Context, userID int64, orderID int64) error {
	record, err := s.rechargeRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("recharge order not found: %w", err)
	}

	if record.UserID != userID {
		return fmt.Errorf("access denied")
	}

	if !record.CanCancel() {
		return fmt.Errorf("order cannot be cancelled")
	}

	record.Status = entities.RechargeStatusCancelled
	return s.rechargeRepo.Update(ctx, record)
}

// QueryRechargeRecords 查询充值记录
func (s *paymentServiceImpl) QueryRechargeRecords(ctx context.Context, req *dto.QueryRechargeRecordsRequest) (*dto.PaginatedRechargeResponse, error) {
	offset := (req.Page - 1) * req.PageSize

	var records []*entities.RechargeRecord
	var total int64
	var err error

	if req.StartTime != nil && req.EndTime != nil {
		records, total, err = s.rechargeRepo.GetByDateRange(ctx, req.UserID, *req.StartTime, *req.EndTime, req.PageSize, offset)
	} else {
		if req.UserID != nil {
			records, total, err = s.rechargeRepo.GetByUserID(ctx, *req.UserID, req.PageSize, offset)
		} else {
			// 管理员查询所有记录的逻辑需要在仓储层实现
			return nil, fmt.Errorf("admin query not implemented yet")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query recharge records: %w", err)
	}

	// 转换为响应格式
	responses := make([]*dto.RechargeResponse, len(records))
	for i, record := range records {
		responses[i] = s.toRechargeResponse(record)
	}

	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &dto.PaginatedRechargeResponse{
		Data:       responses,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetRechargeOptions 获取充值金额选项列表
func (s *paymentServiceImpl) GetRechargeOptions(ctx context.Context) ([]*dto.RechargeOptionResponse, error) {
	options, err := s.rechargeOptionRepo.GetEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recharge options: %w", err)
	}

	// 转换为DTO
	result := make([]*dto.RechargeOptionResponse, len(options))
	for i, option := range options {
		result[i] = &dto.RechargeOptionResponse{
			ID:          option.ID,
			Amount:      option.Amount,
			DisplayText: option.DisplayText,
			Tag:         option.Tag,
			TagColor:    option.TagColor,
			BonusAmount: option.BonusAmount,
			BonusText:   option.BonusText,
			TotalAmount: option.GetTotalAmount(),
		}
	}

	return result, nil
}

// GetPaymentMethods 获取支付方式列表
func (s *paymentServiceImpl) GetPaymentMethods(ctx context.Context, activeOnly bool) ([]*dto.PaymentMethodResponse, error) {
	methods, err := s.paymentMethodRepo.GetWithProvider(ctx, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment methods: %w", err)
	}

	// 转换为DTO
	result := make([]*dto.PaymentMethodResponse, len(methods))
	for i, method := range methods {
		result[i] = s.toPaymentMethodResponse(method)
	}

	return result, nil
}

// 辅助方法

// generateOrderNo 生成订单号
func (s *paymentServiceImpl) generateOrderNo() string {
	return fmt.Sprintf("R%d%s", time.Now().Unix(), uuid.New().String()[:8])
}

// calculateExpiredTime 计算订单过期时间
func (s *paymentServiceImpl) calculateExpiredTime() *time.Time {
	expired := time.Now().Add(30 * time.Minute) // 30分钟过期
	return &expired
}

// generatePaymentURL 生成支付链接
func (s *paymentServiceImpl) generatePaymentURL(record *entities.RechargeRecord, req *dto.CreateRechargeRequest) string {
	// 这里需要根据具体的支付提供商实现
	// 暂时返回空字符串
	return ""
}

// verifySignature 验证签名
func (s *paymentServiceImpl) verifySignature(req *dto.PaymentCallbackRequest) bool {
	// 这里需要根据具体的支付提供商实现签名验证
	// 暂时返回true
	return true
}

// toRechargeResponse 转换为充值响应
func (s *paymentServiceImpl) toRechargeResponse(record *entities.RechargeRecord) *dto.RechargeResponse {
	return &dto.RechargeResponse{
		ID:                record.ID,
		OrderNo:           record.OrderNo,
		Amount:            record.Amount,
		ActualAmount:      record.ActualAmount,
		PaymentMethodID:   record.PaymentMethodID,
		PaymentMethodCode: record.PaymentMethodCode,
		ProviderID:        record.ProviderID,
		PaymentMethod:     record.PaymentMethod,   // 兼容字段
		PaymentProvider:   record.PaymentProvider, // 兼容字段
		Status:            record.Status,
		PaymentURL:        record.PaymentURL,
		ExpiredAt:         record.ExpiredAt,
		CreatedAt:         record.CreatedAt,
	}
}

// toPaymentMethodResponse 转换为支付方式响应
func (s *paymentServiceImpl) toPaymentMethodResponse(method *entities.PaymentMethod) *dto.PaymentMethodResponse {
	response := &dto.PaymentMethodResponse{
		ID:          method.ID,
		Code:        method.Code,
		Name:        method.Name,
		DisplayName: method.DisplayName,
		MinAmount:   method.MinAmount,
		MaxAmount:   method.MaxAmount,
		FeeRate:     method.FeeRate,
		FixedFee:    method.FixedFee,
		Status:      string(method.Status),
		SortOrder:   method.SortOrder,
	}

	// 处理可选字段，确保不返回nil指针
	if method.Icon != nil && *method.Icon != "" {
		response.Icon = method.Icon
	}
	if method.Description != nil && *method.Description != "" {
		response.Description = method.Description
	}

	return response
}
