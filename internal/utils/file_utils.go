package utils

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FileTypeValidationConfig 文件类型验证配置
type FileTypeValidationConfig struct {
	AllowedTypes []string // 允许的文件类型列表，支持通配符如 "image/*"
}

// FileKeyGenerationConfig 文件键生成配置
type FileKeyGenerationConfig struct {
	Prefix      string // 文件键前缀，默认为 "uploads"
	UseDateTime bool   // 是否使用日期时间路径，默认为 true
	UseUUID     bool   // 是否使用UUID，默认为 true
}

// DefaultFileKeyConfig 默认文件键生成配置
var DefaultFileKeyConfig = FileKeyGenerationConfig{
	Prefix:      "uploads",
	UseDateTime: true,
	UseUUID:     true,
}

// IsAllowedFileType 检查文件类型是否被允许
// 这是一个纯函数，可以直接调用
func IsAllowedFileType(contentType string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		return true // 如果没有限制，允许所有类型
	}

	for _, allowedType := range allowedTypes {
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

// IsAllowedFileTypeWithConfig 使用配置检查文件类型是否被允许
func IsAllowedFileTypeWithConfig(contentType string, config FileTypeValidationConfig) bool {
	return IsAllowedFileType(contentType, config.AllowedTypes)
}

// InferMimeType 从文件名推断MIME类型
// 这是一个纯函数，可以直接调用
func InferMimeType(filename string) string {
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}

// GenerateFileKey 生成文件键
// 这是一个纯函数，可以直接调用
func GenerateFileKey(filename string) string {
	return GenerateFileKeyWithConfig(filename, DefaultFileKeyConfig)
}

// GenerateFileKeyWithConfig 使用配置生成文件键
func GenerateFileKeyWithConfig(filename string, config FileKeyGenerationConfig) string {
	var parts []string

	// 添加前缀
	if config.Prefix != "" {
		parts = append(parts, config.Prefix)
	}

	// 添加日期路径
	if config.UseDateTime {
		now := time.Now()
		datePath := fmt.Sprintf("%d/%02d/%02d", now.Year(), now.Month(), now.Day())
		parts = append(parts, datePath)
	}

	// 生成文件名部分
	var fileName string
	if config.UseUUID {
		// 使用UUID作为唯一标识
		id := uuid.New().String()
		ext := filepath.Ext(filename)
		fileName = fmt.Sprintf("%s%s", id, ext)
	} else {
		// 直接使用原文件名（可能需要额外的唯一性处理）
		fileName = filename
	}

	parts = append(parts, fileName)
	return strings.Join(parts, "/")
}

// GenerateUniqueFileKey 生成唯一文件键（总是使用UUID）
func GenerateUniqueFileKey(filename string, prefix string) string {
	config := FileKeyGenerationConfig{
		Prefix:      prefix,
		UseDateTime: true,
		UseUUID:     true,
	}
	return GenerateFileKeyWithConfig(filename, config)
}

// GetFileExtension 获取文件扩展名（包含点号）
func GetFileExtension(filename string) string {
	return filepath.Ext(filename)
}

// GetFileExtensionWithoutDot 获取文件扩展名（不包含点号）
func GetFileExtensionWithoutDot(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) > 0 && ext[0] == '.' {
		return ext[1:]
	}
	return ext
}

// GetFileNameWithoutExtension 获取不包含扩展名的文件名
func GetFileNameWithoutExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return filename
	}
	return filename[:len(filename)-len(ext)]
}

// ValidateFileName 验证文件名是否合法
func ValidateFileName(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// 检查文件名长度
	if len(filename) > 255 {
		return fmt.Errorf("filename too long (max 255 characters)")
	}

	// 检查是否包含非法字符
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(filename, char) {
			return fmt.Errorf("filename contains invalid character: %s", char)
		}
	}

	return nil
}

// SanitizeFileName 清理文件名，移除或替换非法字符
func SanitizeFileName(filename string) string {
	// 替换非法字符为下划线
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := filename
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// 限制长度
	if len(sanitized) > 255 {
		ext := filepath.Ext(sanitized)
		nameWithoutExt := sanitized[:len(sanitized)-len(ext)]
		maxNameLength := 255 - len(ext)
		if maxNameLength > 0 {
			sanitized = nameWithoutExt[:maxNameLength] + ext
		} else {
			sanitized = sanitized[:255]
		}
	}

	return sanitized
}

// FormatFileSize 格式化文件大小为人类可读的格式
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsImageFile 检查是否为图片文件（基于MIME类型）
func IsImageFile(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// IsVideoFile 检查是否为视频文件（基于MIME类型）
func IsVideoFile(contentType string) bool {
	return strings.HasPrefix(contentType, "video/")
}

// IsAudioFile 检查是否为音频文件（基于MIME类型）
func IsAudioFile(contentType string) bool {
	return strings.HasPrefix(contentType, "audio/")
}

// IsTextFile 检查是否为文本文件（基于MIME类型）
func IsTextFile(contentType string) bool {
	return strings.HasPrefix(contentType, "text/") || 
		   contentType == "application/json" ||
		   contentType == "application/xml" ||
		   contentType == "application/javascript"
}
