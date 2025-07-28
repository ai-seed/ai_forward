package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"
)

// StabilityHandler Stability.ai处理器
type StabilityHandler struct {
	stabilityService services.StabilityService
	logger           logger.Logger
}

// NewStabilityHandler 创建Stability.ai处理器
func NewStabilityHandler(stabilityService services.StabilityService, logger logger.Logger) *StabilityHandler {
	return &StabilityHandler{
		stabilityService: stabilityService,
		logger:           logger,
	}
}

// StabilityTextToImageRequest 兼容302AI格式的请求结构
type StabilityTextToImageRequest struct {
	TextPrompts        []StabilityTextPrompt `json:"text_prompts" binding:"required"`
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
	Text   string  `json:"text" binding:"required"`
	Weight float64 `json:"weight,omitempty"`
}

// StabilityResponse 响应结构
type StabilityResponse struct {
	Artifacts []StabilityArtifact `json:"artifacts"`
}

// StabilityArtifact 图像工件
type StabilityArtifact struct {
	Base64       string `json:"base64"`
	Seed         int64  `json:"seed"`
	FinishReason string `json:"finishReason"`
}

// TextToImage 文本生成图像端点
// @Summary 文本生成图像
// @Description 使用Stability.ai API生成图像
// @Tags Stability
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body StabilityTextToImageRequest true "图像生成请求"
// @Success 200 {object} StabilityResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /sd/v1/generation/stable-diffusion-xl-1024-v1-0/text-to-image [post]
func (h *StabilityHandler) TextToImage(c *gin.Context) {
	// 获取用户ID和API密钥ID
	userID := c.GetInt64("user_id")
	apiKeyID := c.GetInt64("api_key_id")

	// 解析请求
	var req StabilityTextToImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("Invalid text-to-image request")

		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request parameters: " + err.Error(),
			},
		})
		return
	}

	// 设置默认值
	if req.Height == 0 {
		req.Height = 1024
	}
	if req.Width == 0 {
		req.Width = 1024
	}
	if req.Samples == 0 {
		req.Samples = 1
	}

	// 转换为客户端请求格式
	clientRequest := &clients.StabilityTextToImageRequest{
		TextPrompts:        make([]clients.StabilityTextPrompt, len(req.TextPrompts)),
		Height:             req.Height,
		Width:              req.Width,
		CfgScale:           req.CfgScale,
		ClipGuidancePreset: req.ClipGuidancePreset,
		Sampler:            req.Sampler,
		Samples:            req.Samples,
		Seed:               req.Seed,
		Steps:              req.Steps,
		StylePreset:        req.StylePreset,
	}

	// 转换文本提示
	for i, prompt := range req.TextPrompts {
		clientRequest.TextPrompts[i] = clients.StabilityTextPrompt{
			Text:   prompt.Text,
			Weight: prompt.Weight,
		}
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":      userID,
		"api_key_id":   apiKeyID,
		"text_prompts": req.TextPrompts,
		"height":       req.Height,
		"width":        req.Width,
		"samples":      req.Samples,
	}).Info("Processing text-to-image request")

	// 调用服务
	response, err := h.stabilityService.TextToImage(c.Request.Context(), userID, apiKeyID, clientRequest)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("Failed to generate image")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "GENERATION_FAILED",
				"message": "Failed to generate image: " + err.Error(),
			},
		})
		return
	}

	// 转换响应格式
	stabilityResponse := StabilityResponse{
		Artifacts: make([]StabilityArtifact, len(response.Artifacts)),
	}

	for i, artifact := range response.Artifacts {
		stabilityResponse.Artifacts[i] = StabilityArtifact{
			Base64:       artifact.Base64,
			Seed:         artifact.Seed,
			FinishReason: artifact.FinishReason,
		}
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":         userID,
		"api_key_id":      apiKeyID,
		"artifacts_count": len(stabilityResponse.Artifacts),
	}).Info("Successfully generated image")

	c.JSON(http.StatusOK, stabilityResponse)
}

// GenerateSD2 SD2图片生成
func (h *StabilityHandler) GenerateSD2(c *gin.Context) {
	h.handleGenerateRequest(c, "sd2", func(ctx context.Context, userID, apiKeyID int64, req *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
		return h.stabilityService.GenerateSD2(ctx, userID, apiKeyID, req)
	})
}

// GenerateSD3 SD3图片生成
func (h *StabilityHandler) GenerateSD3(c *gin.Context) {
	h.handleGenerateRequest(c, "sd3", func(ctx context.Context, userID, apiKeyID int64, req *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
		return h.stabilityService.GenerateSD3(ctx, userID, apiKeyID, req)
	})
}

// GenerateSD3Ultra SD3 Ultra图片生成
func (h *StabilityHandler) GenerateSD3Ultra(c *gin.Context) {
	h.handleGenerateRequest(c, "sd3-ultra", func(ctx context.Context, userID, apiKeyID int64, req *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
		return h.stabilityService.GenerateSD3Ultra(ctx, userID, apiKeyID, req)
	})
}

// GenerateSD35Large SD3.5 Large图片生成
func (h *StabilityHandler) GenerateSD35Large(c *gin.Context) {
	h.handleGenerateRequest(c, "sd3.5-large", func(ctx context.Context, userID, apiKeyID int64, req *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
		return h.stabilityService.GenerateSD35Large(ctx, userID, apiKeyID, req)
	})
}

// GenerateSD35Medium SD3.5 Medium图片生成
func (h *StabilityHandler) GenerateSD35Medium(c *gin.Context) {
	h.handleGenerateRequest(c, "sd3.5-medium", func(ctx context.Context, userID, apiKeyID int64, req *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error) {
		return h.stabilityService.GenerateSD35Medium(ctx, userID, apiKeyID, req)
	})
}

// handleGenerateRequest 处理通用生成请求
func (h *StabilityHandler) handleGenerateRequest(c *gin.Context, modelName string, serviceFunc func(context.Context, int64, int64, *clients.StabilityGenerateRequest) (*clients.StabilityImageResponse, error)) {
	userID := c.GetInt64("user_id")
	apiKeyID := c.GetInt64("api_key_id")

	var req clients.StabilityGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"model":      modelName,
		}).Error("Invalid generate request")

		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request parameters: " + err.Error(),
			},
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"api_key_id": apiKeyID,
		"model":      modelName,
		"prompt":     req.Prompt,
	}).Info("Processing generate request")

	response, err := serviceFunc(c.Request.Context(), userID, apiKeyID, &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"model":      modelName,
		}).Error("Failed to generate image")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "GENERATION_FAILED",
				"message": "Failed to generate image: " + err.Error(),
			},
		})
		return
	}

	stabilityResponse := StabilityResponse{
		Artifacts: make([]StabilityArtifact, len(response.Artifacts)),
	}

	for i, artifact := range response.Artifacts {
		stabilityResponse.Artifacts[i] = StabilityArtifact{
			Base64:       artifact.Base64,
			Seed:         artifact.Seed,
			FinishReason: artifact.FinishReason,
		}
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":         userID,
		"api_key_id":      apiKeyID,
		"model":           modelName,
		"artifacts_count": len(stabilityResponse.Artifacts),
	}).Info("Successfully generated image")

	c.JSON(http.StatusOK, stabilityResponse)
}

// handleGenericRequest 处理通用请求的辅助方法
func (h *StabilityHandler) handleGenericRequest(c *gin.Context, requestType string, bindFunc func() (interface{}, error), serviceFunc func(context.Context, int64, int64, interface{}) (*clients.StabilityImageResponse, error)) {
	userID := c.GetInt64("user_id")
	apiKeyID := c.GetInt64("api_key_id")

	request, err := bindFunc()
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":        err.Error(),
			"user_id":      userID,
			"api_key_id":   apiKeyID,
			"request_type": requestType,
		}).Error("Invalid request")

		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request parameters: " + err.Error(),
			},
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":      userID,
		"api_key_id":   apiKeyID,
		"request_type": requestType,
	}).Info("Processing request")

	response, err := serviceFunc(c.Request.Context(), userID, apiKeyID, request)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":        err.Error(),
			"user_id":      userID,
			"api_key_id":   apiKeyID,
			"request_type": requestType,
		}).Error("Failed to process request")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "PROCESSING_FAILED",
				"message": "Failed to process request: " + err.Error(),
			},
		})
		return
	}

	stabilityResponse := StabilityResponse{
		Artifacts: make([]StabilityArtifact, len(response.Artifacts)),
	}

	for i, artifact := range response.Artifacts {
		stabilityResponse.Artifacts[i] = StabilityArtifact{
			Base64:       artifact.Base64,
			Seed:         artifact.Seed,
			FinishReason: artifact.FinishReason,
		}
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":         userID,
		"api_key_id":      apiKeyID,
		"request_type":    requestType,
		"artifacts_count": len(stabilityResponse.Artifacts),
	}).Info("Successfully processed request")

	c.JSON(http.StatusOK, stabilityResponse)
}

// 图生图处理器方法
func (h *StabilityHandler) ImageToImageSD3(c *gin.Context) {
	h.handleGenericRequest(c, "image-to-image-sd3",
		func() (interface{}, error) {
			var req clients.StabilityImageToImageRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.ImageToImageSD3(ctx, userID, apiKeyID, req.(*clients.StabilityImageToImageRequest))
		})
}

func (h *StabilityHandler) ImageToImageSD35Large(c *gin.Context) {
	h.handleGenericRequest(c, "image-to-image-sd3.5-large",
		func() (interface{}, error) {
			var req clients.StabilityImageToImageRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.ImageToImageSD35Large(ctx, userID, apiKeyID, req.(*clients.StabilityImageToImageRequest))
		})
}

func (h *StabilityHandler) ImageToImageSD35Medium(c *gin.Context) {
	h.handleGenericRequest(c, "image-to-image-sd3.5-medium",
		func() (interface{}, error) {
			var req clients.StabilityImageToImageRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.ImageToImageSD35Medium(ctx, userID, apiKeyID, req.(*clients.StabilityImageToImageRequest))
		})
}

// 图片放大处理器方法
func (h *StabilityHandler) FastUpscale(c *gin.Context) {
	h.handleGenericRequest(c, "fast-upscale",
		func() (interface{}, error) {
			var req clients.StabilityUpscaleRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.FastUpscale(ctx, userID, apiKeyID, req.(*clients.StabilityUpscaleRequest))
		})
}

func (h *StabilityHandler) CreativeUpscale(c *gin.Context) {
	h.handleGenericRequest(c, "creative-upscale",
		func() (interface{}, error) {
			var req clients.StabilityUpscaleRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.CreativeUpscale(ctx, userID, apiKeyID, req.(*clients.StabilityUpscaleRequest))
		})
}

func (h *StabilityHandler) ConservativeUpscale(c *gin.Context) {
	h.handleGenericRequest(c, "conservative-upscale",
		func() (interface{}, error) {
			var req clients.StabilityUpscaleRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.ConservativeUpscale(ctx, userID, apiKeyID, req.(*clients.StabilityUpscaleRequest))
		})
}

func (h *StabilityHandler) FetchCreativeUpscale(c *gin.Context) {
	userID := c.GetInt64("user_id")
	apiKeyID := c.GetInt64("api_key_id")
	requestID := c.Param("id")

	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Request ID is required",
			},
		})
		return
	}

	response, err := h.stabilityService.FetchCreativeUpscale(c.Request.Context(), userID, apiKeyID, requestID)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
			"request_id": requestID,
		}).Error("Failed to fetch creative upscale result")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "FETCH_FAILED",
				"message": "Failed to fetch result: " + err.Error(),
			},
		})
		return
	}

	stabilityResponse := StabilityResponse{
		Artifacts: make([]StabilityArtifact, len(response.Artifacts)),
	}

	for i, artifact := range response.Artifacts {
		stabilityResponse.Artifacts[i] = StabilityArtifact{
			Base64:       artifact.Base64,
			Seed:         artifact.Seed,
			FinishReason: artifact.FinishReason,
		}
	}

	c.JSON(http.StatusOK, stabilityResponse)
}

// 图片编辑处理器方法
func (h *StabilityHandler) Erase(c *gin.Context) {
	h.handleGenericRequest(c, "erase",
		func() (interface{}, error) {
			var req clients.StabilityEraseRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.Erase(ctx, userID, apiKeyID, req.(*clients.StabilityEraseRequest))
		})
}

func (h *StabilityHandler) Inpaint(c *gin.Context) {
	h.handleGenericRequest(c, "inpaint",
		func() (interface{}, error) {
			var req clients.StabilityInpaintRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.Inpaint(ctx, userID, apiKeyID, req.(*clients.StabilityInpaintRequest))
		})
}

func (h *StabilityHandler) Outpaint(c *gin.Context) {
	h.handleGenericRequest(c, "outpaint",
		func() (interface{}, error) {
			var req clients.StabilityOutpaintRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.Outpaint(ctx, userID, apiKeyID, req.(*clients.StabilityOutpaintRequest))
		})
}

func (h *StabilityHandler) SearchAndReplace(c *gin.Context) {
	h.handleGenericRequest(c, "search-and-replace",
		func() (interface{}, error) {
			var req clients.StabilitySearchReplaceRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.SearchAndReplace(ctx, userID, apiKeyID, req.(*clients.StabilitySearchReplaceRequest))
		})
}

func (h *StabilityHandler) SearchAndRecolor(c *gin.Context) {
	h.handleGenericRequest(c, "search-and-recolor",
		func() (interface{}, error) {
			var req clients.StabilitySearchRecolorRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.SearchAndRecolor(ctx, userID, apiKeyID, req.(*clients.StabilitySearchRecolorRequest))
		})
}

func (h *StabilityHandler) RemoveBackground(c *gin.Context) {
	h.handleGenericRequest(c, "remove-background",
		func() (interface{}, error) {
			var req clients.StabilityRemoveBgRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.RemoveBackground(ctx, userID, apiKeyID, req.(*clients.StabilityRemoveBgRequest))
		})
}

// 风格和结构处理器方法
func (h *StabilityHandler) Sketch(c *gin.Context) {
	h.handleGenericRequest(c, "sketch",
		func() (interface{}, error) {
			var req clients.StabilitySketchRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.Sketch(ctx, userID, apiKeyID, req.(*clients.StabilitySketchRequest))
		})
}

func (h *StabilityHandler) Structure(c *gin.Context) {
	h.handleGenericRequest(c, "structure",
		func() (interface{}, error) {
			var req clients.StabilityStructureRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.Structure(ctx, userID, apiKeyID, req.(*clients.StabilityStructureRequest))
		})
}

func (h *StabilityHandler) Style(c *gin.Context) {
	h.handleGenericRequest(c, "style",
		func() (interface{}, error) {
			var req clients.StabilityStyleRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.Style(ctx, userID, apiKeyID, req.(*clients.StabilityStyleRequest))
		})
}

func (h *StabilityHandler) StyleTransfer(c *gin.Context) {
	h.handleGenericRequest(c, "style-transfer",
		func() (interface{}, error) {
			var req clients.StabilityStyleTransferRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.StyleTransfer(ctx, userID, apiKeyID, req.(*clients.StabilityStyleTransferRequest))
		})
}

func (h *StabilityHandler) ReplaceBackground(c *gin.Context) {
	h.handleGenericRequest(c, "replace-background",
		func() (interface{}, error) {
			var req clients.StabilityReplaceBgRequest
			err := c.ShouldBindJSON(&req)
			return &req, err
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.StabilityImageResponse, error) {
			return h.stabilityService.ReplaceBackground(ctx, userID, apiKeyID, req.(*clients.StabilityReplaceBgRequest))
		})
}
