package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ai-api-gateway/internal/billing/domain"
	"ai-api-gateway/internal/billing/service"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BillingInterceptor 计费拦截器 - 确保所有API调用都会经过计费流程
type BillingInterceptor struct {
	billingManager *service.BillingManager
	logger         logger.Logger
	policy         *BillingPolicy // 计费策略
}

// NewBillingInterceptor 创建计费拦截器
func NewBillingInterceptor(billingManager *service.BillingManager, logger logger.Logger) *BillingInterceptor {
	return &BillingInterceptor{
		billingManager: billingManager,
		logger:         logger,
		policy:         NewBillingPolicy(), // 使用默认计费策略
	}
}

// NewBillingInterceptorWithPolicy 使用自定义策略创建计费拦截器
func NewBillingInterceptorWithPolicy(billingManager *service.BillingManager, logger logger.Logger, policy *BillingPolicy) *BillingInterceptor {
	return &BillingInterceptor{
		billingManager: billingManager,
		logger:         logger,
		policy:         policy,
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
		userID, apiKeyID, modelID, err := bi.extractRequestInfo(c, billingBehavior)
		if err != nil {
			// 对于只记录日志的请求，即使获取信息失败也继续
			if billingBehavior == BillingBehaviorLogOnly {
				bi.logger.WithFields(map[string]interface{}{
					"path":     c.Request.URL.Path,
					"behavior": "log_only",
					"error":    err.Error(),
				}).Debug("Admin request with incomplete billing info")
				c.Next()
				return
			}
			
			bi.logger.WithFields(map[string]interface{}{
				"path":  c.Request.URL.Path,
				"error": err.Error(),
			}).Warn("Failed to extract request info for billing")
			c.Next()
			return
		}

		// 创建计费上下文
		billingCtx := bi.createBillingContext(c, userID, apiKeyID, modelID, billingBehavior)

		// 存储到上下文中，供后续中间件和处理器使用
		c.Set("billing_context", billingCtx)
		c.Set("billing_behavior", billingBehavior)

		// 只有正常计费的请求才需要预检查
		if billingBehavior == BillingBehaviorNormal {
			// 进行预检查
			preCheckResult, err := bi.billingManager.PreCheck(c.Request.Context(), billingCtx)
			if err != nil {
				bi.logger.WithFields(map[string]interface{}{
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
					"request_id": billingCtx.RequestID,
					"reason":     preCheckResult.Reason,
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
		}

		c.Next()
	}
}

// PostRequestMiddleware 请求后中间件 - 进行实际计费
func (bi *BillingInterceptor) PostRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 获取计费行为
		billingBehaviorInterface, exists := c.Get("billing_behavior")
		if !exists {
			return // 如果没有计费行为，说明在PreRequest阶段就被跳过了
		}
		
		billingBehavior, ok := billingBehaviorInterface.(BillingBehavior)
		if !ok {
			return
		}

		// 完全跳过的请求不处理
		if billingBehavior == BillingBehaviorSkip {
			return
		}

		// 获取计费上下文
		billingCtxInterface, exists := c.Get("billing_context")
		if !exists {
			bi.logger.WithFields(map[string]interface{}{
				"path": c.Request.URL.Path,
			}).Warn("Billing context not found in post-request middleware")
			return
		}

		billingCtx, ok := billingCtxInterface.(*domain.BillingContext)
		if !ok {
			bi.logger.WithFields(map[string]interface{}{
				"path": c.Request.URL.Path,
			}).Error("Invalid billing context type")
			return
		}

		// 更新响应信息
		bi.updateBillingContextWithResponse(c, billingCtx)

		// 根据计费行为决定处理方式
		switch billingBehavior {
		case BillingBehaviorNormal:
			// 正常计费 - 异步任务除外
			if billingCtx.RequestType != entities.RequestTypeMidjourney {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					_, err := bi.billingManager.ProcessRequest(ctx, billingCtx)
					if err != nil {
						bi.logger.WithFields(map[string]interface{}{
							"request_id":     billingCtx.RequestID,
							"billing_behavior": "normal",
							"error":          err.Error(),
						}).Error("Failed to process billing in post-request middleware")
					}
				}()
			}
			
		case BillingBehaviorLogOnly:
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
						"request_id":     billingCtx.RequestID,
						"billing_behavior": "log_only",
						"error":          err.Error(),
					}).Warn("Failed to create usage log for admin request")
				}
			}()
		}
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
		// 这里可以解析请求体获取模型信息，但要注意不能消费请求体
		// 可以通过查询参数或者其他方式获取
		if modelQuery := c.Query("model"); modelQuery != "" {
			if mid, parseErr := strconv.ParseInt(modelQuery, 10, 64); parseErr == nil {
				modelID = mid
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