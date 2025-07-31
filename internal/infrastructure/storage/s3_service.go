package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3Service S3存储服务接口
type S3Service interface {
	// UploadFile 上传文件
	UploadFile(ctx context.Context, filename string, contentType string, content io.Reader) (*UploadResult, error)
	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, key string) error
	// GetFileURL 获取文件URL
	GetFileURL(ctx context.Context, key string) (string, error)
	// IsEnabled 检查S3服务是否启用
	IsEnabled() bool
}

// UploadResult 上传结果
type UploadResult struct {
	Key      string `json:"key"`       // S3对象键
	URL      string `json:"url"`       // 文件访问URL
	Filename string `json:"filename"`  // 原始文件名
	Size     int64  `json:"size"`      // 文件大小
	MimeType string `json:"mime_type"` // MIME类型
}

// s3ServiceImpl S3服务实现
type s3ServiceImpl struct {
	client *s3.Client
	config *config.S3Config
	logger logger.Logger
}

// NewS3Service 创建S3服务
func NewS3Service(cfg *config.S3Config, logger logger.Logger) (S3Service, error) {
	if !cfg.Enabled {
		return &s3ServiceImpl{
			config: cfg,
			logger: logger,
		}, nil
	}

	// 创建AWS配置
	var awsCfg aws.Config
	var err error

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		// 使用静态凭证
		awsCfg, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(cfg.Region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			)),
		)
	} else {
		// 使用默认凭证链（环境变量、IAM角色等）
		awsCfg, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 创建S3客户端选项
	var options []func(*s3.Options)

	// 如果有自定义端点（如MinIO）
	if cfg.Endpoint != "" {
		options = append(options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = cfg.UsePathStyle
		})
	}

	// 创建S3客户端
	client := s3.NewFromConfig(awsCfg, options...)

	return &s3ServiceImpl{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// IsEnabled 检查S3服务是否启用
func (s *s3ServiceImpl) IsEnabled() bool {
	return s.config.Enabled
}

// UploadFile 上传文件
func (s *s3ServiceImpl) UploadFile(ctx context.Context, filename string, contentType string, content io.Reader) (*UploadResult, error) {
	if !s.config.Enabled {
		return nil, fmt.Errorf("S3 service is not enabled")
	}

	// 如果没有提供contentType，尝试从文件名推断
	if contentType == "" {
		contentType = utils.InferMimeType(filename)
	}

	// 验证文件类型
	if !utils.IsAllowedFileType(contentType, s.config.AllowedTypes) {
		return nil, fmt.Errorf("file type %s is not allowed", contentType)
	}

	// 生成唯一的文件键
	key := utils.GenerateFileKey(filename)

	// 上传文件到S3
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.config.Bucket),
		Key:         aws.String(key),
		Body:        content,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"bucket":   s.config.Bucket,
			"key":      key,
			"filename": filename,
		}).Error("Failed to upload file to S3")
		return nil, fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// 生成文件URL
	url, err := s.GetFileURL(ctx, key)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"key":   key,
		}).Warn("Failed to generate file URL")
		// 即使URL生成失败，文件已经上传成功，所以不返回错误
		url = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.config.Bucket, s.config.Region, key)
	}

	s.logger.WithFields(map[string]interface{}{
		"key":      key,
		"filename": filename,
		"url":      url,
	}).Info("File uploaded successfully to S3")

	return &UploadResult{
		Key:      key,
		URL:      url,
		Filename: filename,
		MimeType: contentType,
	}, nil
}

// DeleteFile 删除文件
func (s *s3ServiceImpl) DeleteFile(ctx context.Context, key string) error {
	if !s.config.Enabled {
		return fmt.Errorf("S3 service is not enabled")
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"bucket": s.config.Bucket,
			"key":    key,
		}).Error("Failed to delete file from S3")
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"key": key,
	}).Info("File deleted successfully from S3")

	return nil
}

// GetFileURL 获取文件URL
func (s *s3ServiceImpl) GetFileURL(ctx context.Context, key string) (string, error) {
	if !s.config.Enabled {
		return "", fmt.Errorf("S3 service is not enabled")
	}

	// 如果有自定义端点，构建URL
	if s.config.Endpoint != "" {
		if s.config.UsePathStyle {
			return fmt.Sprintf("%s/%s/%s", s.config.Endpoint, s.config.Bucket, key), nil
		}
		return fmt.Sprintf("%s/%s", s.config.Endpoint, key), nil
	}

	// 使用AWS S3的标准URL格式
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.config.Bucket, s.config.Region, key), nil
}

// isAllowedType 检查文件类型是否被允许
func (s *s3ServiceImpl) isAllowedType(contentType string) bool {
	if len(s.config.AllowedTypes) == 0 {
		return true // 如果没有限制，允许所有类型
	}

	for _, allowedType := range s.config.AllowedTypes {
		if contentType == allowedType {
			return true
		}
		// 支持通配符匹配，如 "image/*"
		if strings.HasSuffix(allowedType, "/*") {
			prefix := strings.TrimSuffix(allowedType, "/*")
			if strings.HasPrefix(contentType, prefix+"/") {
				return true
			}
		}
	}

	return false
}

// generateFileKey 生成文件键
func (s *s3ServiceImpl) generateFileKey(filename string) string {
	// 生成UUID作为唯一标识
	id := uuid.New().String()

	// 获取文件扩展名
	ext := filepath.Ext(filename)

	// 生成日期路径
	now := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", now.Year(), now.Month(), now.Day())

	// 组合最终的键
	return fmt.Sprintf("uploads/%s/%s%s", datePath, id, ext)
}
