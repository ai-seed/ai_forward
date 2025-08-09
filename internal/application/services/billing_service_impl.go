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
	s.logger.WithFields(map[string]interface{}{
		"event":         "cost_calculation_start",
		"model_id":      modelID,
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
	}).Debug("Starting cost calculation")

	// 一次性获取模型的所有有效定价
	pricings, err := s.modelPricingRepo.GetCurrentPricing(ctx, modelID)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"event":    "cost_calculation_failed",
			"model_id": modelID,
			"error":    err.Error(),
		}).Error("Failed to get model pricing")
		return 0, fmt.Errorf("failed to get model pricing: %w", err)
	}

	if len(pricings) == 0 {
		s.logger.WithFields(map[string]interface{}{
			"event":    "cost_calculation_using_defaults",
			"model_id": modelID,
		}).Warn("No pricing found for model, using default values")
		
		// 使用默认定价计算
		return s.calculateWithDefaults(inputTokens, outputTokens), nil
	}

	s.logger.WithFields(map[string]interface{}{
		"model_id":       modelID,
		"pricing_count":  len(pricings),
	}).Debug("Retrieved model pricing records")

	var totalCost float64
	var costDetails []map[string]interface{}
	var foundRequestPricing bool

	// 遍历所有定价记录，根据类型计算成本
	for _, pricing := range pricings {
		var cost float64
		var units float64

		switch pricing.PricingType {
		case entities.PricingTypeRequest:
			// 基于请求的定价（如 Midjourney）
			cost = pricing.PricePerUnit * pricing.Multiplier
			units = 1
			foundRequestPricing = true
			
			s.logger.WithFields(map[string]interface{}{
				"pricing_type":   "request",
				"price_per_unit": pricing.PricePerUnit,
				"multiplier":     pricing.Multiplier,
				"cost":           cost,
			}).Debug("Calculated request-based cost")

		case entities.PricingTypeInput:
			// 输入token定价
			if inputTokens > 0 {
				units = float64(inputTokens)
				cost = units * pricing.PricePerUnit * pricing.Multiplier / 1000.0
				
				s.logger.WithFields(map[string]interface{}{
					"pricing_type":   "input",
					"tokens":         inputTokens,
					"price_per_unit": pricing.PricePerUnit,
					"multiplier":     pricing.Multiplier,
					"cost":           cost,
				}).Debug("Calculated input token cost")
			}

		case entities.PricingTypeOutput:
			// 输出token定价
			if outputTokens > 0 {
				units = float64(outputTokens)
				cost = units * pricing.PricePerUnit * pricing.Multiplier / 1000.0
				
				s.logger.WithFields(map[string]interface{}{
					"pricing_type":   "output",
					"tokens":         outputTokens,
					"price_per_unit": pricing.PricePerUnit,
					"multiplier":     pricing.Multiplier,
					"cost":           cost,
				}).Debug("Calculated output token cost")
			}

		default:
			s.logger.WithFields(map[string]interface{}{
				"pricing_type": pricing.PricingType,
				"model_id":     modelID,
			}).Warn("Unknown pricing type, skipping")
			continue
		}

		// 累加成本
		totalCost += cost

		// 记录成本详情
		costDetails = append(costDetails, map[string]interface{}{
			"type":           pricing.PricingType,
			"price_per_unit": pricing.PricePerUnit,
			"multiplier":     pricing.Multiplier,
			"units":          units,
			"cost":           cost,
			"currency":       pricing.Currency,
		})
	}

	// 如果找到基于请求的定价，则忽略token定价（请求定价优先级更高）
	if foundRequestPricing {
		// 重新计算，只考虑请求定价
		totalCost = 0
		var requestCostDetails []map[string]interface{}
		
		for _, pricing := range pricings {
			if pricing.PricingType == entities.PricingTypeRequest {
				cost := pricing.PricePerUnit * pricing.Multiplier
				totalCost += cost
				
				requestCostDetails = append(requestCostDetails, map[string]interface{}{
					"type":           pricing.PricingType,
					"price_per_unit": pricing.PricePerUnit,
					"multiplier":     pricing.Multiplier,
					"units":          1,
					"cost":           cost,
					"currency":       pricing.Currency,
				})
			}
		}
		costDetails = requestCostDetails
		
		s.logger.WithFields(map[string]interface{}{
			"event":      "cost_calculation_request_based",
			"model_id":   modelID,
			"total_cost": totalCost,
		}).Info("Using request-based pricing (ignoring token pricing)")
	}

	s.logger.WithFields(map[string]interface{}{
		"event":        "cost_calculation_completed",
		"model_id":     modelID,
		"input_tokens": inputTokens,
		"output_tokens": outputTokens,
		"total_cost":   totalCost,
		"cost_details": costDetails,
		"pricing_mode": func() string {
			if foundRequestPricing {
				return "request_based"
			}
			return "token_based"
		}(),
	}).Info("Cost calculation completed")

	return totalCost, nil
}

// calculateWithDefaults 使用默认定价计算成本
func (s *billingServiceImpl) calculateWithDefaults(inputTokens, outputTokens int) float64 {
	// 默认定价
	inputPricePerUnit := 0.001  // 每1000个token $0.001
	outputPricePerUnit := 0.002 // 每1000个token $0.002
	multiplier := 1.5           // 1.5倍率

	inputCost := float64(inputTokens) * inputPricePerUnit * multiplier / 1000.0
	outputCost := float64(outputTokens) * outputPricePerUnit * multiplier / 1000.0
	totalCost := inputCost + outputCost

	s.logger.WithFields(map[string]interface{}{
		"input_tokens":        inputTokens,
		"output_tokens":       outputTokens,
		"input_price_unit":    inputPricePerUnit,
		"output_price_unit":   outputPricePerUnit,
		"multiplier":          multiplier,
		"input_cost":          inputCost,
		"output_cost":         outputCost,
		"total_cost":          totalCost,
		"pricing_mode":        "default",
	}).Info("Calculated cost using default pricing")

	return totalCost
}

// CalculateRequestCost 计算基于请求的成本（适用于 Midjourney 等模型）
func (s *billingServiceImpl) CalculateRequestCost(ctx context.Context, modelID int64) (float64, error) {
	s.logger.WithFields(map[string]interface{}{
		"event":    "request_cost_calculation_start",
		"model_id": modelID,
	}).Debug("Starting request-based cost calculation")

	// 获取基于请求的定价
	requestPricing, err := s.modelPricingRepo.GetPricingByType(ctx, modelID, entities.PricingTypeRequest)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"event":    "request_cost_calculation_failed",
			"model_id": modelID,
			"error":    err.Error(),
		}).Error("Failed to get request pricing")
		return 0, fmt.Errorf("failed to get request pricing: %w", err)
	}

	// 计算成本（应用倍率）
	cost := requestPricing.PricePerUnit * requestPricing.Multiplier
	
	s.logger.WithFields(map[string]interface{}{
		"event":          "request_cost_calculation_completed",
		"model_id":       modelID,
		"price_per_unit": requestPricing.PricePerUnit,
		"multiplier":     requestPricing.Multiplier,
		"final_cost":     cost,
		"currency":       requestPricing.Currency,
	}).Info("Request-based cost calculated")
	
	return cost, nil
}

// ProcessBilling 处理计费
func (s *billingServiceImpl) ProcessBilling(ctx context.Context, usageLog *entities.UsageLog) error {
	s.logger.WithFields(map[string]interface{}{
		"event":        "billing_process_start",
		"request_id":   usageLog.RequestID,
		"usage_log_id": usageLog.ID,
		"user_id":      usageLog.UserID,
		"cost":         usageLog.Cost,
	}).Info("Starting billing process")

	if usageLog.Cost <= 0 {
		s.logger.WithFields(map[string]interface{}{
			"event":        "billing_process_skipped",
			"request_id":   usageLog.RequestID,
			"usage_log_id": usageLog.ID,
			"cost":         usageLog.Cost,
			"reason":       "zero_or_negative_cost",
		}).Debug("Billing process skipped due to zero or negative cost")
		return nil
	}

	// 获取用户信息
	s.logger.WithFields(map[string]interface{}{
		"user_id": usageLog.UserID,
	}).Debug("Fetching user information")

	user, err := s.userRepo.GetByID(ctx, usageLog.UserID)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"event":      "billing_process_failed",
			"request_id": usageLog.RequestID,
			"user_id":    usageLog.UserID,
			"error":      err.Error(),
			"step":       "get_user",
		}).Error("Failed to get user information")
		return fmt.Errorf("failed to get user: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":         usageLog.UserID,
		"current_balance": user.Balance,
		"deduct_amount":   usageLog.Cost,
	}).Info("User information retrieved, preparing to deduct balance")

	// 扣减用户余额
	originalBalance := user.Balance
	if err := user.DeductBalance(usageLog.Cost); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"event":           "billing_process_failed",
			"request_id":      usageLog.RequestID,
			"user_id":         usageLog.UserID,
			"original_balance": originalBalance,
			"deduct_amount":   usageLog.Cost,
			"error":           err.Error(),
			"step":            "deduct_balance",
		}).Error("Failed to deduct user balance")
		return fmt.Errorf("failed to deduct balance: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":         usageLog.UserID,
		"original_balance": originalBalance,
		"new_balance":     user.Balance,
		"deducted_amount": usageLog.Cost,
	}).Info("User balance deducted successfully")

	// 更新用户余额
	if err := s.userRepo.Update(ctx, user); err != nil {
		// 尝试回滚余额
		rollbackErr := user.AddBalance(usageLog.Cost)
		s.logger.WithFields(map[string]interface{}{
			"event":           "billing_process_failed",
			"request_id":      usageLog.RequestID,
			"user_id":         usageLog.UserID,
			"error":           err.Error(),
			"rollback_error":  rollbackErr,
			"step":            "update_user_balance",
		}).Error("Failed to update user balance")
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":     usageLog.UserID,
		"new_balance": user.Balance,
	}).Info("User balance updated successfully")

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

	s.logger.WithFields(map[string]interface{}{
		"request_id":   usageLog.RequestID,
		"usage_log_id": usageLog.ID,
		"amount":       billingRecord.Amount,
		"currency":     billingRecord.Currency,
	}).Debug("Creating billing record")

	if err := s.billingRepo.Create(ctx, billingRecord); err != nil {
		// 如果创建计费记录失败，需要回滚用户余额
		s.logger.WithFields(map[string]interface{}{
			"event":        "billing_record_creation_failed",
			"request_id":   usageLog.RequestID,
			"usage_log_id": usageLog.ID,
			"error":        err.Error(),
		}).Error("Failed to create billing record, attempting rollback")

		if rollbackErr := user.AddBalance(usageLog.Cost); rollbackErr != nil {
			s.logger.WithFields(map[string]interface{}{
				"event":          "rollback_failed",
				"request_id":     usageLog.RequestID,
				"user_id":        usageLog.UserID,
				"rollback_error": rollbackErr.Error(),
			}).Error("Failed to rollback user balance after billing record creation failure")
		} else {
			if updateErr := s.userRepo.Update(ctx, user); updateErr != nil {
				s.logger.WithFields(map[string]interface{}{
					"event":       "rollback_update_failed",
					"request_id":  usageLog.RequestID,
					"user_id":     usageLog.UserID,
					"update_error": updateErr.Error(),
				}).Error("Failed to update user balance during rollback")
			} else {
				s.logger.WithFields(map[string]interface{}{
					"event":      "rollback_successful",
					"request_id": usageLog.RequestID,
					"user_id":    usageLog.UserID,
					"amount":     usageLog.Cost,
				}).Info("Successfully rolled back user balance")
			}
		}
		return fmt.Errorf("failed to create billing record: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"event":             "billing_process_completed",
		"request_id":        usageLog.RequestID,
		"usage_log_id":      usageLog.ID,
		"billing_record_id": billingRecord.ID,
		"user_id":           usageLog.UserID,
		"amount":            usageLog.Cost,
		"final_balance":     user.Balance,
	}).Info("Billing process completed successfully")

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
