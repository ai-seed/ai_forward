package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/functioncall"
	"ai-api-gateway/internal/infrastructure/gateway"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// AIHandler AI请求处理器
type AIHandler struct {
	gatewayService      gateway.GatewayService
	modelService        services.ModelService
	logger              logger.Logger
	config              *config.Config
	functionCallHandler functioncall.FunctionCallHandler
}

// NewAIHandler 创建AI请求处理器
func NewAIHandler(
	gatewayService gateway.GatewayService,
	modelService services.ModelService,
	logger logger.Logger,
	config *config.Config,
	functionCallHandler functioncall.FunctionCallHandler,
) *AIHandler {
	return &AIHandler{
		gatewayService:      gatewayService,
		modelService:        modelService,
		logger:              logger,
		config:              config,
		functionCallHandler: functionCallHandler,
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

	// 检查是否启用了Function Call
	if h.config.FunctionCall.Enabled && len(gatewayRequest.Request.Tools) > 0 {
		// 对于有工具的流式请求，需要特殊处理
		h.handleStreamingRequestWithFunctionCall(c, gatewayRequest, requestID, userID, apiKeyID)
		return
	}

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

		err := h.gatewayService.ProcessStreamRequest(c.Request.Context(), gatewayRequest, streamChan)
		if err != nil {
			select {
			case errorChan <- err:
			case <-c.Request.Context().Done():
				// 如果上下文已取消，不发送错误
			}
		}
	}()

	// 发送流式数据
	var totalTokens int
	var totalCost float64

	for {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				// 流结束，发送结束标记
				_, err := w.Write([]byte("data: [DONE]\n\n"))
				if err != nil {
					h.logger.WithFields(map[string]interface{}{
						"request_id": requestID,
						"error":      err.Error(),
					}).Error("Failed to write stream end marker")
				}
				w.Flush()

				// 输出流式AI提供商响应结果
				h.logger.WithFields(map[string]interface{}{
					"request_id":   requestID,
					"user_id":      userID,
					"api_key_id":   apiKeyID,
					"total_tokens": totalTokens,
					"total_cost":   totalCost,
					"stream_type":  "completed",
				}).Info("AI provider streaming response completed successfully")

				// 设置使用量到上下文
				c.Set("tokens_used", totalTokens)
				c.Set("cost_used", totalCost)
				return
			}

			// 累计使用量
			if chunk.Usage != nil {
				totalTokens += chunk.Usage.TotalTokens
			}
			if chunk.Cost != nil {
				totalCost += chunk.Cost.TotalCost
			}

			// 构造SSE数据
			data := map[string]interface{}{
				"id":      chunk.ID,
				"object":  "chat.completion.chunk",
				"created": chunk.Created,
				"model":   chunk.Model,
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": chunk.Content,
						},
						"finish_reason": chunk.FinishReason,
					},
				},
			}

			// 序列化为JSON
			jsonData, err := json.Marshal(data)
			if err != nil {
				h.logger.WithFields(map[string]interface{}{
					"request_id": requestID,
					"error":      err.Error(),
				}).Error("Failed to marshal stream chunk")
				continue
			}

			// 发送SSE数据
			sseMessage := fmt.Sprintf("data: %s\n\n", jsonData)
			_, err = w.Write([]byte(sseMessage))
			if err != nil {
				h.logger.WithFields(map[string]interface{}{
					"request_id": requestID,
					"error":      err.Error(),
				}).Error("Failed to write stream chunk")
				return
			}

			// 立即刷新缓冲区
			w.Flush()

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
// @Security ApiKeyAuth
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
	}

	// 如果开启了联网搜索且没有提供工具，自动添加可用工具
	if h.functionCallHandler != nil && len(aiRequest.Tools) == 0 && aiRequest.WebSearch {
		aiRequest.Tools = h.functionCallHandler.GetAvailableTools()
		aiRequest.ToolChoice = "auto"
	}

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

	// 设置使用量到上下文（用于配额中间件）
	c.Set("tokens_used", response.Usage.TotalTokens)
	c.Set("cost_used", response.Cost.TotalCost)

	// 设置响应头
	c.Header("X-Request-ID", requestID)
	c.Header("X-Provider", response.Provider)
	c.Header("X-Model", response.Model)
	c.Header("X-Duration-Ms", strconv.FormatInt(response.Duration.Milliseconds(), 10))

	// 检查是否需要处理 Function Call
	if h.functionCallHandler != nil && response.Response != nil {
		finalResponse, err := h.handleFunctionCallResponse(c.Request.Context(), response.Response, aiRequest, gatewayRequest)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("Failed to handle function call")

			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
				"FUNCTION_CALL_FAILED",
				"Failed to process function call",
				map[string]interface{}{
					"request_id": requestID,
				},
			))
			return
		}

		if finalResponse != nil {
			c.JSON(http.StatusOK, finalResponse)
			return
		}
	}

	// 返回AI响应（保持与OpenAI API兼容的格式）
	c.JSON(http.StatusOK, response.Response)
}

// Completions 处理文本完成请求
// @Summary 文本补全
// @Description 创建文本补全请求，兼容OpenAI API格式
// @Tags AI接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
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

	// 如果开启了联网搜索，自动添加可用工具
	if h.functionCallHandler != nil && aiRequest.WebSearch {
		// 将 prompt 转换为 messages 格式以支持 function call
		aiRequest.Messages = []clients.AIMessage{
			{
				Role:    "user",
				Content: aiRequest.Prompt,
			},
		}
		aiRequest.Prompt = "" // 清空 prompt，使用 messages
		aiRequest.Tools = h.functionCallHandler.GetAvailableTools()
		aiRequest.ToolChoice = "auto"
	}

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

	// 设置使用量到上下文（用于配额中间件）
	c.Set("tokens_used", response.Usage.TotalTokens)
	c.Set("cost_used", response.Cost.TotalCost)

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
// @Description 获取可用的AI模型列表
// @Tags AI接口
// @Produce json
// @Security ApiKeyAuth
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

	// 转换为 OpenAI API 格式
	var modelList []map[string]interface{}
	for _, model := range models {
		displayName := model.Name
		if model.DisplayName != nil {
			displayName = *model.DisplayName
		}

		modelData := map[string]interface{}{
			"id":       model.Slug,
			"object":   "model",
			"created":  model.CreatedAt.Unix(),
			"owned_by": "system",
		}

		// 添加可选字段
		if model.Description != nil {
			modelData["description"] = *model.Description
		}

		// 添加扩展信息
		modelData["display_name"] = displayName
		modelData["model_type"] = string(model.ModelType)
		modelData["status"] = string(model.Status)

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

	// 返回 OpenAI API 兼容格式
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   modelList,
	})
}

// Usage 获取使用情况
// @Summary 使用统计
// @Description 获取当前用户的API使用统计
// @Tags AI接口
// @Produce json
// @Security ApiKeyAuth
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

	// TODO: 实现获取使用情况
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"usage":   gin.H{},
	})
}

// handleFunctionCallResponse 处理包含 Function Call 的响应
func (h *AIHandler) handleFunctionCallResponse(ctx context.Context, response *clients.AIResponse, originalRequest *clients.AIRequest, gatewayRequest *gateway.GatewayRequest) (*clients.AIResponse, error) {
	// 检查响应中是否包含工具调用
	if len(response.Choices) == 0 {
		return nil, nil
	}

	choice := response.Choices[0]

	// 检查是否有工具调用
	var toolCalls []clients.ToolCall
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls = choice.Message.ToolCalls
	} else if len(choice.ToolCalls) > 0 {
		toolCalls = choice.ToolCalls
	}

	if len(toolCalls) == 0 {
		return nil, nil // 没有工具调用，返回原响应
	}

	h.logger.WithFields(map[string]interface{}{
		"tool_calls_count": len(toolCalls),
		"request_id":       gatewayRequest.RequestID,
	}).Info("Processing function calls")

	// 将助手的消息（包含工具调用）添加到消息历史
	messages := append(originalRequest.Messages, choice.Message)

	// 执行工具调用
	toolMessages, err := h.functionCallHandler.HandleFunctionCalls(ctx, messages, toolCalls)
	if err != nil {
		return nil, fmt.Errorf("failed to handle function calls: %w", err)
	}

	// 将工具响应消息添加到消息历史
	messages = append(messages, toolMessages...)

	// 创建新的请求，包含完整的消息历史
	newRequest := &clients.AIRequest{
		Model:       originalRequest.Model,
		Messages:    messages,
		MaxTokens:   originalRequest.MaxTokens,
		Temperature: originalRequest.Temperature,
		Stream:      false, // Function call 后的请求不使用流式
		// 不再包含 tools，让模型生成最终回复
	}

	// 创建新的网关请求
	newGatewayRequest := &gateway.GatewayRequest{
		UserID:    gatewayRequest.UserID,
		APIKeyID:  gatewayRequest.APIKeyID,
		ModelSlug: gatewayRequest.ModelSlug,
		Request:   newRequest,
		RequestID: gatewayRequest.RequestID,
	}

	// 发送第二次请求获取最终回复
	finalResponse, err := h.gatewayService.ProcessRequest(ctx, newGatewayRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to process final request after function calls: %w", err)
	}

	h.logger.WithFields(map[string]interface{}{
		"request_id":   gatewayRequest.RequestID,
		"final_tokens": finalResponse.Usage.TotalTokens,
		"final_cost":   finalResponse.Cost.TotalCost,
	}).Info("Function call processing completed")

	return finalResponse.Response, nil
}

// handleStreamingRequestWithFunctionCall 处理带有Function Call的流式请求
func (h *AIHandler) handleStreamingRequestWithFunctionCall(c *gin.Context, gatewayRequest *gateway.GatewayRequest, requestID string, userID, apiKeyID int64) {
	w := c.Writer

	h.logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"tools_count": len(gatewayRequest.Request.Tools),
	}).Info("Processing streaming request with function call support")

	// 首先发送非流式请求来检查是否有tool calls
	nonStreamRequest := *gatewayRequest.Request
	nonStreamRequest.Stream = false

	nonStreamGatewayRequest := &gateway.GatewayRequest{
		UserID:    gatewayRequest.UserID,
		APIKeyID:  gatewayRequest.APIKeyID,
		ModelSlug: gatewayRequest.ModelSlug,
		Request:   &nonStreamRequest,
		RequestID: gatewayRequest.RequestID,
	}

	// 处理第一次请求
	response, err := h.gatewayService.ProcessRequest(c.Request.Context(), nonStreamGatewayRequest)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to process initial request for function call detection")

		// 发送错误事件
		errorData := map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Failed to process request",
				"type":    "internal_error",
			},
		}
		errorJSON, _ := json.Marshal(errorData)
		w.Write([]byte(fmt.Sprintf("data: %s\n\n", errorJSON)))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	// 检查是否需要处理 Function Call
	if h.functionCallHandler != nil && response.Response != nil {
		finalResponse, err := h.handleFunctionCallResponse(c.Request.Context(), response.Response, &nonStreamRequest, nonStreamGatewayRequest)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("Failed to handle function call in streaming request")

			// 发送错误事件
			errorData := map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Failed to process function call",
					"type":    "function_call_error",
				},
			}
			errorJSON, _ := json.Marshal(errorData)
			w.Write([]byte(fmt.Sprintf("data: %s\n\n", errorJSON)))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			return
		}

		// 如果有function call处理结果，使用最终响应
		if finalResponse != nil {
			response.Response = finalResponse
		}
	}

	// 现在以流式方式发送最终响应
	if response.Response != nil && len(response.Response.Choices) > 0 {
		choice := response.Response.Choices[0]
		content := choice.Message.Content

		// 将内容分块发送，模拟流式输出
		h.streamContent(w, content, response.Response.ID, response.Response.Model, requestID)
	}

	// 发送结束标记
	w.Write([]byte("data: [DONE]\n\n"))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// 设置使用量到上下文
	if response.Usage != nil {
		c.Set("tokens_used", response.Usage.TotalTokens)
	}
	if response.Cost != nil {
		c.Set("cost_used", response.Cost.TotalCost)
	}
}

// streamContent 将内容以流式方式发送
func (h *AIHandler) streamContent(w http.ResponseWriter, content, responseID, model, requestID string) {
	// 获取Flusher接口
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.WithFields(map[string]interface{}{
			"request_id": requestID,
		}).Error("ResponseWriter does not support flushing")
		return
	}

	// 将内容按字符分块发送
	for i, char := range content {
		chunk := map[string]interface{}{
			"id":      responseID,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{
						"content": string(char),
					},
					"finish_reason": func() interface{} {
						if i == len(content)-1 {
							return "stop"
						}
						return nil
					}(),
				},
			},
		}

		chunkJSON, err := json.Marshal(chunk)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("Failed to marshal stream chunk")
			continue
		}

		w.Write([]byte(fmt.Sprintf("data: %s\n\n", chunkJSON)))
		flusher.Flush()

		// 添加小延迟以模拟真实的流式输出
		time.Sleep(10 * time.Millisecond)
	}
}
