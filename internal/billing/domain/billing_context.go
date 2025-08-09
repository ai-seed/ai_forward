package domain

import (
	"time"

	"ai-api-gateway/internal/domain/entities"
)

// BillingContext 计费上下文 - 包含计费所需的所有信息
type BillingContext struct {
	// 基础信息
	RequestID    string    `json:"request_id"`
	UserID       int64     `json:"user_id"`
	APIKeyID     int64     `json:"api_key_id"`
	ModelID      int64     `json:"model_id"`
	ProviderID   int64     `json:"provider_id"`
	RequestTime  time.Time `json:"request_time"`
	
	// 请求详情
	Method       string                 `json:"method"`
	Endpoint     string                 `json:"endpoint"`
	RequestType  entities.RequestType   `json:"request_type"`
	
	// Token信息
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
	
	// 成本信息
	EstimatedCost float64 `json:"estimated_cost"`
	ActualCost    float64 `json:"actual_cost"`
	
	// 状态信息
	Status       int                    `json:"status"`
	DurationMs   int                    `json:"duration_ms"`
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	
	// 计费状态
	BillingStage BillingStage          `json:"billing_stage"`
	IsBilled     bool                   `json:"is_billed"`
	BillingError string                 `json:"billing_error,omitempty"`
}

// BillingStage 计费阶段
type BillingStage string

const (
	BillingStagePreCheck  BillingStage = "pre_check"   // 预检查（余额、配额）
	BillingStagePending   BillingStage = "pending"     // 待计费
	BillingStageProcessed BillingStage = "processed"   // 已计费
	BillingStageError     BillingStage = "error"       // 计费失败
	BillingStageRefunded  BillingStage = "refunded"    // 已退费
	BillingStageLogOnly   BillingStage = "log_only"    // 仅记录日志（管理员请求）
)

// BillingResult 计费结果
type BillingResult struct {
	Success         bool                      `json:"success"`
	Amount          float64                   `json:"amount"`
	UsageLogID      int64                     `json:"usage_log_id,omitempty"`
	BillingRecordID int64                     `json:"billing_record_id,omitempty"`
	Error           string                    `json:"error,omitempty"`
	Details         map[string]interface{}    `json:"details,omitempty"`
}

// PreCheckResult 预检查结果
type PreCheckResult struct {
	BalanceOK    bool                      `json:"balance_ok"`
	QuotaOK      bool                      `json:"quota_ok"`
	CanProceed   bool                      `json:"can_proceed"`
	Reason       string                    `json:"reason,omitempty"`
	EstimatedCost float64                  `json:"estimated_cost"`
	Details      map[string]interface{}    `json:"details,omitempty"`
}

// CalculateInputTokens 计算输入token数量
func (bc *BillingContext) CalculateInputTokens() int {
	if bc.TotalTokens > 0 && bc.OutputTokens > 0 {
		return bc.TotalTokens - bc.OutputTokens
	}
	return bc.InputTokens
}

// CalculateOutputTokens 计算输出token数量  
func (bc *BillingContext) CalculateOutputTokens() int {
	return bc.OutputTokens
}

// CalculateTotalTokens 计算总token数量
func (bc *BillingContext) CalculateTotalTokens() int {
	if bc.TotalTokens > 0 {
		return bc.TotalTokens
	}
	return bc.InputTokens + bc.OutputTokens
}

// IsSuccessful 判断请求是否成功
func (bc *BillingContext) IsSuccessful() bool {
	return bc.Success && bc.Status >= 200 && bc.Status < 300
}

// ShouldBill 判断是否应该计费
func (bc *BillingContext) ShouldBill() bool {
	// 只有成功的请求才计费
	if !bc.IsSuccessful() {
		return false
	}
	
	// 对于异步任务（如Midjourney），在任务完成时才计费
	if bc.RequestType == entities.RequestTypeMidjourney {
		return bc.BillingStage == BillingStageProcessed
	}
	
	// 同步请求立即计费
	return true
}

// ToUsageLog 转换为使用日志实体
func (bc *BillingContext) ToUsageLog() *entities.UsageLog {
	return &entities.UsageLog{
		UserID:       bc.UserID,
		APIKeyID:     bc.APIKeyID,
		ProviderID:   bc.ProviderID,
		ModelID:      bc.ModelID,
		RequestID:    bc.RequestID,
		RequestType:  bc.RequestType,
		Method:       bc.Method,
		Endpoint:     bc.Endpoint,
		InputTokens:  bc.CalculateInputTokens(),
		OutputTokens: bc.CalculateOutputTokens(),
		TotalTokens:  bc.CalculateTotalTokens(),
		DurationMs:   bc.DurationMs,
		StatusCode:   bc.Status,
		Cost:         bc.ActualCost,
		IsBilled:     bc.IsBilled,
		ErrorMessage: func() *string {
			if bc.ErrorMessage != "" {
				return &bc.ErrorMessage
			}
			return nil
		}(),
		CreatedAt: bc.RequestTime,
	}
}