package service

import (
	"encoding/json"
	"time"

	"ai-api-gateway/internal/billing/domain"
	"ai-api-gateway/internal/infrastructure/logger"
)

// BillingAuditLogger 计费审计日志
type BillingAuditLogger struct {
	logger logger.Logger
}

// NewBillingAuditLogger 创建计费审计日志
func NewBillingAuditLogger(logger logger.Logger) *BillingAuditLogger {
	return &BillingAuditLogger{
		logger: logger,
	}
}

// LogPreCheckStart 记录预检查开始
func (bal *BillingAuditLogger) LogPreCheckStart(billingCtx *domain.BillingContext) {
	bal.logger.WithFields(map[string]interface{}{
		"event":      "billing_precheck_start",
		"request_id": billingCtx.RequestID,
		"user_id":    billingCtx.UserID,
		"api_key_id": billingCtx.APIKeyID,
		"model_id":   billingCtx.ModelID,
		"timestamp":  time.Now().Unix(),
	}).Info("Billing pre-check started")
}

// LogPreCheckResult 记录预检查结果
func (bal *BillingAuditLogger) LogPreCheckResult(billingCtx *domain.BillingContext, result *domain.PreCheckResult) {
	resultJSON, _ := json.Marshal(result)
	
	bal.logger.WithFields(map[string]interface{}{
		"event":           "billing_precheck_result",
		"request_id":      billingCtx.RequestID,
		"user_id":         billingCtx.UserID,
		"api_key_id":      billingCtx.APIKeyID,
		"can_proceed":     result.CanProceed,
		"balance_ok":      result.BalanceOK,
		"quota_ok":        result.QuotaOK,
		"estimated_cost":  result.EstimatedCost,
		"reason":          result.Reason,
		"result_details":  string(resultJSON),
		"timestamp":       time.Now().Unix(),
	}).Info("Billing pre-check completed")
}

// LogPreCheckError 记录预检查错误
func (bal *BillingAuditLogger) LogPreCheckError(billingCtx *domain.BillingContext, errorType string, err error) {
	bal.logger.WithFields(map[string]interface{}{
		"event":      "billing_precheck_error",
		"request_id": billingCtx.RequestID,
		"user_id":    billingCtx.UserID,
		"api_key_id": billingCtx.APIKeyID,
		"error_type": errorType,
		"error":      err.Error(),
		"timestamp":  time.Now().Unix(),
	}).Error("Billing pre-check error occurred")
}

// LogBillingStart 记录计费开始
func (bal *BillingAuditLogger) LogBillingStart(billingCtx *domain.BillingContext) {
	contextJSON, _ := json.Marshal(billingCtx)
	
	bal.logger.WithFields(map[string]interface{}{
		"event":         "billing_start",
		"request_id":    billingCtx.RequestID,
		"user_id":       billingCtx.UserID,
		"api_key_id":    billingCtx.APIKeyID,
		"model_id":      billingCtx.ModelID,
		"provider_id":   billingCtx.ProviderID,
		"request_type":  billingCtx.RequestType,
		"input_tokens":  billingCtx.InputTokens,
		"output_tokens": billingCtx.OutputTokens,
		"total_tokens":  billingCtx.TotalTokens,
		"estimated_cost": billingCtx.EstimatedCost,
		"billing_stage": billingCtx.BillingStage,
		"context":       string(contextJSON),
		"timestamp":     time.Now().Unix(),
	}).Info("Billing process started")
}

// LogBillingResult 记录计费结果
func (bal *BillingAuditLogger) LogBillingResult(billingCtx *domain.BillingContext, result *domain.BillingResult, reason string) {
	resultJSON, _ := json.Marshal(result)
	
	logLevel := "Info"
	if !result.Success {
		logLevel = "Error"
	}
	
	fields := map[string]interface{}{
		"event":             "billing_result",
		"request_id":        billingCtx.RequestID,
		"user_id":           billingCtx.UserID,
		"api_key_id":        billingCtx.APIKeyID,
		"success":           result.Success,
		"amount":            result.Amount,
		"usage_log_id":      result.UsageLogID,
		"billing_record_id": result.BillingRecordID,
		"reason":            reason,
		"result_details":    string(resultJSON),
		"timestamp":         time.Now().Unix(),
	}
	
	if result.Error != "" {
		fields["error"] = result.Error
	}
	
	logEntry := bal.logger.WithFields(fields)
	if logLevel == "Error" {
		logEntry.Error("Billing process completed with error")
	} else {
		logEntry.Info("Billing process completed successfully")
	}
}

// LogBillingError 记录计费错误
func (bal *BillingAuditLogger) LogBillingError(billingCtx *domain.BillingContext, errorType string, err error) {
	bal.logger.WithFields(map[string]interface{}{
		"event":      "billing_error",
		"request_id": billingCtx.RequestID,
		"user_id":    billingCtx.UserID,
		"api_key_id": billingCtx.APIKeyID,
		"model_id":   billingCtx.ModelID,
		"error_type": errorType,
		"error":      err.Error(),
		"timestamp":  time.Now().Unix(),
	}).Error("Billing error occurred")
}

// LogAsyncCompletionStart 记录异步完成处理开始
func (bal *BillingAuditLogger) LogAsyncCompletionStart(billingCtx *domain.BillingContext, success bool) {
	bal.logger.WithFields(map[string]interface{}{
		"event":        "billing_async_completion_start",
		"request_id":   billingCtx.RequestID,
		"user_id":      billingCtx.UserID,
		"api_key_id":   billingCtx.APIKeyID,
		"request_type": billingCtx.RequestType,
		"task_success": success,
		"timestamp":    time.Now().Unix(),
	}).Info("Async completion billing started")
}

// LogAsyncCompletionResult 记录异步完成处理结果
func (bal *BillingAuditLogger) LogAsyncCompletionResult(billingCtx *domain.BillingContext, amount float64, reason string) {
	bal.logger.WithFields(map[string]interface{}{
		"event":      "billing_async_completion_result",
		"request_id": billingCtx.RequestID,
		"user_id":    billingCtx.UserID,
		"api_key_id": billingCtx.APIKeyID,
		"amount":     amount,
		"reason":     reason,
		"is_billed":  billingCtx.IsBilled,
		"timestamp":  time.Now().Unix(),
	}).Info("Async completion billing completed")
}

// LogRefund 记录退费
func (bal *BillingAuditLogger) LogRefund(requestID string, userID, usageLogID int64, amount float64, reason string) {
	bal.logger.WithFields(map[string]interface{}{
		"event":        "billing_refund",
		"request_id":   requestID,
		"user_id":      userID,
		"usage_log_id": usageLogID,
		"amount":       amount,
		"reason":       reason,
		"timestamp":    time.Now().Unix(),
	}).Info("Billing refund processed")
}

// LogConsistencyCheck 记录一致性检查
func (bal *BillingAuditLogger) LogConsistencyCheck(checkType string, results map[string]interface{}) {
	bal.logger.WithFields(map[string]interface{}{
		"event":      "billing_consistency_check",
		"check_type": checkType,
		"results":    results,
		"timestamp":  time.Now().Unix(),
	}).Info("Billing consistency check completed")
}

// LogQuotaConsumption 记录配额消费
func (bal *BillingAuditLogger) LogQuotaConsumption(requestID string, apiKeyID int64, quotaType string, value float64, success bool, err error) {
	fields := map[string]interface{}{
		"event":      "quota_consumption",
		"request_id": requestID,
		"api_key_id": apiKeyID,
		"quota_type": quotaType,
		"value":      value,
		"success":    success,
		"timestamp":  time.Now().Unix(),
	}
	
	if err != nil {
		fields["error"] = err.Error()
	}
	
	logEntry := bal.logger.WithFields(fields)
	if success {
		logEntry.Debug("Quota consumption completed")
	} else {
		logEntry.Warn("Quota consumption failed")
	}
}