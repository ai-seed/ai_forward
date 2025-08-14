package entities

import "time"

// RechargeOption 充值金额选项
type RechargeOption struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Amount      float64   `json:"amount" gorm:"type:numeric(15,6);not null;index"`     // 充值金额
	DisplayText string    `json:"display_text" gorm:"not null;size:100"`               // 显示文本，如"10元"
	Tag         string    `json:"tag" gorm:"size:50"`                                  // 标签：推荐、热门、特惠等
	TagColor    string    `json:"tag_color" gorm:"size:20"`                            // 标签颜色
	BonusAmount float64   `json:"bonus_amount" gorm:"type:numeric(15,6);default:0"`    // 赠送金额
	BonusText   string    `json:"bonus_text" gorm:"size:100"`                          // 赠送说明，如"送5元"
	Enabled     bool      `json:"enabled" gorm:"default:true;index"`                   // 是否启用
	SortOrder   int       `json:"sort_order" gorm:"default:0"`                         // 排序
	CreatedAt   time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

// TableName 指定表名
func (RechargeOption) TableName() string {
	return "recharge_options"
}

// IsEnabled 检查选项是否启用
func (r *RechargeOption) IsEnabled() bool {
	return r.Enabled
}

// GetTotalAmount 获取总金额（充值金额+赠送金额）
func (r *RechargeOption) GetTotalAmount() float64 {
	return r.Amount + r.BonusAmount
}

// HasBonus 是否有赠送
func (r *RechargeOption) HasBonus() bool {
	return r.BonusAmount > 0
}

// HasTag 是否有标签
func (r *RechargeOption) HasTag() bool {
	return r.Tag != ""
}
