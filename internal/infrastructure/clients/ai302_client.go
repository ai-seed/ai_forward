package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"ai-api-gateway/internal/domain/entities"
)

// AI302Client 302.AI客户端接口
type AI302Client interface {
	// Upscale 图片放大
	Upscale(ctx context.Context, provider *entities.Provider, request *AI302UpscaleRequest) (*AI302UpscaleResponse, error)
}

// AI302UpscaleRequest 302.AI图片放大请求
type AI302UpscaleRequest struct {
	Image       []byte `json:"-"`                      // 图片字节数据
	Scale       int    `json:"scale,omitempty"`        // 放大倍数，0-10，默认4
	FaceEnhance bool   `json:"face_enhance,omitempty"` // 人脸增强，默认true
}

// AI302UpscaleJSONRequest 302.AI图片放大JSON请求
type AI302UpscaleJSONRequest struct {
	Image       string `json:"image" binding:"required"` // base64编码的图片
	Scale       int    `json:"scale,omitempty"`          // 放大倍数，0-10，默认4
	FaceEnhance bool   `json:"face_enhance,omitempty"`   // 人脸增强，默认true
}

// AI302UpscaleResponse 302.AI图片放大响应
type AI302UpscaleResponse struct {
	ID          string `json:"id"`
	Model       string `json:"model"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Output      string `json:"output"` // url
	Error       string `json:"error,omitempty"`
}

// ai302ClientImpl 302.AI客户端实现
type ai302ClientImpl struct {
	httpClient *http.Client
}

// NewAI302Client 创建302.AI客户端
func NewAI302Client(httpClient *http.Client) AI302Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}
	return &ai302ClientImpl{
		httpClient: httpClient,
	}
}

// Upscale 图片放大
func (c *ai302ClientImpl) Upscale(ctx context.Context, provider *entities.Provider, request *AI302UpscaleRequest) (*AI302UpscaleResponse, error) {
	// 构造请求URL
	url := fmt.Sprintf("%s/302/submit/upscale", provider.BaseURL)

	// 创建multipart form数据
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 直接使用传入的图片字节数据
	imageData := request.Image
	if len(imageData) == 0 {
		return nil, fmt.Errorf("image data is empty")
	}

	// 添加图片文件
	part, err := writer.CreateFormFile("image", "image.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(imageData); err != nil {
		return nil, fmt.Errorf("failed to write image data: %w", err)
	}

	// 添加scale参数
	scale := request.Scale
	if scale == 0 {
		scale = 4 // 默认值
	}
	if err := writer.WriteField("scale", strconv.Itoa(scale)); err != nil {
		return nil, fmt.Errorf("failed to write scale field: %w", err)
	}

	// 添加face_enhance参数
	faceEnhance := request.FaceEnhance
	if err := writer.WriteField("face_enhance", strconv.FormatBool(faceEnhance)); err != nil {
		return nil, fmt.Errorf("failed to write face_enhance field: %w", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	if provider.APIKeyEncrypted != nil {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted))
	}

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response AI302UpscaleResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
