package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	_ "ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// AI302Handler 302.AI处理器
type AI302Handler struct {
	ai302Service services.AI302Service
	logger       logger.Logger
}

// NewAI302Handler 创建302.AI处理器
func NewAI302Handler(ai302Service services.AI302Service, logger logger.Logger) *AI302Handler {
	return &AI302Handler{
		ai302Service: ai302Service,
		logger:       logger,
	}
}

// Upscale 图片放大
// @Summary 图片放大
// @Description 使用302.AI进行图片放大处理
// @Tags 302.AI
// @Accept multipart/form-data,json
// @Produce json
// @Security BearerAuth
// @Param image formData file true "要放大的图片文件"
// @Param scale formData int false "放大倍数，0-10，默认4"
// @Param face_enhance formData bool false "人脸增强，默认true"
// @Success 200 {object} clients.AI302UpscaleResponse "放大成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 429 {object} dto.Response "请求过于频繁"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /ai/upscale [post]
func (h *AI302Handler) Upscale(c *gin.Context) {
	h.handleGenericRequest(c, "upscale",
		func() (interface{}, error) {
			contentType := c.GetHeader("Content-Type")
			h.logger.WithFields(map[string]interface{}{
				"content_type": contentType,
			}).Info("Processing request")

			// 检查Content-Type来决定解析方式
			if strings.Contains(contentType, "multipart/form-data") {
				return h.parseMultipartRequest(c)
			} else {
				return h.parseJSONRequest(c)
			}
		},
		func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.AI302UpscaleResponse, error) {
			return h.ai302Service.Upscale(ctx, userID, apiKeyID, req.(*clients.AI302UpscaleRequest))
		})
}

// handleGenericRequest 通用请求处理器
func (h *AI302Handler) handleGenericRequest(
	c *gin.Context,
	requestType string,
	bindFunc func() (interface{}, error),
	serviceFunc func(ctx context.Context, userID, apiKeyID int64, req interface{}) (*clients.AI302UpscaleResponse, error),
) {
	// 获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "User not authenticated",
				"type":    "authentication_error",
				"code":    "user_not_found",
			},
		})
		return
	}

	apiKeyID, exists := c.Get("api_key_id")
	if !exists {
		h.logger.Error("API Key ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "API Key not found",
				"type":    "authentication_error",
				"code":    "api_key_not_found",
			},
		})
		return
	}

	// 绑定请求参数
	req, err := bindFunc()
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":        err.Error(),
			"request_type": requestType,
		}).Error("Failed to bind request")

		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request parameters",
				"type":    "invalid_request_error",
				"code":    "invalid_parameters",
				"details": err.Error(),
			},
		})
		return
	}

	// 调用服务
	response, err := serviceFunc(c.Request.Context(), userID.(int64), apiKeyID.(int64), req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":        err.Error(),
			"user_id":      userID,
			"api_key_id":   apiKeyID,
			"request_type": requestType,
		}).Error("Failed to process request")

		// 根据错误类型返回不同的状态码
		statusCode := http.StatusInternalServerError
		errorType := "internal_error"
		errorCode := "processing_failed"

		if err.Error() == "insufficient quota" {
			statusCode = http.StatusTooManyRequests
			errorType = "quota_exceeded"
			errorCode = "insufficient_quota"
		}

		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    errorType,
				"code":    errorCode,
			},
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":      userID,
		"api_key_id":   apiKeyID,
		"request_type": requestType,
		"status":       response.Status,
	}).Info("Successfully processed request")

	c.JSON(http.StatusOK, response)
}

// parseMultipartRequest 解析multipart form data请求
func (h *AI302Handler) parseMultipartRequest(c *gin.Context) (*clients.AI302UpscaleRequest, error) {
	// 获取上传的文件
	file, err := c.FormFile("image")
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to get form file")
		return nil, err
	}

	h.logger.WithFields(map[string]interface{}{
		"filename": file.Filename,
		"size":     file.Size,
	}).Info("Successfully got form file")

	// 打开文件
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// 读取文件内容
	imageData, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}

	// 获取可选参数
	scaleStr := c.PostForm("scale")
	faceEnhanceStr := c.PostForm("face_enhance")

	h.logger.WithFields(map[string]interface{}{
		"scale_str":        scaleStr,
		"face_enhance_str": faceEnhanceStr,
	}).Info("Parsing form parameters")

	scale := 4 // 默认值
	if scaleStr != "" {
		s, err := strconv.Atoi(scaleStr)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"scale_str": scaleStr,
				"error":     err.Error(),
			}).Error("Failed to parse scale parameter")
			return nil, fmt.Errorf("invalid scale parameter '%s': %w", scaleStr, err)
		}
		scale = s
	}

	faceEnhance := true // 默认值
	if faceEnhanceStr != "" {
		fe, err := strconv.ParseBool(faceEnhanceStr)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"face_enhance_str": faceEnhanceStr,
				"error":            err.Error(),
			}).Error("Failed to parse face_enhance parameter")
			return nil, fmt.Errorf("invalid face_enhance parameter '%s': %w", faceEnhanceStr, err)
		}
		faceEnhance = fe
	}

	req := &clients.AI302UpscaleRequest{
		Image:       imageData,
		Scale:       scale,
		FaceEnhance: faceEnhance,
	}
	return req, nil
}

// parseJSONRequest 解析JSON请求
func (h *AI302Handler) parseJSONRequest(c *gin.Context) (*clients.AI302UpscaleRequest, error) {
	var jsonReq clients.AI302UpscaleJSONRequest
	err := c.ShouldBindJSON(&jsonReq)
	if err != nil {
		return nil, err
	}

	// 处理base64图片数据
	imageBase64 := jsonReq.Image
	if strings.HasPrefix(imageBase64, "data:") {
		// 移除数据URL前缀 (例如: data:image/png;base64,)
		if idx := strings.Index(imageBase64, ","); idx != -1 {
			imageBase64 = imageBase64[idx+1:]
		}
	}

	// 解码base64图片数据
	imageData, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// 转换为AI302UpscaleRequest
	req := &clients.AI302UpscaleRequest{
		Image:       imageData,
		Scale:       jsonReq.Scale,
		FaceEnhance: jsonReq.FaceEnhance,
	}

	return req, nil
}
