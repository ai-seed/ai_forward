# OAuth 登录部署指南

本指南将帮助你完整部署和配置 Google 和 GitHub OAuth 登录功能。

## 🚀 快速开始

### 1. 验证代码完整性

确保以下文件已正确添加到项目中：

**后端文件：**
- `internal/application/services/oauth_service.go` - OAuth 服务实现
- `internal/presentation/handlers/oauth_handler.go` - OAuth HTTP 处理器
- `internal/domain/entities/user.go` - 扩展的用户实体（包含 OAuth 字段）
- `internal/application/dto/user_dto.go` - OAuth 相关 DTO
- `internal/infrastructure/config/config.go` - OAuth 配置结构

**前端文件：**
- `web/src/components/oauth-buttons/` - OAuth 登录按钮组件
- `web/src/pages/auth/oauth-callback.tsx` - OAuth 回调页面
- `web/src/services/auth.ts` - 扩展的认证服务
- `web/src/contexts/auth-context.tsx` - 扩展的认证上下文

**配置文件：**
- `configs/config.yaml` - 更新的配置文件
- `.env.example` - 环境变量示例
- `docs/oauth-setup.md` - 详细配置说明

### 2. 安装依赖

**后端依赖：**
```bash
go mod tidy
```

**前端依赖：**
```bash
cd web
npm install
```

### 3. 配置 OAuth 应用

#### Google OAuth 配置

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建项目或选择现有项目
3. 启用 Google+ API
4. 创建 OAuth 2.0 客户端 ID
5. 配置重定向 URI：`http://localhost:8080/auth/oauth/google/callback`

#### GitHub OAuth 配置

1. 访问 [GitHub Developer Settings](https://github.com/settings/developers)
2. 创建新的 OAuth App
3. 配置回调 URL：`http://localhost:8080/auth/oauth/github/callback`

### 4. 环境变量配置

创建 `.env` 文件：

```bash
# 数据库配置
DATABASE_URL=postgres://username:password@localhost:5432/dbname

# JWT 配置
JWT_SECRET=your-super-secret-jwt-key

# Google OAuth
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=your_google_client_id
OAUTH_GOOGLE_CLIENT_SECRET=your_google_client_secret
OAUTH_GOOGLE_REDIRECT_URL=http://localhost:8080/auth/oauth/google/callback

# GitHub OAuth
OAUTH_GITHUB_ENABLED=true
OAUTH_GITHUB_CLIENT_ID=your_github_client_id
OAUTH_GITHUB_CLIENT_SECRET=your_github_client_secret
OAUTH_GITHUB_REDIRECT_URL=http://localhost:8080/auth/oauth/github/callback
```

### 5. 启动应用

**启动后端：**
```bash
go run cmd/server/main.go
```

**启动前端：**
```bash
cd web
npm run dev
```

### 6. 测试 OAuth 功能

**使用测试脚本：**
```bash
chmod +x scripts/test-oauth.sh
./scripts/test-oauth.sh
```

**使用配置验证工具：**
```bash
go run cmd/oauth-test/main.go
```

**手动测试：**
1. 访问 `http://localhost:3000/sign-in`
2. 点击 "使用 Google 继续" 或 "使用 GitHub 继续"
3. 完成 OAuth 授权流程
4. 验证是否成功登录

## 🔧 API 端点

OAuth 相关的 API 端点：

- `GET /auth/oauth/{provider}/url` - 获取 OAuth 认证 URL
- `POST /auth/oauth/{provider}/callback` - 处理 OAuth 回调（JSON）
- `GET /auth/oauth/{provider}/redirect` - 直接重定向到 OAuth 提供商
- `GET /auth/oauth/{provider}/callback` - 处理 OAuth 回调（查询参数）

支持的提供商：`google`, `github`

## 🗄️ 数据库变更

OAuth 功能添加了以下数据库字段到 `users` 表：

- `google_id` - Google OAuth ID
- `github_id` - GitHub OAuth ID  
- `avatar` - 用户头像 URL
- `auth_method` - 认证方式（password/google/github）

数据库会在应用启动时自动迁移。

## 🔒 安全考虑

1. **环境变量保护**：确保 OAuth 客户端密钥不会暴露在前端代码中
2. **HTTPS 要求**：生产环境必须使用 HTTPS
3. **状态验证**：OAuth 流程中包含状态参数验证
4. **令牌安全**：JWT 令牌使用安全的密钥签名

## 🚨 故障排除

### 常见问题

1. **redirect_uri_mismatch**
   - 检查 OAuth 应用配置中的重定向 URI
   - 确保与代码中的 URL 完全一致

2. **invalid_client**
   - 验证客户端 ID 和密钥是否正确
   - 检查环境变量是否正确加载

3. **CORS 错误**
   - 确保后端 CORS 配置允许前端域名
   - 检查前端 API 基础 URL 配置

4. **数据库连接错误**
   - 确保数据库正在运行
   - 验证数据库连接字符串

### 调试技巧

1. 查看后端日志中的详细错误信息
2. 使用浏览器开发者工具检查网络请求
3. 运行配置验证工具检查配置
4. 使用测试脚本验证基础功能

## 📈 生产环境部署

### 环境变量更新

```bash
# 更新重定向 URL 为生产域名
OAUTH_GOOGLE_REDIRECT_URL=https://yourdomain.com/auth/oauth/google/callback
OAUTH_GITHUB_REDIRECT_URL=https://yourdomain.com/auth/oauth/github/callback
```

### OAuth 应用更新

1. 在 Google Cloud Console 中添加生产域名的重定向 URI
2. 在 GitHub OAuth 应用中更新回调 URL
3. 确保生产环境使用 HTTPS

### 监控和日志

- 监控 OAuth 登录成功率
- 记录 OAuth 错误日志
- 设置告警机制

## 🎯 下一步

1. 添加更多 OAuth 提供商（微信、钉钉等）
2. 实现账号绑定功能
3. 添加 OAuth 登录统计
4. 优化用户体验

## 📞 支持

如果遇到问题，请：
1. 查看详细的配置文档 `docs/oauth-setup.md`
2. 运行诊断工具进行问题排查
3. 检查项目的 GitHub Issues
