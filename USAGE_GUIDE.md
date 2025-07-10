# AI API Gateway 使用指南

## 🚀 快速开始

### 1. 初始化数据
已经为您创建了测试数据，包括：
- 3个测试用户（admin, testuser, developer）
- 2个AI提供商（OpenAI, Anthropic）
- 3个AI模型（gpt-3.5-turbo, gpt-4, claude-3）
- 1个真实的API密钥

### 2. 启动服务器
```bash
go run cmd/server/main.go -config configs/config.yaml
```

### 3. 测试API密钥
您的测试API密钥：
```
ak_4278fc65b1d32cc99fe69fc25bf352261fab3aa0b08488d919dce0097b0f3915
```

## 🔧 API测试

### 健康检查
```bash
curl http://localhost:8080/health
```

### 获取模型列表
```bash
curl -H "Authorization: Bearer ak_4278fc65b1d32cc99fe69fc25bf352261fab3aa0b08488d919dce0097b0f3915" \
     http://localhost:8080/v1/models
```

### 聊天完成（需要配置真实的提供商API密钥）
```bash
curl -X POST \
  -H "Authorization: Bearer ak_4278fc65b1d32cc99fe69fc25bf352261fab3aa0b08488d919dce0097b0f3915" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }' \
  http://localhost:8080/v1/chat/completions
```

## ⚙️ 配置提供商API密钥

要让网关实际调用AI提供商，您需要通过管理界面或API配置提供商信息和API密钥。

提供商信息现在存储在数据库中，可以通过以下方式管理：

1. **通过管理界面**: 访问管理界面添加和配置提供商
2. **通过API**: 使用提供商管理API添加、更新提供商信息
3. **数据库初始化**: 系统启动时会自动创建默认的OpenAI和Anthropic提供商

## 📊 管理API

### 创建用户
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "email": "newuser@example.com",
    "full_name": "新用户"
  }' \
  http://localhost:8080/admin/users/
```

### 创建API密钥
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "name": "我的API密钥",
    "permissions": {
      "allowed_providers": ["openai"],
      "allowed_models": ["gpt-3.5-turbo"]
    }
  }' \
  http://localhost:8080/admin/api-keys/
```

### 查看用户信息
```bash
curl http://localhost:8080/admin/users/1
```

### 给用户充值
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 50.0,
    "operation": "add",
    "description": "充值"
  }' \
  http://localhost:8080/admin/users/1/balance
```

## 🔍 监控和统计

### 获取统计信息
```bash
curl http://localhost:8080/health/stats
```

### 获取监控指标（Prometheus格式）
```bash
curl http://localhost:8080/metrics
```

### 查看用户使用情况
```bash
curl http://localhost:8080/v1/usage \
  -H "Authorization: Bearer ak_4278fc65b1d32cc99fe69fc25bf352261fab3aa0b08488d919dce0097b0f3915"
```

## 🛠️ 开发和调试

### 查看日志
服务器会输出详细的日志信息，包括：
- 请求处理日志
- 错误信息
- 性能指标

### 数据库查看
```bash
# 连接SQLite数据库
sqlite3 data/gateway.db

# 查看表结构
.schema

# 查看用户
SELECT * FROM users;

# 查看API密钥
SELECT * FROM api_keys;

# 查看使用日志
SELECT * FROM usage_logs;
```

## 🔒 安全注意事项

1. **API密钥安全**: 请妥善保管API密钥，不要在代码中硬编码
2. **HTTPS**: 生产环境请使用HTTPS
3. **速率限制**: 已配置基本的速率限制，可根据需要调整
4. **权限控制**: API密钥支持细粒度的权限控制

## 🚨 常见问题

### Q: 服务器启动后没有输出？
A: 这是正常的，服务器在后台运行。可以通过健康检查确认服务状态。

### Q: API调用返回401错误？
A: 检查API密钥是否正确，确保使用Bearer认证格式。

### Q: 调用AI模型返回错误？
A: 确保已配置对应提供商的真实API密钥。

### Q: 如何查看详细错误信息？
A: 检查服务器日志输出，或将日志级别设置为debug。

## 📝 下一步

1. **配置真实的AI提供商API密钥**
2. **测试各种API端点**
3. **根据需要调整配置**
4. **部署到生产环境**

## 🆘 获取帮助

如果遇到问题，请：
1. 检查服务器日志
2. 确认配置文件正确
3. 验证数据库连接
4. 测试网络连接

---

**重要提醒**: 这是一个完整的AI API网关系统，支持负载均衡、故障转移、配额管理等企业级功能。在生产环境使用前，请确保进行充分的测试和安全配置。
