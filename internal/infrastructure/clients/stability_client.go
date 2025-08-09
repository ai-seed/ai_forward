package clients

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	Artifacts    []StabilityArtifact `json:"artifacts,omitempty"`     // 用于文本生成图像等接口
	FinishReason string              `json:"finish_reason,omitempty"` // 用于背景移除等编辑接口
	Image        string              `json:"image,omitempty"`         // 用于背景移除等编辑接口（base64格式）
	Seed         int64               `json:"seed,omitempty"`          // 用于背景移除等编辑接口
	Cost         float64             `json:"cost,omitempty"`          // 成本信息
	ProviderID   int64               `json:"provider_id,omitempty"`   // 提供商ID
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
	url := fmt.Sprintf("%s/sd/v1/generation/stable-diffusion-xl-1024-v1-0/text-to-image", provider.BaseURL)

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
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/generate/sd", provider.BaseURL)
	return c.sendGenerateRequest(ctx, provider, url, request)
}

// GenerateSD3 SD3图片生成
func (c *stabilityClientImpl) GenerateSD3(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/generate/sd3", provider.BaseURL)
	return c.sendGenerateRequest(ctx, provider, url, request)
}

// GenerateSD3Ultra SD3 Ultra图片生成
func (c *stabilityClientImpl) GenerateSD3Ultra(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/generate/ultra", provider.BaseURL)
	return c.sendGenerateRequest(ctx, provider, url, request)
}

// GenerateSD35Large SD3.5 Large图片生成
func (c *stabilityClientImpl) GenerateSD35Large(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/generate/sd3-large", provider.BaseURL)
	return c.sendGenerateRequest(ctx, provider, url, request)
}

// GenerateSD35Medium SD3.5 Medium图片生成
func (c *stabilityClientImpl) GenerateSD35Medium(ctx context.Context, provider *entities.Provider, request *StabilityGenerateRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/generate/sd3-medium", provider.BaseURL)
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
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/control/sd3", provider.BaseURL)
	return c.sendImageToImageRequest(ctx, provider, url, request)
}

// ImageToImageSD35Large SD3.5 Large图生图
func (c *stabilityClientImpl) ImageToImageSD35Large(ctx context.Context, provider *entities.Provider, request *StabilityImageToImageRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/control/sd3-large", provider.BaseURL)
	return c.sendImageToImageRequest(ctx, provider, url, request)
}

// ImageToImageSD35Medium SD3.5 Medium图生图
func (c *stabilityClientImpl) ImageToImageSD35Medium(ctx context.Context, provider *entities.Provider, request *StabilityImageToImageRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/control/sd3-medium", provider.BaseURL)
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
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/upscale/fast", provider.BaseURL)
	return c.sendUpscaleRequest(ctx, provider, url, request)
}

// CreativeUpscale 创意图片放大
func (c *stabilityClientImpl) CreativeUpscale(ctx context.Context, provider *entities.Provider, request *StabilityUpscaleRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/upscale/creative", provider.BaseURL)
	return c.sendUpscaleRequest(ctx, provider, url, request)
}

// ConservativeUpscale 保守图片放大
func (c *stabilityClientImpl) ConservativeUpscale(ctx context.Context, provider *entities.Provider, request *StabilityUpscaleRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/upscale/conservative", provider.BaseURL)
	return c.sendUpscaleRequest(ctx, provider, url, request)
}

// FetchCreativeUpscale 获取创意放大结果
func (c *stabilityClientImpl) FetchCreativeUpscale(ctx context.Context, provider *entities.Provider, requestID string) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/upscale/creative/result/%s", provider.BaseURL, requestID)

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
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/edit/erase", provider.BaseURL)
	return c.sendMultipartRequest(ctx, provider, url, request)
}

// Inpaint 图片修改
func (c *stabilityClientImpl) Inpaint(ctx context.Context, provider *entities.Provider, request *StabilityInpaintRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/edit/inpaint", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// Outpaint 图片扩展
func (c *stabilityClientImpl) Outpaint(ctx context.Context, provider *entities.Provider, request *StabilityOutpaintRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/edit/outpaint", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// SearchAndReplace 内容替换
func (c *stabilityClientImpl) SearchAndReplace(ctx context.Context, provider *entities.Provider, request *StabilitySearchReplaceRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/edit/search-and-replace", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// SearchAndRecolor 内容重着色
func (c *stabilityClientImpl) SearchAndRecolor(ctx context.Context, provider *entities.Provider, request *StabilitySearchRecolorRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/edit/search-and-recolor", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// RemoveBackground 背景消除
func (c *stabilityClientImpl) RemoveBackground(ctx context.Context, provider *entities.Provider, request *StabilityRemoveBgRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/edit/remove-background", provider.BaseURL)
	return c.sendMultipartRequest(ctx, provider, url, request)
}

// sendEditRequest 发送编辑请求的通用方法
func (c *stabilityClientImpl) sendEditRequest(ctx context.Context, provider *entities.Provider, url string, request interface{}) (*StabilityImageResponse, error) {
	fmt.Printf("[DEBUG] 准备发送Stability.ai编辑请求: provider_id=%d, provider_name=%s, url=%s, has_api_key=%t\n",
		provider.ID, provider.Name, url, provider.APIKeyEncrypted != nil)

	requestBody, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("[ERROR] 序列化请求失败: %v\n", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		fmt.Printf("[ERROR] 创建HTTP请求失败: %v, url=%s\n", err, url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

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

	// 输出完整的响应头信息
	fmt.Printf("[INFO] 响应头信息:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// 检查响应内容类型
	contentType := resp.Header.Get("Content-Type")
	fmt.Printf("[INFO] 响应Content-Type: %s, response_length=%d\n", contentType, len(responseBody))

	// 打印响应的详细信息
	if len(responseBody) > 0 {
		// 打印前100字节
		preview := responseBody
		if len(preview) > 100 {
			preview = preview[:100]
		}
		fmt.Printf("[DEBUG] 响应前100字节: %v\n", preview)
		fmt.Printf("[DEBUG] 响应前100字节(字符串): %q\n", string(preview))

		// 如果响应看起来像JSON，打印完整内容
		if len(responseBody) < 10000 && (responseBody[0] == '{' || responseBody[0] == '[') {
			fmt.Printf("[DEBUG] 完整JSON响应: %s\n", string(responseBody))
		}

		// 如果是图片，打印图片信息
		if responseBody[0] == 0x89 && len(responseBody) > 3 {
			fmt.Printf("[DEBUG] PNG图片信息: 文件大小=%d字节\n", len(responseBody))
		}
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[ERROR] API请求失败: status_code=%d, url=%s\n",
			resp.StatusCode, url)
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// 将图片转换为base64放在Image字段中传回给用户（与背景消除接口一致）
	fmt.Printf("[INFO] 将图片转换为base64，原始长度: %d字节\n", len(responseBody))

	base64Image := base64.StdEncoding.EncodeToString(responseBody)
	fmt.Printf("[INFO] base64转换完成，长度: %d字符\n", len(base64Image))

	return &StabilityImageResponse{
		FinishReason: "SUCCESS",
		Image:        base64Image,
		Seed:         0,
	}, nil
}

// sendMultipartRequest 发送multipart/form-data请求的方法
func (c *stabilityClientImpl) sendMultipartRequest(ctx context.Context, provider *entities.Provider, url string, request interface{}) (*StabilityImageResponse, error) {
	fmt.Printf("[DEBUG] 准备发送Stability.ai multipart请求: provider_id=%d, provider_name=%s, url=%s, has_api_key=%t\n",
		provider.ID, provider.Name, url, provider.APIKeyEncrypted != nil)

	// 创建multipart/form-data请求
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 根据请求类型添加字段
	switch req := request.(type) {
	case *StabilityRemoveBgRequest:
		// 添加图片文件 - Image字段包含base64编码的图片数据
		if req.Image != "" {
			// 解码base64图片数据
			imageData, err := base64.StdEncoding.DecodeString(req.Image)
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

		// 添加其他字段
		if req.OutputFormat != "" {
			writer.WriteField("output_format", req.OutputFormat)
		}

	case *StabilityEraseRequest:
		// 添加图片文件 - Image字段包含base64编码的图片数据
		if req.Image != "" {
			// 解码base64图片数据
			imageData, err := base64.StdEncoding.DecodeString(req.Image)
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

		// 添加遮罩文件 - Mask字段包含base64编码的遮罩数据
		if req.Mask != "" {
			// 解码base64遮罩数据
			maskData, err := base64.StdEncoding.DecodeString(req.Mask)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 mask: %w", err)
			}

			part, err := writer.CreateFormFile("mask", "mask.png")
			if err != nil {
				return nil, fmt.Errorf("failed to create form file for mask: %w", err)
			}

			// 写入解码后的遮罩数据
			_, err = part.Write(maskData)
			if err != nil {
				return nil, fmt.Errorf("failed to write mask data: %w", err)
			}
		}

		// 添加其他字段
		if req.OutputFormat != "" {
			writer.WriteField("output_format", req.OutputFormat)
		}

	default:
		writer.Close()
		return nil, fmt.Errorf("unsupported request type for multipart: %T", request)
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
	httpReq.Header.Set("Accept", "image/*")

	fmt.Printf("[DEBUG] 请求头信息:\n")
	fmt.Printf("  Content-Type: %s\n", writer.FormDataContentType())
	fmt.Printf("  Accept: image/*\n")

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

	// 输出完整的响应头信息
	fmt.Printf("[INFO] 响应头信息:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[ERROR] API请求失败: status_code=%d, response_body=%s, url=%s\n",
			resp.StatusCode, string(responseBody), url)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 检查响应内容类型，判断是JSON还是图片文件
	contentType := resp.Header.Get("Content-Type")
	fmt.Printf("[INFO] 响应Content-Type: %s, response_length=%d\n", contentType, len(responseBody))

	// 打印响应的详细信息
	if len(responseBody) > 0 {
		// 打印前100字节
		preview := responseBody
		if len(preview) > 100 {
			preview = preview[:100]
		}
		fmt.Printf("[DEBUG] 响应前100字节: %v\n", preview)
		fmt.Printf("[DEBUG] 响应前100字节(字符串): %q\n", string(preview))

		// 如果响应看起来像JSON，打印完整内容
		if len(responseBody) < 10000 && (responseBody[0] == '{' || responseBody[0] == '[') {
			fmt.Printf("[DEBUG] 完整JSON响应: %s\n", string(responseBody))
		}

		// 如果是图片，打印图片信息
		if responseBody[0] == 0x89 && len(responseBody) > 3 {
			fmt.Printf("[DEBUG] PNG图片信息: 文件大小=%d字节\n", len(responseBody))
		}
	}

	// 将图片转换为base64放在Image字段中传回给用户
	fmt.Printf("[INFO] 将图片转换为base64，原始长度: %d字节\n", len(responseBody))

	base64Image := base64.StdEncoding.EncodeToString(responseBody)
	fmt.Printf("[INFO] base64转换完成，长度: %d字符\n", len(base64Image))

	return &StabilityImageResponse{
		FinishReason: "SUCCESS",
		Image:        base64Image,
		Seed:         0,
	}, nil
}

// Sketch 草图转图片
func (c *stabilityClientImpl) Sketch(ctx context.Context, provider *entities.Provider, request *StabilitySketchRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/control/sketch", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// Structure 以图生图
func (c *stabilityClientImpl) Structure(ctx context.Context, provider *entities.Provider, request *StabilityStructureRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/control/structure", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// Style 风格一致性
func (c *stabilityClientImpl) Style(ctx context.Context, provider *entities.Provider, request *StabilityStyleRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/control/style", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// StyleTransfer 风格迁移
func (c *stabilityClientImpl) StyleTransfer(ctx context.Context, provider *entities.Provider, request *StabilityStyleTransferRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/edit/style-transfer", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}

// ReplaceBackground 更换背景
func (c *stabilityClientImpl) ReplaceBackground(ctx context.Context, provider *entities.Provider, request *StabilityReplaceBgRequest) (*StabilityImageResponse, error) {
	url := fmt.Sprintf("%s/sd/v2beta/stable-image/edit/replace-background", provider.BaseURL)
	return c.sendEditRequest(ctx, provider, url, request)
}
