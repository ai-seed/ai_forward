package dto

import (
	"time"

	"ai-api-gateway/internal/domain/entities"
)

// CreateRechargeRequest 创建充值订单请求
type CreateRechargeRequest struct {
	Amount          float64 `json:"amount" validate:"required,gt=0"`            // 充值金额
	PaymentMethodID int64   `json:"payment_method_id" validate:"required,gt=0"` // 支付方式ID
	ReturnURL       *string `json:"return_url,omitempty"`                       // 支付成功返回URL
	NotifyURL       *string `json:"notify_url,omitempty"`                       // 支付回调URL
}

// RechargeResponse 充值订单响应
type RechargeResponse struct {
	ID                int64                   `json:"id"`
	OrderNo           string                  `json:"order_no"`
	Amount            float64                 `json:"amount"`
	ActualAmount      float64                 `json:"actual_amount"`
	PaymentMethodID   int64                   `json:"payment_method_id"`   // 支付方式ID
	PaymentMethodCode string                  `json:"payment_method_code"` // 支付方式代码
	ProviderID        int64                   `json:"provider_id"`         // 服务商ID
	PaymentMethod     string                  `json:"payment_method"`      // 兼容字段
	PaymentProvider   string                  `json:"payment_provider"`    // 兼容字段
	Status            entities.RechargeStatus `json:"status"`
	PaymentURL        *string                 `json:"payment_url,omitempty"`
	ExpiredAt         *time.Time              `json:"expired_at,omitempty"`
	CreatedAt         time.Time               `json:"created_at"`
}

// PaymentMethodResponse 支付方式响应
type PaymentMethodResponse struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Icon        *string `json:"icon,omitempty"`
	Description *string `json:"description,omitempty"`
	MinAmount   float64 `json:"min_amount"`
	MaxAmount   float64 `json:"max_amount"`
	FeeRate     float64 `json:"fee_rate"`
	FixedFee    float64 `json:"fixed_fee"`
	Status      string  `json:"status"`
	SortOrder   int     `json:"sort_order"`
}

// PaymentProviderResponse 支付服务商响应
type PaymentProviderResponse struct {
	ID     int64  `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// PaymentCallbackRequest 支付回调请求
type PaymentCallbackRequest struct {
	OrderNo   string                 `json:"order_no"`
	PaymentID string                 `json:"payment_id"`
	Status    string                 `json:"status"`
	Amount    float64                `json:"amount"`
	PaidAt    *time.Time             `json:"paid_at,omitempty"`
	ExtraData map[string]interface{} `json:"extra_data,omitempty"`
	Signature string                 `json:"signature"`
}

// CreateGiftRequest 创建赠送请求
type CreateGiftRequest struct {
	UserID       int64             `json:"user_id" validate:"required"`
	Amount       float64           `json:"amount" validate:"required,gt=0"`
	GiftType     entities.GiftType `json:"gift_type" validate:"required"`
	TriggerEvent *string           `json:"trigger_event,omitempty"`
	RelatedID    *int64            `json:"related_id,omitempty"`
	RuleID       *int64            `json:"rule_id,omitempty"`
	Reason       string            `json:"reason" validate:"required"`
}

// GiftResponse 赠送响应
type GiftResponse struct {
	ID           int64               `json:"id"`
	UserID       int64               `json:"user_id"`
	Amount       float64             `json:"amount"`
	GiftType     entities.GiftType   `json:"gift_type"`
	TriggerEvent string              `json:"trigger_event"`
	RelatedID    *int64              `json:"related_id,omitempty"`
	RuleID       *int64              `json:"rule_id,omitempty"`
	Reason       string              `json:"reason"`
	Status       entities.GiftStatus `json:"status"`
	ProcessedAt  *time.Time          `json:"processed_at,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
}

// TransactionResponse 交易记录响应
type TransactionResponse struct {
	ID            int64                    `json:"id"`
	TransactionNo string                   `json:"transaction_no"`
	Type          entities.TransactionType `json:"type"`
	Amount        float64                  `json:"amount"`
	BalanceBefore float64                  `json:"balance_before"`
	BalanceAfter  float64                  `json:"balance_after"`
	RelatedType   *string                  `json:"related_type,omitempty"`
	RelatedID     *int64                   `json:"related_id,omitempty"`
	Description   string                   `json:"description"`
	CreatedAt     time.Time                `json:"created_at"`
}

// RechargeOptionResponse 充值金额选项响应
type RechargeOptionResponse struct {
	ID          int64   `json:"id"`
	Amount      float64 `json:"amount"`       // 充值金额
	DisplayText string  `json:"display_text"` // 显示文本
	Tag         string  `json:"tag"`          // 标签
	TagColor    string  `json:"tag_color"`    // 标签颜色
	BonusAmount float64 `json:"bonus_amount"` // 赠送金额
	BonusText   string  `json:"bonus_text"`   // 赠送说明
	TotalAmount float64 `json:"total_amount"` // 总金额
}

// GiftRuleResponse 赠送规则响应
type GiftRuleResponse struct {
	ID           int64                   `json:"id"`
	Name         string                  `json:"name"`
	Type         entities.GiftType       `json:"type"`
	TriggerEvent string                  `json:"trigger_event"`
	Conditions   string                  `json:"conditions"`
	GiftAmount   *float64                `json:"gift_amount,omitempty"`
	GiftRate     *float64                `json:"gift_rate,omitempty"`
	MaxGift      *float64                `json:"max_gift,omitempty"`
	Status       entities.GiftRuleStatus `json:"status"`
	StartTime    *time.Time              `json:"start_time,omitempty"`
	EndTime      *time.Time              `json:"end_time,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
}

// CreateGiftRuleRequest 创建赠送规则请求
type CreateGiftRuleRequest struct {
	Name         string                  `json:"name" validate:"required"`
	Type         entities.GiftType       `json:"type" validate:"required"`
	TriggerEvent string                  `json:"trigger_event" validate:"required"`
	Conditions   *string                 `json:"conditions,omitempty"`
	GiftAmount   *float64                `json:"gift_amount,omitempty" validate:"omitempty,gt=0"`
	GiftRate     *float64                `json:"gift_rate,omitempty" validate:"omitempty,gte=0,lte=1"`
	MaxGift      *float64                `json:"max_gift,omitempty" validate:"omitempty,gt=0"`
	Status       entities.GiftRuleStatus `json:"status" validate:"required"`
	StartTime    *time.Time              `json:"start_time,omitempty"`
	EndTime      *time.Time              `json:"end_time,omitempty"`
}

// UpdateGiftRuleRequest 更新赠送规则请求
type UpdateGiftRuleRequest struct {
	Name         *string                  `json:"name,omitempty"`
	TriggerEvent *string                  `json:"trigger_event,omitempty"`
	Conditions   *string                  `json:"conditions,omitempty"`
	GiftAmount   *float64                 `json:"gift_amount,omitempty" validate:"omitempty,gt=0"`
	GiftRate     *float64                 `json:"gift_rate,omitempty" validate:"omitempty,gte=0,lte=1"`
	MaxGift      *float64                 `json:"max_gift,omitempty" validate:"omitempty,gt=0"`
	Status       *entities.GiftRuleStatus `json:"status,omitempty"`
	StartTime    *time.Time               `json:"start_time,omitempty"`
	EndTime      *time.Time               `json:"end_time,omitempty"`
}

// BalanceResponse 余额响应
type BalanceResponse struct {
	UserID        int64   `json:"user_id"`
	Balance       float64 `json:"balance"`
	FrozenBalance float64 `json:"frozen_balance"`
	TotalBalance  float64 `json:"total_balance"`
}

// TransactionSummaryResponse 交易汇总响应
type TransactionSummaryResponse struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	NetAmount    float64 `json:"net_amount"`
	Count        int64   `json:"count"`
}

// QueryTransactionsRequest 查询交易记录请求
type QueryTransactionsRequest struct {
	UserID    *int64                    `json:"user_id,omitempty"`
	Type      *entities.TransactionType `json:"type,omitempty"`
	StartTime *time.Time                `json:"start_time,omitempty"`
	EndTime   *time.Time                `json:"end_time,omitempty"`
	Page      int                       `json:"page" validate:"min=1"`
	PageSize  int                       `json:"page_size" validate:"min=1,max=100"`
}

// QueryRechargeRecordsRequest 查询充值记录请求
type QueryRechargeRecordsRequest struct {
	UserID    *int64                   `json:"user_id,omitempty"`
	Status    *entities.RechargeStatus `json:"status,omitempty"`
	Method    *string                  `json:"method,omitempty"`
	StartTime *time.Time               `json:"start_time,omitempty"`
	EndTime   *time.Time               `json:"end_time,omitempty"`
	Page      int                      `json:"page" validate:"min=1"`
	PageSize  int                      `json:"page_size" validate:"min=1,max=100"`
}

// QueryGiftRecordsRequest 查询赠送记录请求
type QueryGiftRecordsRequest struct {
	UserID    *int64               `json:"user_id,omitempty"`
	GiftType  *entities.GiftType   `json:"gift_type,omitempty"`
	Status    *entities.GiftStatus `json:"status,omitempty"`
	StartTime *time.Time           `json:"start_time,omitempty"`
	EndTime   *time.Time           `json:"end_time,omitempty"`
	Page      int                  `json:"page" validate:"min=1"`
	PageSize  int                  `json:"page_size" validate:"min=1,max=100"`
}

// PaginatedRechargeResponse 充值记录分页响应
type PaginatedRechargeResponse struct {
	Data       []*RechargeResponse `json:"data"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

// PaginatedGiftResponse 赠送记录分页响应
type PaginatedGiftResponse struct {
	Data       []*GiftResponse `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// PaginatedTransactionResponse 交易记录分页响应
type PaginatedTransactionResponse struct {
	Data       []*TransactionResponse `json:"data"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}
