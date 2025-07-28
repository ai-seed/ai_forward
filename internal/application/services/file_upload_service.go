package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/infrastructure/storage"
)

// FileUploadService 文件上传服务接口
type FileUploadService interface {
	// UploadFile 上传文件
	UploadFile(ctx context.Context, file *multipart.FileHeader) (*dto.FileUploadResponse, error)
	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, key string) error
	// GetFileInfo 获取文件信息
	GetFileInfo(ctx context.Context, key string) (*dto.FileInfoResponse, error)
	// IsEnabled 检查文件上传服务是否启用
	IsEnabled() bool
}

// fileUploadServiceImpl 文件上传服务实现
type fileUploadServiceImpl struct {
	s3Service storage.S3Service
	logger    logger.Logger
}

// NewFileUploadService 创建文件上传服务
func NewFileUploadService(s3Service storage.S3Service, logger logger.Logger) FileUploadService {
	return &fileUploadServiceImpl{
		s3Service: s3Service,
		logger:    logger,
	}
}

// IsEnabled 检查文件上传服务是否启用
func (s *fileUploadServiceImpl) IsEnabled() bool {
	return s.s3Service.IsEnabled()
}

// UploadFile 上传文件
func (s *fileUploadServiceImpl) UploadFile(ctx context.Context, file *multipart.FileHeader) (*dto.FileUploadResponse, error) {
	if !s.s3Service.IsEnabled() {
		return nil, fmt.Errorf("file upload service is not enabled")
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"filename": file.Filename,
		}).Error("Failed to open uploaded file")
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// 获取文件内容类型
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		// 如果没有Content-Type，尝试从文件名推断
		contentType = "application/octet-stream"
	}

	s.logger.WithFields(map[string]interface{}{
		"filename":     file.Filename,
		"size":         file.Size,
		"content_type": contentType,
	}).Info("Starting file upload")

	// 上传到S3
	result, err := s.s3Service.UploadFile(ctx, file.Filename, contentType, src)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"filename": file.Filename,
		}).Error("Failed to upload file to S3")
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// 构建响应
	response := &dto.FileUploadResponse{
		Key:        result.Key,
		URL:        result.URL,
		Filename:   result.Filename,
		Size:       file.Size,
		MimeType:   result.MimeType,
		UploadedAt: time.Now(),
	}

	s.logger.WithFields(map[string]interface{}{
		"key":      result.Key,
		"filename": result.Filename,
		"size":     file.Size,
		"url":      result.URL,
	}).Info("File uploaded successfully")

	return response, nil
}

// DeleteFile 删除文件
func (s *fileUploadServiceImpl) DeleteFile(ctx context.Context, key string) error {
	if !s.s3Service.IsEnabled() {
		return fmt.Errorf("file upload service is not enabled")
	}

	if key == "" {
		return fmt.Errorf("file key is required")
	}

	s.logger.WithFields(map[string]interface{}{
		"key": key,
	}).Info("Starting file deletion")

	err := s.s3Service.DeleteFile(ctx, key)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"key":   key,
		}).Error("Failed to delete file from S3")
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"key": key,
	}).Info("File deleted successfully")

	return nil
}

// GetFileInfo 获取文件信息
func (s *fileUploadServiceImpl) GetFileInfo(ctx context.Context, key string) (*dto.FileInfoResponse, error) {
	if !s.s3Service.IsEnabled() {
		return nil, fmt.Errorf("file upload service is not enabled")
	}

	if key == "" {
		return nil, fmt.Errorf("file key is required")
	}

	// 获取文件URL
	url, err := s.s3Service.GetFileURL(ctx, key)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"key":   key,
		}).Error("Failed to get file URL")
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 构建响应（注意：这里只能返回基本信息，因为S3不存储原始文件名等元数据）
	response := &dto.FileInfoResponse{
		Key: key,
		URL: url,
		// 其他字段需要从数据库或其他存储中获取
	}

	return response, nil
}

// ValidateFile 验证文件
func (s *fileUploadServiceImpl) ValidateFile(file *multipart.FileHeader, maxSize int64, allowedTypes []string) error {
	// 检查文件大小
	if maxSize > 0 && file.Size > maxSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size %d bytes", file.Size, maxSize)
	}

	// 检查文件类型
	if len(allowedTypes) > 0 {
		contentType := file.Header.Get("Content-Type")
		allowed := false
		for _, allowedType := range allowedTypes {
			if contentType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file type %s is not allowed", contentType)
		}
	}

	return nil
}
