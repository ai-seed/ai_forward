package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// GenericProxyHandler 通用代理处理器
type GenericProxyHandler struct {
	providerRepo             repositories.ProviderRepository
	providerModelSupportRepo repositories.ProviderModelSupportRepository
	logger                   logger.Logger
}

// NewGenericProxyHandler 创建通用代理处理器
func NewGenericProxyHandler(
	providerRepo repositories.ProviderRepository,
	providerModelSupportRepo repositories.ProviderModelSupportRepository,
	logger logger.Logger,
) *GenericProxyHandler {
	return &GenericProxyHandler{
		providerRepo:             providerRepo,
		providerModelSupportRepo: providerModelSupportRepo,
		logger:                   logger,
	}
}

// ProxyRequest 通用的请求转发处理器
func (h *GenericProxyHandler) ProxyRequest(c *gin.Context) {
	// 获取请求方法和路径
	method := c.Request.Method
	path := c.Request.URL.Path

	// 获取查询参数
	query := c.Request.URL.Query()

	// 获取请求头
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// 读取请求体
	var body []byte
	var err error
	if c.Request.Body != nil {
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
				"path":  path,
			}).Error("Failed to read request body")

			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Failed to read request body",
				},
			})
			return
		}
	}

	// 检查是否为流式请求
	isStreamRequest := h.isStreamRequest(headers, body)

	h.logger.WithFields(map[string]interface{}{
		"method":    method,
		"path":      path,
		"body_size": len(body),
		"query":     query,
		"is_stream": isStreamRequest,
	}).Info("Processing generic proxy request")

	// 从请求体中提取模型信息并获取提供商配置
	provider, authType, selectedModelInfo, err := h.getProviderByRequest(c.Request.Context(), path, body)
	if err != nil {
		// 根据错误类型返回不同的状态码和错误信息
		statusCode, errorCode, errorMessage := h.categorizeProviderError(err)

		h.logger.WithFields(map[string]interface{}{
			"error":       err.Error(),
			"path":        path,
			"method":      method,
			"body_size":   len(body),
			"status_code": statusCode,
			"error_code":  errorCode,
		}).Error("Failed to get provider for request")

		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"code":    errorCode,
				"message": errorMessage,
				"details": err.Error(),
			},
		})
		return
	}

	// 设置提供商信息到上下文供计费中间件使用
	h.setBillingContext(c, provider, selectedModelInfo)

	// 保存原始请求体用于token估算
	c.Set("original_request_body", body)

	// 转换请求体（如果需要）
	transformedBody, transformedPath, err := h.transformRequest(provider, selectedModelInfo, body, path)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":         err.Error(),
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
			"model_slug":    selectedModelInfo.ModelSlug,
		}).Error("Failed to transform request")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "REQUEST_TRANSFORM_ERROR",
				"message": "Failed to transform request for upstream provider",
				"details": err.Error(),
			},
		})
		return
	}

	// 创建代理客户端
	proxyClient := clients.NewGenericProxyClient(
		provider.BaseURL,
		*provider.APIKeyEncrypted,
		authType,
		h.logger,
	)

	// 根据是否为流式请求选择处理方式
	if isStreamRequest {
		h.handleStreamRequest(c, proxyClient, method, transformedPath, headers, transformedBody, query)
	} else {
		h.handleNormalRequest(c, proxyClient, method, transformedPath, headers, transformedBody, query)
	}
}

// handleNormalRequest 处理普通请求
func (h *GenericProxyHandler) handleNormalRequest(c *gin.Context, proxyClient clients.GenericProxyClient, method, path string, headers map[string]string, body []byte, query url.Values) {
	// 转发请求
	response, err := proxyClient.ForwardRequest(
		c.Request.Context(),
		method,
		path,
		headers,
		body,
		query,
	)

	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"path":  path,
		}).Error("Failed to forward request")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "FORWARD_ERROR",
				"message": "Failed to forward request to upstream service",
			},
		})
		return
	}

	// 设置响应头
	for key, value := range response.Headers {
		c.Header(key, value)
	}

	// 尝试从响应中提取使用量信息并设置到上下文
	h.extractAndSetUsageFromResponse(c, response.Body, path)

	// 返回响应
	c.Data(response.StatusCode, c.GetHeader("Content-Type"), response.Body)
}

// handleStreamRequest 处理流式请求
func (h *GenericProxyHandler) handleStreamRequest(c *gin.Context, proxyClient clients.GenericProxyClient, method, path string, headers map[string]string, body []byte, query url.Values) {
	// 转发流式请求
	streamResponse, err := proxyClient.ForwardStreamRequest(
		c.Request.Context(),
		method,
		path,
		headers,
		body,
		query,
	)

	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"path":  path,
		}).Error("Failed to forward stream request")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "STREAM_FORWARD_ERROR",
				"message": "Failed to forward stream request to upstream service",
			},
		})
		return
	}
	defer streamResponse.Reader.Close()

	// 设置流式响应头
	for key, value := range streamResponse.Headers {
		c.Header(key, value)
	}

	// 确保设置正确的流式响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 设置状态码
	c.Status(streamResponse.StatusCode)

	// 获取响应写入器
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Error("Response writer does not support flushing")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "STREAM_ERROR",
				"message": "Stream response not supported",
			},
		})
		return
	}

	// 逐行读取并转发流式响应
	scanner := bufio.NewScanner(streamResponse.Reader)
	lineCount := 0
	var totalInputTokens, totalOutputTokens int
	var lastInputTokens, lastOutputTokens int // 记录最后一次的token数，用于处理累积值
	var hasUsageInfo bool
	var outputContent strings.Builder // 收集输出内容用于token估算

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		h.logger.WithFields(map[string]interface{}{
			"path":        path,
			"line_count":  lineCount,
			"line":        line,
			"line_length": len(line),
		}).Info("Processing stream line") // 改为Info级别便于观察原始数据

		// 处理 SSE 格式的数据
		if h.isValidSSELine(line) {
			// 转发有效的 SSE 行
			fmt.Fprintf(w, "%s\n", line)
			flusher.Flush()

			// 尝试从流式数据中提取token使用量和内容
			if strings.HasPrefix(line, "data: ") {
				inputTokens, outputTokens := h.extractTokensFromStreamLine(line)
				if inputTokens > 0 || outputTokens > 0 {
					hasUsageInfo = true

					// 对于某些提供商，usage信息是累积的，我们需要取最大值
					if inputTokens > lastInputTokens {
						lastInputTokens = inputTokens
					}
					if outputTokens > lastOutputTokens {
						lastOutputTokens = outputTokens
					}

					// 同时也累积增量（适用于增量式的提供商）
					totalInputTokens += inputTokens
					totalOutputTokens += outputTokens

					h.logger.WithFields(map[string]interface{}{
						"path":           path,
						"line_count":     lineCount,
						"current_input":  inputTokens,
						"current_output": outputTokens,
						"total_input":    totalInputTokens,
						"total_output":   totalOutputTokens,
						"last_input":     lastInputTokens,
						"last_output":    lastOutputTokens,
					}).Info("Updated token usage from stream") // 改为Info级别便于观察

					// 实时更新上下文，确保计费中间件能获取到最新的token信息
					h.updateStreamTokenContext(c, lastInputTokens, lastOutputTokens, path)
				}

				// 提取输出内容用于token估算（当没有usage信息时）
				content := h.extractContentFromStreamLine(line)
				if content != "" {
					outputContent.WriteString(content)
				}
			}

			// 检查是否为结束标记
			if strings.Contains(line, "data: [DONE]") {
				h.logger.WithFields(map[string]interface{}{
					"path":       path,
					"line_count": lineCount,
				}).Info("Stream completed with [DONE] marker")
				break
			}
		} else {
			// 记录无效行但不转发
			h.logger.WithFields(map[string]interface{}{
				"path":         path,
				"line_count":   lineCount,
				"invalid_line": line,
			}).Debug("Skipping invalid SSE line")
		}
	}

	// 决定使用哪种token计数方式
	var finalInputTokens, finalOutputTokens int
	if hasUsageInfo {
		// 如果有usage信息，优先使用最后一次的值（通常是累积值）
		// 如果最后一次的值为0，则使用累积值
		if lastInputTokens > 0 {
			finalInputTokens = lastInputTokens
		} else {
			finalInputTokens = totalInputTokens
		}

		if lastOutputTokens > 0 {
			finalOutputTokens = lastOutputTokens
		} else {
			finalOutputTokens = totalOutputTokens
		}
	} else {
		// 没有usage信息时，使用备用估算方法
		finalInputTokens, finalOutputTokens = h.estimateTokensFromContent(c, outputContent.String(), path)
	}

	// 设置最终的token使用量到上下文
	h.setStreamTokenUsage(c, finalInputTokens, finalOutputTokens, path)

	// 调试：检查上下文中的所有相关值
	h.debugContextValues(c, path)

	if err := scanner.Err(); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"path":       path,
			"line_count": lineCount,
		}).Error("Error reading stream response")

		// 发送错误事件
		fmt.Fprintf(w, "event: error\n")
		fmt.Fprintf(w, "data: {\"error\": \"Stream reading error\", \"details\": \"%s\"}\n\n", err.Error())
		flusher.Flush()
	}

	h.logger.WithFields(map[string]interface{}{
		"path":       path,
		"line_count": lineCount,
	}).Info("Stream response completed")
}

// isValidSSELine 检查是否为有效的 SSE 行
func (h *GenericProxyHandler) isValidSSELine(line string) bool {
	// 空行是有效的（用于分隔事件）
	if line == "" {
		return true
	}

	// 检查是否为有效的 SSE 字段
	validPrefixes := []string{
		"data:",
		"event:",
		"id:",
		"retry:",
		":", // 注释行
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}

// isStreamRequest 检查是否为流式请求
func (h *GenericProxyHandler) isStreamRequest(headers map[string]string, body []byte) bool {
	// 检查 Accept 头
	if accept, exists := headers["Accept"]; exists {
		if strings.Contains(accept, "text/event-stream") {
			return true
		}
	}

	// 检查请求体中的 stream 参数
	if len(body) > 0 {
		var requestData map[string]interface{}
		if err := json.Unmarshal(body, &requestData); err == nil {
			if stream, exists := requestData["stream"]; exists {
				if streamBool, ok := stream.(bool); ok && streamBool {
					return true
				}
			}
		}
	}

	return false
}

// getProviderByRequest 根据请求内容获取对应的提供商配置
func (h *GenericProxyHandler) getProviderByRequest(ctx context.Context, path string, body []byte) (*entities.Provider, string, *entities.ModelSupportInfo, error) {
	// 从请求体中提取模型信息
	modelSlug, err := h.extractModelFromRequest(body)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to extract model from request: %w", err)
	}

	h.logger.WithFields(map[string]interface{}{
		"model_slug": modelSlug,
	}).Info("Extracted model from request")

	// 从数据库获取支持该模型的提供商
	supportInfos, err := h.providerModelSupportRepo.GetSupportingProviders(ctx, modelSlug)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to get supporting providers for model %s: %w", modelSlug, err)
	}

	if len(supportInfos) == 0 {
		return nil, "", nil, fmt.Errorf("no providers support model: %s", modelSlug)
	}

	// 选择最佳可用的提供商（基于优先级和健康状态）
	selectedModelInfo, err := h.selectBestProvider(supportInfos)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to select provider: %w", err)
	}

	// 获取提供商信息（已经在 ModelSupportInfo 中包含）
	provider := selectedModelInfo.Provider
	if provider == nil {
		return nil, "", nil, fmt.Errorf("provider information not found in model support info")
	}

	// 根据提供商类型确定认证方式
	authType := h.getAuthTypeByProvider(provider.Slug)

	h.logger.WithFields(map[string]interface{}{
		"provider_id":   provider.ID,
		"provider_name": provider.Name,
		"provider_slug": provider.Slug,
		"auth_type":     authType,
		"model_slug":    modelSlug,
	}).Info("Selected provider for request")

	return provider, authType, selectedModelInfo, nil
}

// extractModelFromRequest 从请求体中提取模型信息
func (h *GenericProxyHandler) extractModelFromRequest(body []byte) (string, error) {
	if len(body) == 0 {
		return "", fmt.Errorf("empty request body")
	}

	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		return "", fmt.Errorf("failed to parse request body: %w", err)
	}

	// 1. 优先查找 "model" 字段（OpenAI、Anthropic 格式）
	if model, exists := requestData["model"]; exists {
		if modelStr, ok := model.(string); ok && modelStr != "" {
			return modelStr, nil
		}
	}

	// 2. 查找 "engine" 字段（某些 OpenAI 兼容格式）
	if engine, exists := requestData["engine"]; exists {
		if engineStr, ok := engine.(string); ok && engineStr != "" {
			return engineStr, nil
		}
	}

	// 3. 查找 "model_id" 字段（某些自定义格式）
	if modelID, exists := requestData["model_id"]; exists {
		if modelIDStr, ok := modelID.(string); ok && modelIDStr != "" {
			return modelIDStr, nil
		}
	}

	// 4. 对于 Stability AI，可能需要从路径推断模型
	// 这种情况下，我们可以使用默认模型或从其他字段推断
	if mode, exists := requestData["mode"]; exists {
		if modeStr, ok := mode.(string); ok && modeStr != "" {
			// 根据 mode 映射到具体模型
			return h.mapModeToModel(modeStr), nil
		}
	}

	// 5. 检查是否有 style_preset 或其他 Stability AI 特有字段
	if stylePreset, exists := requestData["style_preset"]; exists {
		if _, ok := stylePreset.(string); ok {
			// 这可能是 Stability AI 请求，使用默认模型
			return "stable-diffusion-xl", nil
		}
	}

	// 6. 检查是否有 prompt 字段但没有 messages（可能是 completion 请求）
	if prompt, exists := requestData["prompt"]; exists {
		if _, ok := prompt.(string); ok {
			// 这可能是 completion 请求，需要有模型字段
			return "", fmt.Errorf("completion request found but model field is missing")
		}
	}

	// 7. 检查是否有 messages 字段但没有 model（可能是格式错误）
	if messages, exists := requestData["messages"]; exists {
		if _, ok := messages.([]interface{}); ok {
			return "", fmt.Errorf("chat request found but model field is missing")
		}
	}

	return "", fmt.Errorf("model field not found in request body")
}

// mapModeToModel 将 mode 映射到具体的模型名称
func (h *GenericProxyHandler) mapModeToModel(mode string) string {
	switch mode {
	case "text-to-image", "txt2img":
		return "stable-diffusion-xl"
	case "image-to-image", "img2img":
		return "stable-diffusion-xl-img2img"
	case "upscale":
		return "stable-diffusion-upscale"
	case "inpaint":
		return "stable-diffusion-inpaint"
	case "outpaint":
		return "stable-diffusion-outpaint"
	default:
		return "stable-diffusion-xl" // 默认模型
	}
}

// selectBestProvider 选择最佳可用的提供商
func (h *GenericProxyHandler) selectBestProvider(supportInfos []*entities.ModelSupportInfo) (*entities.ModelSupportInfo, error) {
	if len(supportInfos) == 0 {
		return nil, fmt.Errorf("no provider support info available")
	}

	// 过滤出可用的提供商
	var availableProviders []*entities.ModelSupportInfo
	for _, info := range supportInfos {
		if h.isProviderAvailable(info) {
			availableProviders = append(availableProviders, info)
		}
	}

	if len(availableProviders) == 0 {
		return nil, fmt.Errorf("no available providers found")
	}

	// 如果只有一个可用提供商，直接返回
	if len(availableProviders) == 1 {
		return availableProviders[0], nil
	}

	// 多个提供商时，选择优先级最高的
	// supportInfos 已经按照 priority ASC, provider.priority ASC 排序
	// 所以第一个就是优先级最高的
	selectedProvider := availableProviders[0]

	h.logger.WithFields(map[string]interface{}{
		"selected_provider_id":   selectedProvider.Provider.ID,
		"selected_provider_name": selectedProvider.Provider.Name,
		"selected_priority":      selectedProvider.Priority,
		"total_available":        len(availableProviders),
		"total_candidates":       len(supportInfos),
	}).Info("Selected provider for request")

	return selectedProvider, nil
}

// isProviderAvailable 检查提供商是否可用
func (h *GenericProxyHandler) isProviderAvailable(info *entities.ModelSupportInfo) bool {
	if info == nil || info.Provider == nil {
		return false
	}

	provider := info.Provider

	// 检查提供商状态
	if provider.Status != entities.ProviderStatusActive {
		h.logger.WithFields(map[string]interface{}{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
			"status":        provider.Status,
		}).Debug("Provider not active")
		return false
	}

	// 检查模型支持是否启用
	if !info.Enabled {
		h.logger.WithFields(map[string]interface{}{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
			"model_slug":    info.ModelSlug,
			"enabled":       info.Enabled,
		}).Debug("Model support not enabled for provider")
		return false
	}

	// 检查健康状态
	if provider.HealthStatus == entities.HealthStatusUnhealthy {
		h.logger.WithFields(map[string]interface{}{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
			"health_status": provider.HealthStatus,
		}).Debug("Provider health status is unhealthy")
		return false
	}

	// 检查必要的配置
	if provider.APIKeyEncrypted == nil || *provider.APIKeyEncrypted == "" {
		h.logger.WithFields(map[string]interface{}{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
		}).Debug("Provider missing API key")
		return false
	}

	if provider.BaseURL == "" {
		h.logger.WithFields(map[string]interface{}{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
		}).Debug("Provider missing base URL")
		return false
	}

	return true
}

// categorizeProviderError 根据错误类型分类并返回适当的HTTP状态码和错误信息
func (h *GenericProxyHandler) categorizeProviderError(err error) (int, string, string) {
	errMsg := err.Error()

	// 模型相关错误
	if strings.Contains(errMsg, "model field not found") ||
		strings.Contains(errMsg, "model field is empty") ||
		strings.Contains(errMsg, "failed to extract model") {
		return http.StatusBadRequest, "INVALID_MODEL", "Invalid or missing model in request"
	}

	// 请求体格式错误
	if strings.Contains(errMsg, "failed to parse request body") ||
		strings.Contains(errMsg, "empty request body") {
		return http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body format"
	}

	// 模型不支持错误
	if strings.Contains(errMsg, "no providers support model") ||
		strings.Contains(errMsg, "failed to get supporting providers") {
		return http.StatusNotFound, "MODEL_NOT_SUPPORTED", "The requested model is not supported"
	}

	// 提供商不可用错误
	if strings.Contains(errMsg, "no available providers") ||
		strings.Contains(errMsg, "provider") && strings.Contains(errMsg, "not active") ||
		strings.Contains(errMsg, "failed to select provider") {
		return http.StatusServiceUnavailable, "PROVIDER_UNAVAILABLE", "No available providers for the requested model"
	}

	// 数据库连接错误
	if strings.Contains(errMsg, "database") || strings.Contains(errMsg, "connection") {
		return http.StatusInternalServerError, "DATABASE_ERROR", "Internal database error"
	}

	// 默认为内部服务器错误
	return http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error"
}

// transformRequest 转换请求体和路径以适配上游提供商
func (h *GenericProxyHandler) transformRequest(provider *entities.Provider, modelInfo *entities.ModelSupportInfo, body []byte, path string) ([]byte, string, error) {
	// 如果没有请求体，直接返回
	if len(body) == 0 {
		return body, path, nil
	}

	// 解析请求体
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		// 如果不是 JSON 格式，直接返回原始数据
		return body, path, nil
	}

	// 转换模型名称
	if modelInfo.UpstreamModelName != "" && modelInfo.UpstreamModelName != modelInfo.ModelSlug {
		requestData["model"] = modelInfo.UpstreamModelName
		h.logger.WithFields(map[string]interface{}{
			"original_model": modelInfo.ModelSlug,
			"upstream_model": modelInfo.UpstreamModelName,
			"provider_name":  provider.Name,
		}).Debug("Transformed model name for upstream")
	}

	// 应用提供商特定的配置
	if modelInfo.Config != nil {
		config, err := modelInfo.Support.GetConfig()
		if err == nil && config != nil {
			// 应用参数映射
			if config.ParameterMapping != nil {
				for originalParam, mappedParam := range config.ParameterMapping {
					if value, exists := requestData[originalParam]; exists {
						delete(requestData, originalParam)
						requestData[mappedParam] = value
					}
				}
			}

			// 应用默认参数
			if config.MaxTokens != nil && requestData["max_tokens"] == nil {
				requestData["max_tokens"] = *config.MaxTokens
			}

			if config.Temperature != nil && requestData["temperature"] == nil {
				requestData["temperature"] = *config.Temperature
			}
		}
	}

	// 转换路径（如果需要）
	transformedPath := h.transformPath(provider, path)

	// 重新序列化请求体
	transformedBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal transformed request: %w", err)
	}

	return transformedBody, transformedPath, nil
}

// transformPath 转换请求路径以适配上游提供商
func (h *GenericProxyHandler) transformPath(provider *entities.Provider, path string) string {
	// 根据提供商类型进行路径转换
	switch strings.ToLower(provider.Slug) {
	case "anthropic":
		// Anthropic 使用 /v1/messages 端点
		if strings.HasPrefix(path, "/v1/chat/completions") {
			return "/v1/messages"
		}
	case "openai":
		// OpenAI 保持原路径
		return path
	case "stability":
		// Stability AI 可能需要特殊的路径映射
		// 这里可以根据具体需求进行映射
		return path
	default:
		// 默认保持原路径
		return path
	}

	return path
}

// setBillingContext 设置计费上下文信息
func (h *GenericProxyHandler) setBillingContext(c *gin.Context, provider *entities.Provider, modelInfo *entities.ModelSupportInfo) {
	// 设置提供商信息供计费中间件使用
	c.Set("provider_id", provider.ID)
	c.Set("provider_name", provider.Name)

	// 设置模型信息
	c.Set("model_name", modelInfo.ModelSlug)

	h.logger.WithFields(map[string]interface{}{
		"provider_id":   provider.ID,
		"provider_name": provider.Name,
		"model_slug":    modelInfo.ModelSlug,
		"context_set":   true,
	}).Debug("Set billing context for generic proxy request")
}

// extractAndSetUsageFromResponse 从响应中提取使用量信息并设置到上下文
func (h *GenericProxyHandler) extractAndSetUsageFromResponse(c *gin.Context, responseBody []byte, path string) {
	if len(responseBody) == 0 {
		return
	}

	// 尝试解析响应体
	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"path":  path,
		}).Debug("Failed to parse response body for usage extraction")
		return
	}

	// 提取 usage 信息（OpenAI 格式）
	if usage, exists := responseData["usage"]; exists {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			h.setTokenUsageFromMap(c, usageMap, path)
		}
	}

	// 提取 cost 信息（如果有）
	if cost, exists := responseData["cost"]; exists {
		if costFloat, ok := cost.(float64); ok {
			c.Set("cost_used", costFloat)
			h.logger.WithFields(map[string]interface{}{
				"cost": costFloat,
				"path": path,
			}).Debug("Set cost information from response")
		}
	}
}

// setTokenUsageFromMap 从usage map中设置token使用量
func (h *GenericProxyHandler) setTokenUsageFromMap(c *gin.Context, usageMap map[string]interface{}, path string) {
	var inputTokens, outputTokens, totalTokens int

	// 提取输入token
	if promptTokens, exists := usageMap["prompt_tokens"]; exists {
		if tokens, ok := promptTokens.(float64); ok {
			inputTokens = int(tokens)
		}
	}

	// 提取输出token
	if completionTokens, exists := usageMap["completion_tokens"]; exists {
		if tokens, ok := completionTokens.(float64); ok {
			outputTokens = int(tokens)
		}
	}

	// 提取总token
	if total, exists := usageMap["total_tokens"]; exists {
		if tokens, ok := total.(float64); ok {
			totalTokens = int(tokens)
		}
	}

	// 如果没有总token，计算一个
	if totalTokens == 0 && (inputTokens > 0 || outputTokens > 0) {
		totalTokens = inputTokens + outputTokens
	}

	// 设置到上下文
	if inputTokens > 0 {
		c.Set("input_tokens", inputTokens)
	}
	if outputTokens > 0 {
		c.Set("output_tokens", outputTokens)
	}
	if totalTokens > 0 {
		c.Set("total_tokens", totalTokens)
		c.Set("tokens_used", totalTokens)
	}

	h.logger.WithFields(map[string]interface{}{
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"total_tokens":  totalTokens,
		"path":          path,
	}).Debug("Set token usage information from response")
}

// extractTokensFromStreamLine 从流式响应行中提取token使用量
func (h *GenericProxyHandler) extractTokensFromStreamLine(line string) (int, int) {
	// 移除 "data: " 前缀
	if !strings.HasPrefix(line, "data: ") {
		return 0, 0
	}

	dataContent := strings.TrimPrefix(line, "data: ")

	// 跳过特殊标记
	if dataContent == "[DONE]" || dataContent == "" {
		return 0, 0
	}

	// 尝试解析JSON
	var streamData map[string]interface{}
	if err := json.Unmarshal([]byte(dataContent), &streamData); err != nil {
		// 记录解析失败的详细信息
		h.logger.WithFields(map[string]interface{}{
			"error":        err.Error(),
			"data_content": dataContent,
			"line_length":  len(dataContent),
		}).Debug("Failed to parse stream line JSON")
		return 0, 0
	}

	var inputTokens, outputTokens int

	// 查找usage信息 - 支持多种格式
	if usage, exists := streamData["usage"]; exists {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			// OpenAI格式
			if promptTokens, exists := usageMap["prompt_tokens"]; exists {
				if tokens, ok := promptTokens.(float64); ok {
					inputTokens = int(tokens)
				}
			}
			if completionTokens, exists := usageMap["completion_tokens"]; exists {
				if tokens, ok := completionTokens.(float64); ok {
					outputTokens = int(tokens)
				}
			}

			// Anthropic格式
			if inputTokensField, exists := usageMap["input_tokens"]; exists {
				if tokens, ok := inputTokensField.(float64); ok {
					inputTokens = int(tokens)
				}
			}
			if outputTokensField, exists := usageMap["output_tokens"]; exists {
				if tokens, ok := outputTokensField.(float64); ok {
					outputTokens = int(tokens)
				}
			}

			// 记录找到的usage信息
			if inputTokens > 0 || outputTokens > 0 {
				h.logger.WithFields(map[string]interface{}{
					"input_tokens":  inputTokens,
					"output_tokens": outputTokens,
					"usage_format":  "standard",
				}).Debug("Extracted token usage from stream line")
			}
		}
	}

	// 检查是否为最终的usage信息（某些提供商在最后一个chunk中提供完整usage）
	if finishReason, exists := streamData["finish_reason"]; exists && finishReason != nil {
		h.logger.WithFields(map[string]interface{}{
			"finish_reason": finishReason,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"is_final":      true,
		}).Debug("Found final chunk with finish_reason")
	}

	return inputTokens, outputTokens
}

// updateStreamTokenContext 实时更新流式请求的token上下文
func (h *GenericProxyHandler) updateStreamTokenContext(c *gin.Context, inputTokens, outputTokens int, path string) {
	totalTokens := inputTokens + outputTokens

	// 实时更新上下文
	c.Set("input_tokens", inputTokens)
	c.Set("output_tokens", outputTokens)
	c.Set("total_tokens", totalTokens)
	c.Set("tokens_used", totalTokens)

	h.logger.WithFields(map[string]interface{}{
		"input_tokens":    inputTokens,
		"output_tokens":   outputTokens,
		"total_tokens":    totalTokens,
		"path":            path,
		"stream":          true,
		"context_updated": true,
	}).Info("Real-time updated token context for billing")
}

// setStreamTokenUsage 设置流式请求的token使用量到上下文
func (h *GenericProxyHandler) setStreamTokenUsage(c *gin.Context, inputTokens, outputTokens int, path string) {
	totalTokens := inputTokens + outputTokens

	// 即使token为0也要记录，这样计费中间件知道我们尝试过获取token信息
	c.Set("input_tokens", inputTokens)
	c.Set("output_tokens", outputTokens)
	c.Set("total_tokens", totalTokens)
	c.Set("tokens_used", totalTokens)

	logLevel := "Info"
	if inputTokens <= 0 && outputTokens <= 0 {
		logLevel = "Warn"
	}

	logFields := map[string]interface{}{
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"total_tokens":  totalTokens,
		"path":          path,
		"stream":        true,
		"context_set":   true,
		"final":         true,
	}

	if logLevel == "Warn" {
		logFields["issue"] = "no_token_usage_found"
		h.logger.WithFields(logFields).Warn("No token usage found in stream response - billing may be inaccurate")
	} else {
		h.logger.WithFields(logFields).Info("Set final token usage information from stream response")
	}
}

// getAuthTypeByProvider 根据提供商类型获取认证方式
func (h *GenericProxyHandler) getAuthTypeByProvider(providerSlug string) string {
	// 转换为小写进行匹配
	slug := strings.ToLower(providerSlug)

	switch {
	case strings.Contains(slug, "anthropic") || strings.Contains(slug, "claude"):
		return "anthropic" // x-api-key + anthropic-version
	case strings.Contains(slug, "openai") || strings.Contains(slug, "gpt"):
		return "bearer"
	case strings.Contains(slug, "stability") || strings.Contains(slug, "stable"):
		return "bearer"
	case strings.Contains(slug, "midjourney") || strings.Contains(slug, "mj"):
		return "mj-api-secret"
	case strings.Contains(slug, "google") || strings.Contains(slug, "gemini"):
		return "bearer"
	case strings.Contains(slug, "cohere"):
		return "bearer"
	case strings.Contains(slug, "huggingface") || strings.Contains(slug, "hf"):
		return "bearer"
	case strings.Contains(slug, "replicate"):
		return "bearer"
	case strings.Contains(slug, "together"):
		return "bearer"
	case strings.Contains(slug, "perplexity"):
		return "bearer"
	default:
		h.logger.WithFields(map[string]interface{}{
			"provider_slug": providerSlug,
			"auth_type":     "bearer",
		}).Debug("Using default bearer auth for unknown provider")
		return "bearer" // 默认使用 Bearer token
	}
}

// debugContextValues 调试上下文中的计费相关值
func (h *GenericProxyHandler) debugContextValues(c *gin.Context, path string) {
	// 获取所有计费相关的上下文值
	contextValues := map[string]interface{}{
		"provider_id":   c.GetInt64("provider_id"),
		"provider_name": c.GetString("provider_name"),
		"model_name":    c.GetString("model_name"),
		"input_tokens":  c.GetInt("input_tokens"),
		"output_tokens": c.GetInt("output_tokens"),
		"total_tokens":  c.GetInt("total_tokens"),
		"tokens_used":   c.GetInt("tokens_used"),
		"cost_used":     c.GetFloat64("cost_used"),
	}

	// 检查哪些值存在
	existingValues := make(map[string]interface{})
	missingValues := make([]string, 0)

	for key, value := range contextValues {
		if exists, ok := c.Get(key); ok && exists != nil {
			existingValues[key] = value
		} else {
			missingValues = append(missingValues, key)
		}
	}

	h.logger.WithFields(map[string]interface{}{
		"path":            path,
		"existing_values": existingValues,
		"missing_values":  missingValues,
		"context_debug":   true,
	}).Info("Context values for billing middleware")
}

// extractContentFromStreamLine 从流式响应行中提取内容
func (h *GenericProxyHandler) extractContentFromStreamLine(line string) string {
	// 移除 "data: " 前缀
	if !strings.HasPrefix(line, "data: ") {
		return ""
	}

	dataContent := strings.TrimPrefix(line, "data: ")

	// 跳过特殊标记
	if dataContent == "[DONE]" || dataContent == "" {
		return ""
	}

	// 尝试解析JSON
	var streamData map[string]interface{}
	if err := json.Unmarshal([]byte(dataContent), &streamData); err != nil {
		return ""
	}

	// 提取choices中的content
	if choices, exists := streamData["choices"]; exists {
		if choicesArray, ok := choices.([]interface{}); ok && len(choicesArray) > 0 {
			if choice, ok := choicesArray[0].(map[string]interface{}); ok {
				if delta, exists := choice["delta"]; exists {
					if deltaMap, ok := delta.(map[string]interface{}); ok {
						if content, exists := deltaMap["content"]; exists {
							if contentStr, ok := content.(string); ok {
								return contentStr
							}
						}
					}
				}
			}
		}
	}

	return ""
}

// estimateTokensFromContent 从内容估算token数量
func (h *GenericProxyHandler) estimateTokensFromContent(c *gin.Context, outputContent, path string) (int, int) {
	// 估算输入token（从原始请求中）
	var inputTokens int

	// 尝试从请求体中提取输入内容进行估算
	if requestBody, exists := c.Get("original_request_body"); exists {
		if bodyBytes, ok := requestBody.([]byte); ok {
			inputTokens = h.estimateInputTokensFromRequest(bodyBytes)
		}
	}

	// 估算输出token
	outputTokens := h.estimateOutputTokens(outputContent)

	h.logger.WithFields(map[string]interface{}{
		"path":               path,
		"input_tokens":       inputTokens,
		"output_tokens":      outputTokens,
		"output_content_len": len(outputContent),
		"estimation_method":  "fallback",
		"stream":             true,
	}).Info("Estimated token usage from content (fallback method)")

	return inputTokens, outputTokens
}

// estimateInputTokensFromRequest 从请求体估算输入token
func (h *GenericProxyHandler) estimateInputTokensFromRequest(body []byte) int {
	if len(body) == 0 {
		return 0
	}

	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		return 0
	}

	var inputText strings.Builder

	// 提取messages中的内容
	if messages, exists := requestData["messages"]; exists {
		if messagesArray, ok := messages.([]interface{}); ok {
			for _, msg := range messagesArray {
				if msgMap, ok := msg.(map[string]interface{}); ok {
					if content, exists := msgMap["content"]; exists {
						if contentStr, ok := content.(string); ok {
							inputText.WriteString(contentStr)
							inputText.WriteString(" ")
						}
					}
				}
			}
		}
	}

	// 提取prompt内容（如果有）
	if prompt, exists := requestData["prompt"]; exists {
		if promptStr, ok := prompt.(string); ok {
			inputText.WriteString(promptStr)
		}
	}

	return h.estimateTokensFromText(inputText.String())
}

// estimateOutputTokens 估算输出token数量
func (h *GenericProxyHandler) estimateOutputTokens(content string) int {
	return h.estimateTokensFromText(content)
}

// estimateTokensFromText 从文本估算token数量
func (h *GenericProxyHandler) estimateTokensFromText(text string) int {
	if text == "" {
		return 0
	}

	// 简单的token估算算法
	// 英文：约4个字符 = 1个token
	// 中文：约1个字符 = 1个token
	// 这是一个粗略的估算，实际情况可能有差异

	runes := []rune(text)
	var tokens float64

	for _, r := range runes {
		if r >= 0x4e00 && r <= 0x9fff { // 中文字符
			tokens += 1.0
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			tokens += 0.25 // 英文字符约4个=1token
		} else {
			tokens += 0.5 // 其他字符
		}
	}

	return int(tokens + 0.5) // 四舍五入
}
