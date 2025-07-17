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

// MidjourneyProxyClient Midjourney代理客户端 - 直接转发请求
type MidjourneyProxyClient interface {
	// ForwardRequest 转发请求到上游服务
	ForwardRequest(ctx context.Context, method, path string, headers map[string]string, body []byte, query url.Values) (*ProxyResponse, error)
}

// ProxyResponse 代理响应
type ProxyResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// midjourneyProxyClientImpl 代理客户端实现
type midjourneyProxyClientImpl struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     logger.Logger
}

// NewMidjourneyProxyClient 创建Midjourney代理客户端
func NewMidjourneyProxyClient(baseURL, apiKey string, logger logger.Logger) MidjourneyProxyClient {
	return &midjourneyProxyClientImpl{
		baseURL: strings.TrimSuffix(baseURL, "/"), // 移除末尾的斜杠
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// ForwardRequest 转发请求到上游服务
func (c *midjourneyProxyClientImpl) ForwardRequest(ctx context.Context, method, path string, headers map[string]string, body []byte, query url.Values) (*ProxyResponse, error) {
	c.logger.WithFields(map[string]interface{}{
		"method":        method,
		"path":          path,
		"headers_input": headers,
		"body_length":   len(body),
		"body_content":  string(body),
		"query":         query,
		"base_url":      c.baseURL,
	}).Info("=== PROXY CLIENT: ForwardRequest called with parameters ===")

	// 构造完整的上游URL
	upstreamURL := fmt.Sprintf("%s%s", c.baseURL, path)

	// 添加查询参数
	if len(query) > 0 {
		upstreamURL += "?" + query.Encode()
	}

	c.logger.WithFields(map[string]interface{}{
		"method":       method,
		"upstream_url": upstreamURL,
		"body_size":    len(body),
		"headers":      headers,
		"api_key_set":  c.apiKey != "",
	}).Info("=== PROXY CLIENT: Forwarding request to upstream Midjourney service ===")

	if len(body) > 0 {
		c.logger.WithFields(map[string]interface{}{
			"body_content": string(body),
		}).Debug("Request body content")
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

	c.logger.WithFields(map[string]interface{}{
		"headers_before_processing": headers,
		"api_key_length":            len(c.apiKey),
	}).Info("=== PROXY CLIENT: Processing headers ===")

	// 转发所有原始头部
	for key, value := range headers {
		// 跳过一些不应该转发的头部
		if c.shouldSkipHeader(key) {
			c.logger.WithFields(map[string]interface{}{
				"header_key": key,
				"skipped":    true,
			}).Debug("Skipping header")
			continue
		}
		req.Header.Set(key, value)
		c.logger.WithFields(map[string]interface{}{
			"header_key":   key,
			"header_value": value,
			"set":          true,
		}).Debug("Setting header")
	}

	// 设置或覆盖认证头
	if c.apiKey != "" {
		req.Header.Set("mj-api-secret", c.apiKey)
		c.logger.WithFields(map[string]interface{}{
			"api_key_first_10": c.apiKey[:10] + "...",
		}).Info("=== PROXY CLIENT: Set mj-api-secret header ===")
	} else {
		c.logger.Info("=== PROXY CLIENT: No API key to set ===")
	}

	// 确保Content-Type正确设置
	if len(body) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
		c.logger.Info("=== PROXY CLIENT: Set default Content-Type ===")
	}

	// 打印最终的请求头
	finalHeaders := make(map[string]string)
	for key, values := range req.Header {
		if len(values) > 0 {
			finalHeaders[key] = values[0]
		}
	}
	c.logger.WithFields(map[string]interface{}{
		"final_headers": finalHeaders,
		"body_length":   len(body),
	}).Info("=== PROXY CLIENT: Final request headers and body ===")

	if len(body) > 0 {
		c.logger.WithFields(map[string]interface{}{
			"body_content": string(body),
		}).Info("=== PROXY CLIENT: Request body content ===")
	}

	c.logger.WithFields(map[string]interface{}{
		"upstream_url": upstreamURL,
		"method":       method,
	}).Info("=== PROXY CLIENT: About to send HTTP request ===")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithFields(map[string]interface{}{
			"error":        err.Error(),
			"upstream_url": upstreamURL,
		}).Error("=== PROXY CLIENT: Failed to send request to upstream ===")
		return nil, fmt.Errorf("failed to send upstream request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.WithFields(map[string]interface{}{
		"upstream_url": upstreamURL,
		"status_code":  resp.StatusCode,
	}).Info("=== PROXY CLIENT: Received HTTP response ===")

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upstream response: %w", err)
	}

	// 构造响应头映射
	respHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders[key] = values[0]
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"status_code":   resp.StatusCode,
		"response_size": len(respBody),
		"upstream_url":  upstreamURL,
	}).Debug("Received response from upstream")

	return &ProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    respHeaders,
		Body:       respBody,
	}, nil
}

// shouldSkipHeader 判断是否应该跳过某个头部
func (c *midjourneyProxyClientImpl) shouldSkipHeader(headerName string) bool {
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

// MidjourneyForwardingService 基于转发的Midjourney服务
type MidjourneyForwardingService struct {
	proxyClient MidjourneyProxyClient
	logger      logger.Logger
}

// NewMidjourneyForwardingService 创建转发服务
func NewMidjourneyForwardingService(proxyClient MidjourneyProxyClient, logger logger.Logger) *MidjourneyForwardingService {
	return &MidjourneyForwardingService{
		proxyClient: proxyClient,
		logger:      logger,
	}
}

// ForwardMidjourneyRequest 转发Midjourney请求的便捷方法
func (s *MidjourneyForwardingService) ForwardMidjourneyRequest(ctx context.Context, method, path string, headers map[string]string, body []byte, query url.Values) (*ProxyResponse, error) {
	return s.proxyClient.ForwardRequest(ctx, method, path, headers, body, query)
}

// 一些常用的转发方法

// ForwardImagineRequest 转发imagine请求
func (s *MidjourneyForwardingService) ForwardImagineRequest(ctx context.Context, headers map[string]string, body []byte) (*ProxyResponse, error) {
	return s.proxyClient.ForwardRequest(ctx, "POST", "/mj/submit/imagine", headers, body, nil)
}

// ForwardActionRequest 转发action请求
func (s *MidjourneyForwardingService) ForwardActionRequest(ctx context.Context, headers map[string]string, body []byte) (*ProxyResponse, error) {
	return s.proxyClient.ForwardRequest(ctx, "POST", "/mj/submit/action", headers, body, nil)
}

// ForwardBlendRequest 转发blend请求
func (s *MidjourneyForwardingService) ForwardBlendRequest(ctx context.Context, headers map[string]string, body []byte) (*ProxyResponse, error) {
	return s.proxyClient.ForwardRequest(ctx, "POST", "/mj/submit/blend", headers, body, nil)
}

// ForwardDescribeRequest 转发describe请求
func (s *MidjourneyForwardingService) ForwardDescribeRequest(ctx context.Context, headers map[string]string, body []byte) (*ProxyResponse, error) {
	return s.proxyClient.ForwardRequest(ctx, "POST", "/mj/submit/describe", headers, body, nil)
}

// ForwardModalRequest 转发modal请求
func (s *MidjourneyForwardingService) ForwardModalRequest(ctx context.Context, headers map[string]string, body []byte) (*ProxyResponse, error) {
	return s.proxyClient.ForwardRequest(ctx, "POST", "/mj/submit/modal", headers, body, nil)
}

// ForwardCancelRequest 转发cancel请求
func (s *MidjourneyForwardingService) ForwardCancelRequest(ctx context.Context, headers map[string]string, body []byte) (*ProxyResponse, error) {
	return s.proxyClient.ForwardRequest(ctx, "POST", "/mj/submit/cancel", headers, body, nil)
}

// ForwardFetchRequest 转发fetch请求
func (s *MidjourneyForwardingService) ForwardFetchRequest(ctx context.Context, taskID string, headers map[string]string) (*ProxyResponse, error) {
	path := fmt.Sprintf("/mj/task/%s/fetch", taskID)
	return s.proxyClient.ForwardRequest(ctx, "GET", path, headers, nil, nil)
}
