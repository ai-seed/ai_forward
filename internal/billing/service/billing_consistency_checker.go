package service

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/logger"
)

// BillingConsistencyChecker 计费一致性检查器
type BillingConsistencyChecker struct {
	usageLogRepo      repositories.UsageLogRepository
	billingRecordRepo repositories.BillingRecordRepository
	userRepo          repositories.UserRepository
	billingManager    *BillingManager
	auditLogger       *BillingAuditLogger
	logger            logger.Logger
}

// NewBillingConsistencyChecker 创建计费一致性检查器
func NewBillingConsistencyChecker(
	usageLogRepo repositories.UsageLogRepository,
	billingRecordRepo repositories.BillingRecordRepository,
	userRepo repositories.UserRepository,
	billingManager *BillingManager,
	logger logger.Logger,
) *BillingConsistencyChecker {
	return &BillingConsistencyChecker{
		usageLogRepo:      usageLogRepo,
		billingRecordRepo: billingRecordRepo,
		userRepo:          userRepo,
		billingManager:    billingManager,
		auditLogger:       NewBillingAuditLogger(logger),
		logger:            logger,
	}
}

// ConsistencyCheckResult 一致性检查结果
type ConsistencyCheckResult struct {
	CheckType           string                 `json:"check_type"`
	TotalChecked        int64                  `json:"total_checked"`
	InconsistentCount   int64                  `json:"inconsistent_count"`
	FixedCount          int64                  `json:"fixed_count"`
	ErrorCount          int64                  `json:"error_count"`
	Issues              []ConsistencyIssue     `json:"issues"`
	Summary             map[string]interface{} `json:"summary"`
	CheckTime           time.Time              `json:"check_time"`
	Duration            time.Duration          `json:"duration"`
}

// ConsistencyIssue 一致性问题
type ConsistencyIssue struct {
	Type        string                 `json:"type"`
	RequestID   string                 `json:"request_id"`
	UsageLogID  int64                  `json:"usage_log_id"`
	UserID      int64                  `json:"user_id"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Fixed       bool                   `json:"fixed"`
	FixError    string                 `json:"fix_error,omitempty"`
}

// CheckUnbilledUsageLogs 检查未计费的使用日志
func (bcc *BillingConsistencyChecker) CheckUnbilledUsageLogs(ctx context.Context, lookBackHours int, autoFix bool) (*ConsistencyCheckResult, error) {
	startTime := time.Now()
	result := &ConsistencyCheckResult{
		CheckType: "unbilled_usage_logs",
		CheckTime: startTime,
		Issues:    make([]ConsistencyIssue, 0),
		Summary:   make(map[string]interface{}),
	}

	// 查找指定时间范围内未计费的成功请求日志
	since := time.Now().Add(-time.Duration(lookBackHours) * time.Hour)
	
	// 这里需要实现一个查询未计费日志的方法
	// 暂时使用模拟查询
	unbilledLogs, err := bcc.findUnbilledSuccessfulLogs(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to find unbilled logs: %w", err)
	}

	result.TotalChecked = int64(len(unbilledLogs))

	for _, log := range unbilledLogs {
		issue := ConsistencyIssue{
			Type:        "unbilled_successful_request",
			RequestID:   log.RequestID,
			UsageLogID:  log.ID,
			UserID:      log.UserID,
			Description: fmt.Sprintf("Successful request not billed: %s", log.RequestID),
			Data: map[string]interface{}{
				"status_code":   log.StatusCode,
				"cost":          log.Cost,
				"created_at":    log.CreatedAt,
				"request_type":  log.RequestType,
			},
		}

		if autoFix {
			if err := bcc.fixUnbilledLog(ctx, log); err != nil {
				issue.FixError = err.Error()
				result.ErrorCount++
			} else {
				issue.Fixed = true
				result.FixedCount++
			}
		}

		result.Issues = append(result.Issues, issue)
		result.InconsistentCount++
	}

	result.Duration = time.Since(startTime)
	result.Summary["unbilled_logs_found"] = result.InconsistentCount
	result.Summary["auto_fix_enabled"] = autoFix
	
	// 记录检查结果
	bcc.auditLogger.LogConsistencyCheck("unbilled_usage_logs", map[string]interface{}{
		"total_checked":      result.TotalChecked,
		"inconsistent_count": result.InconsistentCount,
		"fixed_count":        result.FixedCount,
		"error_count":        result.ErrorCount,
		"duration_ms":        result.Duration.Milliseconds(),
	})

	return result, nil
}

// CheckBillingRecordConsistency 检查计费记录一致性
func (bcc *BillingConsistencyChecker) CheckBillingRecordConsistency(ctx context.Context, lookBackHours int, autoFix bool) (*ConsistencyCheckResult, error) {
	startTime := time.Now()
	result := &ConsistencyCheckResult{
		CheckType: "billing_record_consistency",
		CheckTime: startTime,
		Issues:    make([]ConsistencyIssue, 0),
		Summary:   make(map[string]interface{}),
	}

	since := time.Now().Add(-time.Duration(lookBackHours) * time.Hour)
	
	// 查找有计费记录但使用日志标记为未计费的情况
	inconsistentRecords, err := bcc.findInconsistentBillingRecords(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to find inconsistent billing records: %w", err)
	}

	result.TotalChecked = int64(len(inconsistentRecords))

	for _, record := range inconsistentRecords {
		issue := ConsistencyIssue{
			Type:        "billing_record_inconsistency",
			RequestID:   record["request_id"].(string),
			UsageLogID:  record["usage_log_id"].(int64),
			UserID:      record["user_id"].(int64),
			Description: "Usage log and billing record status mismatch",
			Data:        record,
		}

		if autoFix {
			if err := bcc.fixInconsistentBillingRecord(ctx, record); err != nil {
				issue.FixError = err.Error()
				result.ErrorCount++
			} else {
				issue.Fixed = true
				result.FixedCount++
			}
		}

		result.Issues = append(result.Issues, issue)
		result.InconsistentCount++
	}

	result.Duration = time.Since(startTime)
	result.Summary["inconsistent_records_found"] = result.InconsistentCount
	
	bcc.auditLogger.LogConsistencyCheck("billing_record_consistency", map[string]interface{}{
		"total_checked":      result.TotalChecked,
		"inconsistent_count": result.InconsistentCount,
		"fixed_count":        result.FixedCount,
		"error_count":        result.ErrorCount,
		"duration_ms":        result.Duration.Milliseconds(),
	})

	return result, nil
}

// CheckUserBalanceConsistency 检查用户余额一致性
func (bcc *BillingConsistencyChecker) CheckUserBalanceConsistency(ctx context.Context, userID int64) (*ConsistencyCheckResult, error) {
	startTime := time.Now()
	result := &ConsistencyCheckResult{
		CheckType: "user_balance_consistency",
		CheckTime: startTime,
		Issues:    make([]ConsistencyIssue, 0),
		Summary:   make(map[string]interface{}),
	}

	// 获取用户信息
	user, err := bcc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 计算应有余额（这需要根据充值记录和扣费记录计算）
	expectedBalance, err := bcc.calculateExpectedBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate expected balance: %w", err)
	}

	result.TotalChecked = 1

	// 检查余额是否一致
	balanceDiff := user.Balance - expectedBalance
	if abs(balanceDiff) > 0.00000001 { // 考虑浮点数精度问题
		issue := ConsistencyIssue{
			Type:        "balance_mismatch",
			UserID:      userID,
			Description: fmt.Sprintf("User balance mismatch: actual=%.8f, expected=%.8f, diff=%.8f", user.Balance, expectedBalance, balanceDiff),
			Data: map[string]interface{}{
				"actual_balance":   user.Balance,
				"expected_balance": expectedBalance,
				"difference":       balanceDiff,
			},
		}

		result.Issues = append(result.Issues, issue)
		result.InconsistentCount++
	}

	result.Duration = time.Since(startTime)
	result.Summary["balance_difference"] = balanceDiff
	
	bcc.auditLogger.LogConsistencyCheck("user_balance_consistency", map[string]interface{}{
		"user_id":            userID,
		"actual_balance":     user.Balance,
		"expected_balance":   expectedBalance,
		"difference":         balanceDiff,
		"is_consistent":      result.InconsistentCount == 0,
	})

	return result, nil
}

// RunFullConsistencyCheck 运行完整的一致性检查
func (bcc *BillingConsistencyChecker) RunFullConsistencyCheck(ctx context.Context, lookBackHours int, autoFix bool) (map[string]*ConsistencyCheckResult, error) {
	results := make(map[string]*ConsistencyCheckResult)

	// 检查未计费的使用日志
	if result, err := bcc.CheckUnbilledUsageLogs(ctx, lookBackHours, autoFix); err != nil {
		bcc.logger.WithFields(map[string]interface{}{
			"check_type": "unbilled_usage_logs",
			"error": err.Error(),
		}).Error("Consistency check failed")
	} else {
		results["unbilled_usage_logs"] = result
	}

	// 检查计费记录一致性
	if result, err := bcc.CheckBillingRecordConsistency(ctx, lookBackHours, autoFix); err != nil {
		bcc.logger.WithFields(map[string]interface{}{
			"check_type": "billing_record_consistency",
			"error": err.Error(),
		}).Error("Consistency check failed")
	} else {
		results["billing_record_consistency"] = result
	}

	return results, nil
}

// 私有辅助方法

func (bcc *BillingConsistencyChecker) findUnbilledSuccessfulLogs(ctx context.Context, since time.Time) ([]*entities.UsageLog, error) {
	// 这里需要实现查询逻辑，查找：
	// 1. 创建时间在since之后
	// 2. 状态码200-299（成功）
	// 3. IsBilled = false
	// 4. 不是Midjourney类型或者是已完成的Midjourney任务
	
	// 模拟实现，实际需要根据repository接口实现
	return []*entities.UsageLog{}, nil
}

func (bcc *BillingConsistencyChecker) findInconsistentBillingRecords(ctx context.Context, since time.Time) ([]map[string]interface{}, error) {
	// 这里需要实现查询逻辑，查找：
	// 1. 存在计费记录但使用日志未标记为已计费
	// 2. 使用日志标记为已计费但没有计费记录
	
	// 模拟实现
	return []map[string]interface{}{}, nil
}

func (bcc *BillingConsistencyChecker) calculateExpectedBalance(ctx context.Context, userID int64) (float64, error) {
	// 这里需要实现余额计算逻辑：
	// 1. 获取所有充值记录
	// 2. 获取所有扣费记录
	// 3. 计算净余额
	
	// 模拟实现
	return 0.0, nil
}

func (bcc *BillingConsistencyChecker) fixUnbilledLog(ctx context.Context, log *entities.UsageLog) error {
	// 对于未计费的日志，调用计费处理
	return bcc.billingManager.ProcessAsyncCompletion(ctx, log.RequestID, log.IsSuccessful())
}

func (bcc *BillingConsistencyChecker) fixInconsistentBillingRecord(ctx context.Context, record map[string]interface{}) error {
	// 修复不一致的计费记录
	// 具体逻辑依赖于不一致的类型
	return nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}