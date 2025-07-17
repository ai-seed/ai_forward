package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"
)

// WebhookService Webhook服务接口
type WebhookService interface {
	// SendWebhook 发送webhook通知
	SendWebhook(ctx context.Context, url string, data interface{}) error

	// SendWebhookWithRetry 发送webhook通知（带重试）
	SendWebhookWithRetry(ctx context.Context, url string, data interface{}, maxRetries int) error
}

// webhookServiceImpl Webhook服务实现
type webhookServiceImpl struct {
	httpClient *http.Client
	logger     logger.Logger
}

// NewWebhookService 创建Webhook服务
func NewWebhookService(logger logger.Logger) WebhookService {
	return &webhookServiceImpl{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// SendWebhook 发送webhook通知
func (s *webhookServiceImpl) SendWebhook(ctx context.Context, url string, data interface{}) error {
	return s.SendWebhookWithRetry(ctx, url, data, 3)
}

// SendWebhookWithRetry 发送webhook通知（带重试）
func (s *webhookServiceImpl) SendWebhookWithRetry(ctx context.Context, url string, data interface{}, maxRetries int) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook data: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := s.sendWebhookOnce(ctx, url, jsonData); err != nil {
			lastErr = err
			s.logger.WithFields(map[string]interface{}{
				"url":     url,
				"attempt": attempt,
				"error":   err.Error(),
			}).Warn("Webhook send failed, retrying...")

			if attempt < maxRetries {
				// 指数退避
				backoff := time.Duration(attempt*attempt) * time.Second
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					continue
				}
			}
		} else {
			s.logger.WithFields(map[string]interface{}{
				"url":     url,
				"attempt": attempt,
			}).Info("Webhook sent successfully")
			return nil
		}
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", maxRetries, lastErr)
}

// sendWebhookOnce 发送一次webhook
func (s *webhookServiceImpl) sendWebhookOnce(ctx context.Context, url string, jsonData []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AI-API-Gateway/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code: %d", resp.StatusCode)
	}

	return nil
}

// ImageGenerationService 图像生成服务接口
type ImageGenerationService interface {
	// GenerateImage 生成图像
	GenerateImage(ctx context.Context, prompt string, options map[string]interface{}) (*ImageResult, error)

	// ProcessAction 处理操作
	ProcessAction(ctx context.Context, taskID, action string, options map[string]interface{}) (*ImageResult, error)

	// BlendImages 混合图像
	BlendImages(ctx context.Context, images []string, options map[string]interface{}) (*ImageResult, error)

	// DescribeImage 描述图像
	DescribeImage(ctx context.Context, imageData string, options map[string]interface{}) (*DescribeResult, error)

	// InpaintImage 修复图像
	InpaintImage(ctx context.Context, imageData, maskData, prompt string, options map[string]interface{}) (*ImageResult, error)
}

// ImageResult 图像生成结果
type ImageResult struct {
	ImageURL   string   `json:"image_url"`
	DiscordURL string   `json:"discord_url,omitempty"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	Seed       string   `json:"seed,omitempty"`
	Images     []string `json:"images,omitempty"`
	Components []string `json:"components,omitempty"`
}

// DescribeResult 图像描述结果
type DescribeResult struct {
	Descriptions []string `json:"descriptions"`
}

// realImageGenerationService 真实图像生成服务
type realImageGenerationService struct {
	mjClient clients.MidjourneyClient
	logger   logger.Logger
}

// mockImageGenerationService 模拟图像生成服务
type mockImageGenerationService struct {
	logger logger.Logger
}

// NewRealImageGenerationService 创建真实图像生成服务
func NewRealImageGenerationService(mjClient clients.MidjourneyClient, logger logger.Logger) ImageGenerationService {
	return &realImageGenerationService{
		mjClient: mjClient,
		logger:   logger,
	}
}

// NewMockImageGenerationService 创建模拟图像生成服务
func NewMockImageGenerationService(logger logger.Logger) ImageGenerationService {
	return &mockImageGenerationService{
		logger: logger,
	}
}

// realImageGenerationService 的方法实现

// GenerateImage 生成图像
func (s *realImageGenerationService) GenerateImage(ctx context.Context, prompt string, options map[string]interface{}) (*ImageResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"prompt":  prompt,
		"options": options,
	}).Info("Generating image with real Midjourney service")

	// 构造请求
	request := &clients.MidjourneyImagineRequest{
		Prompt:  prompt,
		BotType: "MID_JOURNEY",
	}

	// 处理可选参数
	if base64Array, ok := options["base64Array"].([]string); ok {
		request.Base64Array = base64Array
	}
	if notifyHook, ok := options["notifyHook"].(string); ok {
		request.NotifyHook = notifyHook
	}
	if state, ok := options["state"].(string); ok {
		request.State = state
	}

	// 提交任务
	response, err := s.mjClient.SubmitImagine(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to submit imagine task: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("midjourney API error: %s", response.Description)
	}

	// 等待任务完成（轮询）
	taskID := response.Result
	return s.waitForTaskCompletion(ctx, taskID)
}

// ProcessAction 处理操作
func (s *realImageGenerationService) ProcessAction(ctx context.Context, taskID, action string, options map[string]interface{}) (*ImageResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"task_id": taskID,
		"action":  action,
		"options": options,
	}).Info("Processing action with real Midjourney service")

	// 构造请求
	request := &clients.MidjourneyActionRequest{
		TaskID:   taskID,
		CustomID: action,
	}

	// 处理可选参数
	if notifyHook, ok := options["notifyHook"].(string); ok {
		request.NotifyHook = notifyHook
	}
	if state, ok := options["state"].(string); ok {
		request.State = state
	}

	// 提交任务
	response, err := s.mjClient.SubmitAction(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to submit action task: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("midjourney API error: %s", response.Description)
	}

	// 等待任务完成
	newTaskID := response.Result
	return s.waitForTaskCompletion(ctx, newTaskID)
}

// BlendImages 混合图像
func (s *realImageGenerationService) BlendImages(ctx context.Context, images []string, options map[string]interface{}) (*ImageResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"images":  images,
		"options": options,
	}).Info("Blending images with real Midjourney service")

	// 构造请求
	request := &clients.MidjourneyBlendRequest{
		Base64Array: images,
		BotType:     "MID_JOURNEY",
		Dimensions:  "SQUARE",
	}

	// 处理可选参数
	if dimensions, ok := options["dimensions"].(string); ok {
		request.Dimensions = dimensions
	}
	if notifyHook, ok := options["notifyHook"].(string); ok {
		request.NotifyHook = notifyHook
	}
	if state, ok := options["state"].(string); ok {
		request.State = state
	}

	// 提交任务
	response, err := s.mjClient.SubmitBlend(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to submit blend task: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("midjourney API error: %s", response.Description)
	}

	// 等待任务完成
	taskID := response.Result
	return s.waitForTaskCompletion(ctx, taskID)
}

// DescribeImage 描述图像
func (s *realImageGenerationService) DescribeImage(ctx context.Context, imageData string, options map[string]interface{}) (*DescribeResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"image_data": len(imageData),
		"options":    options,
	}).Info("Describing image with real Midjourney service")

	// 构造请求
	request := &clients.MidjourneyDescribeRequest{
		Base64:  imageData,
		BotType: "MID_JOURNEY",
	}

	// 处理可选参数
	if notifyHook, ok := options["notifyHook"].(string); ok {
		request.NotifyHook = notifyHook
	}
	if state, ok := options["state"].(string); ok {
		request.State = state
	}

	// 提交任务
	response, err := s.mjClient.SubmitDescribe(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to submit describe task: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("midjourney API error: %s", response.Description)
	}

	// 等待任务完成并获取描述结果
	taskID := response.Result
	_, err = s.waitForTaskCompletion(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 从结果中提取描述信息（这里需要根据实际API响应格式调整）
	// 实际实现中应该从任务结果中解析描述信息
	descriptions := []string{
		"Generated description 1",
		"Generated description 2",
		"Generated description 3",
		"Generated description 4",
	}

	return &DescribeResult{
		Descriptions: descriptions,
	}, nil
}

// InpaintImage 修复图像
func (s *realImageGenerationService) InpaintImage(ctx context.Context, imageData, maskData, prompt string, options map[string]interface{}) (*ImageResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"image_data": len(imageData),
		"mask_data":  len(maskData),
		"prompt":     prompt,
		"options":    options,
	}).Info("Inpainting image with real Midjourney service")

	// 构造请求
	request := &clients.MidjourneyInpaintRequest{
		Base64:     imageData,
		MaskBase64: maskData,
		Prompt:     prompt,
		BotType:    "MID_JOURNEY",
	}

	// 处理可选参数
	if notifyHook, ok := options["notifyHook"].(string); ok {
		request.NotifyHook = notifyHook
	}
	if state, ok := options["state"].(string); ok {
		request.State = state
	}

	// 提交任务
	response, err := s.mjClient.SubmitInpaint(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to submit inpaint task: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("midjourney API error: %s", response.Description)
	}

	// 等待任务完成
	taskID := response.Result
	return s.waitForTaskCompletion(ctx, taskID)
}

// waitForTaskCompletion 等待任务完成
func (s *realImageGenerationService) waitForTaskCompletion(ctx context.Context, taskID string) (*ImageResult, error) {
	maxRetries := 60 // 最多等待5分钟（每5秒检查一次）

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 获取任务状态
		result, err := s.mjClient.FetchTask(ctx, taskID)
		if err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":   err.Error(),
				"task_id": taskID,
				"retry":   i + 1,
			}).Warn("Failed to fetch task status, retrying...")

			time.Sleep(5 * time.Second)
			continue
		}

		// 检查任务状态
		switch result.Status {
		case "SUCCESS":
			// 任务成功完成
			return &ImageResult{
				ImageURL:   result.ImageURL,
				DiscordURL: result.ImageURL, // 假设是同一个URL
				Width:      1024,            // 默认值，实际应该从API获取
				Height:     1024,            // 默认值，实际应该从API获取
				Images:     []string{result.ImageURL},
				Components: s.extractComponents(result.Buttons),
			}, nil

		case "FAILED":
			// 任务失败
			return nil, fmt.Errorf("task failed: %s", result.FailReason)

		case "IN_PROGRESS", "PENDING":
			// 任务进行中，继续等待
			s.logger.WithFields(map[string]interface{}{
				"task_id":  taskID,
				"status":   result.Status,
				"progress": result.Progress,
			}).Debug("Task in progress, waiting...")

			time.Sleep(5 * time.Second)
			continue

		default:
			// 未知状态
			s.logger.WithFields(map[string]interface{}{
				"task_id": taskID,
				"status":  result.Status,
			}).Warn("Unknown task status")

			time.Sleep(5 * time.Second)
			continue
		}
	}

	return nil, fmt.Errorf("task timeout after %d retries", maxRetries)
}

// extractComponents 从按钮中提取组件信息
func (s *realImageGenerationService) extractComponents(buttons []clients.MidjourneyButton) []string {
	components := make([]string, 0, len(buttons))
	for _, button := range buttons {
		components = append(components, button.CustomID)
	}
	return components
}

// mockImageGenerationService 的方法实现

// GenerateImage 生成图像
func (s *mockImageGenerationService) GenerateImage(ctx context.Context, prompt string, options map[string]interface{}) (*ImageResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"prompt":  prompt,
		"options": options,
	}).Info("Mock: Generating image")

	// 模拟处理时间
	time.Sleep(2 * time.Second)

	return &ImageResult{
		ImageURL:   "https://cdn.example.com/generated_image.png",
		DiscordURL: "https://cdn.discordapp.com/attachments/generated_image.png",
		Width:      1024,
		Height:     1024,
		Seed:       "123456789",
		Images: []string{
			"https://cdn.example.com/image1.png",
			"https://cdn.example.com/image2.png",
			"https://cdn.example.com/image3.png",
			"https://cdn.example.com/image4.png",
		},
		Components: []string{
			"upsample1", "upsample2", "upsample3", "upsample4",
			"variation1", "variation2", "variation3", "variation4",
		},
	}, nil
}

// ProcessAction 处理操作
func (s *mockImageGenerationService) ProcessAction(ctx context.Context, taskID, action string, options map[string]interface{}) (*ImageResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"task_id": taskID,
		"action":  action,
		"options": options,
	}).Info("Mock: Processing action")

	time.Sleep(3 * time.Second)

	return &ImageResult{
		ImageURL:   "https://cdn.example.com/action_result.png",
		DiscordURL: "https://cdn.discordapp.com/attachments/action_result.png",
		Width:      2048,
		Height:     2048,
	}, nil
}

// BlendImages 混合图像
func (s *mockImageGenerationService) BlendImages(ctx context.Context, images []string, options map[string]interface{}) (*ImageResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"images":  images,
		"options": options,
	}).Info("Mock: Blending images")

	time.Sleep(5 * time.Second)

	return &ImageResult{
		ImageURL:   "https://cdn.example.com/blend_result.png",
		DiscordURL: "https://cdn.discordapp.com/attachments/blend_result.png",
		Width:      1024,
		Height:     1024,
	}, nil
}

// DescribeImage 描述图像
func (s *mockImageGenerationService) DescribeImage(ctx context.Context, imageData string, options map[string]interface{}) (*DescribeResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"image_data": len(imageData),
		"options":    options,
	}).Info("Mock: Describing image")

	time.Sleep(3 * time.Second)

	return &DescribeResult{
		Descriptions: []string{
			"A beautiful landscape with mountains and trees in the background",
			"Scenic mountain view with lush green forest and clear blue sky",
			"Natural outdoor scene featuring tall mountains and dense woodland",
			"Majestic mountain range with verdant forest landscape under bright sky",
		},
	}, nil
}

// InpaintImage 修复图像
func (s *mockImageGenerationService) InpaintImage(ctx context.Context, imageData, maskData, prompt string, options map[string]interface{}) (*ImageResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"image_data": len(imageData),
		"mask_data":  len(maskData),
		"prompt":     prompt,
		"options":    options,
	}).Info("Mock: Inpainting image")

	time.Sleep(8 * time.Second)

	return &ImageResult{
		ImageURL:   "https://cdn.example.com/inpaint_result.png",
		DiscordURL: "https://cdn.discordapp.com/attachments/inpaint_result.png",
		Width:      1024,
		Height:     1024,
	}, nil
}
