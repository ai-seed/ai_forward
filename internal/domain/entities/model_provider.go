package entities

import "time"

// ModelProvider AI模型厂商实体
type ModelProvider struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"not null;size:100;uniqueIndex"`             // 厂商名称，如 openai
	DisplayName string    `json:"display_name" gorm:"not null;size:200"`                 // 显示名称，如 OpenAI
	Description *string   `json:"description,omitempty" gorm:"type:text"`                // 厂商描述
	Website     *string   `json:"website,omitempty" gorm:"size:500"`                     // 官网地址
	LogoURL     *string   `json:"logo_url,omitempty" gorm:"size:500"`                    // Logo URL
	Color       string    `json:"color" gorm:"not null;size:20;default:'#1976d2'"`       // 品牌颜色
	SortOrder   int       `json:"sort_order" gorm:"not null;default:0;index"`            // 排序顺序，数字越小越靠前
	Status      string    `json:"status" gorm:"not null;size:20;default:'active';index"` // active, inactive
	CreatedAt   time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

// TableName 指定表名
func (ModelProvider) TableName() string {
	return "model_providers"
}
