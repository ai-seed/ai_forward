package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-api-gateway/internal/domain/entities"
)

// StabilityClient Stability.ai客户端接口
type StabilityClient interface {
	// TextToImage 文本生成图像 (V1)
	TextToImage(ctx context.Context, provider *entities.Provider, request *StabilityTextToImageRequest) (*StabilityImageResponse, error)

	// 图片生成接口
	GenerateSD2(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error)
	GenerateSD3(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error)
	GenerateSD3Ultra(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error)
	GenerateSD35Large(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error)
	GenerateSD35Medium(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error)

	// 图生图接口
	ImageToImageSD3(ctx context.Context, provider *entities.Provider, request *StabilityImageToImageRequest) (*StabilityImageResponse, error)
	ImageToImageSD35Large(ctx context.Context, provider *entities.Provider, request *StabilityImageToImageRequest) (*StabilityImageResponse, error)
	ImageToImageSD35Medium(ctx context.Context, provider *entities.Provider, request *StabilityImageToImageRequest) (*StabilityImageResponse, error)

	// 图片处理接口
	FastUpscale(ctx context.Context, provider *entities.Provider, request *StabilityUpscaleRequest) (*StabilityImageResponse, error)
	CreativeUpscale(ctx context.Context, provider *entities.Provider, request *StabilityUpscaleRequest) (*StabilityImageResponse, error)
	ConservativeUpscale(ctx context.Context, provider *entities.Provider, request *StabilityUpscaleRequest) (*StabilityImageResponse, error)
	FetchCreativeUpscale(ctx context.Context, provider *entities.Provider, requestID string) (*StabilityImageResponse, error)

	// 图片编辑接口
	Erase(ctx context.Context, provider *entities.Provider, request *StabilityEraseRequest) (*StabilityImageResponse, error)
	Inpaint(ctx context.Context, provider *entities.Provider, request *StabilityInpaintRequest) (*StabilityImageResponse, error)
	Outpaint(ctx context.Context, provider *entities.Provider, request *StabilityOutpaintRequest) (*StabilityImageResponse, error)
	SearchAndReplace(ctx context.Context, provider *entities.Provider, request *StabilitySearchReplaceRequest) (*StabilityImageResponse, error)
	SearchAndRecolor(ctx context.Context, provider *entities.Provider, request *StabilitySearchRecolorRequest) (*StabilityImageResponse, error)
	RemoveBackground(ctx context.Context, provider *entities.Provider, request *StabilityRemoveBgRequest) (*StabilityImageResponse, error)

	// 风格和结构接口
	Sketch(ctx context.Context, provider *entities.Provider, request *StabilitySketchRequest) (*StabilityImageResponse, error)
	Structure(ctx context.Context, provider *entities.Provider, request *StabilityStructureRequest) (*StabilityImageResponse, error)
	Style(ctx context.Context, provider *entities.Provider, request *StabilityStyleRequest) (*StabilityImageResponse, error)
	StyleTransfer(ctx context.Context, provider *entities.Provider, request *StabilityStyleTransferRequest) (*StabilityImageResponse, error)
	ReplaceBackground(ctx context.Context, provider *entities.Provider, request *StabilityReplaceBgRequest) (*StabilityImageResponse, error)
}

// StabilityTextToImageRequest Stability.ai文本生成图像请求
type StabilityTextToImageRequest struct {
	TextPrompts        []StabilityTextPrompt `json:"text_prompts"`
	Height             int                   `json:"height,omitempty"`
	Width              int                   `json:"width,omitempty"`
	CfgScale           float64               `json:"cfg_scale,omitempty"`
	ClipGuidancePreset string                `json:"clip_guidance_preset,omitempty"`
	Sampler            string                `json:"sampler,omitempty"`
	Samples            int                   `json:"samples,omitempty"`
	Seed               int64                 `json:"seed,omitempty"`
	Steps              int                   `json:"steps,omitempty"`
	StylePreset        string                `json:"style_preset,omitempty"`
}

// StabilityTextPrompt 文本提示
type StabilityTextPrompt struct {
	Text   string  `json:"text"`
	Weight float64 `json:"weight,omitempty"`
}

// StabilityImageResponse Stability.ai图像响应
type StabilityImageResponse struct {
	Artifacts []StabilityArtifact `json:"artifacts"`
}

// StabilityArtifact 图像工件
type StabilityArtifact struct {
	Base64       string `json:"base64"`
	Seed         int64  `json:"seed"`
	FinishReason string `json:"finishReason"`
}

// StabilityGenerateRequest 通用图片生成请求
type StabilityGenerateRequest struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	AspectRatio    string `json:"aspect_ratio,omitempty"`
	Seed           int64  `json:"seed,omitempty"`
	OutputFormat   string `json:"output_format,omitempty"`
	Model          string `json:"model,omitempty"`
	Mode           string `json:"mode,omitempty"`
}

// StabilityImageToImageRequest 图生图请求
type StabilityImageToImageRequest struct {
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt,omitempty"`
	Image          string  `json:"image"` // base64 encoded
	Strength       float64 `json:"strength,omitempty"`
	Seed           int64   `json:"seed,omitempty"`
	OutputFormat   string  `json:"output_format,omitempty"`
	Mode           string  `json:"mode,omitempty"`
}

// StabilityUpscaleRequest 图片放大请求
type StabilityUpscaleRequest struct {
	Image        string `json:"image"` // base64 encoded
	Prompt       string `json:"prompt,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
	Creativity   string `json:"creativity,omitempty"`
}

// StabilityEraseRequest 物体消除请求
type StabilityEraseRequest struct {
	Image        string `json:"image"` // base64 encoded
	Mask         string `json:"mask"`  // base64 encoded
	OutputFormat string `json:"output_format,omitempty"`
}

// StabilityInpaintRequest 图片修改请求
type StabilityInpaintRequest struct {
	Image        string `json:"image"` // base64 encoded
	Mask         string `json:"mask"`  // base64 encoded
	Prompt       string `json:"prompt"`
	OutputFormat string `json:"output_format,omitempty"`
	Seed         int64  `json:"seed,omitempty"`
}

// StabilityOutpaintRequest 图片扩展请求
type StabilityOutpaintRequest struct {
	Image        string `json:"image"` // base64 encoded
	Prompt       string `json:"prompt,omitempty"`
	Left         int    `json:"left,omitempty"`
	Right        int    `json:"right,omitempty"`
	Up           int    `json:"up,omitempty"`
	Down         int    `json:"down,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
	Seed         int64  `json:"seed,omitempty"`
}

// StabilitySearchReplaceRequest 内容替换请求
type StabilitySearchReplaceRequest struct {
	Image        string `json:"image"` // base64 encoded
	Prompt       string `json:"prompt"`
	SearchPrompt string `json:"search_prompt"`
	OutputFormat string `json:"output_format,omitempty"`
	Seed         int64  `json:"seed,omitempty"`
}

// StabilitySearchRecolorRequest 内容重着色请求
type StabilitySearchRecolorRequest struct {
	Image        string `json:"image"` // base64 encoded
	Prompt       string `json:"prompt"`
	SelectPrompt string `json:"select_prompt"`
	OutputFormat string `json:"output_format,omitempty"`
	Seed         int64  `json:"seed,omitempty"`
}

// StabilityRemoveBgRequest 背景消除请求
type StabilityRemoveBgRequest struct {
	Image        string `json:"image"` // base64 encoded
	OutputFormat string `json:"output_format,omitempty"`
}

// StabilitySketchRequest 草图转图片请求
type StabilitySketchRequest struct {
	Image           string  `json:"image"` // base64 encoded
	Prompt          string  `json:"prompt"`
	ControlStrength float64 `json:"control_strength,omitempty"`
	OutputFormat    string  `json:"output_format,omitempty"`
	Seed            int64   `json:"seed,omitempty"`
}

// StabilityStructureRequest 以图生图请求
type StabilityStructureRequest struct {
	Image           string  `json:"image"` // base64 encoded
	Prompt          string  `json:"prompt"`
	ControlStrength float64 `json:"control_strength,omitempty"`
	OutputFormat    string  `json:"output_format,omitempty"`
	Seed            int64   `json:"seed,omitempty"`
}

// StabilityStyleRequest 风格一致性请求
type StabilityStyleRequest struct {
	Image            string  `json:"image"` // base64 encoded
	Prompt           string  `json:"prompt"`
	FidelityStrength float64 `json:"fidelity_strength,omitempty"`
	OutputFormat     string  `json:"output_format,omitempty"`
	Seed             int64   `json:"seed,omitempty"`
}

// StabilityStyleTransferRequest 风格迁移请求
type StabilityStyleTransferRequest struct {
	Image        string `json:"image"`       // base64 encoded
	StyleImage   string `json:"style_image"` // base64 encoded
	Prompt       string `json:"prompt,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
	Seed         int64  `json:"seed,omitempty"`
}

// StabilityReplaceBgRequest 更换背景请求
type StabilityReplaceBgRequest struct {
	Image        string `json:"image"` // base64 encoded
	Prompt       string `json:"prompt"`
	OutputFormat string `json:"output_format,omitempty"`
	Seed         int64  `json:"seed,omitempty"`
}

// stabilityClientImpl Stability.ai客户端实现
type stabilityClientImpl struct {
	httpClient *http.Client
}

// NewStabilityClient 创建Stability.ai客户端
func NewStabilityClient(httpClient *http.Client) StabilityClient {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &stabilityClientImpl{
		httpClient: httpClient,
	}
}

// TextToImage 文本生成图像
func (c *stabilityClientImpl) TextToImage(ctx context.Context, provider *entities.Provider, request *StabilityTextToImageRequest) (*StabilityImageResponse, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/v1/generation/stable-diffusion-xl-1024-v1-0/text-to-image", provider.BaseURL)

	// 序列化请求体
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// 设置认证头
	if provider.APIKeyEncrypted != nil {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted))
	}

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 解析响应
	var response StabilityImageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// GenerateSD2 SD2图片生成
func (c *stabilityClientImpl) GenerateSD2(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/generate/sd", provider.BaseURL)
	return c.sendGenerateRequest(ctx, provider, url, request)
}

// GenerateSD3 SD3图片生成
func (c *stabilityClientImpl) GenerateSD3(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/generate/sd3", provider.BaseURL)
	return c.sendGenerateRequest(ctx, provider, url, request)
}

// GenerateSD3Ultra SD3 Ultra图片生成
func (c *stabilityClientImpl) GenerateSD3Ultra(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/generate/ultra", provider.BaseURL)
	return c.sendGenerateRequest(ctx, provider, url, request)
}

// GenerateSD35Large SD3.5 Large图片生成
func (c *stabilityClientImpl) GenerateSD35Large(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/generate/sd3-large", provider.BaseURL)
	return c.sendGenerateRequest(ctx, provider, url, request)
}

// GenerateSD35Medium SD3.5 Medium图片生成
func (c *stabilityClientImpl) GenerateSD35Medium(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/generate/sd3-medium", provider.BaseURL)
	return c.sendGenerateRequest(ctx, provider, url, request)
}

// sendGenerateRequest 发送通用生成请求
func (c *stabilityClientImpl) sendGenerateRequest(ctx context.Context, provider *entities.Provider, url string, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	if provider.APIKeyEncrypted != nil {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted))
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var response StabilityImageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// ImageToImageSD3 SD3图生图
func (c *stabilityClientImpl) ImageToImageSD3(ctx context.Context, provider *entities.Provider, request *StabilityImageToImageRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/control/sd3", provider.BaseURL)
	return c.sendImageToImageRequest(ctx, provider, url, request)
}

// ImageToImageSD35Large SD3.5 Large图生图
func (c *stabilityClientImpl) ImageToImageSD35Large(ctx context.Context, provider *entities.Provider, request *StabilityImageToImageRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/control/sd3-large", provider.BaseURL)
	return c.sendImageToImageRequest(ctx, provider, url, request)
}

// ImageToImageSD35Medium SD3.5 Medium图生图
func (c *stabilityClientImpl) ImageToImageSD35Medium(ctx context.Context, provider *entities.Provider, request *StabilityImageToImageRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/control/sd3-medium", provider.BaseURL)
	return c.sendImageToImageRequest(ctx, provider, url, request)
}

// sendImageToImageRequest 发送图生图请求
func (c *stabilityClientImpl) sendImageToImageRequest(ctx context.Context, provider *entities.Provider, url string, request *StabilityImageToImageRequest) (*StabilityImageResponse, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	if provider.APIKeyEncrypted != nil {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted))
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var response StabilityImageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// FastUpscale 快速图片放大
func (c *stabilityClientImpl) FastUpscale(ctx context.Context, provider *entities.Provider, request *StabilityUpscaleRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/upscale/fast", provider.BaseURL)
	return c.sendUpscaleRequest(ctx, provider, url, request)
}

// CreativeUpscale 创意图片放大
func (c *stabilityClientImpl) CreativeUpscale(ctx context.Context, provider *entities.Provider, request *StabilityUpscaleRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/upscale/creative", provider.BaseURL)
	return c.sendUpscaleRequest(ctx, provider, url, request)
}

// ConservativeUpscale 保守图片放大
func (c *stabilityClientImpl) ConservativeUpscale(ctx context.Context, provider *entities.Provider, request *StabilityUpscaleRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/upscale/conservative", provider.BaseURL)
	return c.sendUpscaleRequest(ctx, provider, url, request)
}

// FetchCreativeUpscale 获取创意放大结果
func (c *stabilityClientImpl) FetchCreativeUpscale(ctx context.Context, provider *entities.Provider, requestID string) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/upscale/creative/result/%s", provider.BaseURL, requestID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")

	if provider.APIKeyEncrypted != nil {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted))
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var response StabilityImageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// sendUpscaleRequest 发送放大请求
func (c *stabilityClientImpl) sendUpscaleRequest(ctx context.Context, provider *entities.Provider, url string, request *StabilityUpscaleRequest) (*StabilityImageResponse, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	if provider.APIKeyEncrypted != nil {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted))
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var response StabilityImageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// Erase 物体消除
func (c *stabilityClientImpl) Erase(ctx context.Context, provider *entities.Provider, request *StabilityEraseRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/edit/erase", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// Inpaint 图片修改
func (c *stabilityClientImpl) Inpaint(ctx context.Context, provider *entities.Provider, request *StabilityInpaintRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/edit/inpaint", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// Outpaint 图片扩展
func (c *stabilityClientImpl) Outpaint(ctx context.Context, provider *entities.Provider, request *StabilityOutpaintRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/edit/outpaint", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// SearchAndReplace 内容替换
func (c *stabilityClientImpl) SearchAndReplace(ctx context.Context, provider *entities.Provider, request *StabilitySearchReplaceRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/edit/search-and-replace", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// SearchAndRecolor 内容重着色
func (c *stabilityClientImpl) SearchAndRecolor(ctx context.Context, provider *entities.Provider, request *StabilitySearchRecolorRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/edit/search-and-recolor", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// RemoveBackground 背景消除
func (c *stabilityClientImpl) RemoveBackground(ctx context.Context, provider *entities.Provider, request *StabilityRemoveBgRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/edit/remove-background", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// sendEditRequest 发送编辑请求的通用方法
func (c *stabilityClientImpl) sendEditRequest(ctx context.Context, provider *entities.Provider, url string, request interface{}) (*StabilityImageResponse, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	if provider.APIKeyEncrypted != nil {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted))
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var response StabilityImageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// Sketch 草图转图片
func (c *stabilityClientImpl) Sketch(ctx context.Context, provider *entities.Provider, request *StabilitySketchRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/control/sketch", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// Structure 以图生图
func (c *stabilityClientImpl) Structure(ctx context.Context, provider *entities.Provider, request *StabilityStructureRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/control/structure", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// Style 风格一致性
func (c *stabilityClientImpl) Style(ctx context.Context, provider *entities.Provider, request *StabilityStyleRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/control/style", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// StyleTransfer 风格迁移
func (c *stabilityClientImpl) StyleTransfer(ctx context.Context, provider *entities.Provider, request *StabilityStyleTransferRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/edit/style-transfer", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// ReplaceBackground 更换背景
func (c *stabilityClientImpl) ReplaceBackground(ctx context.Context, provider *entities.Provider, request *StabilityReplaceBgRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/v2beta/stable-image/edit/replace-background", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}
