package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/functioncall"
	"ai-api-gateway/internal/infrastructure/gateway"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/infrastructure/tokenizer"
	"ai-api-gateway/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// AIHandler AI请求处理器
type AIHandler struct {
	gatewayService           gateway.GatewayService
	modelService             services.ModelService
	usageLogService          services.UsageLogService
	logger                   logger.Logger
	config                   *config.Config
	functionCallHandler      functioncall.FunctionCallHandler
	providerModelSupportRepo repositories.ProviderModelSupportRepository
	httpClient               clients.HTTPClient
	aiClient                 clients.AIProviderClient
	thinkingService          services.ThinkingService
	tokenizer                *tokenizer.SimpleTokenizer
}

// NewAIHandler 创建AI请求处理器
func NewAIHandler(
	gatewayService gateway.GatewayService,
	modelService services.ModelService,
	usageLogService services.UsageLogService,
	logger logger.Logger,
	config *config.Config,
	functionCallHandler functioncall.FunctionCallHandler,
	providerModelSupportRepo repositories.ProviderModelSupportRepository,
	httpClient clients.HTTPClient,
	aiClient clients.AIProviderClient,
	thinkingService services.ThinkingService,
) *AIHandler {
	return &AIHandler{
		gatewayService:           gatewayService,
		modelService:             modelService,
		usageLogService:          usageLogService,
		logger:                   logger,
		config:                   config,
		functionCallHandler:      functionCallHandler,
		providerModelSupportRepo: providerModelSupportRepo,
		httpClient:               httpClient,
		aiClient:                 aiClient,
		thinkingService:          thinkingService,
		tokenizer:                tokenizer.NewSimpleTokenizer(),
	}
}

// handleStreamingRequest 处理流式请求
func (h *AIHandler) handleStreamingRequest(c *gin.Context, gatewayRequest *gateway.GatewayRequest, requestID string, userID, apiKeyID int64) {
	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Request-ID", requestID)

	// 获取响应写入器
	w := c.Writer

	// 立即发送头部并刷新，确保浏览器识别为 EventStream
	c.Status(http.StatusOK)
	w.Flush()

	// 预先获取并设置 provider 信息，确保 billing 中间件能够获取到
	if err := h.presetProviderInfo(c.Request.Context(), c, gatewayRequest.ModelSlug); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Warn("Failed to preset provider info for streaming request")
	}

	// 移除了 Function Call 和 Thinking 模式的特殊处理
	// 所有请求都直接转发原始数据

	// 创建流式响应通道
	streamChan := make(chan *gateway.StreamChunk, 100)
	errorChan := make(chan error, 1)

	// 启动流式处理
	go func() {
		defer func() {
			// 安全关闭channels
			select {
			case <-streamChan:
			default:
				close(streamChan)
			}

			select {
			case <-errorChan:
			default:
				close(errorChan)
			}
		}()

		routeResponse, err := h.gatewayService.ProcessStreamRequest(c.Request.Context(), gatewayRequest, streamChan)
		if err != nil {
			select {
			case errorChan <- err:
			case <-c.Request.Context().Done():
				// 如果上下文已取消，不发送错误
			}
		}

		// 记录路由响应信息（暂时记录日志，后续会用于设置provider_id）
		if routeResponse != nil {
			h.logger.WithFields(map[string]interface{}{
				"request_id":    requestID,
				"provider_id":   routeResponse.Provider.ID,
				"provider_name": routeResponse.Provider.Name,
			}).Debug("Got route response from streaming request")
		}
	}()

	// 发送流式数据
	var totalTokens int
	var totalCost float64
	var outputContent strings.Builder // 收集输出内容用于token计算
	var inputTokens int

	// 计算输入token数量
	if gatewayRequest.Request.Messages != nil {
		var messages []map[string]interface{}
		for _, msg := range gatewayRequest.Request.Messages {
			messages = append(messages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
		inputTokens = h.tokenizer.CountTokensFromMessages(messages)
	} else if gatewayRequest.Request.Prompt != "" {
		inputTokens = h.tokenizer.CountTokens(gatewayRequest.Request.Prompt)
	}

	for {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				// 流结束，处理token统计
				outputTokens := 0

				// 如果从chunk中获取到了准确的token信息，使用它
				if totalTokens > 0 {
					// 使用AI提供商提供的准确token统计
					if inputTokens == 0 {
						// 如果没有单独的输入token统计，估算一下
						inputTokens = h.tokenizer.CountTokensFromMessages([]map[string]interface{}{
							{"role": "user", "content": gatewayRequest.Request.Messages[len(gatewayRequest.Request.Messages)-1].Content},
						})
						outputTokens = totalTokens - inputTokens
						if outputTokens < 0 {
							outputTokens = totalTokens / 3 // 粗略估算输出约占1/3
							inputTokens = totalTokens - outputTokens
						}
					}
				} else {
					// 没有准确token信息，使用估算
					outputTokens = h.tokenizer.EstimateOutputTokensFromContent(outputContent.String())
					if inputTokens == 0 {
						if gatewayRequest.Request.Messages != nil {
							var messages []map[string]interface{}
							for _, msg := range gatewayRequest.Request.Messages {
								messages = append(messages, map[string]interface{}{
									"role":    msg.Role,
									"content": msg.Content,
								})
							}
							inputTokens = h.tokenizer.CountTokensFromMessages(messages)
						}
					}
					totalTokens = inputTokens + outputTokens

					// 记录详细的估算信息，方便调试和优化
					debugInfo := h.tokenizer.DebugTokenCount(outputContent.String())
					h.logger.WithFields(map[string]interface{}{
						"request_id":     requestID,
						"estimated":      true,
						"input_tokens":   inputTokens,
						"output_tokens":  outputTokens,
						"content_length": outputContent.Len(),
						"debug_info":     debugInfo,
					}).Warn("Using estimated token count for streaming response - compare with actual provider tokens")
				}

				// 流结束时不需要发送额外的结束标记，因为原始数据中已经包含了
				// 但仍需要flush确保数据发送完成
				w.Flush()

				// 输出流式AI提供商响应结果
				h.logger.WithFields(map[string]interface{}{
					"request_id":     requestID,
					"user_id":        userID,
					"api_key_id":     apiKeyID,
					"input_tokens":   inputTokens,
					"output_tokens":  outputTokens,
					"total_tokens":   totalTokens,
					"total_cost":     totalCost,
					"stream_type":    "completed",
					"content_length": outputContent.Len(),
				}).Info("AI provider streaming response completed successfully")

				// 设置使用量到上下文
				c.Set("tokens_used", totalTokens)
				c.Set("cost_used", totalCost)
				c.Set("input_tokens", inputTokens)
				c.Set("output_tokens", outputTokens)
				c.Set("total_tokens", totalTokens)

				// 设置 provider 信息供计费中间件使用
				// TODO: 这是一个临时解决方案，需要完善流式架构来直接传递provider信息
				providerID, providerName := h.extractProviderInfoFromRequestID(requestID)
				if providerID > 0 {
					h.logger.WithFields(map[string]interface{}{
						"request_id":    requestID,
						"provider_id":   providerID,
						"provider_name": providerName,
						"stream":        true,
					}).Debug("Setting provider information for billing middleware (streaming)")
					c.Set("provider_id", providerID)
					c.Set("provider_name", providerName)
				} else {
					h.logger.WithFields(map[string]interface{}{
						"request_id": requestID,
						"issue":      "streaming_provider_info_missing",
					}).Warn("Could not extract provider information from streaming request")
				}

				return
			}

			// 收集输出内容（用于token统计）
			if chunk.Content != "" {
				outputContent.WriteString(chunk.Content)
			}

			// 累计使用量（如果chunk中包含usage信息）
			if chunk.Usage != nil {
				totalTokens += chunk.Usage.TotalTokens
			}
			if chunk.Cost != nil {
				totalCost += chunk.Cost.TotalCost
			}

			// 处理原始SSE数据，为每行添加 event type
			if len(chunk.RawData) > 0 {
				// 解析原始数据并添加 event type
				rawDataStr := string(chunk.RawData)
				lines := strings.Split(rawDataStr, "\n")

				for _, line := range lines {
					if strings.HasPrefix(line, "data: ") {
						// 为数据行添加 event type
						w.Write([]byte("event: message\n"))
						w.Write([]byte(line + "\n"))
						w.Write([]byte("\n")) // 添加空行分隔
					} else if line != "" {
						// 保持其他行不变（如注释行）
						w.Write([]byte(line + "\n"))
					}
				}

				w.Flush()
			} else {
				// 如果没有原始数据，记录警告但继续处理
				h.logger.WithFields(map[string]interface{}{
					"request_id": requestID,
					"chunk_id":   chunk.ID,
				}).Warn("Stream chunk missing raw data, skipping")
			}

		case err := <-errorChan:
			if err != nil {
				h.logger.WithFields(map[string]interface{}{
					"request_id": requestID,
					"user_id":    userID,
					"api_key_id": apiKeyID,
					"error":      err.Error(),
				}).Error("Stream processing failed")

				// 发送错误事件
				errorData := map[string]interface{}{
					"error": map[string]interface{}{
						"message": "Stream processing failed",
						"type":    "server_error",
						"code":    "stream_error",
					},
				}

				jsonData, _ := json.Marshal(errorData)
				w.Write([]byte(fmt.Sprintf("data: %s\n\n", jsonData)))
				w.Flush()
			}
			return

		case <-c.Request.Context().Done():
			// 客户端断开连接
			h.logger.WithFields(map[string]interface{}{
				"request_id": requestID,
			}).Info("Client disconnected from stream")
			return
		}
	}
}

// ChatCompletions 处理聊天完成请求
// @Summary 聊天补全
// @Description 创建聊天补全请求，兼容OpenAI API格式。支持流式和非流式响应。
// @Tags AI接口
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body clients.ChatCompletionRequest true "聊天补全请求"
// @Success 200 {object} clients.AIResponse "聊天补全响应"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 429 {object} dto.Response "请求过于频繁"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /v1/chat/completions [post]
func (h *AIHandler) ChatCompletions(c *gin.Context) {
	// 获取认证信息
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"Authentication required",
			nil,
		))
		return
	}

	apiKeyID, exists := middleware.GetAPIKeyIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"API key required",
			nil,
		))
		return
	}

	// 解析请求体
	var chatRequest clients.ChatCompletionRequest
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"error":      err.Error(),
		}).Warn("Invalid request body")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	// 验证必需字段
	if chatRequest.Model == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"MISSING_MODEL",
			"Model is required",
			nil,
		))
		return
	}

	if len(chatRequest.Messages) == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"MISSING_MESSAGES",
			"Messages array is required for chat completions",
			nil,
		))
		return
	}

	// 获取请求ID
	requestID := middleware.GetRequestIDFromContext(c)

	// 转换为通用 AIRequest 结构
	aiRequest := &clients.AIRequest{
		Model:       chatRequest.Model,
		Messages:    chatRequest.Messages,
		MaxTokens:   chatRequest.MaxTokens,
		Temperature: chatRequest.Temperature,
		Stream:      chatRequest.Stream,
		Tools:       chatRequest.Tools,
		ToolChoice:  chatRequest.ToolChoice,
		WebSearch:   chatRequest.WebSearch,
		Thinking:    chatRequest.Thinking,
	}

	// 如果启用了思考模式，处理思考请求
	if h.thinkingService.IsThinkingEnabled(aiRequest) {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"user_id":    userID,
			"thinking":   true,
		}).Info("Processing request with thinking mode")

		processedRequest, err := h.thinkingService.ProcessThinkingRequest(c.Request.Context(), aiRequest)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("Failed to process thinking request")

			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
				"THINKING_PROCESS_FAILED",
				"Failed to process thinking request",
				map[string]interface{}{
					"request_id": requestID,
				},
			))
			return
		}
		aiRequest = processedRequest
	}

	// 删除了 Function Call 和联网搜索的自动工具添加

	// 设置模型信息到上下文，供计费系统使用
	c.Set("model_name", aiRequest.Model)

	// 构造网关请求
	gatewayRequest := &gateway.GatewayRequest{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		ModelSlug: aiRequest.Model,
		Request:   aiRequest,
		RequestID: requestID,
	}

	// 检查是否为流式请求
	if aiRequest.Stream {
		h.handleStreamingRequest(c, gatewayRequest, requestID, userID, apiKeyID)
		return
	}

	// 处理非流式请求
	response, err := h.gatewayService.ProcessRequest(c.Request.Context(), gatewayRequest)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"model":      aiRequest.Model,
			"error":      err.Error(),
		}).Error("Failed to process AI request")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"REQUEST_FAILED",
			"Failed to process request",
			map[string]interface{}{
				"request_id": requestID,
			},
		))
		return
	}

	// 输出AI提供商响应结果
	h.logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"user_id":       userID,
		"api_key_id":    apiKeyID,
		"model":         aiRequest.Model,
		"provider":      response.Provider,
		"duration_ms":   response.Duration.Milliseconds(),
		"total_tokens":  response.Usage.TotalTokens,
		"input_tokens":  response.Usage.InputTokens,
		"output_tokens": response.Usage.OutputTokens,
		"total_cost":    response.Cost.TotalCost,
		"response_data": response.Response,
	}).Info("AI provider response received successfully")

	// 设置使用量到上下文（用于配额中间件和计费系统）
	if response.Usage != nil {
		c.Set("tokens_used", response.Usage.TotalTokens)
		c.Set("input_tokens", response.Usage.InputTokens)
		c.Set("output_tokens", response.Usage.OutputTokens)
		c.Set("total_tokens", response.Usage.TotalTokens)
	}
	if response.Cost != nil {
		c.Set("cost_used", response.Cost.TotalCost)
	}

	// 设置 provider 信息供计费中间件使用（从网关响应中获取）
	h.logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"provider_id":   response.ProviderID,
		"provider_name": response.Provider,
	}).Debug("Setting provider information for billing middleware")
	c.Set("provider_id", response.ProviderID)
	c.Set("provider_name", response.Provider)

	// 设置响应头
	c.Header("X-Request-ID", requestID)
	c.Header("X-Provider", response.Provider)
	c.Header("X-Model", response.Model)
	c.Header("X-Duration-Ms", strconv.FormatInt(response.Duration.Milliseconds(), 10))

	// 移除了 Function Call 的特殊处理
	// 直接转发原始数据

	// 直接返回上游原始响应数据，不做数据结构化
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", response.RawResponse)
}

// Completions 处理文本完成请求
// @Summary 文本补全
// @Description 创建文本补全请求，兼容OpenAI API格式
// @Tags AI接口
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body clients.CompletionRequest true "文本补全请求"
// @Success 200 {object} clients.AIResponse "文本补全响应"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 429 {object} dto.Response "请求过于频繁"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /v1/completions [post]
func (h *AIHandler) Completions(c *gin.Context) {
	// 获取认证信息
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"Authentication required",
			nil,
		))
		return
	}

	apiKeyID, exists := middleware.GetAPIKeyIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"API key required",
			nil,
		))
		return
	}

	// 解析请求体
	var completionRequest clients.CompletionRequest
	if err := c.ShouldBindJSON(&completionRequest); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"error":      err.Error(),
		}).Warn("Invalid request body")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	// 验证必需字段
	if completionRequest.Model == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"MISSING_MODEL",
			"Model is required",
			nil,
		))
		return
	}

	if completionRequest.Prompt == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"MISSING_PROMPT",
			"Prompt is required",
			nil,
		))
		return
	}

	// 获取请求ID
	requestID := middleware.GetRequestIDFromContext(c)

	// 转换为通用 AIRequest 结构
	aiRequest := &clients.AIRequest{
		Model:       completionRequest.Model,
		Prompt:      completionRequest.Prompt,
		MaxTokens:   completionRequest.MaxTokens,
		Temperature: completionRequest.Temperature,
		Stream:      completionRequest.Stream,
		WebSearch:   completionRequest.WebSearch,
	}

	// 删除了联网搜索的自动工具添加和 prompt 转换

	// 设置模型信息到上下文，供计费系统使用
	c.Set("model_name", aiRequest.Model)

	// 构造网关请求
	gatewayRequest := &gateway.GatewayRequest{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		ModelSlug: aiRequest.Model,
		Request:   aiRequest,
		RequestID: requestID,
	}

	// 处理请求
	response, err := h.gatewayService.ProcessRequest(c.Request.Context(), gatewayRequest)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"model":      aiRequest.Model,
			"error":      err.Error(),
		}).Error("Failed to process AI request")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"REQUEST_FAILED",
			"Failed to process request",
			map[string]interface{}{
				"request_id": requestID,
			},
		))
		return
	}

	// 输出AI提供商响应结果
	h.logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"user_id":       userID,
		"api_key_id":    apiKeyID,
		"model":         aiRequest.Model,
		"provider":      response.Provider,
		"duration_ms":   response.Duration.Milliseconds(),
		"total_tokens":  response.Usage.TotalTokens,
		"input_tokens":  response.Usage.InputTokens,
		"output_tokens": response.Usage.OutputTokens,
		"total_cost":    response.Cost.TotalCost,
		"response_data": response.Response,
	}).Info("AI provider response received successfully")

	// 设置使用量到上下文（用于配额中间件和计费系统）
	if response.Usage != nil {
		c.Set("tokens_used", response.Usage.TotalTokens)
		c.Set("input_tokens", response.Usage.InputTokens)
		c.Set("output_tokens", response.Usage.OutputTokens)
		c.Set("total_tokens", response.Usage.TotalTokens)
	}
	if response.Cost != nil {
		c.Set("cost_used", response.Cost.TotalCost)
	}

	// 设置 provider 信息供计费中间件使用（从网关响应中获取）
	h.logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"provider_id":   response.ProviderID,
		"provider_name": response.Provider,
	}).Debug("Setting provider information for billing middleware")
	c.Set("provider_id", response.ProviderID)
	c.Set("provider_name", response.Provider)

	// 设置响应头
	c.Header("X-Request-ID", requestID)
	c.Header("X-Provider", response.Provider)
	c.Header("X-Model", response.Model)
	c.Header("X-Duration-Ms", strconv.FormatInt(response.Duration.Milliseconds(), 10))

	// 返回AI响应
	c.JSON(http.StatusOK, response.Response)
}

// Models 获取可用模型列表
// @Summary 列出模型
// @Description 获取可用的AI模型列表，包含多语言显示名称和描述
// @Tags AI接口
// @Produce json
// @Security BearerAuth
// @Success 200 {object} clients.ModelsResponse "模型列表"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /v1/models [get]
func (h *AIHandler) Models(c *gin.Context) {
	// 获取可用模型列表
	models, err := h.modelService.GetAvailableModels(c.Request.Context(), 0) // 0 表示获取所有提供商的模型
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to get available models")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to get models",
				"type":    "internal_error",
				"code":    "models_fetch_failed",
			},
		})
		return
	}

	// 转换为 OpenAI API 格式，包含多语言字段
	var modelList []map[string]interface{}
	for _, model := range models {
		modelData := map[string]interface{}{
			"id":       model.Slug,
			"object":   "model",
			"created":  model.CreatedAt.Unix(),
			"owned_by": "system",
		}

		// 添加默认显示名称（向后兼容）
		if model.DisplayName != nil && *model.DisplayName != "" {
			modelData["display_name"] = *model.DisplayName
		} else {
			modelData["display_name"] = model.Name
		}

		// 由于数据库表中没有display_name的多语言字段，暂时跳过
		// 后续可以考虑添加这些字段到数据库表中

		// 添加默认描述（向后兼容）
		if model.Description != nil && *model.Description != "" {
			modelData["description"] = *model.Description
		}

		// 添加多语言描述字段
		if model.DescriptionEN != nil && *model.DescriptionEN != "" {
			modelData["description_en"] = *model.DescriptionEN
		}
		if model.DescriptionZH != nil && *model.DescriptionZH != "" {
			modelData["description_zh"] = *model.DescriptionZH
		}
		if model.DescriptionJP != nil && *model.DescriptionJP != "" {
			modelData["description_ja"] = *model.DescriptionJP
		}

		// 添加扩展信息
		modelData["model_type"] = string(model.ModelType)
		modelData["status"] = string(model.Status)

		// 添加多语言模型类型 - 从数据库字段获取
		if model.ModelTypeEN != nil && *model.ModelTypeEN != "" {
			modelData["model_type_en"] = *model.ModelTypeEN
		} else {
			modelData["model_type_en"] = string(model.ModelType)
		}
		if model.ModelTypeZH != nil && *model.ModelTypeZH != "" {
			modelData["model_type_zh"] = *model.ModelTypeZH
		} else {
			modelData["model_type_zh"] = string(model.ModelType)
		}
		if model.ModelTypeJP != nil && *model.ModelTypeJP != "" {
			modelData["model_type_ja"] = *model.ModelTypeJP
		} else {
			modelData["model_type_ja"] = string(model.ModelType)
		}

		if model.ContextLength != nil {
			modelData["context_length"] = *model.ContextLength
		}

		if model.MaxTokens != nil {
			modelData["max_tokens"] = *model.MaxTokens
		}

		modelData["supports_streaming"] = model.SupportsStreaming
		modelData["supports_functions"] = model.SupportsFunctions

		modelList = append(modelList, modelData)
	}

	// 返回标准 OpenAI API 兼容格式
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   modelList,
	})
}

// extractPreferredLanguage 从Accept-Language头提取首选语言
func (h *AIHandler) extractPreferredLanguage(acceptLanguage string) string {
	if acceptLanguage == "" {
		return "en" // 默认英文
	}

	// 支持的语言列表
	supportedLangs := map[string]bool{
		"en": true,
		"zh": true,
		"ja": true,
	}

	// 简单解析Accept-Language头
	// 支持格式: "zh-CN,zh;q=0.9,en;q=0.8"
	languages := strings.Split(acceptLanguage, ",")

	for _, lang := range languages {
		// 移除权重标识和空白
		lang = strings.TrimSpace(strings.Split(lang, ";")[0])

		// 标准化语言代码
		if strings.HasPrefix(lang, "zh") {
			lang = "zh"
		} else if strings.HasPrefix(lang, "en") {
			lang = "en"
		} else if strings.HasPrefix(lang, "ja") {
			lang = "ja"
		}

		// 检查是否支持
		if supportedLangs[lang] {
			return lang
		}
	}

	return "en" // 默认回退到英文
}

// Usage 获取使用情况
// @Summary 使用统计
// @Description 获取当前用户的API使用统计
// @Tags AI接口
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.UsageResponse "使用统计信息"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /v1/usage [get]
func (h *AIHandler) Usage(c *gin.Context) {
	// 获取认证信息
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"Authentication required",
			nil,
		))
		return
	}

	// 获取使用统计
	stats, err := h.usageLogService.GetUsageStats(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to get usage stats")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"USAGE_STATS_ERROR",
			"Failed to get usage statistics",
			nil,
		))
		return
	}

	// 构造响应数据
	usageResponse := dto.UsageResponse{
		TotalRequests: int(stats.TotalRequests),
		TotalTokens:   int(stats.TotalTokens),
		TotalCost:     stats.TotalCost,
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(usageResponse, "Usage statistics retrieved successfully"))
}

// 删除了 handleFunctionCallResponse 方法

// 删除了 Function Call 相关的流式处理方法

// AnthropicMessages 处理Anthropic Messages API请求
// @Summary Anthropic Messages API - 创建消息
// @Description 完全兼容 Anthropic Messages API 的消息创建接口。支持文本对话、工具调用、流式响应等功能。
// @Description
// @Description **支持的功能：**
// @Description - 文本对话（单轮和多轮）
// @Description - 系统提示（system prompt）
// @Description - 流式响应（Server-Sent Events）
// @Description - 工具调用（Function Calling）
// @Description - 温度控制、Top-K、Top-P 采样
// @Description - 停止序列、最大token限制
// @Description
// @Description **认证方式：**
// @Description - Bearer Token: `Authorization: Bearer YOUR_API_KEY`
// @Description - API Key Header: `x-api-key: YOUR_API_KEY`
// @Description
// @Description **版本控制：**
// @Description - 推荐添加版本头: `anthropic-version: 2023-06-01`
// @Description
// @Description **流式响应：**
// @Description - 设置 `stream: true` 启用流式响应
// @Description - 响应格式为 Server-Sent Events (text/event-stream)
// @Description - 每个数据块以 `data: ` 开头，结束时发送 `data: [DONE]`
// @Tags AI接口
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Security BearerAuth
// @Param anthropic-version header string false "Anthropic API版本" default(2023-06-01)
// @Param x-api-key header string false "API密钥（可替代Authorization头）"
// @Param body body clients.AnthropicMessageRequest true "Anthropic消息请求"
// @Success 200 {object} clients.AnthropicMessageResponse "成功响应"
// @Success 200 {string} string "流式响应 (当stream=true时)" format(text/event-stream)
// @Failure 400 {object} object "请求参数错误" example({"type":"error","error":{"type":"invalid_request_error","message":"model is required"}})
// @Failure 401 {object} object "认证失败" example({"type":"error","error":{"type":"authentication_error","message":"Authentication required"}})
// @Failure 429 {object} object "请求过于频繁" example({"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}})
// @Failure 500 {object} object "服务器内部错误" example({"type":"error","error":{"type":"api_error","message":"Internal server error"}})
// @Router /v1/messages [post]
func (h *AIHandler) AnthropicMessages(c *gin.Context) {
	// 获取认证信息
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"type":    "authentication_error",
				"message": "Authentication required",
			},
		})
		return
	}

	apiKeyID, exists := middleware.GetAPIKeyIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"type":    "authentication_error",
				"message": "API key required",
			},
		})
		return
	}

	// 解析请求体
	var anthropicRequest clients.AnthropicMessageRequest
	if err := c.ShouldBindJSON(&anthropicRequest); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"error":      err.Error(),
		}).Warn("Invalid request body")

		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"type":    "invalid_request_error",
				"message": "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	// 验证必需字段
	if anthropicRequest.Model == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"type":    "invalid_request_error",
				"message": "model is required",
			},
		})
		return
	}

	if len(anthropicRequest.Messages) == 0 {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"type":    "invalid_request_error",
				"message": "messages is required",
			},
		})
		return
	}

	if anthropicRequest.MaxTokens <= 0 {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"type":    "invalid_request_error",
				"message": "max_tokens must be greater than 0",
			},
		})
		return
	}

	// 获取请求ID
	requestID := middleware.GetRequestIDFromContext(c)

	// 设置模型信息到上下文，供计费系统使用
	c.Set("model_name", anthropicRequest.Model)

	// 处理流式请求
	if anthropicRequest.Stream {
		h.handleAnthropicStreamingRequest(c, &anthropicRequest, requestID, userID, apiKeyID)
		return
	}

	// 处理非流式请求
	response, providerInfo, err := h.processAnthropicRequest(c.Request.Context(), &anthropicRequest, userID, apiKeyID, requestID)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"model":      anthropicRequest.Model,
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to process Anthropic request")

		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"type":    "api_error",
				"message": "Failed to process request",
			},
		})
		return
	}

	// 设置使用量到上下文（用于计费中间件）
	if response.Usage.InputTokens > 0 || response.Usage.OutputTokens > 0 {
		totalTokens := response.Usage.InputTokens + response.Usage.OutputTokens
		c.Set("tokens_used", totalTokens)
		c.Set("input_tokens", response.Usage.InputTokens)
		c.Set("output_tokens", response.Usage.OutputTokens)
		c.Set("total_tokens", totalTokens)
	}

	// 设置 provider 信息供计费中间件使用
	if providerInfo != nil {
		h.logger.WithFields(map[string]interface{}{
			"request_id":    requestID,
			"provider_id":   providerInfo.ProviderID,
			"provider_name": providerInfo.ProviderName,
		}).Debug("Setting provider information for billing middleware (Anthropic Messages)")
		c.Set("provider_id", providerInfo.ProviderID)
		c.Set("provider_name", providerInfo.ProviderName)

		// 设置响应头
		c.Header("X-Request-ID", requestID)
		c.Header("X-Provider", providerInfo.ProviderName)
		c.Header("X-Model", response.Model)
	}

	c.JSON(http.StatusOK, response)
}

// convertToClaudeResponse 将通用AI响应转换为Claude格式
func (h *AIHandler) convertToClaudeResponse(response *clients.AIResponse) *clients.ClaudeMessageResponse {
	claudeResponse := &clients.ClaudeMessageResponse{
		ID:    response.ID,
		Type:  "message",
		Role:  "assistant",
		Model: response.Model,
		Usage: clients.ClaudeUsage{
			InputTokens:  response.Usage.PromptTokens,
			OutputTokens: response.Usage.CompletionTokens,
		},
	}

	// 处理错误
	if response.Error != nil {
		claudeResponse.Error = response.Error
		return claudeResponse
	}

	// 转换内容
	var content []clients.ClaudeContent
	for _, choice := range response.Choices {
		if choice.Message.Content != "" {
			content = append(content, clients.ClaudeContent{
				Type: "text",
				Text: choice.Message.Content,
			})
		}

		// 处理工具调用
		for _, toolCall := range choice.Message.ToolCalls {
			content = append(content, clients.ClaudeContent{
				Type:    "tool_use",
				ID:      toolCall.ID,
				Name:    toolCall.Function.Name,
				Input:   toolCall.Function.Arguments,
				ToolUse: &toolCall,
			})
		}

		// 设置停止原因
		switch choice.FinishReason {
		case "stop":
			claudeResponse.StopReason = "end_turn"
		case "length":
			claudeResponse.StopReason = "max_tokens"
		case "tool_calls":
			claudeResponse.StopReason = "tool_use"
		default:
			claudeResponse.StopReason = "end_turn"
		}
	}

	claudeResponse.Content = content
	return claudeResponse
}

// convertToAnthropicResponse 将通用AI响应转换为Anthropic Messages API格式
func (h *AIHandler) convertToAnthropicResponse(response *clients.AIResponse) *clients.AnthropicMessageResponse {
	anthropicResponse := &clients.AnthropicMessageResponse{
		ID:    response.ID,
		Type:  "message",
		Role:  "assistant",
		Model: response.Model,
		Usage: clients.AnthropicUsage{
			InputTokens:  response.Usage.PromptTokens,
			OutputTokens: response.Usage.CompletionTokens,
		},
	}

	// 转换内容
	var content []clients.AnthropicContentBlock
	for _, choice := range response.Choices {
		if choice.Message.Content != "" {
			content = append(content, clients.AnthropicContentBlock{
				Type: "text",
				Text: choice.Message.Content,
			})
		}

		// 处理工具调用
		for _, toolCall := range choice.Message.ToolCalls {
			content = append(content, clients.AnthropicContentBlock{
				Type:  "tool_use",
				ID:    toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: toolCall.Function.Arguments,
			})
		}

		// 设置停止原因
		switch choice.FinishReason {
		case "stop":
			anthropicResponse.StopReason = "end_turn"
		case "length":
			anthropicResponse.StopReason = "max_tokens"
		case "tool_calls":
			anthropicResponse.StopReason = "tool_use"
		default:
			anthropicResponse.StopReason = "end_turn"
		}
	}

	anthropicResponse.Content = content
	return anthropicResponse
}

// handleClaudeStreamingRequest 处理Claude流式请求
func (h *AIHandler) handleClaudeStreamingRequest(c *gin.Context, gatewayRequest *gateway.GatewayRequest, requestID string, userID, apiKeyID int64) {
	// 设置流式响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Request-ID", requestID)

	// 获取响应写入器
	w := c.Writer

	// 立即发送头部并刷新，确保浏览器识别为 EventStream
	c.Status(http.StatusOK)
	w.Flush()

	// 删除了 Function Call 的特殊处理

	// 创建流式响应通道
	streamChan := make(chan *gateway.StreamChunk, 100)
	errorChan := make(chan error, 1)

	// 启动流式处理
	go func() {
		defer func() {
			// 安全关闭channels
			select {
			case <-streamChan:
			default:
				close(streamChan)
			}

			select {
			case <-errorChan:
			default:
				close(errorChan)
			}
		}()

		routeResponse, err := h.gatewayService.ProcessStreamRequest(c.Request.Context(), gatewayRequest, streamChan)
		if err != nil {
			select {
			case errorChan <- err:
			case <-c.Request.Context().Done():
				// 如果上下文已取消，不发送错误
			}
		}

		// 记录路由响应信息（暂时记录日志，后续会用于设置provider_id）
		if routeResponse != nil {
			h.logger.WithFields(map[string]interface{}{
				"request_id":    requestID,
				"provider_id":   routeResponse.Provider.ID,
				"provider_name": routeResponse.Provider.Name,
			}).Debug("Got route response from streaming request")
		}
	}()

	// 发送流式数据
	var totalTokens int
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
		}).Error("Streaming unsupported")
		return
	}

	for {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				// 流已结束
				return
			}

			// 处理原始SSE数据，为每行添加 event type
			if len(chunk.RawData) > 0 {
				// 解析原始数据并添加 event type
				rawDataStr := string(chunk.RawData)
				lines := strings.Split(rawDataStr, "\n")

				for _, line := range lines {
					if strings.HasPrefix(line, "data: ") {
						// 为数据行添加 event type
						w.Write([]byte("event: message\n"))
						w.Write([]byte(line + "\n"))
						w.Write([]byte("\n")) // 添加空行分隔
					} else if line != "" {
						// 保持其他行不变（如注释行）
						w.Write([]byte(line + "\n"))
					}
				}
			} else {
				// 如果没有原始数据，记录警告但继续处理
				h.logger.WithFields(map[string]interface{}{
					"request_id": requestID,
					"chunk_id":   chunk.ID,
				}).Warn("Claude stream chunk missing raw data, skipping")
			}

			flusher.Flush()

			if chunk.Usage != nil {
				totalTokens = chunk.Usage.TotalTokens
			}

		case err := <-errorChan:
			h.logger.WithFields(map[string]interface{}{
				"user_id":    userID,
				"api_key_id": apiKeyID,
				"model":      gatewayRequest.Request.Model,
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("Failed to process Claude stream request")

			// 发送错误事件
			errorEvent := map[string]interface{}{
				"error": map[string]interface{}{
					"type":    "api_error",
					"message": "Failed to process request",
				},
			}
			errorJSON, _ := json.Marshal(errorEvent)
			w.Write([]byte(fmt.Sprintf("data: %s\n\n", errorJSON)))
			flusher.Flush()
			return

		case <-c.Request.Context().Done():
			// 客户端断开连接
			h.logger.WithFields(map[string]interface{}{
				"request_id":   requestID,
				"total_tokens": totalTokens,
			}).Info("Claude stream request cancelled by client")
			return
		}
	}
}

// convertToClaudeStreamChunk 将流式响应块转换为Claude格式
func (h *AIHandler) convertToClaudeStreamChunk(chunk *gateway.StreamChunk) map[string]interface{} {
	claudeChunk := map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]interface{}{
			"type": "text_delta",
			"text": chunk.Content,
		},
	}

	// 如果是最后一个块，添加完成信息
	if chunk.FinishReason != nil {
		claudeChunk["type"] = "message_delta"
		claudeChunk["delta"] = map[string]interface{}{
			"stop_reason": h.convertFinishReasonToClaude(*chunk.FinishReason),
		}

		// 添加使用情况信息
		if chunk.Usage != nil {
			claudeChunk["usage"] = map[string]interface{}{
				"input_tokens":  chunk.Usage.PromptTokens,
				"output_tokens": chunk.Usage.CompletionTokens,
			}
		}
	}

	return claudeChunk
}

// convertFinishReasonToClaude 转换完成原因为Claude格式
func (h *AIHandler) convertFinishReasonToClaude(finishReason string) string {
	switch finishReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}

// extractSystemContent 从system字段中提取文本内容
func (h *AIHandler) extractSystemContent(system interface{}) string {
	if system == nil {
		return ""
	}

	// 如果是字符串，直接返回
	if str, ok := system.(string); ok {
		return str
	}

	// 如果是数组格式，提取text内容
	if arr, ok := system.([]interface{}); ok {
		for _, item := range arr {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, exists := itemMap["type"]; exists && itemType == "text" {
					if text, exists := itemMap["text"]; exists {
						if textStr, ok := text.(string); ok {
							return textStr
						}
					}
				}
			}
		}
	}

	// 如果是单个对象格式
	if objMap, ok := system.(map[string]interface{}); ok {
		if objType, exists := objMap["type"]; exists && objType == "text" {
			if text, exists := objMap["text"]; exists {
				if textStr, ok := text.(string); ok {
					return textStr
				}
			}
		}
	}

	return ""
}

// processAnthropicRequest 处理Anthropic非流式请求
// AnthropicProviderInfo provider信息结构
type AnthropicProviderInfo struct {
	ProviderID   int64
	ProviderName string
}

func (h *AIHandler) processAnthropicRequest(ctx context.Context, request *clients.AnthropicMessageRequest, userID, apiKeyID int64, requestID string) (*clients.AnthropicMessageResponse, *AnthropicProviderInfo, error) {
	// 检测是否为thinking模型
	isThinkingModel := strings.Contains(request.Model, "thinking")

	h.logger.WithFields(map[string]interface{}{
		"request_id":        requestID,
		"user_id":           userID,
		"api_key_id":        apiKeyID,
		"model":             request.Model,
		"max_tokens":        request.MaxTokens,
		"stream":            request.Stream,
		"thinking":          request.Thinking,
		"is_thinking_model": isThinkingModel,
	}).Info("开始处理 Anthropic 请求")

	// 从数据库获取支持该模型的提供商
	supportInfos, err := h.providerModelSupportRepo.GetSupportingProviders(ctx, request.Model)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"model":      request.Model,
			"error":      err.Error(),
		}).Error("获取支持该模型的提供商失败")
		return nil, nil, fmt.Errorf("failed to get supporting providers: %w", err)
	}

	h.logger.WithFields(map[string]interface{}{
		"request_id":      requestID,
		"model":           request.Model,
		"providers_count": len(supportInfos),
	}).Info("从数据库获取到支持的提供商")

	if len(supportInfos) == 0 {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"model":      request.Model,
		}).Error("没有提供商支持该模型")
		return nil, nil, fmt.Errorf("no providers support model: %s", request.Model)
	}

	// 选择第一个可用的提供商（可以后续优化为负载均衡）
	var selectedProvider *entities.Provider
	var selectedModelInfo *entities.ModelSupportInfo
	for i, info := range supportInfos {
		h.logger.WithFields(map[string]interface{}{
			"request_id":    requestID,
			"provider_id":   info.Provider.ID,
			"provider_name": info.Provider.Name,
			"provider_slug": info.Provider.Slug,
			"base_url":      info.Provider.BaseURL,
			"status":        info.Provider.Status,
			"priority":      info.Provider.Priority,
			"index":         i,
		}).Info("检查提供商可用性")

		if info.Provider.IsAvailable() {
			selectedProvider = info.Provider
			selectedModelInfo = info
			h.logger.WithFields(map[string]interface{}{
				"request_id":    requestID,
				"provider_id":   info.Provider.ID,
				"provider_name": info.Provider.Name,
				"provider_slug": info.Provider.Slug,
				"base_url":      info.Provider.BaseURL,
			}).Info("选择了可用的提供商")
			break
		} else {
			h.logger.WithFields(map[string]interface{}{
				"request_id":    requestID,
				"provider_id":   info.Provider.ID,
				"provider_name": info.Provider.Name,
				"status":        info.Provider.Status,
			}).Warn("提供商不可用，跳过")
		}
	}

	if selectedProvider == nil {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"model":      request.Model,
		}).Error("没有可用的提供商")
		return nil, nil, fmt.Errorf("no available providers for model: %s", request.Model)
	}

	// 直接发送Anthropic格式请求到提供商
	h.logger.WithFields(map[string]interface{}{
		"request_id":          requestID,
		"provider_id":         selectedProvider.ID,
		"provider_name":       selectedProvider.Name,
		"provider_slug":       selectedProvider.Slug,
		"base_url":            selectedProvider.BaseURL,
		"upstream_model_name": selectedModelInfo.UpstreamModelName,
	}).Info("开始发送请求到上游提供商")

	response, err := h.sendAnthropicRequestToProvider(ctx, selectedProvider, selectedModelInfo, request)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"request_id":    requestID,
			"provider_id":   selectedProvider.ID,
			"provider_name": selectedProvider.Name,
			"base_url":      selectedProvider.BaseURL,
			"error":         err.Error(),
		}).Error("发送请求到提供商失败")
		return nil, nil, fmt.Errorf("failed to send request to provider: %w", err)
	}

	h.logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"provider_id":   selectedProvider.ID,
		"provider_name": selectedProvider.Name,
		"response_id":   response.ID,
		"input_tokens":  response.Usage.InputTokens,
		"output_tokens": response.Usage.OutputTokens,
	}).Info("成功收到提供商响应")

	// 构造 provider 信息
	providerInfo := &AnthropicProviderInfo{
		ProviderID:   selectedProvider.ID,
		ProviderName: selectedProvider.Name,
	}

	return response, providerInfo, nil
}

// sendAnthropicRequestToProvider 发送Anthropic请求到提供商
func (h *AIHandler) sendAnthropicRequestToProvider(ctx context.Context, provider *entities.Provider, modelInfo *entities.ModelSupportInfo, request *clients.AnthropicMessageRequest) (*clients.AnthropicMessageResponse, error) {
	h.logger.WithFields(map[string]interface{}{
		"provider_id":         provider.ID,
		"provider_name":       provider.Name,
		"provider_slug":       provider.Slug,
		"base_url":            provider.BaseURL,
		"original_model":      request.Model,
		"upstream_model_name": modelInfo.UpstreamModelName,
		"max_tokens":          request.MaxTokens,
		"stream":              request.Stream,
		"thinking":            request.Thinking,
	}).Info("开始直接发送 Anthropic 格式请求")

	// 构造请求URL - 强制使用 /messages 端点
	baseURL := strings.TrimSuffix(provider.BaseURL, "/")
	var url string
	if strings.HasSuffix(baseURL, "/v1") {
		url = fmt.Sprintf("%s/messages", baseURL)
	} else {
		url = fmt.Sprintf("%s/v1/messages", baseURL)
	}

	// 构造请求头
	headers := map[string]string{
		"Content-Type":      "application/json",
		"anthropic-version": "2023-06-01",
	}

	// 设置认证头
	if provider.APIKeyEncrypted != nil {
		headers["x-api-key"] = *provider.APIKeyEncrypted
	}

	// 使用上游模型名称
	requestCopy := *request
	if modelInfo.UpstreamModelName != "" {
		requestCopy.Model = modelInfo.UpstreamModelName
	}

	h.logger.WithFields(map[string]interface{}{
		"provider_id":   provider.ID,
		"provider_name": provider.Name,
		"url":           url,
		"final_model":   requestCopy.Model,
		"headers":       headers,
	}).Info("准备发送 HTTP 请求到 /v1/messages 端点")

	// 直接发送HTTP请求
	httpResponse, err := h.httpClient.Post(ctx, url, &requestCopy, headers)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
			"url":           url,
			"error":         err.Error(),
		}).Error("HTTP 请求失败")
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}

	h.logger.WithFields(map[string]interface{}{
		"provider_id":    provider.ID,
		"provider_name":  provider.Name,
		"url":            url,
		"status_code":    httpResponse.StatusCode,
		"content_length": len(httpResponse.Body),
	}).Info("收到 HTTP 响应")

	// 检查响应状态
	if httpResponse.StatusCode != 200 {
		h.logger.WithFields(map[string]interface{}{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
			"url":           url,
			"status_code":   httpResponse.StatusCode,
			"response_body": string(httpResponse.Body),
		}).Error("提供商返回错误状态码")
		return nil, fmt.Errorf("provider returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// 解析响应
	var anthropicResponse clients.AnthropicMessageResponse
	if err := json.Unmarshal(httpResponse.Body, &anthropicResponse); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
			"url":           url,
			"error":         err.Error(),
			"response_body": string(httpResponse.Body),
		}).Error("解析响应失败")
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 删除了 thinking 内容的特殊处理

	h.logger.WithFields(map[string]interface{}{
		"provider_id":    provider.ID,
		"provider_name":  provider.Name,
		"url":            url,
		"response_id":    anthropicResponse.ID,
		"response_model": anthropicResponse.Model,
		"content_blocks": len(anthropicResponse.Content),
		"stop_reason":    anthropicResponse.StopReason,
		"input_tokens":   anthropicResponse.Usage.InputTokens,
		"output_tokens":  anthropicResponse.Usage.OutputTokens,
	}).Info("Anthropic 响应解析成功")

	return &anthropicResponse, nil
}

// handleAnthropicStreamingRequest 处理Anthropic流式请求
func (h *AIHandler) handleAnthropicStreamingRequest(c *gin.Context, request *clients.AnthropicMessageRequest, requestID string, userID, apiKeyID int64) {
	// 设置流式响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Request-ID", requestID)

	// 获取响应写入器
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
		}).Error("Streaming unsupported")
		return
	}

	// 从数据库获取支持该模型的提供商
	supportInfos, err := h.providerModelSupportRepo.GetSupportingProviders(c.Request.Context(), request.Model)
	if err != nil {
		h.sendStreamError(w, flusher, "Failed to get supporting providers", err)
		return
	}

	if len(supportInfos) == 0 {
		h.sendStreamError(w, flusher, "No providers support model: "+request.Model, nil)
		return
	}

	// 选择第一个可用的提供商
	var selectedProvider *entities.Provider
	var selectedModelInfo *entities.ModelSupportInfo
	for _, info := range supportInfos {
		if info.Provider.IsAvailable() {
			selectedProvider = info.Provider
			selectedModelInfo = info
			break
		}
	}

	if selectedProvider == nil {
		h.sendStreamError(w, flusher, "No available providers for model: "+request.Model, nil)
		return
	}

	// 提前计算 input tokens（不依赖上游返回）
	var inputTokens int
	if len(request.Messages) > 0 {
		// 转换 Anthropic messages 为 tokenizer 可识别的格式
		var messages []map[string]interface{}
		for _, msg := range request.Messages {
			// 使用现有的 GetTextContent 方法提取文本内容
			content := msg.GetTextContent()
			if content != "" {
				messages = append(messages, map[string]interface{}{
					"role":    msg.Role,
					"content": content,
				})
			}
		}
		inputTokens = h.tokenizer.CountTokensFromMessages(messages)
	}

	h.logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"input_tokens":  inputTokens,
		"message_count": len(request.Messages),
	}).Debug("Calculated input tokens for Anthropic streaming request")

	// 直接发送流式请求到 /v1/messages 端点
	usageInfo, err := h.sendAnthropicStreamRequestToProvider(c, selectedProvider, selectedModelInfo, request, w, flusher)
	if err != nil {
		h.sendStreamError(w, flusher, "Failed to process stream request", err)
		return
	}

	// 设置计费相关的上下文数据
	var finalInputTokens = inputTokens // 使用本地计算的值作为默认
	var finalOutputTokens int

	if usageInfo != nil {
		// 如果上游返回了 input_tokens，并且看起来合理，则使用上游的值
		diff := usageInfo.InputTokens - inputTokens
		if diff < 0 {
			diff = -diff
		}
		if usageInfo.InputTokens > 0 && inputTokens > 0 && diff <= inputTokens/10 {
			finalInputTokens = usageInfo.InputTokens
		}
		finalOutputTokens = usageInfo.OutputTokens
	}

	if finalInputTokens > 0 || finalOutputTokens > 0 {
		totalTokens := finalInputTokens + finalOutputTokens
		c.Set("tokens_used", totalTokens)
		c.Set("input_tokens", finalInputTokens)
		c.Set("output_tokens", finalOutputTokens)
		c.Set("total_tokens", totalTokens)

		h.logger.WithFields(map[string]interface{}{
			"request_id":     requestID,
			"local_input":    inputTokens,
			"upstream_input": usageInfo.InputTokens,
			"final_input":    finalInputTokens,
			"output_tokens":  finalOutputTokens,
			"total_tokens":   totalTokens,
		}).Info("Set billing context for Anthropic streaming request")
	}

	// 设置 provider 信息供计费中间件使用
	h.logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"provider_id":   selectedProvider.ID,
		"provider_name": selectedProvider.Name,
	}).Debug("Setting provider information for billing middleware (Anthropic Streaming)")
	c.Set("provider_id", selectedProvider.ID)
	c.Set("provider_name", selectedProvider.Name)
}

// convertToAnthropicStreamChunk 将流式响应块转换为Anthropic格式
func (h *AIHandler) convertToAnthropicStreamChunk(chunk *gateway.StreamChunk) map[string]interface{} {
	// Anthropic 流式响应格式
	anthropicChunk := map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]interface{}{
			"type": "text_delta",
			"text": chunk.Content,
		},
	}

	// 如果是最后一个块，添加完成信息
	if chunk.FinishReason != nil {
		anthropicChunk["type"] = "message_delta"
		anthropicChunk["delta"] = map[string]interface{}{
			"stop_reason": h.convertFinishReasonToAnthropic(*chunk.FinishReason),
		}

		// 添加使用情况信息
		if chunk.Usage != nil {
			anthropicChunk["usage"] = map[string]interface{}{
				"input_tokens":  chunk.Usage.PromptTokens,
				"output_tokens": chunk.Usage.CompletionTokens,
			}
		}
	}

	return anthropicChunk
}

// convertFinishReasonToAnthropic 转换完成原因为Anthropic格式
func (h *AIHandler) convertFinishReasonToAnthropic(finishReason string) string {
	switch finishReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}

// sendStreamError 发送流式错误响应
func (h *AIHandler) sendStreamError(w http.ResponseWriter, flusher http.Flusher, message string, err error) {
	errorEvent := map[string]interface{}{
		"type": "error",
		"error": map[string]interface{}{
			"type":    "api_error",
			"message": message,
		},
	}

	if err != nil {
		errorEvent["error"].(map[string]interface{})["details"] = err.Error()
	}

	errorJSON, _ := json.Marshal(errorEvent)
	w.Write([]byte(fmt.Sprintf("data: %s\n\n", errorJSON)))
	flusher.Flush()
}

// AnthropicStreamUsageInfo 流式使用量信息
type AnthropicStreamUsageInfo struct {
	InputTokens  int
	OutputTokens int
}

// sendAnthropicStreamRequestToProvider 发送Anthropic流式请求到提供商
func (h *AIHandler) sendAnthropicStreamRequestToProvider(c *gin.Context, provider *entities.Provider, modelInfo *entities.ModelSupportInfo, request *clients.AnthropicMessageRequest, w http.ResponseWriter, flusher http.Flusher) (*AnthropicStreamUsageInfo, error) {
	// 构造请求URL - 强制使用 /messages 端点
	baseURL := strings.TrimSuffix(provider.BaseURL, "/")
	var url string
	if strings.HasSuffix(baseURL, "/v1") {
		url = fmt.Sprintf("%s/messages", baseURL)
	} else {
		url = fmt.Sprintf("%s/v1/messages", baseURL)
	}

	// 检测是否为thinking模型
	isThinkingModel := strings.Contains(request.Model, "thinking")

	h.logger.WithFields(map[string]interface{}{
		"provider_id":         provider.ID,
		"provider_name":       provider.Name,
		"provider_slug":       provider.Slug,
		"base_url":            provider.BaseURL,
		"stream_url":          url,
		"original_model":      request.Model,
		"upstream_model_name": modelInfo.UpstreamModelName,
		"is_thinking_model":   isThinkingModel,
	}).Info("开始发送流式请求到 /v1/messages 端点")

	// 构造请求头
	headers := map[string]string{
		"Content-Type":      "application/json",
		"anthropic-version": "2023-06-01",
		"Accept":            "text/event-stream",
	}

	// 设置认证
	if provider.APIKeyEncrypted != nil {
		headers["x-api-key"] = *provider.APIKeyEncrypted
	}

	// 使用上游模型名称
	requestCopy := *request
	if modelInfo.UpstreamModelName != "" {
		requestCopy.Model = modelInfo.UpstreamModelName
	}

	// 确保是流式请求
	requestCopy.Stream = true

	h.logger.WithFields(map[string]interface{}{
		"provider_id": provider.ID,
		"stream_url":  url,
		"final_model": requestCopy.Model,
		"stream":      requestCopy.Stream,
		"thinking":    requestCopy.Thinking,
		"headers":     headers,
	}).Info("准备发送流式 HTTP 请求")

	// 序列化请求体
	requestBody, err := json.Marshal(&requestCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 设置请求头
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 处理流式响应并收集使用量信息
	usageInfo := &AnthropicStreamUsageInfo{}
	scanner := bufio.NewScanner(resp.Body)

	// 用于跟踪thinking状态
	var currentEvent string
	var isThinkingBlock bool
	var currentBlockIndex int

	for scanner.Scan() {
		line := scanner.Text()

		// 处理 event: 行
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimSpace(line[7:]) // 移除 "event: " 前缀
			w.Write([]byte(line + "\n"))
			flusher.Flush()
		} else if strings.HasPrefix(line, "data: ") {
			data := line[6:] // 移除 "data: " 前缀
			if data == "[DONE]" {
				w.Write([]byte("data: [DONE]\n\n"))
				flusher.Flush()
				break
			}

			// 尝试解析数据以提取使用量信息和thinking状态
			var streamData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &streamData); err == nil {
				// 检查是否包含使用量信息
				if usage, exists := streamData["usage"]; exists {
					if usageMap, ok := usage.(map[string]interface{}); ok {
						if inputTokens, exists := usageMap["input_tokens"]; exists {
							if tokens, ok := inputTokens.(float64); ok {
								usageInfo.InputTokens = int(tokens)
							}
						}
						if outputTokens, exists := usageMap["output_tokens"]; exists {
							if tokens, ok := outputTokens.(float64); ok {
								usageInfo.OutputTokens = int(tokens)
							}
						}
					}
				}

				// 检查thinking相关事件
				if currentEvent == "content_block_start" {
					if contentBlock, exists := streamData["content_block"]; exists {
						if blockMap, ok := contentBlock.(map[string]interface{}); ok {
							if blockType, exists := blockMap["type"]; exists && blockType == "thinking" {
								isThinkingBlock = true
								if index, exists := streamData["index"]; exists {
									if idx, ok := index.(float64); ok {
										currentBlockIndex = int(idx)
									}
								}
								h.logger.WithFields(map[string]interface{}{
									"block_index":       currentBlockIndex,
									"block_type":        "thinking",
									"is_thinking_model": isThinkingModel,
								}).Debug("Detected thinking block start")
							}
						}
					}
				} else if currentEvent == "content_block_stop" {
					if index, exists := streamData["index"]; exists {
						if idx, ok := index.(float64); ok && int(idx) == currentBlockIndex && isThinkingBlock {
							isThinkingBlock = false
							h.logger.WithFields(map[string]interface{}{
								"block_index": currentBlockIndex,
							}).Debug("Thinking block ended")
						}
					}
				} else if currentEvent == "content_block_delta" && isThinkingBlock {
					// 为thinking内容添加特殊标记
					if delta, exists := streamData["delta"]; exists {
						if deltaMap, ok := delta.(map[string]interface{}); ok {
							deltaMap["content_type"] = "thinking"
							streamData["delta"] = deltaMap
						}
					}
					// 重新序列化修改后的数据
					if modifiedData, err := json.Marshal(streamData); err == nil {
						data = string(modifiedData)
					}
				}
			}

			// 转发数据
			w.Write([]byte(fmt.Sprintf("data: %s\n", data)))
			flusher.Flush()
		} else if line == "" {
			// 处理空行（SSE 事件分隔符）
			w.Write([]byte("\n"))
			flusher.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	return usageInfo, nil
}

// convertAnthropicToAIRequest 将 Anthropic 请求转换为通用 AI 请求
func (h *AIHandler) convertAnthropicToAIRequest(request *clients.AnthropicMessageRequest, modelInfo *entities.ModelSupportInfo) *clients.AIRequest {
	// 转换消息格式
	var aiMessages []clients.AIMessage
	for _, anthropicMsg := range request.Messages {
		aiMessages = append(aiMessages, clients.AIMessage{
			Role:    anthropicMsg.Role,
			Content: anthropicMsg.GetTextContent(),
		})
	}

	// 转换工具格式
	var tools []clients.Tool
	for _, anthropicTool := range request.Tools {
		tools = append(tools, clients.Tool{
			Type: "function",
			Function: clients.Function{
				Name:        anthropicTool.Name,
				Description: anthropicTool.Description,
				Parameters:  anthropicTool.InputSchema,
			},
		})
	}

	// 设置温度默认值
	temperature := 1.0
	if request.Temperature != nil {
		temperature = *request.Temperature
	}

	// 使用上游模型名称
	model := request.Model
	if modelInfo.UpstreamModelName != "" {
		model = modelInfo.UpstreamModelName
	}

	return &clients.AIRequest{
		Model:       model,
		Messages:    aiMessages,
		MaxTokens:   request.MaxTokens,
		Temperature: temperature,
		Stream:      request.Stream,
		Tools:       tools,
		ToolChoice:  request.ToolChoice,
		Extra: map[string]interface{}{
			"system":         request.System,
			"stop_sequences": request.StopSequences,
			"top_k":          request.TopK,
			"top_p":          request.TopP,
			"service_tier":   request.ServiceTier,
			"metadata":       request.Metadata,
			"container":      request.Container,
			"mcp_servers":    request.MCPServers,
			"thinking":       request.Thinking,
		},
	}
}

// presetProviderInfo 预先设置 provider 信息到 context 中
// 用于确保 billing 中间件能够获取到正确的 provider_id
func (h *AIHandler) presetProviderInfo(ctx context.Context, c *gin.Context, modelSlug string) error {
	// 如果 providerModelSupportRepo 为 nil（如在测试中），跳过预设置
	if h.providerModelSupportRepo == nil {
		return nil
	}

	// 获取支持该模型的提供商
	supportInfos, err := h.providerModelSupportRepo.GetSupportingProviders(ctx, modelSlug)
	if err != nil {
		return fmt.Errorf("failed to get supporting providers: %w", err)
	}

	if len(supportInfos) == 0 {
		return fmt.Errorf("no available providers for model: %s", modelSlug)
	}

	// 使用第一个可用的 provider（与路由逻辑保持一致）
	supportInfo := supportInfos[0]

	h.logger.WithFields(map[string]interface{}{
		"model_slug":    modelSlug,
		"provider_id":   supportInfo.Provider.ID,
		"provider_name": supportInfo.Provider.Name,
		"preset":        true,
	}).Debug("Presetting provider info for streaming request")

	// 设置到 context 中供 billing 中间件使用
	c.Set("provider_id", supportInfo.Provider.ID)
	c.Set("provider_name", supportInfo.Provider.Name)

	return nil
}

// extractProviderInfoFromRequestID 从请求ID提取provider信息
// 这是一个临时解决方案，通过内存缓存的方式来传递流式请求的provider信息
func (h *AIHandler) extractProviderInfoFromRequestID(requestID string) (int64, string) {
	// 临时解决方案：使用一个简单的内存映射
	// 在更好的架构改进之前，我们先硬编码一个默认的provider
	// TODO: 实现更好的provider信息传递机制
	return 1, "openai" // 默认使用第一个provider
}

// convertAIResponseToAnthropic 将通用 AI 响应转换为 Anthropic 格式
func (h *AIHandler) convertAIResponseToAnthropic(response *clients.AIResponse) *clients.AnthropicMessageResponse {
	anthropicResponse := &clients.AnthropicMessageResponse{
		ID:    response.ID,
		Type:  "message",
		Role:  "assistant",
		Model: response.Model,
		Usage: clients.AnthropicUsage{
			InputTokens:  response.Usage.PromptTokens,
			OutputTokens: response.Usage.CompletionTokens,
		},
	}

	// 转换内容
	var content []clients.AnthropicContentBlock
	for _, choice := range response.Choices {
		if choice.Message.Content != "" {
			content = append(content, clients.AnthropicContentBlock{
				Type: "text",
				Text: choice.Message.Content,
			})
		}

		// 处理工具调用
		for _, toolCall := range choice.Message.ToolCalls {
			content = append(content, clients.AnthropicContentBlock{
				Type:  "tool_use",
				ID:    toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: toolCall.Function.Arguments,
			})
		}

		// 设置停止原因
		switch choice.FinishReason {
		case "stop":
			anthropicResponse.StopReason = "end_turn"
		case "length":
			anthropicResponse.StopReason = "max_tokens"
		case "tool_calls":
			anthropicResponse.StopReason = "tool_use"
		default:
			anthropicResponse.StopReason = "end_turn"
		}
	}

	anthropicResponse.Content = content
	return anthropicResponse
}

// convertStreamChunkToAnthropic 将流式数据块转换为Anthropic格式
func (h *AIHandler) convertStreamChunkToAnthropic(chunk *clients.StreamChunk) map[string]interface{} {
	// Anthropic 流式响应格式
	anthropicChunk := map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]interface{}{
			"type": "text_delta",
			"text": chunk.Content,
		},
	}

	// 如果是最后一个块，添加完成信息
	if chunk.FinishReason != nil {
		anthropicChunk["type"] = "message_delta"
		anthropicChunk["delta"] = map[string]interface{}{
			"stop_reason": h.convertFinishReasonToAnthropic(*chunk.FinishReason),
		}

		// 添加使用情况信息
		if chunk.Usage != nil {
			anthropicChunk["usage"] = map[string]interface{}{
				"input_tokens":  chunk.Usage.PromptTokens,
				"output_tokens": chunk.Usage.CompletionTokens,
			}
		}
	}

	return anthropicChunk
}

// 删除了 Thinking 相关的流式处理方法

// 删除了 processThinkingContentBlocks 方法
