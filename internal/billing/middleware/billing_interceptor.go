package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"ai-api-gateway/internal/billing/domain"
	"ai-api-gateway/internal/billing/service"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BillingInterceptor 计费拦截器 - 确保所有API调用都会经过计费流程
type BillingInterceptor struct {
	billingManager *service.BillingManager
	logger         logger.Logger
	policy         *BillingPolicy // 计费策略
	modelRepo      repositories.ModelRepository // 添加模型仓储以查询模型ID
}

// NewBillingInterceptor 创建计费拦截器
func NewBillingInterceptor(billingManager *service.BillingManager, logger logger.Logger, modelRepo repositories.ModelRepository) *BillingInterceptor {
	return &BillingInterceptor{
		billingManager: billingManager,
		logger:         logger,
		policy:         NewBillingPolicy(), // 使用默认计费策略
		modelRepo:      modelRepo,
	}
}

// NewBillingInterceptorWithPolicy 使用自定义策略创建计费拦截器
func NewBillingInterceptorWithPolicy(billingManager *service.BillingManager, logger logger.Logger, policy *BillingPolicy, modelRepo repositories.ModelRepository) *BillingInterceptor {
	return &BillingInterceptor{
		billingManager: billingManager,
		logger:         logger,
		policy:         policy,
		modelRepo:      modelRepo,
	}
}

// PreRequestMiddleware 请求前中间件 - 进行预检查
func (bi *BillingInterceptor) PreRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取计费行为
		billingBehavior := bi.policy.GetBillingBehavior(c)
		
		// 完全跳过计费的请求
		if billingBehavior == BillingBehaviorSkip {
			c.Next()
			return
		}

		// 获取请求信息 - 对于admin接口可能没有用户ID
		bi.logger.WithFields(map[string]interface{}{
			"event":           "billing_info_extraction_start",
			"path":            c.Request.URL.Path,
			"method":          c.Request.Method,
			"billing_behavior": billingBehavior,
		}).Debug("Starting billing information extraction")

		userID, apiKeyID, modelID, err := bi.extractRequestInfo(c, billingBehavior)
		if err != nil {
			// 对于只记录日志的请求，即使获取信息失败也继续
			if billingBehavior == BillingBehaviorLogOnly {
				bi.logger.WithFields(map[string]interface{}{
					"event":    "billing_info_extraction_incomplete",
					"path":     c.Request.URL.Path,
					"behavior": "log_only",
					"error":    err.Error(),
				}).Debug("Admin request with incomplete billing info")
				c.Next()
				return
			}
			
			bi.logger.WithFields(map[string]interface{}{
				"event": "billing_info_extraction_failed",
				"path":  c.Request.URL.Path,
				"error": err.Error(),
			}).Warn("Failed to extract request info for billing")
			c.Next()
			return
		}

		bi.logger.WithFields(map[string]interface{}{
			"event":           "billing_info_extraction_completed",
			"path":            c.Request.URL.Path,
			"user_id":         userID,
			"api_key_id":      apiKeyID,
			"model_id":        modelID,
			"billing_behavior": billingBehavior,
		}).Debug("Billing information extracted successfully")

		// 创建计费上下文
		bi.logger.WithFields(map[string]interface{}{
			"event":      "billing_context_creation",
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"model_id":   modelID,
		}).Debug("Creating billing context")

		billingCtx := bi.createBillingContext(c, userID, apiKeyID, modelID, billingBehavior)

		bi.logger.WithFields(map[string]interface{}{
			"event":           "billing_context_created",
			"request_id":      billingCtx.RequestID,
			"billing_stage":   billingCtx.BillingStage,
			"request_type":    billingCtx.RequestType,
		}).Debug("Billing context created successfully")

		// 存储到上下文中，供后续中间件和处理器使用
		c.Set("billing_context", billingCtx)
		c.Set("billing_behavior", billingBehavior)

		// 只有正常计费的请求才需要预检查
		if billingBehavior == BillingBehaviorNormal {
			bi.logger.WithFields(map[string]interface{}{
				"event":      "billing_precheck_start",
				"request_id": billingCtx.RequestID,
				"user_id":    billingCtx.UserID,
				"api_key_id": billingCtx.APIKeyID,
			}).Debug("Starting billing pre-check")

			// 进行预检查
			preCheckResult, err := bi.billingManager.PreCheck(c.Request.Context(), billingCtx)
			if err != nil {
				bi.logger.WithFields(map[string]interface{}{
					"event":      "billing_precheck_error",
					"request_id": billingCtx.RequestID,
					"error":      err.Error(),
				}).Error("Billing pre-check failed")

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  "BILLING_PRECHECK_FAILED",
				})
				c.Abort()
				return
			}

			// 检查是否可以继续
			if !preCheckResult.CanProceed {
				bi.logger.WithFields(map[string]interface{}{
					"event":           "billing_precheck_blocked",
					"request_id":      billingCtx.RequestID,
					"reason":          preCheckResult.Reason,
					"estimated_cost":  preCheckResult.EstimatedCost,
					"balance_ok":      preCheckResult.BalanceOK,
					"quota_ok":        preCheckResult.QuotaOK,
				}).Warn("Request blocked by billing pre-check")

				var statusCode int
				var errorCode string
				var message string

				switch preCheckResult.Reason {
				case "insufficient_balance":
					statusCode = http.StatusPaymentRequired
					errorCode = "INSUFFICIENT_BALANCE"
					message = "Insufficient account balance"
				case "tokens_quota_exceeded":
					statusCode = http.StatusTooManyRequests
					errorCode = "TOKEN_QUOTA_EXCEEDED"
					message = "Token quota exceeded"
				case "requests_quota_exceeded":
					statusCode = http.StatusTooManyRequests
					errorCode = "REQUEST_QUOTA_EXCEEDED"
					message = "Request quota exceeded"
				case "cost_quota_exceeded":
					statusCode = http.StatusTooManyRequests
					errorCode = "COST_QUOTA_EXCEEDED"
					message = "Cost quota exceeded"
				default:
					statusCode = http.StatusForbidden
					errorCode = "REQUEST_BLOCKED"
					message = "Request blocked by billing system"
				}

				c.JSON(statusCode, gin.H{
					"error":   message,
					"code":    errorCode,
					"details": preCheckResult.Details,
				})
				c.Abort()
				return
			}

			// 存储预检查结果
			c.Set("billing_precheck_result", preCheckResult)
			
			bi.logger.WithFields(map[string]interface{}{
				"event":          "billing_precheck_passed",
				"request_id":     billingCtx.RequestID,
				"estimated_cost": preCheckResult.EstimatedCost,
			}).Debug("Billing pre-check passed, proceeding with request")
		} else {
			bi.logger.WithFields(map[string]interface{}{
				"event":           "billing_precheck_skipped",
				"request_id":      billingCtx.RequestID,
				"billing_behavior": billingBehavior,
			}).Debug("Billing pre-check skipped for non-normal billing behavior")
		}

		c.Next()
	}
}

// PostRequestMiddleware 请求后中间件 - 进行实际计费
func (bi *BillingInterceptor) PostRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		bi.logger.WithFields(map[string]interface{}{
			"event":  "billing_post_request_start",
			"path":   c.Request.URL.Path,
			"status": c.Writer.Status(),
		}).Debug("Starting post-request billing processing")

		// 获取计费行为
		billingBehaviorInterface, exists := c.Get("billing_behavior")
		if !exists {
			bi.logger.WithFields(map[string]interface{}{
				"event": "billing_post_request_skipped",
				"path":  c.Request.URL.Path,
				"reason": "no_billing_behavior",
			}).Debug("Post-request billing skipped - no billing behavior found")
			return // 如果没有计费行为，说明在PreRequest阶段就被跳过了
		}
		
		billingBehavior, ok := billingBehaviorInterface.(BillingBehavior)
		if !ok {
			bi.logger.WithFields(map[string]interface{}{
				"event": "billing_post_request_skipped",
				"path":  c.Request.URL.Path,
				"reason": "invalid_billing_behavior_type",
			}).Debug("Post-request billing skipped - invalid billing behavior type")
			return
		}

		// 完全跳过的请求不处理
		if billingBehavior == BillingBehaviorSkip {
			bi.logger.WithFields(map[string]interface{}{
				"event":           "billing_post_request_skipped",
				"path":            c.Request.URL.Path,
				"billing_behavior": billingBehavior,
				"reason":          "skip_behavior",
			}).Debug("Post-request billing skipped - skip behavior")
			return
		}

		// 获取计费上下文
		billingCtxInterface, exists := c.Get("billing_context")
		if !exists {
			bi.logger.WithFields(map[string]interface{}{
				"event": "billing_post_request_error",
				"path":  c.Request.URL.Path,
				"reason": "billing_context_not_found",
			}).Warn("Billing context not found in post-request middleware")
			return
		}

		billingCtx, ok := billingCtxInterface.(*domain.BillingContext)
		if !ok {
			bi.logger.WithFields(map[string]interface{}{
				"event": "billing_post_request_error",
				"path":  c.Request.URL.Path,
				"reason": "invalid_billing_context_type",
			}).Error("Invalid billing context type")
			return
		}

		bi.logger.WithFields(map[string]interface{}{
			"event":           "billing_context_update_start",
			"request_id":      billingCtx.RequestID,
			"response_status": c.Writer.Status(),
		}).Debug("Updating billing context with response information")

		// 更新响应信息
		bi.updateBillingContextWithResponse(c, billingCtx)

		bi.logger.WithFields(map[string]interface{}{
			"event":           "billing_context_updated",
			"request_id":      billingCtx.RequestID,
			"success":         billingCtx.Success,
			"input_tokens":    billingCtx.InputTokens,
			"output_tokens":   billingCtx.OutputTokens,
			"total_tokens":    billingCtx.TotalTokens,
			"duration_ms":     billingCtx.DurationMs,
		}).Debug("Billing context updated with response information")

		// 根据计费行为决定处理方式
		switch billingBehavior {
		case BillingBehaviorNormal:
			// 正常计费 - 异步任务除外
			if billingCtx.RequestType != entities.RequestTypeMidjourney {
				bi.logger.WithFields(map[string]interface{}{
					"event":        "billing_async_start",
					"request_id":   billingCtx.RequestID,
					"request_type": billingCtx.RequestType,
				}).Debug("Starting asynchronous billing process")

				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					_, err := bi.billingManager.ProcessRequest(ctx, billingCtx)
					if err != nil {
						bi.logger.WithFields(map[string]interface{}{
							"event":            "billing_async_failed",
							"request_id":       billingCtx.RequestID,
							"billing_behavior": "normal",
							"error":            err.Error(),
						}).Error("Failed to process billing in post-request middleware")
					} else {
						bi.logger.WithFields(map[string]interface{}{
							"event":      "billing_async_completed",
							"request_id": billingCtx.RequestID,
						}).Info("Asynchronous billing completed successfully")
					}
				}()
			} else {
				bi.logger.WithFields(map[string]interface{}{
					"event":        "billing_async_skipped",
					"request_id":   billingCtx.RequestID,
					"request_type": billingCtx.RequestType,
					"reason":       "async_task_will_be_billed_on_completion",
				}).Debug("Billing skipped for async task - will be billed on completion")
			}
			
		case BillingBehaviorLogOnly:
			bi.logger.WithFields(map[string]interface{}{
				"event":      "log_only_billing_start",
				"request_id": billingCtx.RequestID,
			}).Debug("Starting log-only billing process")

			// 只记录使用日志，不计费
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				// 创建使用日志但不计费
				usageLog := billingCtx.ToUsageLog()
				usageLog.Cost = 0 // 不计费
				usageLog.IsBilled = true // 标记为已处理，避免被一致性检查误判
				
				if err := bi.billingManager.CreateUsageLogOnly(ctx, usageLog); err != nil {
					bi.logger.WithFields(map[string]interface{}{
						"event":            "log_only_billing_failed",
						"request_id":       billingCtx.RequestID,
						"billing_behavior": "log_only",
						"error":            err.Error(),
					}).Warn("Failed to create usage log for admin request")
				} else {
					bi.logger.WithFields(map[string]interface{}{
						"event":      "log_only_billing_completed",
						"request_id": billingCtx.RequestID,
					}).Debug("Log-only billing completed successfully")
				}
			}()
		}

		bi.logger.WithFields(map[string]interface{}{
			"event":           "billing_post_request_completed",
			"request_id":      billingCtx.RequestID,
			"billing_behavior": billingBehavior,
		}).Debug("Post-request billing processing completed")
	}
}

// shouldSkipBilling 判断是否应该跳过计费 (已废弃，使用policy.GetBillingBehavior)
func (bi *BillingInterceptor) shouldSkipBilling(c *gin.Context) bool {
	return bi.policy.GetBillingBehavior(c) == BillingBehaviorSkip
}

// extractRequestInfo 提取请求信息
func (bi *BillingInterceptor) extractRequestInfo(c *gin.Context, billingBehavior BillingBehavior) (userID, apiKeyID, modelID int64, err error) {
	// 对于管理员请求，可能没有user_id，尝试获取admin_id
	if billingBehavior == BillingBehaviorLogOnly {
		// 尝试获取admin信息
		if adminInfo := bi.policy.GetAdminInfo(c); adminInfo != nil && adminInfo.IsAdmin {
			userID = adminInfo.AdminID // 使用admin_id作为user_id
		}
		// 对于admin请求，API Key可能不存在，使用默认值
		apiKeyID = -1 // 特殊值表示admin请求
	} else {
		// 普通用户请求
		if userIDInterface, exists := c.Get("user_id"); exists {
			if uid, ok := userIDInterface.(int64); ok {
				userID = uid
			}
		}
	}

	// API Key ID获取
	if apiKeyIDInterface, exists := c.Get("api_key_id"); exists {
		if akid, ok := apiKeyIDInterface.(int64); ok {
			apiKeyID = akid
		}
	}

	// 从请求中提取模型ID
	if modelParam := c.Param("model"); modelParam != "" {
		if mid, parseErr := strconv.ParseInt(modelParam, 10, 64); parseErr == nil {
			modelID = mid
		}
	}

	// 从请求体中提取模型信息（如果是POST请求）
	if modelID == 0 && c.Request.Method == "POST" {
		// 首先尝试从查询参数获取
		if modelQuery := c.Query("model"); modelQuery != "" {
			if mid, parseErr := strconv.ParseInt(modelQuery, 10, 64); parseErr == nil {
				modelID = mid
			}
		}
		
		// 如果还是没有模型ID，尝试从请求体解析模型名称
		if modelID == 0 {
			modelName := bi.extractModelFromRequestBody(c)
			if modelName != "" && bi.modelRepo != nil {
				ctx := context.Background()
				model, err := bi.modelRepo.GetBySlug(ctx, modelName)
				if err != nil {
					bi.logger.WithFields(map[string]interface{}{
						"event":      "model_lookup_in_precheck_failed",
						"model_name": modelName,
						"error":      err.Error(),
					}).Debug("Failed to lookup model ID during pre-check")
				} else if model != nil {
					modelID = model.ID
					bi.logger.WithFields(map[string]interface{}{
						"event":      "model_lookup_in_precheck_success",
						"model_name": modelName,
						"model_id":   model.ID,
					}).Debug("Successfully resolved model ID during pre-check")
				}
			}
		}
	}

	// 对于普通计费请求，userID和apiKeyID都是必须的
	if billingBehavior == BillingBehaviorNormal && (userID == 0 || apiKeyID == 0) {
		return 0, 0, 0, fmt.Errorf("missing required billing information: userID=%d, apiKeyID=%d", userID, apiKeyID)
	}
	
	// 对于只记录日志的请求，如果没有用户信息也可以继续
	if billingBehavior == BillingBehaviorLogOnly && userID == 0 {
		userID = -1 // 使用特殊值表示匿名admin请求
	}

	return userID, apiKeyID, modelID, nil
}

// createBillingContext 创建计费上下文
func (bi *BillingInterceptor) createBillingContext(c *gin.Context, userID, apiKeyID, modelID int64, billingBehavior BillingBehavior) *domain.BillingContext {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
		c.Header("X-Request-ID", requestID)
	}

	// 确定请求类型
	requestType := entities.RequestTypeAPI
	path := c.Request.URL.Path
	
	if path == "/api/v1/midjourney" || path == "/api/v1/midjourney/submit" {
		requestType = entities.RequestTypeMidjourney
	}

	// 管理员请求有特殊标识
	var billingStage domain.BillingStage
	if billingBehavior == BillingBehaviorLogOnly {
		billingStage = domain.BillingStageLogOnly // 新增日志专用阶段
	} else {
		billingStage = domain.BillingStagePreCheck
	}

	return &domain.BillingContext{
		RequestID:    requestID,
		UserID:       userID,
		APIKeyID:     apiKeyID,
		ModelID:      modelID,
		RequestTime:  time.Now(),
		Method:       c.Request.Method,
		Endpoint:     path,
		RequestType:  requestType,
		BillingStage: billingStage,
	}
}

// updateBillingContextWithResponse 使用响应信息更新计费上下文
func (bi *BillingInterceptor) updateBillingContextWithResponse(c *gin.Context, billingCtx *domain.BillingContext) {
	// 更新状态码
	billingCtx.Status = c.Writer.Status()
	billingCtx.Success = billingCtx.Status >= 200 && billingCtx.Status < 300

	// 尝试从AI处理器设置的上下文中获取模型信息
	if billingCtx.ModelID == 0 {
		if modelName, exists := c.Get("model_name"); exists {
			if modelStr, ok := modelName.(string); ok {
				bi.logger.WithFields(map[string]interface{}{
					"event":        "model_lookup_start",
					"request_id":   billingCtx.RequestID,
					"model_name":   modelStr,
					"path":         c.Request.URL.Path,
				}).Debug("Looking up model ID from model name")

				// 通过模型名称查询模型ID
				if bi.modelRepo != nil {
					ctx := context.Background()
					model, err := bi.modelRepo.GetBySlug(ctx, modelStr)
					if err != nil {
						bi.logger.WithFields(map[string]interface{}{
							"event":        "model_lookup_failed",
							"request_id":   billingCtx.RequestID,
							"model_name":   modelStr,
							"error":        err.Error(),
						}).Warn("Failed to lookup model ID from model name")
					} else if model != nil {
						billingCtx.ModelID = model.ID
						bi.logger.WithFields(map[string]interface{}{
							"event":        "model_lookup_success",
							"request_id":   billingCtx.RequestID,
							"model_name":   modelStr,
							"model_id":     model.ID,
						}).Info("Successfully resolved model ID from model name")
					} else {
						bi.logger.WithFields(map[string]interface{}{
							"event":        "model_lookup_not_found",
							"request_id":   billingCtx.RequestID,
							"model_name":   modelStr,
						}).Warn("Model not found in database")
					}
				}
			}
		}
	}

	// 获取Token信息（如果有的话）
	if inputTokens, exists := c.Get("input_tokens"); exists {
		if tokens, ok := inputTokens.(int); ok {
			billingCtx.InputTokens = tokens
		}
	}

	if outputTokens, exists := c.Get("output_tokens"); exists {
		if tokens, ok := outputTokens.(int); ok {
			billingCtx.OutputTokens = tokens
		}
	}

	if totalTokens, exists := c.Get("total_tokens"); exists {
		if tokens, ok := totalTokens.(int); ok {
			billingCtx.TotalTokens = tokens
		}
	}

	// 获取provider信息
	if providerID, exists := c.Get("provider_id"); exists {
		if pid, ok := providerID.(int64); ok {
			billingCtx.ProviderID = pid
			bi.logger.WithFields(map[string]interface{}{
				"event":       "provider_information_retrieved",
				"request_id":  billingCtx.RequestID,
				"provider_id": pid,
				"path":        c.Request.URL.Path,
			}).Debug("Retrieved provider information from handler context")
		} else {
			bi.logger.WithFields(map[string]interface{}{
				"event":         "provider_information_type_mismatch",
				"request_id":    billingCtx.RequestID,
				"provider_type": fmt.Sprintf("%T", providerID),
				"path":          c.Request.URL.Path,
			}).Warn("Provider information exists but is not int64")
		}
	} else {
		bi.logger.WithFields(map[string]interface{}{
			"event":      "provider_information_not_found",
			"request_id": billingCtx.RequestID,
			"path":       c.Request.URL.Path,
		}).Warn("No provider information found in handler context")
	}

	// 获取成本信息（注意：这是AI提供商返回的成本，计费系统会基于自己的定价重新计算）
	if costUsed, exists := c.Get("cost_used"); exists {
		if cost, ok := costUsed.(float64); ok {
			bi.logger.WithFields(map[string]interface{}{
				"event":      "cost_information_retrieved",
				"request_id": billingCtx.RequestID,
				"cost":       cost,
				"path":       c.Request.URL.Path,
			}).Debug("Retrieved cost information from handler context")
			
			// 将AI提供商的成本保存到Details中作为参考
			if billingCtx.BillingStage == domain.BillingStageLogOnly {
				// 对于仅记录日志的请求，可以使用AI提供商的成本
				billingCtx.ActualCost = cost
			} else {
				// 对于正常计费，优先使用Handler提供的准确成本
				// 因为Handler已经基于数据库定价计算了正确的成本
				billingCtx.ActualCost = cost
				if billingCtx.EstimatedCost == 0 {
					billingCtx.EstimatedCost = cost
				}
			}
		} else {
			bi.logger.WithFields(map[string]interface{}{
				"event":      "cost_information_type_mismatch",
				"request_id": billingCtx.RequestID,
				"cost_type":  fmt.Sprintf("%T", costUsed),
				"path":       c.Request.URL.Path,
			}).Warn("Cost information exists but is not float64")
		}
	} else {
		bi.logger.WithFields(map[string]interface{}{
			"event":      "cost_information_not_found",
			"request_id": billingCtx.RequestID,
			"path":       c.Request.URL.Path,
		}).Warn("No cost information found in handler context")
	}

	// 计算响应时间
	if startTime, exists := c.Get("start_time"); exists {
		if st, ok := startTime.(time.Time); ok {
			billingCtx.DurationMs = int(time.Since(st).Milliseconds())
		}
	}

	// 获取错误信息（如果有的话）
	if errorMsg, exists := c.Get("error_message"); exists {
		if msg, ok := errorMsg.(string); ok {
			billingCtx.ErrorMessage = msg
		}
	}

	// 更新计费阶段
	if billingCtx.Success {
		billingCtx.BillingStage = domain.BillingStagePending
	} else {
		billingCtx.BillingStage = domain.BillingStageError
	}

	bi.logger.WithFields(map[string]interface{}{
		"event":           "billing_context_response_update",
		"request_id":      billingCtx.RequestID,
		"final_model_id":  billingCtx.ModelID,
		"success":         billingCtx.Success,
		"input_tokens":    billingCtx.InputTokens,
		"output_tokens":   billingCtx.OutputTokens,
		"total_tokens":    billingCtx.TotalTokens,
	}).Debug("Billing context updated with response information")
}

// GetBillingContext 从Gin上下文中获取计费上下文
func GetBillingContext(c *gin.Context) (*domain.BillingContext, bool) {
	if billingCtxInterface, exists := c.Get("billing_context"); exists {
		if billingCtx, ok := billingCtxInterface.(*domain.BillingContext); ok {
			return billingCtx, true
		}
	}
	return nil, false
}

// extractModelFromRequestBody 从请求体中提取模型名称（不消费请求体）
func (bi *BillingInterceptor) extractModelFromRequestBody(c *gin.Context) string {
	// 只处理包含 JSON 的请求
	if c.GetHeader("Content-Type") != "application/json" && 
	   c.GetHeader("Content-Type") != "application/json; charset=utf-8" {
		return ""
	}

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		bi.logger.WithFields(map[string]interface{}{
			"event": "request_body_read_failed",
			"error": err.Error(),
		}).Debug("Failed to read request body for model extraction")
		return ""
	}

	// 重新设置请求体，这样后续的处理器还能读取
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// 解析 JSON 获取模型信息
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		bi.logger.WithFields(map[string]interface{}{
			"event": "request_body_parse_failed",
			"error": err.Error(),
		}).Debug("Failed to parse request body JSON for model extraction")
		return ""
	}

	// 尝试获取模型名称
	if model, exists := requestData["model"]; exists {
		if modelStr, ok := model.(string); ok {
			bi.logger.WithFields(map[string]interface{}{
				"event":      "model_extracted_from_body",
				"model_name": modelStr,
			}).Debug("Successfully extracted model name from request body")
			return modelStr
		}
	}

	return ""
}