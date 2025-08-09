package clients

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"ai-api-gateway/internal/domain/entities"
)

// VectorizerClient Vectorizer客户端接口
type VectorizerClient interface {
	// Vectorize 矢量化图片
	Vectorize(ctx context.Context, provider *entities.Provider, request *VectorizerRequest) (*VectorizerResponse, error)
}

// VectorizerRequest 矢量化请求
type VectorizerRequest struct {
	Image string `json:"image"` // base64 encoded image
}

// VectorizerResponse 矢量化响应
type VectorizerResponse struct {
	SVGData      string            `json:"svg_data,omitempty"`      // SVG矢量图数据
	FinishReason string            `json:"finish_reason,omitempty"` // 完成原因
	Error        string            `json:"error,omitempty"`         // 错误信息
	Cost         *VectorizerCost   `json:"cost,omitempty"`          // 成本信息
	ProviderID   int64             `json:"provider_id,omitempty"`   // 提供商ID
}

// VectorizerCost 矢量化成本信息
type VectorizerCost struct {
	TotalCost float64 `json:"total_cost"` // 总成本
	Currency  string  `json:"currency"`   // 货币
}

// vectorizerClientImpl Vectorizer客户端实现
type vectorizerClientImpl struct {
	httpClient *http.Client
}

// NewVectorizerClient 创建Vectorizer客户端
func NewVectorizerClient(httpClient *http.Client) VectorizerClient {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 60 * time.Second, // 矢量化可能需要更长时间
		}
	}

	return &vectorizerClientImpl{
		httpClient: httpClient,
	}
}

// Vectorize 矢量化图片
func (c *vectorizerClientImpl) Vectorize(ctx context.Context, provider *entities.Provider, request *VectorizerRequest) (*VectorizerResponse, error) {
	url := fmt.Sprintf("%s/vectorizer/api/v1/vectorize", provider.BaseURL)
	return c.sendVectorizerRequest(ctx, provider, url, request)
}

// sendVectorizerRequest 发送矢量化请求
func (c *vectorizerClientImpl) sendVectorizerRequest(ctx context.Context, provider *entities.Provider, url string, request *VectorizerRequest) (*VectorizerResponse, error) {
	fmt.Printf("[DEBUG] 准备发送Vectorizer请求: provider_id=%d, provider_name=%s, url=%s, has_api_key=%t\n",
		provider.ID, provider.Name, url, provider.APIKeyEncrypted != nil)

	// 创建multipart/form-data请求
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 添加图片文件 - Image字段包含base64编码的图片数据
	if request.Image != "" {
		// 解码base64图片数据
		imageData, err := base64.StdEncoding.DecodeString(request.Image)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 image: %w", err)
		}

		part, err := writer.CreateFormFile("image", "image.png")
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		// 写入解码后的图片数据
		_, err = part.Write(imageData)
		if err != nil {
			return nil, fmt.Errorf("failed to write image data: %w", err)
		}
	}

	writer.Close()

	fmt.Printf("[INFO] 发送HTTP multipart请求: url=%s, method=POST\n", url)
	fmt.Printf("[DEBUG] 请求体大小: %d字节\n", body.Len())

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		fmt.Printf("[ERROR] 创建HTTP请求失败: %v, url=%s\n", err, url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Accept", "image/svg+xml")

	fmt.Printf("[DEBUG] 请求头信息:\n")
	fmt.Printf("  Content-Type: %s\n", writer.FormDataContentType())
	fmt.Printf("  Accept: image/svg+xml\n")

	if provider.APIKeyEncrypted != nil {
		authHeader := fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted)
		httpReq.Header.Set("Authorization", authHeader)
		apiKeyPrefix := ""
		if len(*provider.APIKeyEncrypted) > 10 {
			apiKeyPrefix = (*provider.APIKeyEncrypted)[:10]
		} else {
			apiKeyPrefix = *provider.APIKeyEncrypted
		}
		fmt.Printf("[INFO] 设置认证头: auth_header_length=%d, api_key_prefix=%s\n", len(authHeader), apiKeyPrefix)
	} else {
		fmt.Printf("[ERROR] 提供商没有API密钥\n")
	}

	fmt.Printf("[INFO] 发送HTTP请求: url=%s, method=POST\n", url)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		fmt.Printf("[ERROR] 发送HTTP请求失败: %v, url=%s\n", err, url)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[ERROR] 读取响应失败: %v\n", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("[INFO] 收到API响应: status_code=%d, response_length=%d\n",
		resp.StatusCode, len(responseBody))

	// 输出响应头信息
	fmt.Printf("[INFO] 响应头信息:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// 检查响应内容类型
	contentType := resp.Header.Get("Content-Type")
	fmt.Printf("[INFO] 响应Content-Type: %s, response_length=%d\n", contentType, len(responseBody))

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[ERROR] API请求失败: status_code=%d, url=%s\n",
			resp.StatusCode, url)
		return &VectorizerResponse{
			Error:        fmt.Sprintf("API request failed with status %d", resp.StatusCode),
			FinishReason: "ERROR",
		}, nil
	}

	// 将SVG数据转换为字符串返回
	svgData := string(responseBody)
	fmt.Printf("[INFO] SVG矢量化完成，数据长度: %d字符\n", len(svgData))

	return &VectorizerResponse{
		SVGData:      svgData,
		FinishReason: "SUCCESS",
	}, nil
}
