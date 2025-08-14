package entities

import (
	"time"
)

// RechargeStatus 充值状态枚举
type RechargeStatus string

const (
	RechargeStatusPending   RechargeStatus = "pending"   // 待支付
	RechargeStatusSuccess   RechargeStatus = "success"   // 支付成功
	RechargeStatusFailed    RechargeStatus = "failed"    // 支付失败
	RechargeStatusCancelled RechargeStatus = "cancelled" // 已取消
	RechargeStatusExpired   RechargeStatus = "expired"   // 已过期
)

// GiftType 赠送类型枚举
type GiftType string

const (
	GiftTypeRegister GiftType = "register" // 注册赠送
	GiftTypeRecharge GiftType = "recharge" // 充值赠送
	GiftTypeActivity GiftType = "activity" // 活动赠送
	GiftTypeManual   GiftType = "manual"   // 手动赠送
)

// GiftStatus 赠送状态枚举
type GiftStatus string

const (
	GiftStatusPending GiftStatus = "pending" // 待处理
	GiftStatusSuccess GiftStatus = "success" // 已发放
	GiftStatusFailed  GiftStatus = "failed"  // 发放失败
)

// TransactionType 交易类型枚举
type TransactionType string

const (
	TransactionTypeRecharge TransactionType = "recharge" // 充值
	TransactionTypeGift     TransactionType = "gift"     // 赠送
	TransactionTypeConsume  TransactionType = "consume"  // 消费
	TransactionTypeRefund   TransactionType = "refund"   // 退款
)

// GiftRuleStatus 赠送规则状态枚举
type GiftRuleStatus string

const (
	GiftRuleStatusActive   GiftRuleStatus = "active"   // 启用
	GiftRuleStatusInactive GiftRuleStatus = "inactive" // 禁用
)

// RechargeRecord 充值记录实体
type RechargeRecord struct {
	ID                int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID            int64          `json:"user_id" gorm:"not null;index"`
	OrderNo           string         `json:"order_no" gorm:"uniqueIndex;not null;size:64"`
	Amount            float64        `json:"amount" gorm:"type:numeric(15,6);not null"`
	ActualAmount      float64        `json:"actual_amount" gorm:"type:numeric(15,6);not null"`
	PaymentMethodID   int64          `json:"payment_method_id" gorm:"not null;index"`     // 支付方式ID
	PaymentMethodCode string         `json:"payment_method_code" gorm:"not null;size:50"` // 支付方式代码（快照）
	ProviderID        int64          `json:"provider_id" gorm:"not null;index"`           // 服务商ID（快照）
	PaymentMethod     string         `json:"payment_method" gorm:"not null;size:50"`      // 兼容字段，存储支付方式代码
	PaymentProvider   string         `json:"payment_provider" gorm:"size:50"`             // 兼容字段，存储服务商名称
	Status            RechargeStatus `json:"status" gorm:"not null;size:20;index"`
	PaymentID         *string        `json:"payment_id" gorm:"size:255"`
	PaymentURL        *string        `json:"payment_url" gorm:"size:500"`
	PaidAt            *time.Time     `json:"paid_at"`
	ExpiredAt         *time.Time     `json:"expired_at"`
	Remark            *string        `json:"remark" gorm:"type:text"`
	CreatedAt         time.Time      `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

// TableName 指定表名
func (RechargeRecord) TableName() string {
	return "recharge_records"
}

// IsCompleted 检查充值是否已完成
func (r *RechargeRecord) IsCompleted() bool {
	return r.Status == RechargeStatusSuccess
}

// IsPending 检查充值是否待处理
func (r *RechargeRecord) IsPending() bool {
	return r.Status == RechargeStatusPending
}

// CanCancel 检查是否可以取消
func (r *RechargeRecord) CanCancel() bool {
	return r.Status == RechargeStatusPending
}

// GiftRecord 赠送记录实体
type GiftRecord struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       int64      `json:"user_id" gorm:"not null;index"`
	Amount       float64    `json:"amount" gorm:"type:numeric(15,6);not null"`
	GiftType     GiftType   `json:"gift_type" gorm:"not null;size:50;index"`
	TriggerEvent string     `json:"trigger_event" gorm:"size:100"`
	RelatedID    *int64     `json:"related_id" gorm:"index"`
	RuleID       *int64     `json:"rule_id" gorm:"index"`
	Reason       string     `json:"reason" gorm:"not null;type:text"`
	Status       GiftStatus `json:"status" gorm:"not null;size:20;index"`
	ProcessedAt  *time.Time `json:"processed_at"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

// TableName 指定表名
func (GiftRecord) TableName() string {
	return "gift_records"
}

// IsCompleted 检查赠送是否已完成
func (g *GiftRecord) IsCompleted() bool {
	return g.Status == GiftStatusSuccess
}

// IsPending 检查赠送是否待处理
func (g *GiftRecord) IsPending() bool {
	return g.Status == GiftStatusPending
}

// Transaction 交易流水实体
type Transaction struct {
	ID            int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int64           `json:"user_id" gorm:"not null;index"`
	TransactionNo string          `json:"transaction_no" gorm:"uniqueIndex;not null;size:64"`
	Type          TransactionType `json:"type" gorm:"not null;size:20;index"`
	Amount        float64         `json:"amount" gorm:"type:numeric(15,6);not null"`
	BalanceBefore float64         `json:"balance_before" gorm:"type:numeric(15,6);not null"`
	BalanceAfter  float64         `json:"balance_after" gorm:"type:numeric(15,6);not null"`
	RelatedType   *string         `json:"related_type" gorm:"size:50"`
	RelatedID     *int64          `json:"related_id" gorm:"index"`
	Description   string          `json:"description" gorm:"not null;type:text"`
	CreatedAt     time.Time       `json:"created_at" gorm:"not null;autoCreateTime;index"`
}

// TableName 指定表名
func (Transaction) TableName() string {
	return "transactions"
}

// IsIncome 检查是否为收入交易
func (t *Transaction) IsIncome() bool {
	return t.Amount > 0
}

// IsExpense 检查是否为支出交易
func (t *Transaction) IsExpense() bool {
	return t.Amount < 0
}

// GiftRule 赠送规则实体
type GiftRule struct {
	ID           int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string         `json:"name" gorm:"not null;size:100"`
	Type         GiftType       `json:"type" gorm:"not null;size:50;index"`
	TriggerEvent string         `json:"trigger_event" gorm:"not null;size:100"`
	Conditions   string         `json:"conditions" gorm:"type:text"`
	GiftAmount   *float64       `json:"gift_amount" gorm:"type:numeric(15,6)"`
	GiftRate     *float64       `json:"gift_rate" gorm:"type:numeric(8,6)"`
	MaxGift      *float64       `json:"max_gift" gorm:"type:numeric(15,6)"`
	Status       GiftRuleStatus `json:"status" gorm:"not null;size:20;index"`
	StartTime    *time.Time     `json:"start_time"`
	EndTime      *time.Time     `json:"end_time"`
	CreatedAt    time.Time      `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

// TableName 指定表名
func (GiftRule) TableName() string {
	return "gift_rules"
}

// IsActive 检查规则是否启用
func (g *GiftRule) IsActive() bool {
	if g.Status != GiftRuleStatusActive {
		return false
	}

	now := time.Now()
	if g.StartTime != nil && now.Before(*g.StartTime) {
		return false
	}
	if g.EndTime != nil && now.After(*g.EndTime) {
		return false
	}

	return true
}

// CalculateGiftAmount 计算赠送金额
func (g *GiftRule) CalculateGiftAmount(baseAmount float64) float64 {
	var giftAmount float64

	// 固定金额赠送
	if g.GiftAmount != nil {
		giftAmount = *g.GiftAmount
	}

	// 按比例赠送
	if g.GiftRate != nil {
		giftAmount += baseAmount * (*g.GiftRate)
	}

	// 限制最大赠送金额
	if g.MaxGift != nil && giftAmount > *g.MaxGift {
		giftAmount = *g.MaxGift
	}

	return giftAmount
}

// PaymentMethodStatus 支付方式状态
type PaymentMethodStatus string

const (
	PaymentMethodStatusActive   PaymentMethodStatus = "active"   // 启用
	PaymentMethodStatusInactive PaymentMethodStatus = "inactive" // 禁用
)

// PaymentMethod 支付方式实体（前端显示层）
type PaymentMethod struct {
	ID          int64               `json:"id" gorm:"primaryKey;autoIncrement"`
	Code        string              `json:"code" gorm:"uniqueIndex;not null;size:50"` // alipay, wechat, bank
	Name        string              `json:"name" gorm:"not null;size:100"`            // 内部名称
	DisplayName string              `json:"display_name" gorm:"not null;size:100"`    // 前端显示名称：支付宝、微信支付
	Icon        *string             `json:"icon" gorm:"size:255"`                     // 图标URL
	Description *string             `json:"description" gorm:"size:500"`              // 描述信息
	ProviderID  int64               `json:"provider_id" gorm:"not null;index"`        // 关联的服务商ID（一对一关系）
	MinAmount   float64             `json:"min_amount" gorm:"type:numeric(15,6);not null;default:0"`
	MaxAmount   float64             `json:"max_amount" gorm:"type:numeric(15,6);not null;default:50000"`
	FeeRate     float64             `json:"fee_rate" gorm:"type:numeric(8,6);default:0"`   // 手续费率
	FixedFee    float64             `json:"fixed_fee" gorm:"type:numeric(15,6);default:0"` // 固定手续费
	Status      PaymentMethodStatus `json:"status" gorm:"not null;size:20;index"`
	SortOrder   int                 `json:"sort_order" gorm:"default:0"`
	Config      *string             `json:"config" gorm:"type:text"` // 支付方式特定配置
	// 多语言字段
	DisplayNameZh *string   `json:"display_name_zh" gorm:"column:display_name_zh;size:100"` // 中文显示名称
	DisplayNameEn *string   `json:"display_name_en" gorm:"column:display_name_en;size:100"` // 英文显示名称
	DisplayNameJa *string   `json:"display_name_ja" gorm:"column:display_name_ja;size:100"` // 日文显示名称
	DescriptionZh *string   `json:"description_zh" gorm:"column:description_zh;size:500"`   // 中文描述
	DescriptionEn *string   `json:"description_en" gorm:"column:description_en;size:500"`   // 英文描述
	DescriptionJa *string   `json:"description_ja" gorm:"column:description_ja;size:500"`   // 日文描述
	CreatedAt     time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"not null;autoUpdateTime"`

	// 关联关系
	Provider *PaymentProvider `json:"provider,omitempty" gorm:"foreignKey:ProviderID"`
}

// PaymentProvider 支付服务商实体（实际对接的第三方）
type PaymentProvider struct {
	ID        int64               `json:"id" gorm:"primaryKey;autoIncrement"`
	Code      string              `json:"code" gorm:"uniqueIndex;not null;size:50"` // yeepay, huifu, alipay_direct
	Name      string              `json:"name" gorm:"not null;size:100"`            // 易宝支付、汇付天下、支付宝直连
	Type      string              `json:"type" gorm:"not null;size:50"`             // gateway, direct, bank
	ApiUrl    string              `json:"api_url" gorm:"not null;size:255"`         // API地址
	Config    string              `json:"config" gorm:"type:text"`                  // 配置信息（JSON格式）
	Status    PaymentMethodStatus `json:"status" gorm:"not null;size:20;index"`
	Priority  int                 `json:"priority" gorm:"default:0"` // 优先级，数字越小优先级越高
	CreatedAt time.Time           `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time           `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

// PaymentChannel 支付渠道实体（支付方式+服务商的组合）
type PaymentChannel struct {
	ID           int64               `json:"id" gorm:"primaryKey;autoIncrement"`
	MethodID     int64               `json:"method_id" gorm:"not null;index"`                    // 支付方式ID
	ProviderID   int64               `json:"provider_id" gorm:"not null;index"`                  // 服务商ID
	ChannelCode  string              `json:"channel_code" gorm:"not null;size:100"`              // 渠道代码：alipay_yeepay
	ChannelName  string              `json:"channel_name" gorm:"not null;size:100"`              // 渠道名称：支付宝-易宝支付
	FeeRate      float64             `json:"fee_rate" gorm:"type:numeric(8,6);default:0"`        // 手续费率
	FixedFee     float64             `json:"fixed_fee" gorm:"type:numeric(15,6);default:0"`      // 固定手续费
	MinAmount    float64             `json:"min_amount" gorm:"type:numeric(15,6);default:0"`     // 最小金额
	MaxAmount    float64             `json:"max_amount" gorm:"type:numeric(15,6);default:50000"` // 最大金额
	DailyLimit   *float64            `json:"daily_limit" gorm:"type:numeric(15,6)"`              // 日限额
	MonthlyLimit *float64            `json:"monthly_limit" gorm:"type:numeric(15,6)"`            // 月限额
	Status       PaymentMethodStatus `json:"status" gorm:"not null;size:20;index"`
	Weight       int                 `json:"weight" gorm:"default:100"` // 权重，用于负载均衡
	Config       *string             `json:"config" gorm:"type:text"`   // 渠道特定配置
	CreatedAt    time.Time           `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time           `json:"updated_at" gorm:"not null;autoUpdateTime"`

	// 关联关系
	Method   *PaymentMethod   `json:"method,omitempty" gorm:"foreignKey:MethodID"`
	Provider *PaymentProvider `json:"provider,omitempty" gorm:"foreignKey:ProviderID"`
}

// TableName 指定表名
func (PaymentMethod) TableName() string {
	return "payment_methods"
}

func (PaymentProvider) TableName() string {
	return "payment_providers"
}

func (PaymentChannel) TableName() string {
	return "payment_channels"
}

// IsActive 检查支付方式是否启用
func (p *PaymentMethod) IsActive() bool {
	return p.Status == PaymentMethodStatusActive
}

func (p *PaymentProvider) IsActive() bool {
	return p.Status == PaymentMethodStatusActive
}

func (p *PaymentChannel) IsActive() bool {
	return p.Status == PaymentMethodStatusActive
}

// CalculateFee 计算手续费
func (p *PaymentChannel) CalculateFee(amount float64) float64 {
	return amount*p.FeeRate + p.FixedFee
}

// IsAmountValid 检查金额是否在允许范围内
func (p *PaymentChannel) IsAmountValid(amount float64) bool {
	return amount >= p.MinAmount && amount <= p.MaxAmount
}

// CalculateFee 计算手续费
func (p *PaymentMethod) CalculateFee(amount float64) float64 {
	return amount*p.FeeRate + p.FixedFee
}

// IsAmountValid 检查金额是否在允许范围内
func (p *PaymentMethod) IsAmountValid(amount float64) bool {
	return amount >= p.MinAmount && amount <= p.MaxAmount
}
