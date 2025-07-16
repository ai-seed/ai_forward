# Swagger 文档生成指南

本文档介绍如何为 AI API Gateway 生成和管理 Swagger 文档。

## 概述

AI API Gateway 使用 [swaggo/swag](https://github.com/swaggo/swag) 工具自动生成 Swagger 文档。文档包含了所有的 API 端点、请求/响应模型和认证配置。

## 支持的 API

生成的 Swagger 文档包含以下 API：

### 🤖 OpenAI 兼容 API
- `POST /v1/chat/completions` - 聊天完成
- `POST /v1/completions` - 文本完成
- `GET /v1/models` - 获取模型列表

### 🎨 Midjourney 兼容 API (302AI 格式)
- `POST /mj/submit/imagine` - 图像生成
- `POST /mj/submit/action` - 执行操作 (U1-U4, V1-V4)
- `POST /mj/submit/blend` - 图像混合
- `POST /mj/submit/describe` - 图像描述
- `POST /mj/submit/modal` - 局部重绘
- `POST /mj/submit/cancel` - 取消任务
- `GET /mj/task/{id}/fetch` - 获取任务结果

### 🔐 认证 API
- `POST /auth/login` - 用户登录
- `POST /auth/register` - 用户注册
- `POST /auth/refresh` - 刷新令牌
- `GET /auth/profile` - 获取用户资料
- `POST /auth/change-password` - 修改密码
- `POST /auth/recharge` - 用户充值

### 🏥 健康检查
- `GET /health` - 服务健康检查

## 生成方法

我们提供了多种方式来生成 Swagger 文档：

### 方法 1: 使用 Makefile (推荐)

```bash
# 生成 Swagger 文档
make swagger

# 清理旧文档
make swagger-clean

# 验证文档
make swagger-verify
```

### 方法 2: 使用 Shell 脚本 (Linux/macOS)

```bash
# 给脚本执行权限
chmod +x scripts/generate-swagger.sh

# 生成文档
./scripts/generate-swagger.sh

# 查看帮助
./scripts/generate-swagger.sh --help

# 仅清理
./scripts/generate-swagger.sh --clean

# 仅验证
./scripts/generate-swagger.sh --verify
```

### 方法 3: 使用批处理脚本 (Windows)

```cmd
REM 生成文档
scripts\generate-swagger.bat

REM 查看帮助
scripts\generate-swagger.bat --help

REM 仅清理
scripts\generate-swagger.bat --clean

REM 仅验证
scripts\generate-swagger.bat --verify
```

### 方法 4: 使用 Go 脚本 (跨平台)

```bash
# 生成文档
go run scripts/generate-swagger.go

# 查看帮助
go run scripts/generate-swagger.go --help

# 仅清理
go run scripts/generate-swagger.go --clean

# 仅验证
go run scripts/generate-swagger.go --verify
```

### 方法 5: 直接使用 swag 命令

```bash
# 安装 swag 工具
go install github.com/swaggo/swag/cmd/swag@latest

# 生成文档
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

## 生成的文件

生成过程会在 `docs/` 目录下创建以下文件：

- `docs.go` - Go 代码形式的文档定义
- `swagger.json` - JSON 格式的 Swagger 规范
- `swagger.yaml` - YAML 格式的 Swagger 规范

## 访问文档

启动服务器后，可以通过以下方式访问文档：

### Swagger UI (推荐)
```
http://localhost:8080/swagger/index.html
```

### JSON API 文档
```
http://localhost:8080/swagger/doc.json
```

### 本地文件
- JSON: `docs/swagger.json`
- YAML: `docs/swagger.yaml`

## 认证配置

文档包含两种认证方式：

### 1. Bearer Token (OpenAI API)
- **Header**: `Authorization`
- **格式**: `Bearer {token}`
- **用于**: OpenAI 兼容 API、认证 API

### 2. MJ API Secret (Midjourney API)
- **Header**: `mj-api-secret`
- **格式**: `{api-key}`
- **用于**: Midjourney 兼容 API

## 在 Swagger UI 中测试 API

1. 打开 Swagger UI: http://localhost:8080/swagger/index.html
2. 点击右上角的 "Authorize" 按钮
3. 输入相应的认证信息：
   - **BearerAuth**: 输入 `Bearer your-token`
   - **MJApiSecret**: 输入 `your-api-key`
4. 选择要测试的 API 端点
5. 填写请求参数
6. 点击 "Execute" 执行请求

## 示例请求

### Midjourney 图像生成
```bash
curl -X POST "http://localhost:8080/mj/submit/imagine" \
  -H "mj-api-secret: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A beautiful cat sitting in a garden",
    "botType": "MID_JOURNEY"
  }'
```

### OpenAI 聊天完成
```bash
curl -X POST "http://localhost:8080/v1/chat/completions" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ]
  }'
```

## 故障排除

### 1. swag 工具未安装
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### 2. 文档生成失败
- 检查 Go 代码中的 Swagger 注释格式
- 确保所有引用的类型都存在
- 查看错误信息中的具体问题

### 3. 无法访问 Swagger UI
- 确保服务器正在运行
- 检查端口是否正确 (默认 8080)
- 确认路由配置正确

### 4. 认证失败
- 检查 API 密钥是否正确
- 确认使用了正确的认证方式
- 查看服务器日志获取详细错误信息

## 开发注意事项

### 添加新的 API 端点

1. 在处理器函数上添加 Swagger 注释：
```go
// CreateUser 创建用户
// @Summary 创建新用户
// @Description 创建一个新的用户账户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "创建用户请求"
// @Success 201 {object} CreateUserResponse "创建成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
    // 实现逻辑
}
```

2. 重新生成文档：
```bash
make swagger
```

### 更新认证配置

在 `cmd/server/main.go` 中更新 Swagger 注释：

```go
// @securityDefinitions.apikey NewAuth
// @in header
// @name X-API-Key
// @description New API key authentication
```

## 自动化集成

### CI/CD 集成

在 CI/CD 流水线中添加文档生成步骤：

```yaml
# GitHub Actions 示例
- name: Generate Swagger docs
  run: make swagger

- name: Verify Swagger docs
  run: make swagger-verify
```

### Git Hooks

在 pre-commit hook 中自动生成文档：

```bash
#!/bin/sh
# .git/hooks/pre-commit
make swagger
git add docs/
```

## 相关链接

- [Swagger/OpenAPI 规范](https://swagger.io/specification/)
- [swaggo/swag 文档](https://github.com/swaggo/swag)
- [Gin Swagger 中间件](https://github.com/swaggo/gin-swagger)
- [Swagger UI](https://swagger.io/tools/swagger-ui/)
