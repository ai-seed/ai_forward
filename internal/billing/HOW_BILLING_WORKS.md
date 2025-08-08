# 计费拦截器工作原理详解

## 问题回答

### Q: 拦截器是怎么知道要怎么扣费的？
### A: 拦截器通过多个阶段逐步收集和更新计费信息

## 工作流程图

```
1. 请求前 (PreRequestMiddleware)
   ├── 从认证中间件获取基础信息
   │   ├── user_id (从JWT token或API key验证中间件)
   │   ├── api_key_id (从API key验证中间件)
   │   └── model_id (从URL参数或请求体)
   │
   ├── 创建初始BillingContext
   │   ├── 基础信息: RequestID, UserID, APIKeyID, ModelID
   │   ├── 请求信息: Method, Endpoint, RequestType
   │   ├── 时间信息: RequestTime
   │   └── 计费阶段: PreCheck
   │
   └── 预检查 (估算成本 + 检查余额配额)

2. 请求处理中 (业务处理器)
   ├── 处理器可以更新BillingContext
   │   ├── c.Set("input_tokens", inputTokens)
   │   ├── c.Set("output_tokens", outputTokens) 
   │   ├── c.Set("total_tokens", totalTokens)
   │   └── c.Set("error_message", errorMsg)
   │
   └── 请求处理完成

3. 请求后 (PostRequestMiddleware)
   ├── 从Gin Context更新BillingContext
   │   ├── 状态码: c.Writer.Status()
   │   ├── Token信息: input_tokens, output_tokens
   │   ├── 响应时间: 计算duration
   │   └── 错误信息: error_message
   │
   ├── 判断是否需要计费
   │   ├── 成功请求: Status 200-299
   │   ├── 非异步任务: RequestType != Midjourney  
   │   └── 有实际使用: TokenCount > 0 或 RequestType基于请求计费
   │
   └── 异步执行计费
       └── BillingManager.ProcessRequest(billingContext)
```

## BillingContext 包含的完整信息

```go
type BillingContext struct {
    // === 1. 身份信息 (请求前获取) ===
    RequestID    string    // UUID，用于追踪
    UserID       int64     // 从认证中间件获取
    APIKeyID     int64     // 从API Key验证中间件获取
    ModelID      int64     // 从URL参数或请求体获取
    ProviderID   int64     // 从模型信息查询获取
    
    // === 2. 请求信息 (请求前获取) ===
    Method       string    // HTTP方法: GET/POST/PUT
    Endpoint     string    // API端点: /api/v1/chat/completions
    RequestType  entities.RequestType // API类型: API/Midjourney
    RequestTime  time.Time // 请求开始时间
    
    // === 3. 使用信息 (请求后更新) ===
    InputTokens  int      // 输入token数 (从处理器设置)
    OutputTokens int      // 输出token数 (从处理器设置)
    TotalTokens  int      // 总token数 (从处理器设置或计算)
    
    // === 4. 成本信息 (计费时计算) ===
    EstimatedCost float64  // 预估成本 (预检查时计算)
    ActualCost    float64  // 实际成本 (最终计费时计算)
    
    // === 5. 响应信息 (请求后更新) ===
    Status       int      // HTTP状态码
    DurationMs   int      // 请求耗时(毫秒)
    Success      bool     // 是否成功 (200-299)
    ErrorMessage string   // 错误信息
    
    // === 6. 计费状态 (全流程跟踪) ===
    BillingStage BillingStage // 计费阶段
    IsBilled     bool         // 是否已计费
    BillingError string       // 计费错误信息
}
```

## 具体获取信息的方式

### 1. 基础信息获取 (请求前)

```go
// extractRequestInfo 从现有中间件获取信息
func (bi *BillingInterceptor) extractRequestInfo(c *gin.Context) (userID, apiKeyID, modelID int64, err error) {
    // 用户ID - 从JWT验证中间件获取
    if userIDInterface, exists := c.Get("user_id"); exists {
        userID = userIDInterface.(int64)
    }
    
    // API Key ID - 从API Key验证中间件获取  
    if apiKeyIDInterface, exists := c.Get("api_key_id"); exists {
        apiKeyID = apiKeyIDInterface.(int64)
    }
    
    // 模型ID - 从URL参数获取
    if modelParam := c.Param("model"); modelParam != "" {
        modelID, _ = strconv.ParseInt(modelParam, 10, 64)
    }
    
    // 或从查询参数获取
    if modelID == 0 {
        if modelQuery := c.Query("model"); modelQuery != "" {
            modelID, _ = strconv.ParseInt(modelQuery, 10, 64)
        }
    }
}
```

### 2. 请求类型判断

```go
// createBillingContext 根据URL判断请求类型
func (bi *BillingInterceptor) createBillingContext(...) *domain.BillingContext {
    requestType := entities.RequestTypeAPI
    
    // 特殊路径的特殊处理
    if c.Request.URL.Path == "/api/v1/midjourney" || 
       c.Request.URL.Path == "/api/v1/midjourney/submit" {
        requestType = entities.RequestTypeMidjourney
    }
    
    // 可以根据需要添加更多类型判断
    // if strings.Contains(path, "/vectorizer") {
    //     requestType = entities.RequestTypeVectorizer
    // }
}
```

### 3. Token信息更新 (业务处理器设置)

```go
// 在AI处理器中，处理完AI请求后：
func aiHandler(c *gin.Context) {
    // ... 调用AI服务 ...
    
    response := callAIService(request)
    
    // 设置token信息供计费使用
    c.Set("input_tokens", response.Usage.InputTokens)
    c.Set("output_tokens", response.Usage.OutputTokens) 
    c.Set("total_tokens", response.Usage.TotalTokens)
    
    // 如果有错误也要设置
    if response.Error != nil {
        c.Set("error_message", response.Error.Message)
    }
    
    c.JSON(200, response)
}
```

### 4. 响应信息更新 (请求后)

```go
// updateBillingContextWithResponse 从Gin Context更新响应信息
func (bi *BillingInterceptor) updateBillingContextWithResponse(c *gin.Context, billingCtx *domain.BillingContext) {
    // 状态码
    billingCtx.Status = c.Writer.Status()
    billingCtx.Success = billingCtx.Status >= 200 && billingCtx.Status < 300
    
    // Token信息
    if inputTokens, exists := c.Get("input_tokens"); exists {
        billingCtx.InputTokens = inputTokens.(int)
    }
    
    // 响应时间
    if startTime, exists := c.Get("start_time"); exists {
        billingCtx.DurationMs = int(time.Since(startTime.(time.Time)).Milliseconds())
    }
    
    // 错误信息
    if errorMsg, exists := c.Get("error_message"); exists {
        billingCtx.ErrorMessage = errorMsg.(string)
    }
}
```

## 成本计算逻辑

```go
// BillingManager 根据上下文信息计算成本
func (bm *BillingManager) estimateCost(ctx context.Context, billingCtx *domain.BillingContext) (float64, error) {
    if billingCtx.RequestType == entities.RequestTypeMidjourney {
        // Midjourney按请求计费 - 固定价格
        return bm.billingService.CalculateRequestCost(ctx, billingCtx.ModelID)
    }
    
    // 普通API按token计费 - 基于输入输出token数
    return bm.billingService.CalculateCost(
        ctx, 
        billingCtx.ModelID, 
        billingCtx.CalculateInputTokens(),  // 从上下文获取
        billingCtx.CalculateOutputTokens(), // 从上下文获取
    )
}
```

## 关键设计点

### 1. 分阶段收集信息
- **请求前**: 收集身份和基础信息，进行预检查
- **处理中**: 业务处理器更新使用信息
- **请求后**: 收集响应信息，执行计费

### 2. 依赖现有中间件
- 不需要重复验证，依赖现有的认证和授权中间件
- 通过Gin Context传递信息

### 3. 容错设计
- 信息缺失时有默认值和跳过逻辑
- 计费失败不影响正常响应

### 4. 扩展性
- 可以根据URL路径判断不同的计费类型
- 处理器可以灵活设置计费相关信息

这样设计的好处是：
1. **信息完整**: 收集了计费所需的所有信息
2. **时机准确**: 在正确的时间点获取和更新信息  
3. **依赖清晰**: 明确依赖关系，不重复获取信息
4. **容错处理**: 信息缺失时有合理的默认行为