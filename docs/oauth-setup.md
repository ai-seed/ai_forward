# OAuth 登录配置指南

本文档介绍如何配置 Google 和 GitHub OAuth 登录功能。

## 1. Google OAuth 配置

### 1.1 创建 Google OAuth 应用

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建新项目或选择现有项目
3. 启用 Google+ API 和 Google Identity API
4. 转到 "APIs & Services" > "Credentials"
5. 点击 "Create Credentials" > "OAuth 2.0 Client IDs"
6. 选择应用类型为 "Web application"
7. 配置授权重定向 URI：
   - 开发环境：`http://localhost:8080/auth/oauth/google/callback`
   - 生产环境：`https://yourdomain.com/auth/oauth/google/callback`

### 1.2 配置环境变量

在 `.env` 文件中添加以下配置：

```bash
# Google OAuth
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=your_google_client_id_here
OAUTH_GOOGLE_CLIENT_SECRET=your_google_client_secret_here
OAUTH_GOOGLE_REDIRECT_URL=http://localhost:8080/auth/oauth/google/callback
```

### 1.3 更新配置文件

在 `configs/config.yaml` 中更新：

```yaml
oauth:
  google:
    enabled: true
    client_id: "${OAUTH_GOOGLE_CLIENT_ID}"
    client_secret: "${OAUTH_GOOGLE_CLIENT_SECRET}"
    redirect_url: "${OAUTH_GOOGLE_REDIRECT_URL}"
```

## 2. GitHub OAuth 配置

### 2.1 创建 GitHub OAuth 应用

1. 访问 [GitHub Developer Settings](https://github.com/settings/developers)
2. 点击 "New OAuth App"
3. 填写应用信息：
   - Application name: 你的应用名称
   - Homepage URL: `http://localhost:3000` (开发环境)
   - Authorization callback URL: `http://localhost:8080/auth/oauth/github/callback`
4. 点击 "Register application"
5. 记录 Client ID 和 Client Secret

### 2.2 配置环境变量

在 `.env` 文件中添加以下配置：

```bash
# GitHub OAuth
OAUTH_GITHUB_ENABLED=true
OAUTH_GITHUB_CLIENT_ID=your_github_client_id_here
OAUTH_GITHUB_CLIENT_SECRET=your_github_client_secret_here
OAUTH_GITHUB_REDIRECT_URL=http://localhost:8080/auth/oauth/github/callback
```

### 2.3 更新配置文件

在 `configs/config.yaml` 中更新：

```yaml
oauth:
  github:
    enabled: true
    client_id: "${OAUTH_GITHUB_CLIENT_ID}"
    client_secret: "${OAUTH_GITHUB_CLIENT_SECRET}"
    redirect_url: "${OAUTH_GITHUB_REDIRECT_URL}"
```

## 3. 数据库迁移

运行应用程序时，数据库会自动迁移以支持 OAuth 字段。确保在 `internal/infrastructure/database/gorm.go` 中启用了 User 实体的迁移。

## 4. 前端配置

前端会自动检测后端的 OAuth 配置。确保前端能够访问后端的 OAuth 端点：

- `/auth/oauth/google/url` - 获取 Google 认证 URL
- `/auth/oauth/google/callback` - Google 回调处理
- `/auth/oauth/github/url` - 获取 GitHub 认证 URL  
- `/auth/oauth/github/callback` - GitHub 回调处理

## 5. 测试 OAuth 登录

### 5.1 启动应用

1. 确保数据库正在运行
2. 启动后端服务：`go run cmd/server/main.go`
3. 启动前端服务：`cd web && npm run dev`

### 5.2 测试流程

1. 访问登录页面：`http://localhost:3000/sign-in`
2. 点击 "使用 Google 继续" 或 "使用 GitHub 继续" 按钮
3. 重定向到 OAuth 提供商进行授权
4. 授权成功后重定向回应用
5. 检查是否成功登录并创建用户账户

## 6. 生产环境配置

### 6.1 域名配置

在生产环境中，需要更新重定向 URL：

```bash
OAUTH_GOOGLE_REDIRECT_URL=https://yourdomain.com/auth/oauth/google/callback
OAUTH_GITHUB_REDIRECT_URL=https://yourdomain.com/auth/oauth/github/callback
```

### 6.2 HTTPS 要求

OAuth 提供商通常要求生产环境使用 HTTPS。确保你的应用部署在 HTTPS 环境中。

### 6.3 安全考虑

1. 保护 OAuth 客户端密钥，不要在前端代码中暴露
2. 使用环境变量管理敏感配置
3. 定期轮换 OAuth 客户端密钥
4. 监控 OAuth 登录日志以检测异常活动

## 7. 故障排除

### 7.1 常见错误

1. **redirect_uri_mismatch**: 检查 OAuth 应用配置中的重定向 URI 是否与代码中的一致
2. **invalid_client**: 检查客户端 ID 和密钥是否正确
3. **access_denied**: 用户拒绝了授权请求
4. **invalid_grant**: 授权码已过期或无效

### 7.2 调试技巧

1. 检查后端日志中的 OAuth 相关错误
2. 使用浏览器开发者工具检查网络请求
3. 验证环境变量是否正确加载
4. 确认数据库连接和迁移是否成功

## 8. API 文档

OAuth 相关的 API 端点：

- `GET /auth/oauth/{provider}/url` - 获取 OAuth 认证 URL
- `POST /auth/oauth/{provider}/callback` - 处理 OAuth 回调
- `GET /auth/oauth/{provider}/redirect` - 直接重定向到 OAuth 提供商
- `GET /auth/oauth/{provider}/callback` - 处理查询参数形式的回调

支持的提供商：`google`, `github`
