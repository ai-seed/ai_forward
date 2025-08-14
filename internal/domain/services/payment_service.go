package services

import (
	"context"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
)

// PaymentService 支付服务接口
type PaymentService interface {
	// CreateRechargeOrder 创建充值订单
	CreateRechargeOrder(ctx context.Context, userID int64, req *dto.CreateRechargeRequest) (*dto.RechargeResponse, error)

	// GetRechargeOrder 获取充值订单
	GetRechargeOrder(ctx context.Context, userID int64, orderID int64) (*dto.RechargeResponse, error)

	// GetRechargeOrderByNo 根据订单号获取充值订单
	GetRechargeOrderByNo(ctx context.Context, orderNo string) (*dto.RechargeResponse, error)

	// ProcessPaymentCallback 处理支付回调
	ProcessPaymentCallback(ctx context.Context, req *dto.PaymentCallbackRequest) error

	// CancelRechargeOrder 取消充值订单
	CancelRechargeOrder(ctx context.Context, userID int64, orderID int64) error

	// QueryRechargeRecords 查询充值记录
	QueryRechargeRecords(ctx context.Context, req *dto.QueryRechargeRecordsRequest) (*dto.PaginatedRechargeResponse, error)

	// GetRechargeOptions 获取充值金额选项列表
	GetRechargeOptions(ctx context.Context) ([]*dto.RechargeOptionResponse, error)

	// GetPaymentMethods 获取支付方式列表（包含服务商信息）
	GetPaymentMethods(ctx context.Context, activeOnly bool) ([]*dto.PaymentMethodResponse, error)

	// GetPaymentPage 获取支付页面信息
	GetPaymentPage(ctx context.Context, orderNo string) (*dto.PaymentPageResponse, error)

	// ProcessPayment 处理支付确认（更新订单状态并充值）
	ProcessPayment(ctx context.Context, orderNo string) (*dto.PaymentResultResponse, error)

	// SimulatePaymentSuccess 模拟支付成功（用于测试）
	SimulatePaymentSuccess(ctx context.Context, orderNo string) error
}

// GiftService 赠送服务接口
type GiftService interface {
	// CreateGift 创建赠送记录
	CreateGift(ctx context.Context, req *dto.CreateGiftRequest) (*dto.GiftResponse, error)

	// ProcessGift 处理赠送（发放到用户余额）
	ProcessGift(ctx context.Context, giftID int64) error

	// ProcessGiftsByTrigger 根据触发事件处理赠送
	ProcessGiftsByTrigger(ctx context.Context, userID int64, triggerEvent string, relatedID *int64, baseAmount *float64) error

	// QueryGiftRecords 查询赠送记录
	QueryGiftRecords(ctx context.Context, req *dto.QueryGiftRecordsRequest) (*dto.PaginatedGiftResponse, error)

	// GetGiftRecord 获取赠送记录详情
	GetGiftRecord(ctx context.Context, userID int64, giftID int64) (*dto.GiftResponse, error)

	// GetGiftRules 获取赠送规则列表
	GetGiftRules(ctx context.Context, activeOnly bool) ([]*dto.GiftRuleResponse, error)

	// GetGiftRule 获取赠送规则详情
	GetGiftRule(ctx context.Context, id int64) (*dto.GiftRuleResponse, error)
}

// TransactionService 交易服务接口
type TransactionService interface {
	// CreateTransaction 创建交易记录
	CreateTransaction(ctx context.Context, userID int64, transactionType entities.TransactionType, amount float64, relatedType *string, relatedID *int64, description string) (*dto.TransactionResponse, error)

	// QueryTransactions 查询交易记录
	QueryTransactions(ctx context.Context, req *dto.QueryTransactionsRequest) (*dto.PaginatedTransactionResponse, error)

	// GetTransaction 获取交易记录详情
	GetTransaction(ctx context.Context, userID int64, transactionID int64) (*dto.TransactionResponse, error)

	// GetUserBalance 获取用户余额
	GetUserBalance(ctx context.Context, userID int64) (*dto.BalanceResponse, error)

	// GetTransactionSummary 获取交易汇总
	GetTransactionSummary(ctx context.Context, userID int64, startTime, endTime *time.Time) (*dto.TransactionSummaryResponse, error)

	// UpdateUserBalance 更新用户余额（原子操作）
	UpdateUserBalance(ctx context.Context, userID int64, amount float64, transactionType entities.TransactionType, relatedType *string, relatedID *int64, description string) error
}

// GiftRuleService 赠送规则管理服务接口
type GiftRuleService interface {
	// CreateGiftRule 创建赠送规则
	CreateGiftRule(ctx context.Context, req *dto.CreateGiftRuleRequest) (*dto.GiftRuleResponse, error)

	// UpdateGiftRule 更新赠送规则
	UpdateGiftRule(ctx context.Context, id int64, req *dto.UpdateGiftRuleRequest) (*dto.GiftRuleResponse, error)

	// DeleteGiftRule 删除赠送规则
	DeleteGiftRule(ctx context.Context, id int64) error

	// GetGiftRules 获取赠送规则列表
	GetGiftRules(ctx context.Context, activeOnly bool) ([]*dto.GiftRuleResponse, error)

	// GetGiftRule 获取赠送规则详情
	GetGiftRule(ctx context.Context, id int64) (*dto.GiftRuleResponse, error)

	// UpdateGiftRuleStatus 更新赠送规则状态
	UpdateGiftRuleStatus(ctx context.Context, id int64, status entities.GiftRuleStatus) error

	// GetActiveRulesByTrigger 根据触发事件获取活跃规则
	GetActiveRulesByTrigger(ctx context.Context, triggerEvent string) ([]*dto.GiftRuleResponse, error)
}
