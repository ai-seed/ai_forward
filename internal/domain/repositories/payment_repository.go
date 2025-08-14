package repositories

import (
	"context"
	"time"

	"ai-api-gateway/internal/domain/entities"
)

// RechargeRecordRepository 充值记录仓储接口
type RechargeRecordRepository interface {
	// Create 创建充值记录
	Create(ctx context.Context, record *entities.RechargeRecord) error

	// GetByID 根据ID获取充值记录
	GetByID(ctx context.Context, id int64) (*entities.RechargeRecord, error)

	// GetByOrderNo 根据订单号获取充值记录
	GetByOrderNo(ctx context.Context, orderNo string) (*entities.RechargeRecord, error)

	// GetByUserID 根据用户ID获取充值记录列表
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*entities.RechargeRecord, int64, error)

	// Update 更新充值记录
	Update(ctx context.Context, record *entities.RechargeRecord) error

	// UpdateStatus 更新充值状态
	UpdateStatus(ctx context.Context, id int64, status entities.RechargeStatus) error

	// GetPendingRecords 获取待处理的充值记录
	GetPendingRecords(ctx context.Context, expiredBefore *time.Time) ([]*entities.RechargeRecord, error)

	// GetByDateRange 根据日期范围获取充值记录
	GetByDateRange(ctx context.Context, userID *int64, startTime, endTime time.Time, limit, offset int) ([]*entities.RechargeRecord, int64, error)

	// QueryWithFilters 根据多个条件查询充值记录
	QueryWithFilters(ctx context.Context, filters *RechargeQueryFilters, limit, offset int) ([]*entities.RechargeRecord, int64, error)
}

// RechargeQueryFilters 充值记录查询过滤条件
type RechargeQueryFilters struct {
	UserID    *int64                   `json:"user_id,omitempty"`
	OrderNo   *string                  `json:"order_no,omitempty"`
	Status    *entities.RechargeStatus `json:"status,omitempty"`
	Method    *string                  `json:"method,omitempty"`
	StartTime *time.Time               `json:"start_time,omitempty"`
	EndTime   *time.Time               `json:"end_time,omitempty"`
}

// GiftRecordRepository 赠送记录仓储接口
type GiftRecordRepository interface {
	// Create 创建赠送记录
	Create(ctx context.Context, record *entities.GiftRecord) error

	// GetByID 根据ID获取赠送记录
	GetByID(ctx context.Context, id int64) (*entities.GiftRecord, error)

	// GetByUserID 根据用户ID获取赠送记录列表
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*entities.GiftRecord, int64, error)

	// GetByType 根据赠送类型获取记录
	GetByType(ctx context.Context, userID *int64, giftType entities.GiftType, limit, offset int) ([]*entities.GiftRecord, int64, error)

	// Update 更新赠送记录
	Update(ctx context.Context, record *entities.GiftRecord) error

	// UpdateStatus 更新赠送状态
	UpdateStatus(ctx context.Context, id int64, status entities.GiftStatus) error

	// GetPendingRecords 获取待处理的赠送记录
	GetPendingRecords(ctx context.Context) ([]*entities.GiftRecord, error)

	// GetByRelatedID 根据关联ID获取赠送记录
	GetByRelatedID(ctx context.Context, relatedID int64, giftType entities.GiftType) ([]*entities.GiftRecord, error)

	// GetByDateRange 根据日期范围获取赠送记录
	GetByDateRange(ctx context.Context, userID *int64, startTime, endTime time.Time, limit, offset int) ([]*entities.GiftRecord, int64, error)
}

// TransactionRepository 交易流水仓储接口
type TransactionRepository interface {
	// Create 创建交易记录
	Create(ctx context.Context, transaction *entities.Transaction) error

	// GetByID 根据ID获取交易记录
	GetByID(ctx context.Context, id int64) (*entities.Transaction, error)

	// GetByTransactionNo 根据交易号获取交易记录
	GetByTransactionNo(ctx context.Context, transactionNo string) (*entities.Transaction, error)

	// GetByUserID 根据用户ID获取交易记录列表
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*entities.Transaction, int64, error)

	// GetByType 根据交易类型获取记录
	GetByType(ctx context.Context, userID *int64, transactionType entities.TransactionType, limit, offset int) ([]*entities.Transaction, int64, error)

	// GetByDateRange 根据日期范围获取交易记录
	GetByDateRange(ctx context.Context, userID *int64, startTime, endTime time.Time, limit, offset int) ([]*entities.Transaction, int64, error)

	// GetUserBalance 获取用户最新余额（从最后一笔交易记录）
	GetUserBalance(ctx context.Context, userID int64) (float64, error)

	// GetUserTransactionSummary 获取用户交易汇总
	GetUserTransactionSummary(ctx context.Context, userID int64, startTime, endTime time.Time) (*TransactionSummary, error)
}

// GiftRuleRepository 赠送规则仓储接口
type GiftRuleRepository interface {
	// Create 创建赠送规则
	Create(ctx context.Context, rule *entities.GiftRule) error

	// GetByID 根据ID获取赠送规则
	GetByID(ctx context.Context, id int64) (*entities.GiftRule, error)

	// GetAll 获取所有赠送规则
	GetAll(ctx context.Context) ([]*entities.GiftRule, error)

	// GetActive 获取启用的赠送规则
	GetActive(ctx context.Context) ([]*entities.GiftRule, error)

	// GetByType 根据类型获取赠送规则
	GetByType(ctx context.Context, giftType entities.GiftType) ([]*entities.GiftRule, error)

	// GetByTriggerEvent 根据触发事件获取赠送规则
	GetByTriggerEvent(ctx context.Context, triggerEvent string) ([]*entities.GiftRule, error)

	// Update 更新赠送规则
	Update(ctx context.Context, rule *entities.GiftRule) error

	// UpdateStatus 更新赠送规则状态
	UpdateStatus(ctx context.Context, id int64, status entities.GiftRuleStatus) error

	// Delete 删除赠送规则
	Delete(ctx context.Context, id int64) error
}

// PaymentMethodRepository 支付方式仓储接口
type PaymentMethodRepository interface {
	// Create 创建支付方式
	Create(ctx context.Context, method *entities.PaymentMethod) error

	// GetByID 根据ID获取支付方式
	GetByID(ctx context.Context, id int64) (*entities.PaymentMethod, error)

	// GetByCode 根据代码获取支付方式
	GetByCode(ctx context.Context, code string) (*entities.PaymentMethod, error)

	// GetActive 获取启用的支付方式列表
	GetActive(ctx context.Context) ([]*entities.PaymentMethod, error)

	// GetAll 获取所有支付方式
	GetAll(ctx context.Context) ([]*entities.PaymentMethod, error)

	// Update 更新支付方式
	Update(ctx context.Context, method *entities.PaymentMethod) error

	// UpdateStatus 更新支付方式状态
	UpdateStatus(ctx context.Context, id int64, status entities.PaymentMethodStatus) error

	// Delete 删除支付方式
	Delete(ctx context.Context, id int64) error

	// GetWithProvider 获取包含服务商信息的支付方式列表
	GetWithProvider(ctx context.Context, activeOnly bool) ([]*entities.PaymentMethod, error)
}

// PaymentProviderRepository 支付服务商仓储接口
type PaymentProviderRepository interface {
	// Create 创建支付服务商
	Create(ctx context.Context, provider *entities.PaymentProvider) error

	// GetByID 根据ID获取支付服务商
	GetByID(ctx context.Context, id int64) (*entities.PaymentProvider, error)

	// GetByCode 根据代码获取支付服务商
	GetByCode(ctx context.Context, code string) (*entities.PaymentProvider, error)

	// GetActive 获取启用的支付服务商列表
	GetActive(ctx context.Context) ([]*entities.PaymentProvider, error)

	// GetAll 获取所有支付服务商
	GetAll(ctx context.Context) ([]*entities.PaymentProvider, error)

	// Update 更新支付服务商
	Update(ctx context.Context, provider *entities.PaymentProvider) error

	// UpdateStatus 更新支付服务商状态
	UpdateStatus(ctx context.Context, id int64, status entities.PaymentMethodStatus) error

	// Delete 删除支付服务商
	Delete(ctx context.Context, id int64) error
}

// TransactionSummary 交易汇总信息
type TransactionSummary struct {
	TotalIncome  float64 `json:"total_income"`  // 总收入
	TotalExpense float64 `json:"total_expense"` // 总支出
	NetAmount    float64 `json:"net_amount"`    // 净额
	Count        int64   `json:"count"`         // 交易笔数
}

// PaymentFilter 支付查询过滤器
type PaymentFilter struct {
	UserID    *int64                   `json:"user_id,omitempty"`
	Status    *entities.RechargeStatus `json:"status,omitempty"`
	Method    *string                  `json:"method,omitempty"`
	StartTime *time.Time               `json:"start_time,omitempty"`
	EndTime   *time.Time               `json:"end_time,omitempty"`
	MinAmount *float64                 `json:"min_amount,omitempty"`
	MaxAmount *float64                 `json:"max_amount,omitempty"`
}

// GiftFilter 赠送查询过滤器
type GiftFilter struct {
	UserID    *int64               `json:"user_id,omitempty"`
	GiftType  *entities.GiftType   `json:"gift_type,omitempty"`
	Status    *entities.GiftStatus `json:"status,omitempty"`
	StartTime *time.Time           `json:"start_time,omitempty"`
	EndTime   *time.Time           `json:"end_time,omitempty"`
	MinAmount *float64             `json:"min_amount,omitempty"`
	MaxAmount *float64             `json:"max_amount,omitempty"`
}

// TransactionFilter 交易查询过滤器
type TransactionFilter struct {
	UserID    *int64                    `json:"user_id,omitempty"`
	Type      *entities.TransactionType `json:"type,omitempty"`
	StartTime *time.Time                `json:"start_time,omitempty"`
	EndTime   *time.Time                `json:"end_time,omitempty"`
	MinAmount *float64                  `json:"min_amount,omitempty"`
	MaxAmount *float64                  `json:"max_amount,omitempty"`
}
