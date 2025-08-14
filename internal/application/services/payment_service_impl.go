package services

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/domain/services"
	"ai-api-gateway/internal/infrastructure/config"
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
	config             *config.Config
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
	config *config.Config,
	logger logger.Logger,
) services.PaymentService {
	return &paymentServiceImpl{
		rechargeRepo:       rechargeRepo,
		rechargeOptionRepo: rechargeOptionRepo,
		paymentMethodRepo:  paymentMethodRepo,
		transactionRepo:    transactionRepo,
		userRepo:           userRepo,
		transactionSvc:     transactionSvc,
		config:             config,
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

	// 构建过滤条件
	filters := &repositories.RechargeQueryFilters{
		UserID:    req.UserID,
		OrderNo:   req.OrderNo,
		Status:    req.Status,
		Method:    req.Method,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}

	// 使用新的过滤查询方法
	records, total, err := s.rechargeRepo.QueryWithFilters(ctx, filters, req.PageSize, offset)
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
	// 从配置中获取前端地址
	frontendBaseURL := s.config.OAuth.FrontendURL
	if frontendBaseURL == "" {
		// 如果配置为空，使用默认值
		frontendBaseURL = "http://localhost:3000"
		s.logger.Warn("Frontend URL not configured, using default: http://localhost:3000")
	}

	// 构建前端支付页面URL，包含订单信息
	paymentURL := fmt.Sprintf("%s/admin/pay?order_no=%s&amount=%.2f&method=%s",
		frontendBaseURL, record.OrderNo, record.Amount, record.PaymentMethodCode)

	s.logger.WithFields(map[string]interface{}{
		"order_no":     record.OrderNo,
		"amount":       record.Amount,
		"method":       record.PaymentMethodCode,
		"frontend_url": frontendBaseURL,
		"payment_url":  paymentURL,
	}).Info("Generated payment URL")

	return paymentURL
}

// GetPaymentPage 获取支付页面信息
func (s *paymentServiceImpl) GetPaymentPage(ctx context.Context, orderNo string) (*dto.PaymentPageResponse, error) {
	// 获取充值记录
	record, err := s.rechargeRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, fmt.Errorf("recharge order not found: %w", err)
	}

	// 检查订单是否已过期
	if record.ExpiredAt != nil && time.Now().After(*record.ExpiredAt) {
		return nil, fmt.Errorf("payment order has expired")
	}

	// 检查订单状态
	if record.Status != entities.RechargeStatusPending {
		return nil, fmt.Errorf("payment order is not pending, current status: %s", record.Status)
	}

	// 获取支付方式信息
	paymentMethod, err := s.paymentMethodRepo.GetByID(ctx, record.PaymentMethodID)
	if err != nil {
		s.logger.WithField("payment_method_id", record.PaymentMethodID).Warn("Failed to get payment method")
	}

	displayName := record.PaymentMethodCode
	if paymentMethod != nil {
		displayName = paymentMethod.DisplayName
	}

	return &dto.PaymentPageResponse{
		OrderNo:       record.OrderNo,
		Amount:        record.Amount,
		ActualAmount:  record.ActualAmount,
		PaymentMethod: record.PaymentMethodCode,
		DisplayName:   displayName,
		Status:        string(record.Status),
		ExpiredAt:     record.ExpiredAt,
		CreatedAt:     record.CreatedAt,
	}, nil
}

// ProcessPayment 处理支付确认（更新订单状态并充值）
func (s *paymentServiceImpl) ProcessPayment(ctx context.Context, orderNo string) (*dto.PaymentResultResponse, error) {
	// 获取充值记录
	record, err := s.rechargeRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, fmt.Errorf("recharge order not found: %w", err)
	}

	// 检查订单状态
	if record.Status != entities.RechargeStatusPending {
		paymentID := ""
		if record.PaymentID != nil {
			paymentID = *record.PaymentID
		}
		return &dto.PaymentResultResponse{
			OrderNo:      record.OrderNo,
			Status:       string(record.Status),
			Amount:       record.Amount,
			ActualAmount: record.ActualAmount,
			PaymentID:    paymentID,
			Message:      fmt.Sprintf("Order is already %s", record.Status),
		}, nil
	}

	// 检查订单是否已过期
	if record.ExpiredAt != nil && time.Now().After(*record.ExpiredAt) {
		// 更新订单状态为过期
		record.Status = entities.RechargeStatusExpired
		if err := s.rechargeRepo.Update(ctx, record); err != nil {
			s.logger.WithField("order_no", orderNo).Error("Failed to update expired order status")
		}
		return nil, fmt.Errorf("payment order has expired")
	}

	// 生成支付ID和时间
	paymentID := fmt.Sprintf("PAY_%d_%d", time.Now().Unix(), time.Now().Nanosecond()%10000)
	now := time.Now()

	// 构建支付回调请求，复用现有的回调处理逻辑
	callbackReq := &dto.PaymentCallbackRequest{
		OrderNo:   orderNo,
		PaymentID: paymentID,
		Status:    "success",
		Amount:    record.Amount,
		PaidAt:    &now,
		Signature: fmt.Sprintf("direct_payment_%s_%s", orderNo, paymentID),
		ExtraData: map[string]interface{}{
			"provider":   "direct_payment",
			"source":     "payment_page",
			"direct_pay": true,
		},
	}

	// 处理支付回调
	if err := s.ProcessPaymentCallback(ctx, callbackReq); err != nil {
		return nil, fmt.Errorf("failed to process payment: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"order_no":      record.OrderNo,
		"user_id":       record.UserID,
		"amount":        record.Amount,
		"actual_amount": record.ActualAmount,
		"payment_id":    paymentID,
	}).Info("Payment processed successfully")

	return &dto.PaymentResultResponse{
		OrderNo:      record.OrderNo,
		Status:       string(entities.RechargeStatusSuccess),
		Amount:       record.Amount,
		ActualAmount: record.ActualAmount,
		PaymentID:    paymentID,
		PaidAt:       now,
		Message:      "Payment processed successfully",
	}, nil
}

// SimulatePaymentSuccess 模拟支付成功
func (s *paymentServiceImpl) SimulatePaymentSuccess(ctx context.Context, orderNo string) error {
	// 生成模拟的支付回调数据
	paymentID := fmt.Sprintf("mock_payment_%d_%d", time.Now().Unix(), time.Now().Nanosecond()%10000)
	now := time.Now()

	// 获取订单信息
	record, err := s.rechargeRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return fmt.Errorf("recharge order not found: %w", err)
	}

	// 构建回调请求
	callbackReq := &dto.PaymentCallbackRequest{
		OrderNo:   orderNo,
		PaymentID: paymentID,
		Status:    "success",
		Amount:    record.Amount,
		PaidAt:    &now,
		Signature: fmt.Sprintf("mock_signature_%s_%s", orderNo, paymentID),
		ExtraData: map[string]interface{}{
			"provider":       "mock_provider",
			"transaction_id": paymentID,
			"test_mode":      true,
		},
	}

	// 处理支付回调
	return s.ProcessPaymentCallback(ctx, callbackReq)
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
