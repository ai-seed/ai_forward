# Billing 模块

## 模块概述

这是一个模块化的计费系统，设计为统一的计费入口，确保不漏扣费、防止重复扣费，并提供完整的审计和监控能力。

## 核心特性

### 🛡️ 防漏扣费机制
- **统一入口**: 所有计费都通过 `BillingManager` 处理
- **拦截器保护**: `BillingInterceptor` 确保所有API调用都经过计费流程
- **状态跟踪**: 完整的计费阶段状态管理
- **补偿机制**: 自动检测和修复漏扣费情况

### 📊 完整的审计日志
- **全链路追踪**: 从预检查到最终计费的完整日志
- **结构化日志**: 便于监控和分析的标准化日志格式
- **错误追踪**: 详细的错误信息和堆栈追踪

### 🔧 一致性保障
- **一致性检查**: 定期检查计费数据的一致性
- **自动修复**: 发现问题时自动尝试修复
- **补偿机制**: 处理各种异常情况的补偿逻辑

## 架构设计

```
billing/
├── domain/                    # 领域层
│   └── billing_context.go    # 计费上下文和核心模型
├── service/                   # 服务层
│   ├── billing_manager.go     # 计费管理器（核心入口）
│   ├── billing_audit_logger.go # 审计日志服务
│   ├── billing_consistency_checker.go # 一致性检查器
│   └── billing_compensation_service.go # 补偿服务
├── middleware/                # 中间件层
│   └── billing_interceptor.go # 计费拦截器
└── README.md                 # 文档
```

## 核心组件

### 1. BillingContext (计费上下文)
包含计费所需的所有信息，贯穿整个计费流程：
```go
type BillingContext struct {
    RequestID     string
    UserID        int64
    APIKeyID      int64
    ModelID       int64
    // ... 更多字段
}
```

### 2. BillingManager (计费管理器)
统一的计费处理入口，提供三个核心方法：

#### PreCheck - 预检查
```go
func (bm *BillingManager) PreCheck(ctx context.Context, billingCtx *BillingContext) (*PreCheckResult, error)
```
- 检查用户余额是否足够
- 检查各种配额限制
- 估算请求成本

#### ProcessRequest - 处理同步请求
```go
func (bm *BillingManager) ProcessRequest(ctx context.Context, billingCtx *BillingContext) (*BillingResult, error)
```
- 创建使用日志
- 计算实际成本
- 执行扣费操作
- 消费配额

#### ProcessAsyncCompletion - 处理异步任务完成
```go
func (bm *BillingManager) ProcessAsyncCompletion(ctx context.Context, requestID string, success bool) error
```
- 专门处理 Midjourney 等异步任务的计费

### 3. BillingInterceptor (计费拦截器)
确保所有API调用都经过计费流程：

#### PreRequestMiddleware - 请求前中间件
- 创建计费上下文
- 执行预检查
- 阻止不符合条件的请求

#### PostRequestMiddleware - 请求后中间件  
- 更新响应信息
- 异步执行计费处理

### 4. BillingAuditLogger (审计日志)
记录所有计费相关的操作：
- 预检查日志
- 计费处理日志
- 错误日志
- 异步完成日志
- 补偿操作日志

### 5. BillingConsistencyChecker (一致性检查器)
定期检查计费数据的一致性：
- 检查未计费的成功请求
- 检查计费记录一致性
- 检查用户余额一致性
- 支持自动修复

### 6. BillingCompensationService (补偿服务)
处理各种异常情况的补偿：
- 重试失败的计费
- 处理退费请求
- 调整用户余额

## 使用示例

### 1. 集成到现有代码

```go
// 在路由中添加拦截器
router.Use(billingInterceptor.PreRequestMiddleware())
router.Use(billingInterceptor.PostRequestMiddleware())

// 在处理器中获取计费上下文
if billingCtx, exists := middleware.GetBillingContext(c); exists {
    // 可以更新token信息
    billingCtx.InputTokens = inputTokens
    billingCtx.OutputTokens = outputTokens
}
```

### 2. 处理异步任务完成

```go
// Midjourney任务完成时
err := billingManager.ProcessAsyncCompletion(ctx, jobID, success)
if err != nil {
    logger.Error("Failed to process async billing completion", err)
}
```

### 3. 运行一致性检查

```go
// 检查最近24小时的未计费日志
result, err := consistencyChecker.CheckUnbilledUsageLogs(ctx, 24, true) // autoFix=true
if err != nil {
    logger.Error("Consistency check failed", err)
}
```

### 4. 处理退费

```go
// 处理退费请求
err := compensationService.ProcessRefund(ctx, requestID, refundAmount, "用户投诉")
if err != nil {
    logger.Error("Failed to process refund", err)
}
```

## 防漏扣费保障

### 1. 多层保护机制
- **拦截器保护**: 请求级别的强制计费检查
- **状态跟踪**: 完整的计费状态管理
- **一致性检查**: 定期检查和修复
- **补偿机制**: 异常情况的自动补偿

### 2. 审计能力
- **完整日志链**: 每个计费操作都有详细日志
- **结构化数据**: 便于监控和分析
- **错误追踪**: 失败原因的详细记录

### 3. 监控指标
建议监控的关键指标：
- 计费成功率
- 预检查通过率
- 一致性检查结果
- 补偿任务执行情况

### 4. 告警机制
- 计费成功率低于阈值时告警
- 发现一致性问题时告警
- 补偿任务失败时告警

## 配置和部署

### 1. 依赖注入
```go
// 创建计费管理器
billingManager := service.NewBillingManager(
    billingService,
    quotaService,
    usageLogRepo,
    userRepo,
    billingRecordRepo,
    modelPricingRepo,
    logger,
)

// 创建拦截器
billingInterceptor := middleware.NewBillingInterceptor(billingManager, logger)
```

### 2. 定时任务
```go
// 每小时运行一致性检查
go func() {
    ticker := time.NewTicker(time.Hour)
    for range ticker.C {
        consistencyChecker.RunFullConsistencyCheck(context.Background(), 24, true)
    }
}()
```

## 最佳实践

1. **总是使用拦截器**: 确保所有API端点都使用计费拦截器
2. **监控审计日志**: 建立完善的日志监控系统
3. **定期一致性检查**: 设置定时任务检查数据一致性
4. **及时处理告警**: 建立完善的告警处理流程
5. **测试覆盖**: 确保所有计费场景都有测试覆盖

这个模块化设计既保证了计费的准确性，又保持了与现有系统的良好集成，是防止漏扣费的可靠解决方案。