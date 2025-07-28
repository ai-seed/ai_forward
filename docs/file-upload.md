# 文件上传功能

本文档介绍AI API Gateway的文件上传功能，支持将文件上传到S3兼容的存储服务。

## 功能特性

- 支持AWS S3和S3兼容服务（如MinIO）
- 文件类型验证
- 文件大小限制
- 自动生成唯一文件名
- JWT认证保护
- 完整的Swagger API文档

## 配置

### S3配置

在`configs/config.yaml`中配置S3存储：

```yaml
s3:
  # 是否启用S3存储
  enabled: true
  # AWS区域
  region: "us-east-1"
  # S3存储桶名称
  bucket: "your-s3-bucket-name"
  # AWS访问密钥ID
  access_key_id: "your-aws-access-key-id"
  # AWS秘密访问密钥
  secret_access_key: "your-aws-secret-access-key"
  # 自定义端点（用于兼容S3的服务，如MinIO）
  endpoint: ""
  # 是否使用路径样式URL（MinIO等服务需要设置为true）
  use_path_style: false
  # 最大文件大小（字节，默认10MB）
  max_file_size: 10485760
  # 允许的文件类型
  allowed_types:
    - "image/jpeg"
    - "image/png"
    - "image/gif"
    - "image/webp"
    - "application/pdf"
    - "text/plain"
```

### MinIO配置示例

如果使用MinIO，配置如下：

```yaml
s3:
  enabled: true
  region: "us-east-1"
  bucket: "uploads"
  access_key_id: "minioadmin"
  secret_access_key: "minioadmin"
  endpoint: "http://localhost:9000"
  use_path_style: true
  max_file_size: 10485760
  allowed_types:
    - "image/jpeg"
    - "image/png"
    - "application/pdf"
```

## API接口

### 1. 上传文件

**接口**: `POST /api/files/upload`

**认证**: 需要JWT Token

**请求格式**: `multipart/form-data`

**参数**:
- `file`: 要上传的文件

**响应示例**:
```json
{
  "success": true,
  "message": "File uploaded successfully",
  "data": {
    "key": "uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg",
    "url": "https://your-bucket.s3.us-east-1.amazonaws.com/uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg",
    "filename": "avatar.jpg",
    "size": 1024000,
    "mime_type": "image/jpeg",
    "uploaded_at": "2024-01-15T10:30:00Z"
  }
}
```

### 2. 删除文件

**接口**: `DELETE /api/files/delete`

**认证**: 需要JWT Token

**请求体**:
```json
{
  "key": "uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg"
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "File deleted successfully",
  "data": {
    "key": "uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg",
    "message": "File deleted successfully"
  }
}
```

### 3. 获取文件信息

**接口**: `GET /api/files/{key}`

**认证**: 需要JWT Token

**响应示例**:
```json
{
  "success": true,
  "message": "File info retrieved successfully",
  "data": {
    "key": "uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg",
    "url": "https://your-bucket.s3.us-east-1.amazonaws.com/uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg",
    "filename": "avatar.jpg",
    "size": 1024000,
    "mime_type": "image/jpeg"
  }
}
```

## 使用示例

### cURL示例

```bash
# 上传文件
curl -X POST "http://localhost:8080/api/files/upload" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@/path/to/your/file.jpg"

# 删除文件
curl -X DELETE "http://localhost:8080/api/files/delete" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"key": "uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg"}'

# 获取文件信息
curl -X GET "http://localhost:8080/api/files/uploads%2F2024%2F01%2F15%2F550e8400-e29b-41d4-a716-446655440000.jpg" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### JavaScript示例

```javascript
// 上传文件
const formData = new FormData();
formData.append('file', fileInput.files[0]);

fetch('/api/files/upload', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`
  },
  body: formData
})
.then(response => response.json())
.then(data => {
  console.log('Upload success:', data);
});

// 删除文件
fetch('/api/files/delete', {
  method: 'DELETE',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    key: 'uploads/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg'
  })
})
.then(response => response.json())
.then(data => {
  console.log('Delete success:', data);
});
```

## 错误处理

常见错误响应：

- `401 Unauthorized`: 未提供有效的JWT Token
- `400 Bad Request`: 请求参数错误或文件格式不正确
- `413 Request Entity Too Large`: 文件大小超过限制
- `415 Unsupported Media Type`: 文件类型不被允许
- `500 Internal Server Error`: 服务器内部错误
- `503 Service Unavailable`: S3服务未启用

## 安全考虑

1. **认证**: 所有文件操作都需要有效的JWT Token
2. **文件类型验证**: 只允许配置中指定的文件类型
3. **文件大小限制**: 防止大文件上传占用过多资源
4. **唯一文件名**: 自动生成UUID文件名，防止文件名冲突
5. **路径安全**: 使用日期分层存储，便于管理

## 监控和日志

系统会记录以下操作日志：
- 文件上传成功/失败
- 文件删除操作
- 文件访问记录
- 错误和异常情况

可以通过日志系统监控文件上传的使用情况和性能指标。
