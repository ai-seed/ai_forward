#!/bin/bash

# OAuth 测试脚本
# 用于测试 OAuth 登录功能

set -e

echo "🚀 开始测试 OAuth 功能..."

# 检查服务器是否运行
echo "📡 检查服务器状态..."
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "❌ 服务器未运行，请先启动服务器"
    echo "   运行: go run cmd/server/main.go"
    exit 1
fi

echo "✅ 服务器正在运行"

# 测试 Google OAuth URL 获取
echo "🔍 测试 Google OAuth URL 获取..."
GOOGLE_RESPONSE=$(curl -s http://localhost:8080/auth/oauth/google/url)
echo "Google OAuth 响应: $GOOGLE_RESPONSE"

if echo "$GOOGLE_RESPONSE" | grep -q "auth_url"; then
    echo "✅ Google OAuth URL 获取成功"
else
    echo "❌ Google OAuth URL 获取失败"
fi

# 测试 GitHub OAuth URL 获取
echo "🔍 测试 GitHub OAuth URL 获取..."
GITHUB_RESPONSE=$(curl -s http://localhost:8080/auth/oauth/github/url)
echo "GitHub OAuth 响应: $GITHUB_RESPONSE"

if echo "$GITHUB_RESPONSE" | grep -q "auth_url"; then
    echo "✅ GitHub OAuth URL 获取成功"
else
    echo "❌ GitHub OAuth URL 获取失败"
fi

# 测试无效提供商
echo "🔍 测试无效提供商..."
INVALID_RESPONSE=$(curl -s http://localhost:8080/auth/oauth/invalid/url)
echo "无效提供商响应: $INVALID_RESPONSE"

if echo "$INVALID_RESPONSE" | grep -q "UNSUPPORTED_PROVIDER"; then
    echo "✅ 无效提供商处理正确"
else
    echo "❌ 无效提供商处理失败"
fi

echo "🎉 OAuth 基础功能测试完成！"
echo ""
echo "📋 下一步配置说明："
echo "1. 配置 Google OAuth 应用："
echo "   - 访问 https://console.cloud.google.com/"
echo "   - 创建 OAuth 2.0 客户端 ID"
echo "   - 设置重定向 URI: http://localhost:8080/auth/oauth/google/callback"
echo ""
echo "2. 配置 GitHub OAuth 应用："
echo "   - 访问 https://github.com/settings/developers"
echo "   - 创建新的 OAuth App"
echo "   - 设置回调 URL: http://localhost:8080/auth/oauth/github/callback"
echo ""
echo "3. 更新环境变量："
echo "   - OAUTH_GOOGLE_ENABLED=true"
echo "   - OAUTH_GOOGLE_CLIENT_ID=your_google_client_id"
echo "   - OAUTH_GOOGLE_CLIENT_SECRET=your_google_client_secret"
echo "   - OAUTH_GITHUB_ENABLED=true"
echo "   - OAUTH_GITHUB_CLIENT_ID=your_github_client_id"
echo "   - OAUTH_GITHUB_CLIENT_SECRET=your_github_client_secret"
echo ""
echo "4. 重启服务器并测试完整的 OAuth 流程"
