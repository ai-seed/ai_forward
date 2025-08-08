# Billing Service

独立的计费服务，负责处理AI API网关的计费、配额管理和余额扣减。

## 架构设计

### 服务职责
- **计费管理**：处理基于token和请求的计费逻辑
- **配额控制**：管理API调用的配额限制和使用情况
- **余额管理**：处理用户余额的扣减和充值
- **定价管理**：管理不同AI模型的定价策略

### 微服务架构
```
billing-service/
├── cmd/server/          # 服务入口
├── internal/
│   ├── domain/          # 领域层
│   │   ├── entities/    # 实体
│   │   ├── repositories/ # 仓储接口
│   │   └── services/    # 领域服务接口
│   ├── application/     # 应用层
│   │   ├── services/    # 应用服务实现
│   │   └── dto/         # 数据传输对象
│   ├── infrastructure/  # 基础设施层
│   │   ├── database/    # 数据库
│   │   ├── repositories/ # 仓储实现
│   │   ├── clients/     # 外部客户端
│   │   └── config/      # 配置
│   └── presentation/    # 表现层
│       ├── handlers/    # HTTP处理器
│       └── routes/      # 路由配置
├── configs/             # 配置文件
├── migrations/          # 数据库迁移
└── docker/             # Docker配置

```

### API接口设计

#### 计费相关
- `POST /api/v1/billing/calculate` - 计算费用
- `POST /api/v1/billing/process` - 处理计费
- `GET /api/v1/billing/history` - 查询计费历史
- `POST /api/v1/billing/refund` - 退款处理

#### 配额相关
- `POST /api/v1/quotas/check` - 检查配额
- `POST /api/v1/quotas/consume` - 消费配额
- `GET /api/v1/quotas/status` - 查询配额状态
- `POST /api/v1/quotas/reset` - 重置配额

#### 余额相关
- `GET /api/v1/balance/{userID}` - 查询余额
- `POST /api/v1/balance/deduct` - 扣减余额
- `POST /api/v1/balance/add` - 增加余额

### 数据库设计

核心表：
- `users` - 用户信息（包含余额）
- `quotas` - 配额设置
- `quota_usage` - 配额使用记录
- `usage_logs` - 使用日志
- `billing_records` - 计费记录
- `model_pricing` - 模型定价

### 技术栈
- **语言**: Go 1.21+
- **框架**: Gin
- **数据库**: PostgreSQL
- **缓存**: Redis
- **监控**: Prometheus + Grafana
- **部署**: Docker + Kubernetes

### 服务通信
- **同步调用**: HTTP REST API
- **异步通信**: 消息队列（Redis Streams）
- **服务发现**: 环境变量配置