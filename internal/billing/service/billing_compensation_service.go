package service

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/billing/domain"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/logger"
)

// BillingCompensationService 计费补偿服务 - 处理计费失败的补偿逻辑
type BillingCompensationService struct {
	billingManager     *BillingManager
	usageLogRepo       repositories.UsageLogRepository
	billingRecordRepo  repositories.BillingRecordRepository
	userRepo           repositories.UserRepository
	auditLogger        *BillingAuditLogger
	logger             logger.Logger
	retryAttempts      int
	retryInterval      time.Duration
}

// NewBillingCompensationService 创建计费补偿服务
func NewBillingCompensationService(
	billingManager *BillingManager,
	usageLogRepo repositories.UsageLogRepository,
	billingRecordRepo repositories.BillingRecordRepository,
	userRepo repositories.UserRepository,
	logger logger.Logger,
) *BillingCompensationService {
	return &BillingCompensationService{
		billingManager:    billingManager,
		usageLogRepo:      usageLogRepo,
		billingRecordRepo: billingRecordRepo,
		userRepo:          userRepo,
		auditLogger:       NewBillingAuditLogger(logger),
		logger:            logger,
		retryAttempts:     3,
		retryInterval:     time.Minute * 5,
	}
}

// CompensationTask 补偿任务
type CompensationTask struct {
	ID          string                 `json:"id"`
	Type        CompensationTaskType   `json:"type"`
	RequestID   string                 `json:"request_id"`
	UsageLogID  int64                  `json:"usage_log_id"`
	UserID      int64                  `json:"user_id"`
	Amount      float64                `json:"amount"`
	Reason      string                 `json:"reason"`
	Status      CompensationStatus     `json:"status"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// CompensationTaskType 补偿任务类型
type CompensationTaskType string

const (
	CompensationTaskRetryBilling CompensationTaskType = "retry_billing"  // 重试计费
	CompensationTaskRefund       CompensationTaskType = "refund"         // 退费
	CompensationTaskAdjustment   CompensationTaskType = "adjustment"     // 余额调整
)

// CompensationStatus 补偿状态
type CompensationStatus string

const (
	CompensationStatusPending   CompensationStatus = "pending"   // 待处理
	CompensationStatusProcessed CompensationStatus = "processed" // 已处理
	CompensationStatusFailed    CompensationStatus = "failed"    // 处理失败
	CompensationStatusExpired   CompensationStatus = "expired"   // 已过期
)

// ProcessFailedBilling 处理计费失败的补偿
func (bcs *BillingCompensationService) ProcessFailedBilling(ctx context.Context, requestID string, reason string) error {
	// 查找使用日志
	usageLog, err := bcs.usageLogRepo.GetByRequestID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get usage log for request %s: %w", requestID, err)
	}

	if usageLog == nil {
		return fmt.Errorf("usage log not found for request %s", requestID)
	}

	// 创建重试计费任务
	task := &CompensationTask{
		ID:          fmt.Sprintf("retry_%s_%d", requestID, time.Now().Unix()),
		Type:        CompensationTaskRetryBilling,
		RequestID:   requestID,
		UsageLogID:  usageLog.ID,
		UserID:      usageLog.UserID,
		Amount:      usageLog.Cost,
		Reason:      reason,
		Status:      CompensationStatusPending,
		Attempts:    0,
		MaxAttempts: bcs.retryAttempts,
		Data:        make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	task.Data["original_request_type"] = usageLog.RequestType
	task.Data["original_cost"] = usageLog.Cost
	task.Data["original_status_code"] = usageLog.StatusCode

	// 执行补偿任务
	return bcs.executeCompensationTask(ctx, task)
}

// ProcessRefund 处理退费补偿
func (bcs *BillingCompensationService) ProcessRefund(ctx context.Context, requestID string, amount float64, reason string) error {
	// 查找使用日志
	usageLog, err := bcs.usageLogRepo.GetByRequestID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get usage log for request %s: %w", requestID, err)
	}

	if usageLog == nil {
		return fmt.Errorf("usage log not found for request %s", requestID)
	}

	// 检查是否已经计费
	if !usageLog.IsBilled {
		return fmt.Errorf("cannot refund unbilled request: %s", requestID)
	}

	// 创建退费任务
	task := &CompensationTask{
		ID:          fmt.Sprintf("refund_%s_%d", requestID, time.Now().Unix()),
		Type:        CompensationTaskRefund,
		RequestID:   requestID,
		UsageLogID:  usageLog.ID,
		UserID:      usageLog.UserID,
		Amount:      amount,
		Reason:      reason,
		Status:      CompensationStatusPending,
		Attempts:    0,
		MaxAttempts: bcs.retryAttempts,
		Data:        make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	task.Data["original_cost"] = usageLog.Cost
	task.Data["refund_amount"] = amount

	// 执行补偿任务
	return bcs.executeCompensationTask(ctx, task)
}

// ProcessBalanceAdjustment 处理余额调整补偿
func (bcs *BillingCompensationService) ProcessBalanceAdjustment(ctx context.Context, userID int64, amount float64, reason string) error {
	// 创建余额调整任务
	task := &CompensationTask{
		ID:          fmt.Sprintf("adjustment_%d_%d", userID, time.Now().Unix()),
		Type:        CompensationTaskAdjustment,
		UserID:      userID,
		Amount:      amount,
		Reason:      reason,
		Status:      CompensationStatusPending,
		Attempts:    0,
		MaxAttempts: bcs.retryAttempts,
		Data:        make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	task.Data["adjustment_amount"] = amount
	task.Data["adjustment_reason"] = reason

	// 执行补偿任务
	return bcs.executeCompensationTask(ctx, task)
}

// executeCompensationTask 执行补偿任务
func (bcs *BillingCompensationService) executeCompensationTask(ctx context.Context, task *CompensationTask) error {
	bcs.logger.WithFields(map[string]interface{}{
		"task_id":   task.ID,
		"task_type": task.Type,
		"user_id":   task.UserID,
		"amount":    task.Amount,
		"reason":    task.Reason,
	}).Info("Executing compensation task")

	for attempt := 1; attempt <= task.MaxAttempts; attempt++ {
		task.Attempts = attempt
		task.UpdatedAt = time.Now()

		var err error
		switch task.Type {
		case CompensationTaskRetryBilling:
			err = bcs.executeRetryBilling(ctx, task)
		case CompensationTaskRefund:
			err = bcs.executeRefund(ctx, task)
		case CompensationTaskAdjustment:
			err = bcs.executeBalanceAdjustment(ctx, task)
		default:
			err = fmt.Errorf("unknown compensation task type: %s", task.Type)
		}

		if err == nil {
			// 任务执行成功
			task.Status = CompensationStatusProcessed
			now := time.Now()
			task.ProcessedAt = &now
			
			bcs.auditLogger.LogBillingResult(&domain.BillingContext{
				RequestID: task.RequestID,
				UserID:    task.UserID,
			}, &domain.BillingResult{
				Success: true,
				Amount:  task.Amount,
			}, fmt.Sprintf("compensation_task_completed_%s", task.Type))

			bcs.logger.WithFields(map[string]interface{}{
				"task_id":   task.ID,
				"task_type": task.Type,
				"attempts":  attempt,
			}).Info("Compensation task completed successfully")

			return nil
		}

		// 任务执行失败，记录错误
		task.Error = err.Error()
		
		bcs.logger.WithFields(map[string]interface{}{
			"task_id":   task.ID,
			"task_type": task.Type,
			"attempt":   attempt,
			"error":     err.Error(),
		}).Warn("Compensation task attempt failed")

		// 如果还有重试机会，等待一段时间再重试
		if attempt < task.MaxAttempts {
			time.Sleep(bcs.retryInterval)
		}
	}

	// 所有重试都失败了
	task.Status = CompensationStatusFailed
	
	bcs.auditLogger.LogBillingError(&domain.BillingContext{
		RequestID: task.RequestID,
		UserID:    task.UserID,
	}, "compensation_task_failed", fmt.Errorf("compensation task failed after %d attempts: %s", task.MaxAttempts, task.Error))

	bcs.logger.WithFields(map[string]interface{}{
		"task_id":      task.ID,
		"task_type":    task.Type,
		"max_attempts": task.MaxAttempts,
		"final_error":  task.Error,
	}).Error("Compensation task failed after all attempts")

	return fmt.Errorf("compensation task failed after %d attempts: %s", task.MaxAttempts, task.Error)
}

// executeRetryBilling 执行重试计费
func (bcs *BillingCompensationService) executeRetryBilling(ctx context.Context, task *CompensationTask) error {
	// 重新处理计费
	return bcs.billingManager.ProcessAsyncCompletion(ctx, task.RequestID, true)
}

// executeRefund 执行退费
func (bcs *BillingCompensationService) executeRefund(ctx context.Context, task *CompensationTask) error {
	// 获取用户信息
	user, err := bcs.userRepo.GetByID(ctx, task.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// 退费（增加余额）
	if err := user.AddBalance(task.Amount); err != nil {
		return fmt.Errorf("failed to add balance for refund: %w", err)
	}

	// 更新用户余额
	if err := bcs.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user balance for refund: %w", err)
	}

	// 创建退费记录
	description := fmt.Sprintf("Refund for request %s: %s", task.RequestID, task.Reason)
	processedAt := time.Now()
	
	billingRecord := &entities.BillingRecord{
		UserID:      task.UserID,
		UsageLogID:  task.UsageLogID,
		Amount:      -task.Amount, // 负数表示退费
		Currency:    "USD",
		BillingType: entities.BillingTypeRefund,
		Description: &description,
		ProcessedAt: &processedAt,
		Status:      entities.BillingStatusProcessed,
		CreatedAt:   time.Now(),
	}

	if err := bcs.billingRecordRepo.Create(ctx, billingRecord); err != nil {
		// 如果创建退费记录失败，需要回滚用户余额
		user.DeductBalance(task.Amount) // 回滚
		bcs.userRepo.Update(ctx, user)
		return fmt.Errorf("failed to create refund record: %w", err)
	}

	// 记录退费审计日志
	bcs.auditLogger.LogRefund(task.RequestID, task.UserID, task.UsageLogID, task.Amount, task.Reason)

	return nil
}

// executeBalanceAdjustment 执行余额调整
func (bcs *BillingCompensationService) executeBalanceAdjustment(ctx context.Context, task *CompensationTask) error {
	// 获取用户信息
	user, err := bcs.userRepo.GetByID(ctx, task.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// 调整余额
	if task.Amount > 0 {
		if err := user.AddBalance(task.Amount); err != nil {
			return fmt.Errorf("failed to add balance: %w", err)
		}
	} else if task.Amount < 0 {
		if err := user.DeductBalance(-task.Amount); err != nil {
			return fmt.Errorf("failed to deduct balance: %w", err)
		}
	} else {
		// 金额为0，无需调整
		return nil
	}

	// 更新用户余额
	if err := bcs.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// 创建调整记录
	description := fmt.Sprintf("Balance adjustment: %s", task.Reason)
	processedAt := time.Now()
	
	billingRecord := &entities.BillingRecord{
		UserID:      task.UserID,
		UsageLogID:  0, // 余额调整通常不关联特定的使用日志
		Amount:      task.Amount,
		Currency:    "USD",
		BillingType: entities.BillingTypeAdjustment,
		Description: &description,
		ProcessedAt: &processedAt,
		Status:      entities.BillingStatusProcessed,
		CreatedAt:   time.Now(),
	}

	if err := bcs.billingRecordRepo.Create(ctx, billingRecord); err != nil {
		// 如果创建调整记录失败，需要回滚用户余额
		if task.Amount > 0 {
			user.DeductBalance(task.Amount)
		} else {
			user.AddBalance(-task.Amount)
		}
		bcs.userRepo.Update(ctx, user)
		return fmt.Errorf("failed to create adjustment record: %w", err)
	}

	return nil
}

// GetCompensationTaskHistory 获取补偿任务历史（这里需要实现任务存储）
func (bcs *BillingCompensationService) GetCompensationTaskHistory(ctx context.Context, userID int64, limit int) ([]*CompensationTask, error) {
	// 这里需要实现任务存储和查询逻辑
	// 可以存储在数据库中，也可以使用内存缓存
	return []*CompensationTask{}, nil
}