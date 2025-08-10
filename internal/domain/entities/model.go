package entities

import (
	"time"
)

// ModelType 模型类型枚举
type ModelType string

const (
	ModelTypeChat       ModelType = "chat"
	ModelTypeCompletion ModelType = "completion"
	ModelTypeEmbedding  ModelType = "embedding"
	ModelTypeImage      ModelType = "image"
	ModelTypeAudio      ModelType = "audio"
)

// ModelStatus 模型状态枚举
type ModelStatus string

const (
	ModelStatusActive     ModelStatus = "active"
	ModelStatusDeprecated ModelStatus = "deprecated"
	ModelStatusDisabled   ModelStatus = "disabled"
)

// Model AI模型实体
type Model struct {
	ID                int64       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name              string      `json:"name" gorm:"not null;size:100"`
	Slug              string      `json:"slug" gorm:"uniqueIndex;not null;size:100"`
	DisplayName       *string     `json:"display_name,omitempty" gorm:"size:200"`
	Description       *string     `json:"description,omitempty" gorm:"type:text"`
	ModelType         ModelType   `json:"model_type" gorm:"not null;size:50;index"`
	ModelProviderID   int64       `json:"model_provider_id" gorm:"not null;index"` // 模型厂商ID（指向model_providers表）
	ContextLength     *int        `json:"context_length,omitempty"`
	MaxTokens         *int        `json:"max_tokens,omitempty"`
	SupportsStreaming bool        `json:"supports_streaming" gorm:"not null;default:false"`
	SupportsFunctions bool        `json:"supports_functions" gorm:"not null;default:false"`
	Status            ModelStatus `json:"status" gorm:"not null;default:active;size:20;index"`
	CreatedAt         time.Time   `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt         time.Time   `json:"updated_at" gorm:"not null;autoUpdateTime"`

	// 多语言字段 - 与数据库表结构一致，使用 _jp 后缀
	DescriptionEN *string `json:"description_en,omitempty" gorm:"type:text"`
	DescriptionZH *string `json:"description_zh,omitempty" gorm:"type:text"`
	DescriptionJP *string `json:"description_jp,omitempty" gorm:"type:text"`
	ModelTypeEN   *string `json:"model_type_en,omitempty" gorm:"size:50"`
	ModelTypeZH   *string `json:"model_type_zh,omitempty" gorm:"size:50"`
	ModelTypeJP   *string `json:"model_type_jp,omitempty" gorm:"size:50"`

	// 关联关系（不使用外键约束，通过代码逻辑控制）
	ModelProvider *ModelProvider `json:"model_provider,omitempty" gorm:"-"`
}

// TableName 指定表名
func (Model) TableName() string {
	return "models"
}

// IsAvailable 检查模型是否可用
func (m *Model) IsAvailable() bool {
	return m.Status == ModelStatusActive
}

// IsActive 检查模型是否处于活跃状态
func (m *Model) IsActive() bool {
	return m.Status == ModelStatusActive
}

// GetDisplayName 获取显示名称
func (m *Model) GetDisplayName() string {
	if m.DisplayName != nil && *m.DisplayName != "" {
		return *m.DisplayName
	}
	return m.Name
}

// GetDisplayNameI18n 根据语言获取本地化的显示名称
func (m *Model) GetDisplayNameI18n(lang string) string {
	// 由于数据库表中没有display_name的多语言字段，直接返回默认显示名称
	return m.GetDisplayName()
}

// GetDescriptionI18n 根据语言获取本地化的描述
func (m *Model) GetDescriptionI18n(lang string) string {
	switch lang {
	case "en":
		if m.DescriptionEN != nil && *m.DescriptionEN != "" {
			return *m.DescriptionEN
		}
	case "zh":
		if m.DescriptionZH != nil && *m.DescriptionZH != "" {
			return *m.DescriptionZH
		}
	case "ja":
		if m.DescriptionJP != nil && *m.DescriptionJP != "" {
			return *m.DescriptionJP
		}
	}
	// 回退到默认描述
	if m.Description != nil && *m.Description != "" {
		return *m.Description
	}
	return ""
}

// GetAllDisplayNames 获取所有语言的显示名称
func (m *Model) GetAllDisplayNames() map[string]string {
	result := make(map[string]string)
	displayName := m.GetDisplayName()
	
	// 由于数据库表中没有display_name的多语言字段，所有语言都返回相同值
	result["en"] = displayName
	result["zh"] = displayName
	result["ja"] = displayName
	
	return result
}

// GetAllDescriptions 获取所有语言的描述
func (m *Model) GetAllDescriptions() map[string]string {
	result := make(map[string]string)
	defaultDesc := ""
	if m.Description != nil && *m.Description != "" {
		defaultDesc = *m.Description
	}
	
	if m.DescriptionEN != nil && *m.DescriptionEN != "" {
		result["en"] = *m.DescriptionEN
	} else {
		result["en"] = defaultDesc
	}
	
	if m.DescriptionZH != nil && *m.DescriptionZH != "" {
		result["zh"] = *m.DescriptionZH
	} else {
		result["zh"] = defaultDesc
	}
	
	if m.DescriptionJP != nil && *m.DescriptionJP != "" {
		result["ja"] = *m.DescriptionJP
	} else {
		result["ja"] = defaultDesc
	}
	
	return result
}

// GetModelTypeI18n 根据语言获取本地化的模型类型
func (m *Model) GetModelTypeI18n(lang string) string {
	switch lang {
	case "en":
		if m.ModelTypeEN != nil && *m.ModelTypeEN != "" {
			return *m.ModelTypeEN
		}
	case "zh":
		if m.ModelTypeZH != nil && *m.ModelTypeZH != "" {
			return *m.ModelTypeZH
		}
	case "ja":
		if m.ModelTypeJP != nil && *m.ModelTypeJP != "" {
			return *m.ModelTypeJP
		}
	}
	// 回退到原始模型类型
	return string(m.ModelType)
}

// GetAllModelTypes 获取所有语言的模型类型
func (m *Model) GetAllModelTypes() map[string]string {
	result := make(map[string]string)
	defaultType := string(m.ModelType)
	
	if m.ModelTypeEN != nil && *m.ModelTypeEN != "" {
		result["en"] = *m.ModelTypeEN
	} else {
		result["en"] = defaultType
	}
	
	if m.ModelTypeZH != nil && *m.ModelTypeZH != "" {
		result["zh"] = *m.ModelTypeZH
	} else {
		result["zh"] = defaultType
	}
	
	if m.ModelTypeJP != nil && *m.ModelTypeJP != "" {
		result["ja"] = *m.ModelTypeJP
	} else {
		result["ja"] = defaultType
	}
	
	return result
}

// GetContextLength 获取上下文长度
func (m *Model) GetContextLength() int {
	if m.ContextLength != nil {
		return *m.ContextLength
	}
	return 4096 // 默认值
}

// GetMaxTokens 获取最大token数
func (m *Model) GetMaxTokens() int {
	if m.MaxTokens != nil {
		return *m.MaxTokens
	}
	return m.GetContextLength() / 2 // 默认为上下文长度的一半
}

// CanStream 检查是否支持流式输出
func (m *Model) CanStream() bool {
	return m.SupportsStreaming
}

// CanUseFunctions 检查是否支持函数调用
func (m *Model) CanUseFunctions() bool {
	return m.SupportsFunctions
}

// PricingType 定价类型枚举
type PricingType string

const (
	PricingTypeInput   PricingType = "input"
	PricingTypeOutput  PricingType = "output"
	PricingTypeRequest PricingType = "request"
)

// PricingUnit 定价单位枚举
type PricingUnit string

const (
	PricingUnitToken     PricingUnit = "token"
	PricingUnitRequest   PricingUnit = "request"
	PricingUnitCharacter PricingUnit = "character"
)

// ModelPricing 模型定价实体
type ModelPricing struct {
	ID             int64       `json:"id" gorm:"primaryKey;autoIncrement"`
	ModelID        int64       `json:"model_id" gorm:"not null;index"`
	PricingType    PricingType `json:"pricing_type" gorm:"not null;size:20;index"`
	PricePerUnit   float64     `json:"price_per_unit" gorm:"type:numeric(15,8);not null"`
	Multiplier     float64     `json:"multiplier" gorm:"type:numeric(5,2);not null;default:1.5"` // 价格倍率，默认1.5
	Unit           PricingUnit `json:"unit" gorm:"not null;size:20"`
	Currency       string      `json:"currency" gorm:"not null;default:USD;size:3"`
	EffectiveFrom  time.Time   `json:"effective_from" gorm:"not null;default:CURRENT_TIMESTAMP"`
	EffectiveUntil *time.Time  `json:"effective_until,omitempty"`
	CreatedAt      time.Time   `json:"created_at" gorm:"not null;autoCreateTime"`
}

// TableName 指定表名
func (ModelPricing) TableName() string {
	return "model_pricing"
}

// IsEffective 检查定价是否在有效期内
func (mp *ModelPricing) IsEffective(at time.Time) bool {
	if at.Before(mp.EffectiveFrom) {
		return false
	}

	if mp.EffectiveUntil != nil && at.After(*mp.EffectiveUntil) {
		return false
	}

	return true
}

// CalculateCost 计算成本（应用倍率）
func (mp *ModelPricing) CalculateCost(units int) float64 {
	baseCost := float64(units) * mp.PricePerUnit
	return baseCost * mp.Multiplier
}

// CalculateBaseCost 计算基础成本（不应用倍率）
func (mp *ModelPricing) CalculateBaseCost(units int) float64 {
	return float64(units) * mp.PricePerUnit
}

// GetFinalPrice 获取应用倍率后的最终单价
func (mp *ModelPricing) GetFinalPrice() float64 {
	return mp.PricePerUnit * mp.Multiplier
}
