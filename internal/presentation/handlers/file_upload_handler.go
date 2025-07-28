package handlers

import (
	"net/http"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// FileUploadHandler 文件上传处理器
type FileUploadHandler struct {
	fileUploadService services.FileUploadService
	config            *config.S3Config
	logger            logger.Logger
}

// NewFileUploadHandler 创建文件上传处理器
func NewFileUploadHandler(
	fileUploadService services.FileUploadService,
	config *config.S3Config,
	logger logger.Logger,
) *FileUploadHandler {
	return &FileUploadHandler{
		fileUploadService: fileUploadService,
		config:            config,
		logger:            logger,
	}
}

// UploadFile 上传文件
// @Summary 上传文件到S3
// @Description 上传文件到S3存储，支持图片、PDF等多种文件类型
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "要上传的文件"
// @Success 200 {object} dto.Response{data=dto.FileUploadResponse} "上传成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 413 {object} dto.Response "文件过大"
// @Failure 415 {object} dto.Response "不支持的文件类型"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /api/files/upload [post]
func (h *FileUploadHandler) UploadFile(c *gin.Context) {
	// 检查服务是否启用
	if !h.fileUploadService.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse(
			"SERVICE_UNAVAILABLE",
			"File upload service is not enabled",
			nil,
		))
		return
	}

	// 获取认证信息
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"Authentication required",
			nil,
		))
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		}).Error("Failed to get uploaded file")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_FILE",
			"Failed to get uploaded file: "+err.Error(),
			nil,
		))
		return
	}

	// 验证文件大小
	if h.config.MaxFileSize > 0 && file.Size > h.config.MaxFileSize {
		h.logger.WithFields(map[string]interface{}{
			"file_size": file.Size,
			"max_size":  h.config.MaxFileSize,
			"user_id":   userID,
			"filename":  file.Filename,
		}).Warn("File size exceeds limit")
		c.JSON(http.StatusRequestEntityTooLarge, dto.ErrorResponse(
			"FILE_TOO_LARGE",
			"File size exceeds maximum allowed size",
			map[string]interface{}{
				"file_size": file.Size,
				"max_size":  h.config.MaxFileSize,
			},
		))
		return
	}

	// 验证文件类型
	contentType := file.Header.Get("Content-Type")
	if !h.isAllowedType(contentType) {
		h.logger.WithFields(map[string]interface{}{
			"content_type":  contentType,
			"allowed_types": h.config.AllowedTypes,
			"user_id":       userID,
			"filename":      file.Filename,
		}).Warn("File type not allowed")
		c.JSON(http.StatusUnsupportedMediaType, dto.ErrorResponse(
			"UNSUPPORTED_FILE_TYPE",
			"File type is not allowed",
			map[string]interface{}{
				"content_type":  contentType,
				"allowed_types": h.config.AllowedTypes,
			},
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"filename":     file.Filename,
		"size":         file.Size,
		"content_type": contentType,
		"user_id":      userID,
	}).Info("Starting file upload")

	// 上传文件
	result, err := h.fileUploadService.UploadFile(c.Request.Context(), file)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"filename": file.Filename,
			"user_id":  userID,
		}).Error("Failed to upload file")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"UPLOAD_FAILED",
			"Failed to upload file: "+err.Error(),
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"key":      result.Key,
		"url":      result.URL,
		"filename": result.Filename,
		"user_id":  userID,
	}).Info("File uploaded successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(result, "File uploaded successfully"))
}

// DeleteFile 删除文件
// @Summary 删除S3文件
// @Description 根据文件键删除S3中的文件
// @Tags 文件管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.FileDeleteRequest true "删除请求"
// @Success 200 {object} dto.Response{data=dto.FileDeleteResponse} "删除成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 404 {object} dto.Response "文件不存在"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /api/files/delete [delete]
func (h *FileUploadHandler) DeleteFile(c *gin.Context) {
	// 检查服务是否启用
	if !h.fileUploadService.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse(
			"SERVICE_UNAVAILABLE",
			"File upload service is not enabled",
			nil,
		))
		return
	}

	// 获取认证信息
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"Authentication required",
			nil,
		))
		return
	}

	// 解析请求
	var req dto.FileDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		}).Error("Invalid delete request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request parameters: "+err.Error(),
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"key":     req.Key,
		"user_id": userID,
	}).Info("Starting file deletion")

	// 删除文件
	err := h.fileUploadService.DeleteFile(c.Request.Context(), req.Key)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":   err.Error(),
			"key":     req.Key,
			"user_id": userID,
		}).Error("Failed to delete file")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DELETE_FAILED",
			"Failed to delete file: "+err.Error(),
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"key":     req.Key,
		"user_id": userID,
	}).Info("File deleted successfully")

	result := &dto.FileDeleteResponse{
		Key:     req.Key,
		Message: "File deleted successfully",
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(result, "File deleted successfully"))
}

// isAllowedType 检查文件类型是否被允许
func (h *FileUploadHandler) isAllowedType(contentType string) bool {
	if len(h.config.AllowedTypes) == 0 {
		return true // 如果没有限制，允许所有类型
	}

	for _, allowedType := range h.config.AllowedTypes {
		if contentType == allowedType {
			return true
		}
	}

	return false
}
