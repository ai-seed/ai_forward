package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-api-gateway/internal/infrastructure/logger"
)

// MidjourneyClient Midjourney上游服务客户端接口
type MidjourneyClient interface {
	// SubmitImagine 提交图像生成任务
	SubmitImagine(ctx context.Context, request *MidjourneyImagineRequest) (*MidjourneySubmitResponse, error)

	// SubmitAction 提交操作任务 (U1-U4, V1-V4)
	SubmitAction(ctx context.Context, request *MidjourneyActionRequest) (*MidjourneySubmitResponse, error)

	// SubmitBlend 提交图像混合任务
	SubmitBlend(ctx context.Context, request *MidjourneyBlendRequest) (*MidjourneySubmitResponse, error)

	// SubmitDescribe 提交图像描述任务
	SubmitDescribe(ctx context.Context, request *MidjourneyDescribeRequest) (*MidjourneySubmitResponse, error)

	// SubmitInpaint 提交局部重绘任务
	SubmitInpaint(ctx context.Context, request *MidjourneyInpaintRequest) (*MidjourneySubmitResponse, error)

	// FetchTask 获取任务结果
	FetchTask(ctx context.Context, taskID string) (*MidjourneyTaskResult, error)

	// CancelTask 取消任务
	CancelTask(ctx context.Context, taskID string) error
}

// MidjourneyImagineRequest 图像生成请求
type MidjourneyImagineRequest struct {
	Prompt      string   `json:"prompt"`
	BotType     string   `json:"botType,omitempty"`
	Base64Array []string `json:"base64Array,omitempty"`
	NotifyHook  string   `json:"notifyHook,omitempty"`
	State       string   `json:"state,omitempty"`
}

// MidjourneyActionRequest 操作请求
type MidjourneyActionRequest struct {
	TaskID     string `json:"taskId"`
	CustomID   string `json:"customId"`
	NotifyHook string `json:"notifyHook,omitempty"`
	State      string `json:"state,omitempty"`
}

// MidjourneyBlendRequest 混合请求
type MidjourneyBlendRequest struct {
	Base64Array []string `json:"base64Array"`
	BotType     string   `json:"botType,omitempty"`
	Dimensions  string   `json:"dimensions,omitempty"`
	NotifyHook  string   `json:"notifyHook,omitempty"`
	State       string   `json:"state,omitempty"`
}

// MidjourneyDescribeRequest 描述请求
type MidjourneyDescribeRequest struct {
	Base64     string `json:"base64"`
	BotType    string `json:"botType,omitempty"`
	NotifyHook string `json:"notifyHook,omitempty"`
	State      string `json:"state,omitempty"`
}

// MidjourneyInpaintRequest 局部重绘请求
type MidjourneyInpaintRequest struct {
	Base64     string `json:"base64"`
	MaskBase64 string `json:"maskBase64"`
	Prompt     string `json:"prompt"`
	BotType    string `json:"botType,omitempty"`
	NotifyHook string `json:"notifyHook,omitempty"`
	State      string `json:"state,omitempty"`
}

// MidjourneySubmitResponse 提交响应
type MidjourneySubmitResponse struct {
	Code        int                    `json:"code"`
	Description string                 `json:"description"`
	Properties  map[string]interface{} `json:"properties"`
	Result      string                 `json:"result"` // 任务ID
}

// MidjourneyTaskResult 任务结果
type MidjourneyTaskResult struct {
	ID          string                 `json:"id"`
	Action      string                 `json:"action"`
	Status      string                 `json:"status"`
	Progress    string                 `json:"progress"`
	SubmitTime  int64                  `json:"submitTime"`
	StartTime   *int64                 `json:"startTime,omitempty"`
	FinishTime  *int64                 `json:"finishTime,omitempty"`
	ImageURL    string                 `json:"imageUrl,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`
	PromptEn    string                 `json:"promptEn,omitempty"`
	Description string                 `json:"description,omitempty"`
	FailReason  string                 `json:"failReason,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Buttons     []MidjourneyButton     `json:"buttons,omitempty"`
}

// MidjourneyButton 操作按钮
type MidjourneyButton struct {
	CustomID string `json:"customId"`
	Label    string `json:"label"`
	Type     int    `json:"type"`
	Style    int    `json:"style,omitempty"`
	Emoji    string `json:"emoji,omitempty"`
}

// midjourneyClientImpl Midjourney客户端实现
type midjourneyClientImpl struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     logger.Logger
}

// NewMidjourneyClient 创建Midjourney客户端
func NewMidjourneyClient(baseURL, apiKey string, logger logger.Logger) MidjourneyClient {
	return &midjourneyClientImpl{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// SubmitImagine 提交图像生成任务
func (c *midjourneyClientImpl) SubmitImagine(ctx context.Context, request *MidjourneyImagineRequest) (*MidjourneySubmitResponse, error) {
	url := fmt.Sprintf("%s/mj/submit/imagine", c.baseURL)
	return c.submitRequest(ctx, url, request)
}

// SubmitAction 提交操作任务
func (c *midjourneyClientImpl) SubmitAction(ctx context.Context, request *MidjourneyActionRequest) (*MidjourneySubmitResponse, error) {
	url := fmt.Sprintf("%s/mj/submit/action", c.baseURL)
	return c.submitRequest(ctx, url, request)
}

// SubmitBlend 提交图像混合任务
func (c *midjourneyClientImpl) SubmitBlend(ctx context.Context, request *MidjourneyBlendRequest) (*MidjourneySubmitResponse, error) {
	url := fmt.Sprintf("%s/mj/submit/blend", c.baseURL)
	return c.submitRequest(ctx, url, request)
}

// SubmitDescribe 提交图像描述任务
func (c *midjourneyClientImpl) SubmitDescribe(ctx context.Context, request *MidjourneyDescribeRequest) (*MidjourneySubmitResponse, error) {
	url := fmt.Sprintf("%s/mj/submit/describe", c.baseURL)
	return c.submitRequest(ctx, url, request)
}

// SubmitInpaint 提交局部重绘任务
func (c *midjourneyClientImpl) SubmitInpaint(ctx context.Context, request *MidjourneyInpaintRequest) (*MidjourneySubmitResponse, error) {
	url := fmt.Sprintf("%s/mj/submit/modal", c.baseURL)
	return c.submitRequest(ctx, url, request)
}

// FetchTask 获取任务结果
func (c *midjourneyClientImpl) FetchTask(ctx context.Context, taskID string) (*MidjourneyTaskResult, error) {
	url := fmt.Sprintf("%s/mj/task/%s/fetch", c.baseURL, taskID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置认证头
	req.Header.Set("mj-api-secret", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(map[string]interface{}{
		"url":     url,
		"task_id": taskID,
	}).Debug("Fetching Midjourney task")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.WithFields(map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("Midjourney API error")
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result MidjourneyTaskResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// CancelTask 取消任务
func (c *midjourneyClientImpl) CancelTask(ctx context.Context, taskID string) error {
	url := fmt.Sprintf("%s/mj/submit/cancel", c.baseURL)

	request := map[string]string{
		"taskId": taskID,
	}

	_, err := c.submitRequest(ctx, url, request)
	return err
}

// submitRequest 提交请求的通用方法
func (c *midjourneyClientImpl) submitRequest(ctx context.Context, url string, request interface{}) (*MidjourneySubmitResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置认证头
	req.Header.Set("mj-api-secret", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(map[string]interface{}{
		"url":     url,
		"request": string(jsonData),
	}).Debug("Sending Midjourney request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.WithFields(map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("Midjourney API error")
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result MidjourneySubmitResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"task_id": result.Result,
		"code":    result.Code,
	}).Info("Midjourney task submitted successfully")

	return &result, nil
}
