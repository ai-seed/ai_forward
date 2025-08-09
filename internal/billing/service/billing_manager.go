package service

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/billing/domain"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/domain/services"
	"ai-api-gateway/internal/infrastructure/logger"
)

// BillingManager 计费管理器 - 统一的计费入口
type BillingManager struct {
	// 依赖的服务
	billingService    services.BillingService
	quotaService     services.QuotaService
	usageLogRepo     repositories.UsageLogRepository
	userRepo         repositories.UserRepository
	billingRecordRepo repositories.BillingRecordRepository
	modelPricingRepo  repositories.ModelPricingRepository
	
	// 配置和工具
	logger           logger.Logger
	auditLogger      *BillingAuditLogger
}

// NewBillingManager 创建计费管理器
func NewBillingManager(
	billingService services.BillingService,
	quotaService services.QuotaService,
	usageLogRepo repositories.UsageLogRepository,
	userRepo repositories.UserRepository,
	billingRecordRepo repositories.BillingRecordRepository,
	modelPricingRepo repositories.ModelPricingRepository,
	logger logger.Logger,
) *BillingManager {
	return &BillingManager{
		billingService:    billingService,
		quotaService:     quotaService,
		usageLogRepo:     usageLogRepo,
		userRepo:         userRepo,
		billingRecordRepo: billingRecordRepo,
		modelPricingRepo:  modelPricingRepo,
		logger:           logger,
		auditLogger:      NewBillingAuditLogger(logger),
	}
}

// PreCheck 预检查 - 在请求处理前检查余额和配额
func (bm *BillingManager) PreCheck(ctx context.Context, billingCtx *domain.BillingContext) (*domain.PreCheckResult, error) {
	// 记录预检查开始
	bm.auditLogger.LogPreCheckStart(billingCtx)
	
	// 估算成本
	estimatedCost, err := bm.estimateCost(ctx, billingCtx)
	if err != nil {
		bm.auditLogger.LogPreCheckError(billingCtx, "cost_estimation_failed", err)
		return nil, fmt.Errorf("failed to estimate cost: %w", err)
	}
	billingCtx.EstimatedCost = estimatedCost
	
	result := &domain.PreCheckResult{
		EstimatedCost: estimatedCost,
		Details:       make(map[string]interface{}),
	}
	
	// 检查余额
	balanceOK, err := bm.quotaService.CheckBalance(ctx, billingCtx.UserID, estimatedCost)
	if err != nil {
		bm.auditLogger.LogPreCheckError(billingCtx, "balance_check_failed", err)
		return nil, fmt.Errorf("failed to check balance: %w", err)
	}
	result.BalanceOK = balanceOK
	result.Details["balance_check"] = balanceOK
	
	if !balanceOK {
		result.CanProceed = false
		result.Reason = "insufficient_balance"
		bm.auditLogger.LogPreCheckResult(billingCtx, result)
		return result, nil
	}
	
	// 检查配额 - 检查token配额和请求配额
	quotaChecks := []entities.QuotaType{entities.QuotaTypeTokens, entities.QuotaTypeRequests, entities.QuotaTypeCost}
	quotaOK := true
	var quotaReason string
	
	for _, quotaType := range quotaChecks {
		var value float64
		switch quotaType {
		case entities.QuotaTypeTokens:
			value = float64(billingCtx.CalculateTotalTokens())
		case entities.QuotaTypeRequests:
			value = 1
		case entities.QuotaTypeCost:
			value = estimatedCost
		}
		
		if value <= 0 {
			continue
		}
		
		quotaResult, err := bm.quotaService.CheckQuota(ctx, billingCtx.APIKeyID, quotaType, value)
		if err != nil {
			bm.logger.WithFields(map[string]interface{}{
				"request_id": billingCtx.RequestID,
				"quota_type": quotaType,
				"error": err.Error(),
			}).Warn("Failed to check quota, allowing request")
			continue
		}
		
		result.Details[fmt.Sprintf("quota_%s", quotaType)] = quotaResult
		
		if !quotaResult.Allowed {
			quotaOK = false
			quotaReason = fmt.Sprintf("%s_quota_exceeded", quotaType)
			break
		}
	}
	
	result.QuotaOK = quotaOK
	result.CanProceed = balanceOK && quotaOK
	if !quotaOK {
		result.Reason = quotaReason
	}
	
	// 记录预检查结果
	bm.auditLogger.LogPreCheckResult(billingCtx, result)
	
	return result, nil
}

// ProcessRequest 处理请求 - 统一的计费处理入口
func (bm *BillingManager) ProcessRequest(ctx context.Context, billingCtx *domain.BillingContext) (*domain.BillingResult, error) {
	// 记录处理开始
	bm.auditLogger.LogBillingStart(billingCtx)
	
	// 创建使用日志
	usageLog := billingCtx.ToUsageLog()
	if err := bm.usageLogRepo.Create(ctx, usageLog); err != nil {
		bm.auditLogger.LogBillingError(billingCtx, "usage_log_creation_failed", err)
		return nil, fmt.Errorf("failed to create usage log: %w", err)
	}
	
	result := &domain.BillingResult{
		UsageLogID: usageLog.ID,
		Details:    make(map[string]interface{}),
	}
	
	// 如果不应该计费，直接返回
	if !billingCtx.ShouldBill() {
		result.Success = true
		result.Amount = 0
		bm.auditLogger.LogBillingResult(billingCtx, result, "no_billing_required")
		return result, nil
	}
	
	// 计算实际成本
	actualCost, err := bm.calculateActualCost(ctx, billingCtx)
	if err != nil {
		bm.auditLogger.LogBillingError(billingCtx, "cost_calculation_failed", err)
		return nil, fmt.Errorf("failed to calculate actual cost: %w", err)
	}
	billingCtx.ActualCost = actualCost
	usageLog.Cost = actualCost
	
	// 处理计费 - 扣减余额和创建计费记录
	if err := bm.billingService.ProcessBilling(ctx, usageLog); err != nil {
		bm.auditLogger.LogBillingError(billingCtx, "billing_processing_failed", err)
		return &domain.BillingResult{
			Success:    false,
			Amount:     actualCost,
			UsageLogID: usageLog.ID,
			Error:      err.Error(),
		}, err
	}
	
	// 消费配额
	if err := bm.consumeQuotas(ctx, billingCtx); err != nil {
		bm.logger.WithFields(map[string]interface{}{
			"request_id": billingCtx.RequestID,
			"error": err.Error(),
		}).Warn("Failed to consume quotas, but billing completed")
	}
	
	// 标记为已计费
	billingCtx.IsBilled = true
	usageLog.IsBilled = true
	if err := bm.usageLogRepo.Update(ctx, usageLog); err != nil {
		bm.logger.WithFields(map[string]interface{}{
			"request_id": billingCtx.RequestID,
			"usage_log_id": usageLog.ID,
			"error": err.Error(),
		}).Error("Failed to update usage log billing status")
	}
	
	result.Success = true
	result.Amount = actualCost
	
	// 记录计费结果
	bm.auditLogger.LogBillingResult(billingCtx, result, "billing_completed")
	
	return result, nil
}

// ProcessAsyncCompletion 处理异步任务完成 - 用于Midjourney等异步任务
func (bm *BillingManager) ProcessAsyncCompletion(ctx context.Context, requestID string, success bool) error {
	// 查找使用日志
	usageLog, err := bm.usageLogRepo.GetByRequestID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get usage log for request %s: %w", requestID, err)
	}
	
	if usageLog == nil {
		return fmt.Errorf("usage log not found for request %s", requestID)
	}
	
	// 如果已经计费，跳过
	if usageLog.IsBilled {
		bm.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"usage_log_id": usageLog.ID,
		}).Debug("Usage log already billed, skipping")
		return nil
	}
	
	// 创建计费上下文
	billingCtx := &domain.BillingContext{
		RequestID:    requestID,
		UserID:       usageLog.UserID,
		APIKeyID:     usageLog.APIKeyID,
		ModelID:      usageLog.ModelID,
		ProviderID:   usageLog.ProviderID,
		RequestType:  usageLog.RequestType,
		Method:       usageLog.Method,
		Endpoint:     usageLog.Endpoint,
		InputTokens:  usageLog.InputTokens,
		OutputTokens: usageLog.OutputTokens,
		TotalTokens:  usageLog.TotalTokens,
		DurationMs:   usageLog.DurationMs,
		Success:      success,
		Status:       func() int {
			if success {
				return 200
			} else {
				return 500
			}
		}(),
		BillingStage: domain.BillingStageProcessed,
		RequestTime:  usageLog.CreatedAt,
	}
	
	// 记录异步完成处理
	bm.auditLogger.LogAsyncCompletionStart(billingCtx, success)
	
	// 更新使用日志状态
	usageLog.StatusCode = billingCtx.Status
	
	// 只有成功的任务才计费
	if success && billingCtx.ShouldBill() {
		// 计算成本
		actualCost, err := bm.calculateActualCost(ctx, billingCtx)
		if err != nil {
			bm.auditLogger.LogBillingError(billingCtx, "async_cost_calculation_failed", err)
			return fmt.Errorf("failed to calculate cost for async completion: %w", err)
		}
		
		usageLog.Cost = actualCost
		billingCtx.ActualCost = actualCost
		
		// 处理计费
		if err := bm.billingService.ProcessBilling(ctx, usageLog); err != nil {
			bm.auditLogger.LogBillingError(billingCtx, "async_billing_failed", err)
			return fmt.Errorf("failed to process billing for async completion: %w", err)
		}
		
		// 标记为已计费
		usageLog.IsBilled = true
		
		bm.auditLogger.LogAsyncCompletionResult(billingCtx, actualCost, "billing_completed")
	} else {
		bm.auditLogger.LogAsyncCompletionResult(billingCtx, 0, "no_billing_required")
	}
	
	// 更新使用日志
	if err := bm.usageLogRepo.Update(ctx, usageLog); err != nil {
		return fmt.Errorf("failed to update usage log: %w", err)
	}
	
	return nil
}

// estimateCost 估算成本
func (bm *BillingManager) estimateCost(ctx context.Context, billingCtx *domain.BillingContext) (float64, error) {
	bm.logger.WithFields(map[string]interface{}{
		"event":        "cost_estimation_start",
		"request_id":   billingCtx.RequestID,
		"model_id":     billingCtx.ModelID,
		"request_type": billingCtx.RequestType,
		"input_tokens": billingCtx.CalculateInputTokens(),
		"output_tokens": billingCtx.CalculateOutputTokens(),
	}).Debug("Starting cost estimation")

	var cost float64
	var err error

	if billingCtx.RequestType == entities.RequestTypeMidjourney {
		// Midjourney按请求计费
		bm.logger.WithFields(map[string]interface{}{
			"request_id": billingCtx.RequestID,
			"model_id":   billingCtx.ModelID,
		}).Debug("Using request-based cost estimation for Midjourney")

		cost, err = bm.billingService.CalculateRequestCost(ctx, billingCtx.ModelID)
	} else {
		// 普通API按token计费
		bm.logger.WithFields(map[string]interface{}{
			"request_id":    billingCtx.RequestID,
			"model_id":      billingCtx.ModelID,
			"input_tokens":  billingCtx.CalculateInputTokens(),
			"output_tokens": billingCtx.CalculateOutputTokens(),
		}).Debug("Using token-based cost estimation")

		cost, err = bm.billingService.CalculateCost(ctx, billingCtx.ModelID, billingCtx.CalculateInputTokens(), billingCtx.CalculateOutputTokens())
	}

	if err != nil {
		bm.logger.WithFields(map[string]interface{}{
			"event":      "cost_estimation_failed",
			"request_id": billingCtx.RequestID,
			"model_id":   billingCtx.ModelID,
			"error":      err.Error(),
		}).Error("Cost estimation failed")
		return 0, err
	}

	bm.logger.WithFields(map[string]interface{}{
		"event":         "cost_estimation_completed",
		"request_id":    billingCtx.RequestID,
		"model_id":      billingCtx.ModelID,
		"estimated_cost": cost,
	}).Info("Cost estimation completed")

	return cost, nil
}

// calculateActualCost 计算实际成本
func (bm *BillingManager) calculateActualCost(ctx context.Context, billingCtx *domain.BillingContext) (float64, error) {
	// 对于大多数情况，实际成本等于估算成本
	return bm.estimateCost(ctx, billingCtx)
}

// consumeQuotas 消费配额
func (bm *BillingManager) consumeQuotas(ctx context.Context, billingCtx *domain.BillingContext) error {
	bm.logger.WithFields(map[string]interface{}{
		"event":        "quota_consumption_start",
		"request_id":   billingCtx.RequestID,
		"api_key_id":   billingCtx.APIKeyID,
		"total_tokens": billingCtx.CalculateTotalTokens(),
		"actual_cost":  billingCtx.ActualCost,
	}).Debug("Starting quota consumption")

	// 消费token配额
	if billingCtx.CalculateTotalTokens() > 0 {
		tokenValue := float64(billingCtx.CalculateTotalTokens())
		bm.logger.WithFields(map[string]interface{}{
			"request_id": billingCtx.RequestID,
			"api_key_id": billingCtx.APIKeyID,
			"quota_type": "tokens",
			"value":      tokenValue,
		}).Debug("Consuming token quota")

		if err := bm.quotaService.ConsumeQuota(ctx, billingCtx.APIKeyID, entities.QuotaTypeTokens, tokenValue); err != nil {
			bm.auditLogger.LogQuotaConsumption(billingCtx.RequestID, billingCtx.APIKeyID, "tokens", tokenValue, false, err)
			return fmt.Errorf("failed to consume token quota: %w", err)
		}
		bm.auditLogger.LogQuotaConsumption(billingCtx.RequestID, billingCtx.APIKeyID, "tokens", tokenValue, true, nil)
	}
	
	// 消费请求配额
	bm.logger.WithFields(map[string]interface{}{
		"request_id": billingCtx.RequestID,
		"api_key_id": billingCtx.APIKeyID,
		"quota_type": "requests",
		"value":      1,
	}).Debug("Consuming request quota")

	if err := bm.quotaService.ConsumeQuota(ctx, billingCtx.APIKeyID, entities.QuotaTypeRequests, 1); err != nil {
		bm.auditLogger.LogQuotaConsumption(billingCtx.RequestID, billingCtx.APIKeyID, "requests", 1, false, err)
		return fmt.Errorf("failed to consume request quota: %w", err)
	}
	bm.auditLogger.LogQuotaConsumption(billingCtx.RequestID, billingCtx.APIKeyID, "requests", 1, true, nil)
	
	// 消费成本配额
	if billingCtx.ActualCost > 0 {
		bm.logger.WithFields(map[string]interface{}{
			"request_id": billingCtx.RequestID,
			"api_key_id": billingCtx.APIKeyID,
			"quota_type": "cost",
			"value":      billingCtx.ActualCost,
		}).Debug("Consuming cost quota")

		if err := bm.quotaService.ConsumeQuota(ctx, billingCtx.APIKeyID, entities.QuotaTypeCost, billingCtx.ActualCost); err != nil {
			bm.auditLogger.LogQuotaConsumption(billingCtx.RequestID, billingCtx.APIKeyID, "cost", billingCtx.ActualCost, false, err)
			return fmt.Errorf("failed to consume cost quota: %w", err)
		}
		bm.auditLogger.LogQuotaConsumption(billingCtx.RequestID, billingCtx.APIKeyID, "cost", billingCtx.ActualCost, true, nil)
	}

	bm.logger.WithFields(map[string]interface{}{
		"event":      "quota_consumption_completed",
		"request_id": billingCtx.RequestID,
		"api_key_id": billingCtx.APIKeyID,
	}).Info("All quotas consumed successfully")
	
	return nil
}

// CreateUsageLogOnly 只创建使用日志，不进行计费（用于管理员请求等）
func (bm *BillingManager) CreateUsageLogOnly(ctx context.Context, usageLog *entities.UsageLog) error {
	// 记录日志创建
	bm.auditLogger.LogBillingStart(&domain.BillingContext{
		RequestID:    usageLog.RequestID,
		UserID:       usageLog.UserID,
		APIKeyID:     usageLog.APIKeyID,
		BillingStage: domain.BillingStageLogOnly,
	})

	// 创建使用日志
	if err := bm.usageLogRepo.Create(ctx, usageLog); err != nil {
		bm.auditLogger.LogBillingError(&domain.BillingContext{
			RequestID: usageLog.RequestID,
			UserID:    usageLog.UserID,
			APIKeyID:  usageLog.APIKeyID,
		}, "usage_log_creation_failed", err)
		return fmt.Errorf("failed to create usage log: %w", err)
	}

	// 记录完成
	bm.auditLogger.LogBillingResult(&domain.BillingContext{
		RequestID: usageLog.RequestID,
		UserID:    usageLog.UserID,
		APIKeyID:  usageLog.APIKeyID,
	}, &domain.BillingResult{
		Success:    true,
		Amount:     0,
		UsageLogID: usageLog.ID,
	}, "log_only_completed")

	return nil
}