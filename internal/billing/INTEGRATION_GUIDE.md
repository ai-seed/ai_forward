# 计费模块集成指南

## 核心问题解答

### Q: 拦截器怎么知道要扣多少费？
**A: 通过现有中间件获取基础信息 + 业务处理器设置使用信息 + 计费服务计算成本**

## 完整集成步骤

### 1. 确保现有中间件正确设置信息

```go
// 现有的认证中间件需要设置用户信息
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... JWT验证逻辑 ...
        
        // ✅ 设置用户信息供计费使用
        c.Set("user_id", user.ID)
        c.Set("start_time", time.Now()) // 用于计算响应时间
        
        c.Next()
    }
}

// 现有的API Key验证中间件需要设置API Key信息
func APIKeyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... API Key验证逻辑 ...
        
        // ✅ 设置API Key信息供计费使用
        c.Set("api_key_id", apiKey.ID)
        
        c.Next()
    }
}
```

### 2. 添加计费拦截器到路由

```go
func setupRoutes(router *gin.Engine, billingModule *billing.BillingModule) {
    // 先添加认证中间件
    router.Use(AuthMiddleware())
    router.Use(APIKeyMiddleware())
    
    // ✅ 然后添加计费拦截器 (这个顺序很重要!)
    billingModule.SetupMiddleware(router)
    
    // 最后添加业务路由
    api := router.Group("/api/v1")
    {
        api.POST("/chat/completions", aiHandler.HandleChatCompletion)
        api.POST("/midjourney/submit", midjourneyHandler.Submit) 
        // ... 其他路由
    }
}
```

### 3. 在业务处理器中设置使用信息

#### 3.1 AI Chat 处理器示例

```go
func (h *AIHandler) HandleChatCompletion(c *gin.Context) {
    // 解析请求
    var req ChatCompletionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // ✅ 设置模型信息 (如果URL中没有模型参数)
    if modelID := h.getModelIDByName(req.Model); modelID > 0 {
        c.Set("model_id", modelID)
    }
    
    // 调用AI服务
    response, err := h.aiClient.ChatCompletion(c.Request.Context(), req)
    if err != nil {
        // ✅ 设置错误信息
        c.Set("error_message", err.Error())
        c.JSON(500, gin.H{"error": "AI service error"})
        return
    }
    
    // ✅ 设置token使用信息 (关键!)
    if response.Usage != nil {
        c.Set("input_tokens", response.Usage.PromptTokens)
        c.Set("output_tokens", response.Usage.CompletionTokens) 
        c.Set("total_tokens", response.Usage.TotalTokens)
    }
    
    // 返回响应
    c.JSON(200, response)
    
    // 计费拦截器会在这之后自动处理计费
}
```

#### 3.2 Midjourney 处理器示例

```go
func (h *MidjourneyHandler) Submit(c *gin.Context) {
    // 解析请求
    var req MidjourneyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // ✅ 设置Midjourney模型ID
    midjourneyModelID := h.getMidjourneyModelID() 
    c.Set("model_id", midjourneyModelID)
    
    // 提交任务到Midjourney
    jobID, err := h.midjourneyService.Submit(c.Request.Context(), req)
    if err != nil {
        c.Set("error_message", err.Error())
        c.JSON(500, gin.H{"error": "Failed to submit job"})
        return
    }
    
    // 返回任务ID
    c.JSON(200, gin.H{
        "job_id": jobID,
        "status": "submitted",
    })
    
    // ✅ Midjourney不在这里计费，而是在任务完成时计费
    // 计费拦截器知道这是Midjourney请求，不会立即计费
}

// 当Midjourney任务完成时
func (h *MidjourneyHandler) handleJobCompletion(jobID string, success bool) {
    // ✅ 通知计费模块异步任务完成
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := h.billingModule.ProcessAsyncCompletion(ctx, jobID, success); err != nil {
        h.logger.WithFields(map[string]interface{}{
            "job_id": jobID,
            "success": success,
            "error": err.Error(),
        }).Error("Failed to process billing for completed Midjourney job")
    }
}
```

### 4. 成本计算配置

```go
// 模型定价配置 (在数据库中)
INSERT INTO model_pricing (model_id, pricing_type, price_per_unit, multiplier, unit, currency) VALUES 
-- GPT-4 输入token定价: $0.03/1000 tokens
(1, 'input', 0.00003, 1.5, 'token', 'USD'),
-- GPT-4 输出token定价: $0.06/1000 tokens  
(1, 'output', 0.00006, 1.5, 'token', 'USD'),
-- Midjourney 按请求定价: $0.1/request
(2, 'request', 0.1, 1.2, 'request', 'USD');

// BillingService会根据这些配置自动计算成本
```

## 关键信息收集点

### 请求前收集 (PreRequestMiddleware)
```go
// 从现有中间件获取
user_id      ← AuthMiddleware 
api_key_id   ← APIKeyMiddleware
model_id     ← URL参数 /api/v1/models/{model_id} 或查询参数 ?model=xxx
method       ← c.Request.Method
endpoint     ← c.Request.URL.Path
request_type ← 根据endpoint判断 (API/Midjourney/其他)
```

### 请求后收集 (PostRequestMiddleware)  
```go
// 从处理器设置的信息获取
input_tokens  ← c.Get("input_tokens")
output_tokens ← c.Get("output_tokens") 
total_tokens  ← c.Get("total_tokens")
error_message ← c.Get("error_message")

// 从响应获取
status_code   ← c.Writer.Status()
duration      ← time.Since(start_time)
success       ← status_code >= 200 && status_code < 300
```

### 成本计算 (BillingManager)
```go
if requestType == "midjourney" {
    // 固定价格
    cost = model_pricing.price_per_unit * multiplier
} else {
    // 基于token
    input_cost = (input_tokens / 1000.0) * input_price_per_1k * multiplier
    output_cost = (output_tokens / 1000.0) * output_price_per_1k * multiplier  
    total_cost = input_cost + output_cost
}
```

## 常见问题解决

### Q1: 模型ID获取不到怎么办？
```go
// 方案1: URL参数
POST /api/v1/models/{model_id}/chat/completions

// 方案2: 查询参数  
POST /api/v1/chat/completions?model=gpt-4

// 方案3: 处理器中手动设置
func handler(c *gin.Context) {
    var req ChatRequest
    c.ShouldBindJSON(&req)
    
    // 根据模型名称查找ID
    modelID := h.modelService.GetIDByName(req.Model)
    c.Set("model_id", modelID)
}
```

### Q2: Token数量获取不到怎么办？
```go
func handler(c *gin.Context) {
    response, err := h.aiClient.Call(req)
    if err != nil {
        // 即使出错也要设置，方便调试
        c.Set("error_message", err.Error())
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // ✅ 关键: 一定要设置token信息
    if response.Usage != nil {
        c.Set("input_tokens", response.Usage.InputTokens)
        c.Set("output_tokens", response.Usage.OutputTokens)
    } else {
        // 如果AI服务没返回用量，可以估算
        c.Set("input_tokens", h.estimateInputTokens(req))
        c.Set("output_tokens", h.estimateOutputTokens(response))
    }
    
    c.JSON(200, response)
}
```

### Q3: 想跳过某些请求不计费？
```go
// 在BillingInterceptor构造函数中配置
skipPaths := map[string]bool{
    "/health":        true,
    "/metrics":       true, 
    "/api/v1/models": true,  // 模型列表接口
    "/api/v1/auth":   true,  // 认证接口
}
```

### Q4: 想在处理器中获取计费上下文？
```go
func handler(c *gin.Context) {
    // 获取计费上下文
    if billingCtx, exists := middleware.GetBillingContext(c); exists {
        // 可以检查预估成本
        estimatedCost := billingCtx.EstimatedCost
        
        // 可以更新信息
        billingCtx.InputTokens = calculatedInputTokens
    }
}
```

## 集成检查清单

- [ ] 认证中间件设置了 `user_id`
- [ ] API Key中间件设置了 `api_key_id` 
- [ ] 计费拦截器添加到了正确位置 (在认证之后，业务处理器之前)
- [ ] 业务处理器设置了模型信息 `model_id`
- [ ] 业务处理器设置了使用信息 `input_tokens`, `output_tokens`
- [ ] 异步任务完成时调用了 `ProcessAsyncCompletion`
- [ ] 数据库中配置了模型定价信息
- [ ] 启动了后台一致性检查任务

这样设计后，计费拦截器就能获取到所有需要的信息来进行准确的计费了！