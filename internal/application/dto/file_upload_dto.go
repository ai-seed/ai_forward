package dto

import "time"

// FileUploadRequest 文件上传请求
type FileUploadRequest struct {
	// 文件字段名，通过multipart/form-data上传
	// 在Swagger中不需要定义，因为这是通过表单上传的
}

// FileUploadResponse 文件上传响应
type FileUploadResponse struct {
	Key        string    `json:"key" example:"uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg"`                                                // S3对象键
	URL        string    `json:"url" example:"https://your-bucket.s3.us-east-1.amazonaws.com/uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg"` // 文件访问URL
	Filename   string    `json:"filename" example:"avatar.jpg"`                                                                                            // 原始文件名
	Size       int64     `json:"size" example:"1024000"`                                                                                                   // 文件大小（字节）
	MimeType   string    `json:"mime_type" example:"image/jpeg"`                                                                                           // MIME类型
	UploadedAt time.Time `json:"uploaded_at"`                                                                                                              // 上传时间
}

// FileDeleteRequest 文件删除请求
type FileDeleteRequest struct {
	Key string `json:"key" binding:"required" example:"uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg"` // S3对象键
}

// FileDeleteResponse 文件删除响应
type FileDeleteResponse struct {
	Key     string `json:"key" example:"uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg"` // 已删除的S3对象键
	Message string `json:"message" example:"File deleted successfully"`                               // 删除结果消息
}
