package handlers

import (
	"encoding/base64"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"
)

// VectorizerHandler Vectorizer处理器
type VectorizerHandler struct {
	vectorizerService services.VectorizerService
	logger            logger.Logger
}

// NewVectorizerHandler 创建Vectorizer处理器
func NewVectorizerHandler(
	vectorizerService services.VectorizerService,
	logger logger.Logger,
) *VectorizerHandler {
	return &VectorizerHandler{
		vectorizerService: vectorizerService,
		logger:            logger,
	}
}

// VectorizerResponse 矢量化响应格式
type VectorizerResponse struct {
	SVGData      string `json:"svg_data,omitempty"`      // SVG矢量图数据
	FinishReason string `json:"finish_reason,omitempty"` // 完成原因
	Error        string `json:"error,omitempty"`         // 错误信息
}

// Vectorize 矢量化图片
// @Summary 矢量化图片
// @Description 将普通图片转换为可无限放大的矢量图
// @Tags Vectorizer
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param image formData file true "要矢量化的图片文件"
// @Success 200 {object} VectorizerResponse "矢量化成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /vectorizer/api/v1/vectorize [post]
func (h *VectorizerHandler) Vectorize(c *gin.Context) {
	userID := c.GetInt64("user_id")
	apiKeyID := c.GetInt64("api_key_id")

	// 解析multipart form
	err := c.Request.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("Failed to parse multipart form")

		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Failed to parse multipart form: " + err.Error(),
			},
		})
		return
	}

	// 获取图片文件
	imageFile, _, err := c.Request.FormFile("image")
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("Failed to get image file")

		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Image file is required",
			},
		})
		return
	}
	defer imageFile.Close()

	// 读取图片数据
	imageData, err := io.ReadAll(imageFile)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("Failed to read image data")

		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Failed to read image data: " + err.Error(),
			},
		})
		return
	}

	// 转换为base64
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// 构造请求
	request := &clients.VectorizerRequest{
		Image: imageBase64,
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"api_key_id": apiKeyID,
		"image_size": len(imageData),
	}).Info("Processing vectorize request")

	// 调用服务
	response, err := h.vectorizerService.Vectorize(c.Request.Context(), userID, apiKeyID, request)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("Failed to vectorize image")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "PROCESSING_FAILED",
				"message": "Failed to vectorize image: " + err.Error(),
			},
		})
		return
	}

	// 检查是否有错误
	if response.Error != "" {
		h.logger.WithFields(map[string]interface{}{
			"error":      response.Error,
			"user_id":    userID,
			"api_key_id": apiKeyID,
		}).Error("Vectorizer service returned error")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "VECTORIZER_ERROR",
				"message": response.Error,
			},
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":       userID,
		"api_key_id":    apiKeyID,
		"svg_length":    len(response.SVGData),
		"finish_reason": response.FinishReason,
	}).Info("Successfully vectorized image")

	// 返回矢量化结果
	vectorizerResponse := VectorizerResponse{
		SVGData:      response.SVGData,
		FinishReason: response.FinishReason,
	}

	c.JSON(http.StatusOK, vectorizerResponse)
}
