package clients

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ai-api-gateway/internal/infrastructure/logger"
)

// GenericProxyClient 通用代理客户端接口
type GenericProxyClient interface {
	// ForwardRequest 转发请求到上游服务
	ForwardRequest(ctx context.Context, method, path string, headers map[string]string, body []byte, query url.Values) (*GenericProxyResponse, error)
	// ForwardStreamRequest 转发流式请求到上游服务
	ForwardStreamRequest(ctx context.Context, method, path string, headers map[string]string, body []byte, query url.Values) (*StreamResponse, error)
}

// GenericProxyResponse 通用代理响应
type GenericProxyResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	IsStream   bool
}

// StreamResponse 流式响应
type StreamResponse struct {
	StatusCode int
	Headers    map[string]string
	Reader     io.ReadCloser
}

// genericProxyClientImpl 通用代理客户端实现
type genericProxyClientImpl struct {
	baseURL    string
	apiKey     string
	authType   string // "bearer", "x-api-key", "mj-api-secret", "anthropic"
	httpClient *http.Client
	logger     logger.Logger
}

// NewGenericProxyClient 创建通用代理客户端
func NewGenericProxyClient(baseURL, apiKey, authType string, logger logger.Logger) GenericProxyClient {
	return &genericProxyClientImpl{
		baseURL:    strings.TrimSuffix(baseURL, "/"), // 移除末尾的斜杠
		apiKey:     apiKey,
		authType:   authType,
		httpClient: &http.Client{Timeout: 60 * time.Second}, // 增加超时时间以支持流式响应
		logger:     logger,
	}
}

// ForwardRequest 转发普通请求到上游服务
func (c *genericProxyClientImpl) ForwardRequest(ctx context.Context, method, path string, headers map[string]string, body []byte, query url.Values) (*GenericProxyResponse, error) {
	// 构造完整的上游URL
	upstreamURL := fmt.Sprintf("%s%s", c.baseURL, path)

	// 添加查询参数
	if len(query) > 0 {
		upstreamURL += "?" + query.Encode()
	}

	// 创建请求
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, upstreamURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create upstream request: %w", err)
	}

	// 转发所有原始头部
	for key, value := range headers {
		// 跳过一些不应该转发的头部
		if c.shouldSkipHeader(key) {
			continue
		}
		req.Header.Set(key, value)
	}

	// 设置认证头
	c.setAuthHeader(req)

	// 确保Content-Type正确设置
	if len(body) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	c.logger.WithFields(map[string]interface{}{
		"method":       method,
		"upstream_url": upstreamURL,
		"auth_type":    c.authType,
		"body_size":    len(body),
	}).Info("Forwarding request to upstream")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithFields(map[string]interface{}{
			"error":        err.Error(),
			"upstream_url": upstreamURL,
		}).Error("Failed to send request to upstream")
		return nil, fmt.Errorf("failed to send upstream request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upstream response: %w", err)
	}

	// 复制响应头
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	// 检查是否为流式响应
	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	c.logger.WithFields(map[string]interface{}{
		"status_code": resp.StatusCode,
		"body_size":   len(responseBody),
		"is_stream":   isStream,
	}).Info("Received upstream response")

	return &GenericProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       responseBody,
		IsStream:   isStream,
	}, nil
}

// ForwardStreamRequest 转发流式请求到上游服务
func (c *genericProxyClientImpl) ForwardStreamRequest(ctx context.Context, method, path string, headers map[string]string, body []byte, query url.Values) (*StreamResponse, error) {
	// 构造完整的上游URL
	upstreamURL := fmt.Sprintf("%s%s", c.baseURL, path)

	// 添加查询参数
	if len(query) > 0 {
		upstreamURL += "?" + query.Encode()
	}

	// 创建请求
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, upstreamURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create upstream request: %w", err)
	}

	// 转发所有原始头部
	for key, value := range headers {
		if c.shouldSkipHeader(key) {
			continue
		}
		req.Header.Set(key, value)
	}

	// 设置认证头
	c.setAuthHeader(req)

	// 确保流式请求的头部设置
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if len(body) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	c.logger.WithFields(map[string]interface{}{
		"method":       method,
		"upstream_url": upstreamURL,
		"auth_type":    c.authType,
		"body_size":    len(body),
	}).Info("Forwarding stream request to upstream")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithFields(map[string]interface{}{
			"error":        err.Error(),
			"upstream_url": upstreamURL,
		}).Error("Failed to send stream request to upstream")
		return nil, fmt.Errorf("failed to send upstream stream request: %w", err)
	}

	// 复制响应头
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"status_code":  resp.StatusCode,
		"content_type": resp.Header.Get("Content-Type"),
	}).Info("Received upstream stream response")

	return &StreamResponse{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Reader:     resp.Body, // 不关闭，让调用者处理
	}, nil
}

// setAuthHeader 设置认证头
func (c *genericProxyClientImpl) setAuthHeader(req *http.Request) {
	if c.apiKey == "" {
		return
	}

	switch c.authType {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	case "x-api-key":
		req.Header.Set("x-api-key", c.apiKey)
	case "mj-api-secret":
		req.Header.Set("mj-api-secret", c.apiKey)
	case "anthropic":
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		// 默认使用 Bearer token
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// shouldSkipHeader 判断是否应该跳过某个头部
func (c *genericProxyClientImpl) shouldSkipHeader(headerName string) bool {
	// 转换为小写进行比较
	lower := strings.ToLower(headerName)

	// 跳过这些头部，因为它们会被HTTP客户端自动处理或不应该转发
	skipHeaders := []string{
		"host",
		"content-length",
		"connection",
		"upgrade",
		"proxy-connection",
		"proxy-authenticate",
		"proxy-authorization",
		"te",
		"trailers",
		"transfer-encoding",
	}

	for _, skip := range skipHeaders {
		if lower == skip {
			return true
		}
	}

	return false
}
