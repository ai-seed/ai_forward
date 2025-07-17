package services

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/logger"
)

// billingServiceImpl 计费服务实现
type billingServiceImpl struct {
	billingRepo      repositories.BillingRecordRepository
	usageLogRepo     repositories.UsageLogRepository
	modelPricingRepo repositories.ModelPricingRepository
	userRepo         repositories.UserRepository
	logger           logger.Logger
}

// NewBillingService 创建计费服务实例
func NewBillingService(
	billingRepo repositories.BillingRecordRepository,
	usageLogRepo repositories.UsageLogRepository,
	modelPricingRepo repositories.ModelPricingRepository,
	userRepo repositories.UserRepository,
) BillingService {
	return &billingServiceImpl{
		billingRepo:      billingRepo,
		usageLogRepo:     usageLogRepo,
		modelPricingRepo: modelPricingRepo,
		userRepo:         userRepo,
		logger:           logger.GetLogger(), // 使用全局logger
	}
}

// CalculateCost 计算成本
func (s *billingServiceImpl) CalculateCost(ctx context.Context, modelID int64, inputTokens, outputTokens int) (float64, error) {
	// 首先尝试获取基于请求的定价（适用于 Midjourney 等模型）
	requestPricing, err := s.modelPricingRepo.GetPricingByType(ctx, modelID, entities.PricingTypeRequest)
	if err == nil {
		// 使用基于请求的定价
		cost := requestPricing.PricePerUnit * requestPricing.Multiplier
		return cost, nil
	}

	// 获取输入token定价
	inputPricing, err := s.modelPricingRepo.GetPricingByType(ctx, modelID, entities.PricingTypeInput)
	if err != nil {
		// 如果找不到定价，使用默认值（避免服务中断）
		inputPricing = &entities.ModelPricing{
			PricePerUnit: 0.001, // 默认每1000个token $0.001
			Multiplier:   1.5,   // 默认1.5倍率
			Unit:         entities.PricingUnitToken,
			Currency:     "USD",
		}
	}

	// 获取输出token定价
	outputPricing, err := s.modelPricingRepo.GetPricingByType(ctx, modelID, entities.PricingTypeOutput)
	if err != nil {
		// 如果找不到定价，使用默认值
		outputPricing = &entities.ModelPricing{
			PricePerUnit: 0.002, // 默认每1000个token $0.002
			Multiplier:   1.5,   // 默认1.5倍率
			Unit:         entities.PricingUnitToken,
			Currency:     "USD",
		}
	}

	// 计算成本（应用倍率）
	// 注意：价格通常是按1000个token计算的，所以需要除以1000
	// 应用倍率：最终价格 = 基础价格 * 倍率
	inputCost := float64(inputTokens) * inputPricing.PricePerUnit * inputPricing.Multiplier / 1000.0
	outputCost := float64(outputTokens) * outputPricing.PricePerUnit * outputPricing.Multiplier / 1000.0
	totalCost := inputCost + outputCost

	return totalCost, nil
}

// CalculateRequestCost 计算基于请求的成本（适用于 Midjourney 等模型）
func (s *billingServiceImpl) CalculateRequestCost(ctx context.Context, modelID int64) (float64, error) {
	// 获取基于请求的定价
	requestPricing, err := s.modelPricingRepo.GetPricingByType(ctx, modelID, entities.PricingTypeRequest)
	if err != nil {
		return 0, fmt.Errorf("failed to get request pricing: %w", err)
	}

	// 计算成本（应用倍率）
	cost := requestPricing.PricePerUnit * requestPricing.Multiplier
	return cost, nil
}

// ProcessBilling 处理计费
func (s *billingServiceImpl) ProcessBilling(ctx context.Context, usageLog *entities.UsageLog) error {
	if usageLog.Cost <= 0 {
		return nil
	}

	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, usageLog.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// 扣减用户余额
	if err := user.DeductBalance(usageLog.Cost); err != nil {
		return fmt.Errorf("failed to deduct balance: %w", err)
	}

	// 更新用户余额
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// 创建计费记录
	description := fmt.Sprintf("API usage cost for request %s", usageLog.RequestID)
	processedAt := time.Now()
	billingRecord := &entities.BillingRecord{
		UserID:      usageLog.UserID,
		UsageLogID:  usageLog.ID,
		Amount:      usageLog.Cost,
		Currency:    "USD",
		BillingType: entities.BillingTypeUsage,
		Description: &description,
		ProcessedAt: &processedAt,
		Status:      entities.BillingStatusProcessed,
		CreatedAt:   time.Now(),
	}

	if err := s.billingRepo.Create(ctx, billingRecord); err != nil {
		// 如果创建计费记录失败，需要回滚用户余额
		if rollbackErr := user.AddBalance(usageLog.Cost); rollbackErr != nil {
			// 回滚失败，但继续处理
		} else {
			if updateErr := s.userRepo.Update(ctx, user); updateErr != nil {
				// 更新失败，但继续处理
			}
		}
		return fmt.Errorf("failed to create billing record: %w", err)
	}

	return nil
}

// ProcessMidjourneyBilling 处理 Midjourney 任务完成时的计费
func (s *billingServiceImpl) ProcessMidjourneyBilling(ctx context.Context, jobID string, success bool) error {
	// 查找对应的使用日志
	usageLog, err := s.usageLogRepo.GetByRequestID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get usage log for job %s: %w", jobID, err)
	}

	if usageLog == nil {
		return fmt.Errorf("usage log not found for job %s", jobID)
	}

	// 检查是否已经计费
	if usageLog.IsBilled {
		return nil // 已经计费，跳过
	}

	// 更新状态码
	if success {
		usageLog.StatusCode = 200 // 成功
	} else {
		usageLog.StatusCode = 500 // 失败
	}

	// 只有成功的任务才计费
	if success {
		// 标记为已计费
		usageLog.IsBilled = true

		// 更新使用日志
		if err := s.usageLogRepo.Update(ctx, usageLog); err != nil {
			return fmt.Errorf("failed to update usage log: %w", err)
		}

		// 处理计费
		if err := s.ProcessBilling(ctx, usageLog); err != nil {
			// 如果计费失败，回滚 IsBilled 状态
			usageLog.IsBilled = false
			s.usageLogRepo.Update(ctx, usageLog)
			return fmt.Errorf("failed to process billing: %w", err)
		}
	} else {
		// 失败的任务不计费，但更新状态
		if err := s.usageLogRepo.Update(ctx, usageLog); err != nil {
			return fmt.Errorf("failed to update usage log status: %w", err)
		}
	}

	return nil
}

// GetBillingHistory 获取计费历史
func (s *billingServiceImpl) GetBillingHistory(ctx context.Context, userID int64, offset, limit int) ([]*entities.BillingRecord, error) {
	return s.billingRepo.GetByUserID(ctx, userID, offset, limit)
}

// GetBillingStats 获取计费统计
func (s *billingServiceImpl) GetBillingStats(ctx context.Context, userID int64, startTime, endTime time.Time) (*BillingStats, error) {
	records, err := s.billingRepo.GetByDateRange(ctx, startTime, endTime, 0, 1000) // 获取前1000条记录
	if err != nil {
		return nil, fmt.Errorf("failed to get billing records: %w", err)
	}

	// 过滤用户记录
	var userRecords []*entities.BillingRecord
	for _, record := range records {
		if record.UserID == userID {
			userRecords = append(userRecords, record)
		}
	}

	stats := &BillingStats{
		UserID:    userID,
		StartTime: startTime,
		EndTime:   endTime,
	}

	for _, record := range userRecords {
		stats.TotalAmount += record.Amount
		stats.TotalRecords++

		switch record.Status {
		case entities.BillingStatusProcessed:
			stats.ProcessedAmount += record.Amount
			stats.ProcessedRecords++
		case entities.BillingStatusFailed:
			stats.FailedAmount += record.Amount
			stats.FailedRecords++
		case entities.BillingStatusPending:
			stats.PendingAmount += record.Amount
			stats.PendingRecords++
		}
	}

	return stats, nil
}

// BillingStats 计费统计
type BillingStats struct {
	UserID           int64     `json:"user_id"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	TotalAmount      float64   `json:"total_amount"`
	TotalRecords     int       `json:"total_records"`
	ProcessedAmount  float64   `json:"processed_amount"`
	ProcessedRecords int       `json:"processed_records"`
	FailedAmount     float64   `json:"failed_amount"`
	FailedRecords    int       `json:"failed_records"`
	PendingAmount    float64   `json:"pending_amount"`
	PendingRecords   int       `json:"pending_records"`
}
