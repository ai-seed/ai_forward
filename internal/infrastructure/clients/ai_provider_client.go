package clients

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-api-gateway/internal/domain/entities"
)

// AIProviderResponse AI提供商响应（包含原始数据和解析后的结构体）
type AIProviderResponse struct {
	Response    *AIResponse `json:"response"`     // 解析后的结构化响应
	RawResponse []byte      `json:"raw_response"` // 原始响应数据
}

// AIProviderClient AI提供商客户端接口
type AIProviderClient interface {
	// SendRequest 发送请求到AI提供商
	SendRequest(ctx context.Context, provider *entities.Provider, request *AIRequest) (*AIProviderResponse, error)

	// SendStreamRequest 发送流式请求到AI提供商
	SendStreamRequest(ctx context.Context, provider *entities.Provider, request *AIRequest, streamChan chan<- *StreamChunk) error

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context, provider *entities.Provider) error

	// GetModels 获取提供商支持的模型列表
	GetModels(ctx context.Context, provider *entities.Provider) ([]*AIModel, error)
}

// StreamChunk 流式响应数据块
type StreamChunk struct {
	ID               string   `json:"id"`
	Object           string   `json:"object"`
	Created          int64    `json:"created"`
	Model            string   `json:"model"`
	Content          string   `json:"content"`
	ReasoningContent string   `json:"reasoning_content,omitempty"` // Claude thinking模型的推理内容
	ContentType      string   `json:"content_type,omitempty"`      // "thinking" | "response"
	FinishReason     *string  `json:"finish_reason"`
	Usage            *AIUsage `json:"usage,omitempty"`
	Cost             *AICost  `json:"cost,omitempty"`
	RawData          []byte   `json:"-"` // 原始SSE数据，不序列化到JSON
}

// AIRequest AI请求 (通用结构)
type AIRequest struct {
	Model       string                 `json:"model"`
	Messages    []AIMessage            `json:"messages,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Tools       []Tool                 `json:"tools,omitempty"`       // Function call tools
	ToolChoice  interface{}            `json:"tool_choice,omitempty"` // Tool choice strategy
	WebSearch   bool                   `json:"web_search,omitempty"`  // 是否启用联网搜索
	Thinking    *ThinkingConfig        `json:"thinking,omitempty"`    // 深度思考配置
	Extra       map[string]interface{} `json:"-"`                     // 额外参数
}

// ChatCompletionRequest 聊天补全请求
type ChatCompletionRequest struct {
	Model       string          `json:"model" binding:"required" example:"gpt-3.5-turbo"`
	Messages    []AIMessage     `json:"messages" binding:"required,min=1"`
	MaxTokens   int             `json:"max_tokens,omitempty" example:"150"`
	Temperature float64         `json:"temperature,omitempty" example:"0.7"`
	Stream      bool            `json:"stream,omitempty" example:"false"`
	Tools       []Tool          `json:"tools,omitempty"`                      // Function call tools
	ToolChoice  interface{}     `json:"tool_choice,omitempty"`                // Tool choice strategy
	WebSearch   bool            `json:"web_search,omitempty" example:"false"` // 是否启用联网搜索
	Thinking    *ThinkingConfig `json:"thinking,omitempty"`                   // 深度思考配置
}

// CompletionRequest 文本补全请求
type CompletionRequest struct {
	Model       string  `json:"model" binding:"required" example:"gpt-3.5-turbo"`
	Prompt      string  `json:"prompt" binding:"required" example:"Once upon a time"`
	MaxTokens   int     `json:"max_tokens,omitempty" example:"150"`
	Temperature float64 `json:"temperature,omitempty" example:"0.7"`
	Stream      bool    `json:"stream,omitempty" example:"false"`
	WebSearch   bool    `json:"web_search,omitempty" example:"false"` // 是否启用联网搜索
}

// AnthropicMessageRequest Anthropic Messages API 请求结构
type AnthropicMessageRequest struct {
	Model         string                   `json:"model" binding:"required" example:"claude-3-sonnet-20240229" swaggertype:"string" description:"要使用的模型名称，如 claude-3-sonnet-20240229, claude-3-haiku-20240307 等"`
	Messages      []AnthropicMessage       `json:"messages" binding:"required,min=1" description:"对话消息数组，至少包含一条消息"`
	MaxTokens     int                      `json:"max_tokens" binding:"required" example:"1024" minimum:"1" maximum:"4096" description:"生成的最大token数量"`
	Temperature   *float64                 `json:"temperature,omitempty" example:"0.7" minimum:"0" maximum:"1" description:"控制输出随机性，0-1之间，值越高越随机"`
	Stream        bool                     `json:"stream,omitempty" example:"false" description:"是否启用流式响应"`
	System        interface{}              `json:"system,omitempty" swaggertype:"string" example:"You are a helpful assistant." description:"系统提示，可以是字符串或数组格式"`
	StopSequences []string                 `json:"stop_sequences,omitempty" example:"[\"\\n\\n\"]" description:"停止序列，遇到这些字符串时停止生成"`
	TopK          *int                     `json:"top_k,omitempty" example:"5" minimum:"1" description:"Top-K采样，从概率最高的K个token中选择"`
	TopP          *float64                 `json:"top_p,omitempty" example:"0.7" minimum:"0" maximum:"1" description:"Top-P采样，累积概率达到P时停止"`
	Tools         []AnthropicTool          `json:"tools,omitempty" description:"可用的工具列表"`
	ToolChoice    interface{}              `json:"tool_choice,omitempty" swaggertype:"string" example:"auto" description:"工具选择策略：auto, any, none 或指定工具名"`
	Metadata      *AnthropicMetadata       `json:"metadata,omitempty" description:"请求元数据"`
	ServiceTier   string                   `json:"service_tier,omitempty" example:"auto" enum:"auto,standard_only" description:"服务层级"`
	Container     *string                  `json:"container,omitempty" description:"容器标识符"`
	MCPServers    []AnthropicMCPServer     `json:"mcp_servers,omitempty" description:"MCP服务器配置"`
	Thinking      *AnthropicThinkingConfig `json:"thinking,omitempty" description:"思考配置"`
}

// ClaudeMessageRequest Claude消息请求 (保持向后兼容)
type ClaudeMessageRequest struct {
	Model         string          `json:"model" binding:"required" example:"claude-3-sonnet-20240229"`
	Messages      []ClaudeMessage `json:"messages" binding:"required,min=1"`
	MaxTokens     int             `json:"max_tokens" binding:"required" example:"1024"`
	Temperature   *float64        `json:"temperature,omitempty" example:"0.7"`
	Stream        bool            `json:"stream,omitempty" example:"false"`
	System        interface{}     `json:"system,omitempty"` // 支持字符串或数组格式
	StopSequences []string        `json:"stop_sequences,omitempty"`
	TopK          *int            `json:"top_k,omitempty" example:"5"`
	TopP          *float64        `json:"top_p,omitempty" example:"0.7"`
	Tools         []Tool          `json:"tools,omitempty"`
	ToolChoice    interface{}     `json:"tool_choice,omitempty"`
	WebSearch     bool            `json:"web_search,omitempty" example:"false"` // 是否启用联网搜索

	// Claude特有字段
	Metadata    *ClaudeMetadata `json:"metadata,omitempty"`
	ServiceTier string          `json:"service_tier,omitempty"` // "auto", "standard_only"
}

// ClaudeMetadata Claude元数据
type ClaudeMetadata struct {
	UserID string `json:"user_id,omitempty"` // 外部用户标识符
}

// ClaudeContentBlock Claude内容块
type ClaudeContentBlock struct {
	Type   string `json:"type"`           // "text", "image", "tool_use", "tool_result"
	Text   string `json:"text,omitempty"` // 文本内容
	Source *struct {
		Type      string `json:"type"`       // "base64"
		MediaType string `json:"media_type"` // "image/jpeg", "image/png", etc.
		Data      string `json:"data"`       // base64编码的图片数据
	} `json:"source,omitempty"` // 图片源
	ID        string      `json:"id,omitempty"`          // tool_use的ID
	Name      string      `json:"name,omitempty"`        // tool_use的名称
	Input     interface{} `json:"input,omitempty"`       // tool_use的输入
	ToolUseID string      `json:"tool_use_id,omitempty"` // tool_result对应的tool_use_id
	Content   interface{} `json:"content,omitempty"`     // tool_result的内容
	IsError   bool        `json:"is_error,omitempty"`    // tool_result是否为错误
}

// ClaudeMessage Claude消息格式
type ClaudeMessage struct {
	Role    string      `json:"role"`    // "user", "assistant"
	Content interface{} `json:"content"` // 可以是string或[]ClaudeContentBlock
}

// UnmarshalJSON 自定义JSON解析
func (cm *ClaudeMessage) UnmarshalJSON(data []byte) error {
	// 先解析基本结构
	type Alias ClaudeMessage
	aux := &struct {
		*Alias
		Content json.RawMessage `json:"content"`
	}{
		Alias: (*Alias)(cm),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 尝试解析content为字符串
	var str string
	if err := json.Unmarshal(aux.Content, &str); err == nil {
		cm.Content = str
		return nil
	}

	// 尝试解析content为数组
	var blocks []ClaudeContentBlock
	if err := json.Unmarshal(aux.Content, &blocks); err == nil {
		cm.Content = blocks
		return nil
	}

	return fmt.Errorf("content must be string or array of content blocks")
}

// GetTextContent 获取文本内容
func (cm *ClaudeMessage) GetTextContent() string {
	if str, ok := cm.Content.(string); ok {
		return str
	}

	if blocks, ok := cm.Content.([]ClaudeContentBlock); ok {
		for _, block := range blocks {
			if block.Type == "text" {
				return block.Text
			}
		}
	}

	return ""
}

// AIMessage AI消息 (保持向后兼容)
type AIMessage struct {
	Role       string     `json:"role" binding:"required" example:"user" enums:"system,user,assistant,tool"`
	Content    string     `json:"content" example:"Hello, how are you?"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // Function calls made by assistant
	ToolCallID string     `json:"tool_call_id,omitempty"` // ID of the tool call this message is responding to
	Name       string     `json:"name,omitempty"`         // Name of the function for tool messages
}

// AIResponse AI响应
type AIResponse struct {
	ID      string     `json:"id"`
	Object  string     `json:"object"`
	Created int64      `json:"created"`
	Model   string     `json:"model"`
	Choices []AIChoice `json:"choices"`
	Usage   AIUsage    `json:"usage"`
	Error   *AIError   `json:"error,omitempty"`
}

// AIChoice AI选择
type AIChoice struct {
	Index        int        `json:"index"`
	Message      AIMessage  `json:"message,omitempty"`
	Text         string     `json:"text,omitempty"`
	FinishReason string     `json:"finish_reason"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"` // Function calls in streaming mode
}

// AIUsage AI使用情况
type AIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AICost AI成本信息
type AICost struct {
	PromptCost     float64 `json:"prompt_cost"`
	CompletionCost float64 `json:"completion_cost"`
	TotalCost      float64 `json:"total_cost"`
}

// AIError AI错误
type AIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// ClaudeMessageResponse Claude消息响应
type ClaudeMessageResponse struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Content      []ClaudeContent `json:"content"`
	Model        string          `json:"model"`
	StopReason   string          `json:"stop_reason"`
	StopSequence *string         `json:"stop_sequence"`
	Usage        ClaudeUsage     `json:"usage"`
	Error        *AIError        `json:"error,omitempty"`
}

// ClaudeContent Claude内容块
type ClaudeContent struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	ToolUse  *ToolCall   `json:"tool_use,omitempty"`
	ToolCall *ToolCall   `json:"tool_call,omitempty"` // 兼容性字段
	ID       string      `json:"id,omitempty"`
	Name     string      `json:"name,omitempty"`
	Input    interface{} `json:"input,omitempty"`
}

// ClaudeUsage Claude使用情况
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AIModel AI模型信息
type AIModel struct {
	ID         string        `json:"id"`
	Object     string        `json:"object"`
	Created    int64         `json:"created"`
	OwnedBy    string        `json:"owned_by"`
	Permission []interface{} `json:"permission"`
}

// ModelsResponse 模型列表响应
type ModelsResponse struct {
	Object string    `json:"object"`
	Data   []AIModel `json:"data"`
}

// UsageResponse 使用情况响应
type UsageResponse struct {
	TotalRequests int     `json:"total_requests"`
	TotalTokens   int     `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
}

// Tool Function call tool definition
type Tool struct {
	Type     string   `json:"type" example:"function"`
	Function Function `json:"function"`
}

// Function Function definition for tool calls
type Function struct {
	Name        string      `json:"name" example:"search"`
	Description string      `json:"description" example:"Search for information"`
	Parameters  interface{} `json:"parameters"` // JSON Schema for function parameters
}

// ToolCall Function call made by the assistant
type ToolCall struct {
	ID       string       `json:"id" example:"call_123"`
	Type     string       `json:"type" example:"function"`
	Function FunctionCall `json:"function"`
}

// FunctionCall Function call details
type FunctionCall struct {
	Name      string `json:"name" example:"search"`
	Arguments string `json:"arguments"` // JSON string of function arguments
}

// aiProviderClientImpl AI提供商客户端实现
type aiProviderClientImpl struct {
	httpClient HTTPClient
}

// NewAIProviderClient 创建AI提供商客户端
func NewAIProviderClient(httpClient HTTPClient) AIProviderClient {
	return &aiProviderClientImpl{
		httpClient: httpClient,
	}
}

// SendRequest 发送请求到AI提供商
func (c *aiProviderClientImpl) SendRequest(ctx context.Context, provider *entities.Provider, request *AIRequest) (*AIProviderResponse, error) {
	// 构造请求URL
	url := fmt.Sprintf("%s/v1/chat/completions", provider.BaseURL)

	// 构造请求头
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	fmt.Printf("[AI_CLIENT] 开始处理请求 - Provider: %s (%s), Model: %s\n", provider.Name, provider.Slug, request.Model)

	// 根据提供商类型设置认证头
	switch provider.Slug {
	case "302":
		if provider.APIKeyEncrypted != nil {
			// TODO: 解密API密钥
			headers["Authorization"] = fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted)
		}
		fmt.Printf("[AI_CLIENT] 使用 302 格式 - URL: %s\n", url)
	case "openai":
		if provider.APIKeyEncrypted != nil {
			// TODO: 解密API密钥
			headers["Authorization"] = fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted)
		}
		fmt.Printf("[AI_CLIENT] 使用 OpenAI 格式 - URL: %s\n", url)
	case "anthropic":
		if provider.APIKeyEncrypted != nil {
			// TODO: 解密API密钥
			headers["x-api-key"] = *provider.APIKeyEncrypted
			headers["anthropic-version"] = "2023-06-01"
		}
		// Anthropic使用不同的端点 - 根据文档应该是 /v1/messages
		url = fmt.Sprintf("%s/v1/messages", provider.BaseURL)
		fmt.Printf("[AI_CLIENT] 使用 Anthropic 格式 - URL: %s\n", url)
	case "your-proxy-dual":
		// 支持双格式的中转服务
		if provider.APIKeyEncrypted != nil {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted)
		}
		// 根据模型类型决定使用哪种端点
		if c.isAnthropicModel(request.Model) {
			url = fmt.Sprintf("%s/v1/messages", provider.BaseURL)
			headers["anthropic-version"] = "2023-06-01"
		} else {
			url = fmt.Sprintf("%s/v1/chat/completions", provider.BaseURL)
		}
	}

	// 打印请求详情
	fmt.Printf("[AI_CLIENT] 准备发送 HTTP 请求:\n")
	fmt.Printf("  - URL: %s\n", url)
	fmt.Printf("  - Method: POST\n")
	fmt.Printf("  - Headers: %+v\n", headers)
	fmt.Printf("  - Model: %s\n", request.Model)
	fmt.Printf("  - Messages Count: %d\n", len(request.Messages))
	fmt.Printf("  - Max Tokens: %d\n", request.MaxTokens)
	fmt.Printf("  - Stream: %t\n", request.Stream)

	// 发送请求
	fmt.Printf("[AI_CLIENT] 开始发送 HTTP 请求到: %s\n", url)
	resp, err := c.httpClient.Post(ctx, url, request, headers)
	if err != nil {
		fmt.Printf("[AI_CLIENT] HTTP 请求失败: %v\n", err)
		return nil, fmt.Errorf("failed to send request to provider %s: %w", provider.Name, err)
	}

	fmt.Printf("[AI_CLIENT] 收到 HTTP 响应:\n")
	fmt.Printf("  - Status Code: %d\n", resp.StatusCode)
	fmt.Printf("  - Content Length: %d bytes\n", len(resp.Body))

	// 保存原始响应数据
	rawResponse := make([]byte, len(resp.Body))
	copy(rawResponse, resp.Body)

	// 解析响应
	var aiResp AIResponse

	if err := resp.UnmarshalJSON(&aiResp); err != nil {
		fmt.Printf("[AI_CLIENT] 解析响应失败: %v\n", err)
		fmt.Printf("[AI_CLIENT] 原始响应内容: %s\n", string(resp.Body))
		return nil, fmt.Errorf("failed to unmarshal response from provider %s: %w", provider.Name, err)
	}

	fmt.Printf("[AI_CLIENT] 响应解析成功:\n")
	fmt.Printf("  - Response ID: %s\n", aiResp.ID)
	fmt.Printf("  - Model: %s\n", aiResp.Model)
	fmt.Printf("  - Choices Count: %d\n", len(aiResp.Choices))
	fmt.Printf("  - Prompt Tokens: %d\n", aiResp.Usage.PromptTokens)
	fmt.Printf("  - Completion Tokens: %d\n", aiResp.Usage.CompletionTokens)
	fmt.Printf("  - Total Tokens: %d\n", aiResp.Usage.TotalTokens)

	// 构造包含原始数据和解析数据的响应
	providerResponse := &AIProviderResponse{
		Response:    &aiResp,
		RawResponse: rawResponse,
	}

	// 检查是否有错误
	if aiResp.Error != nil {
		fmt.Printf("[AI_CLIENT] 提供商返回错误: %s\n", aiResp.Error.Message)
		return providerResponse, fmt.Errorf("provider %s returned error: %s", provider.Name, aiResp.Error.Message)
	}

	fmt.Printf("[AI_CLIENT] 请求处理完成，返回响应\n")
	return providerResponse, nil
}

// HealthCheck 健康检查
func (c *aiProviderClientImpl) HealthCheck(ctx context.Context, provider *entities.Provider) error {
	// 使用模型列表端点进行健康检查
	url := fmt.Sprintf("%s/models", provider.BaseURL)

	// 构造请求头
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// 根据提供商类型设置认证头
	switch provider.Slug {
	case "openai":
		if provider.APIKeyEncrypted != nil {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted)
		}
	case "anthropic":
		if provider.APIKeyEncrypted != nil {
			headers["x-api-key"] = *provider.APIKeyEncrypted
			headers["anthropic-version"] = "2023-06-01"
		}
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(provider.TimeoutSeconds)*time.Second)
	defer cancel()

	// 发送请求
	resp, err := c.httpClient.Get(ctx, url, headers)
	if err != nil {
		return fmt.Errorf("health check failed for provider %s: %w", provider.Name, err)
	}

	// 检查响应状态
	if !resp.IsSuccess() {
		return fmt.Errorf("health check failed for provider %s: status %d", provider.Name, resp.StatusCode)
	}

	return nil
}

// GetModels 获取提供商支持的模型列表
func (c *aiProviderClientImpl) GetModels(ctx context.Context, provider *entities.Provider) ([]*AIModel, error) {
	// 构造请求URL
	url := fmt.Sprintf("%s/models", provider.BaseURL)

	// 构造请求头
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// 根据提供商类型设置认证头
	switch provider.Slug {
	case "openai":
		if provider.APIKeyEncrypted != nil {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted)
		}
	case "anthropic":
		if provider.APIKeyEncrypted != nil {
			headers["x-api-key"] = *provider.APIKeyEncrypted
			headers["anthropic-version"] = "2023-06-01"
		}
	}

	// 发送请求
	resp, err := c.httpClient.Get(ctx, url, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get models from provider %s: %w", provider.Name, err)
	}

	// 解析响应
	var modelsResp struct {
		Data []AIModel `json:"data"`
	}
	if err := resp.UnmarshalJSON(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response from provider %s: %w", provider.Name, err)
	}

	// 转换为指针切片
	models := make([]*AIModel, len(modelsResp.Data))
	for i := range modelsResp.Data {
		models[i] = &modelsResp.Data[i]
	}

	return models, nil
}

// SendStreamRequest 发送流式请求到AI提供商
func (c *aiProviderClientImpl) SendStreamRequest(ctx context.Context, provider *entities.Provider, request *AIRequest, streamChan chan<- *StreamChunk) error {
	// 确保请求是流式的
	request.Stream = true

	// 构造请求URL
	url := fmt.Sprintf("%s/v1/chat/completions", provider.BaseURL)

	// 构造请求头
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "text/event-stream",
		"Cache-Control": "no-cache",
	}

	// 根据提供商类型设置认证头
	switch provider.Slug {
	case "302":
		if provider.APIKeyEncrypted != nil {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted)
		}
	case "openai":
		if provider.APIKeyEncrypted != nil {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", *provider.APIKeyEncrypted)
		}
	case "anthropic":
		if provider.APIKeyEncrypted != nil {
			headers["x-api-key"] = *provider.APIKeyEncrypted
			headers["anthropic-version"] = "2023-06-01"
		}
		// Anthropic使用不同的端点 - 根据文档应该是 /v1/messages
		url = fmt.Sprintf("%s/v1/messages", provider.BaseURL)
	}

	// 发送流式请求
	return c.sendStreamRequestToProvider(ctx, url, request, headers, streamChan)
}

// 工厂方法
func NewOpenAIClient(httpClient HTTPClient) AIProviderClient {
	return NewAIProviderClient(httpClient)
}

func NewAnthropicClient(httpClient HTTPClient) AIProviderClient {
	return NewAIProviderClient(httpClient)
}

// sendStreamRequestToProvider 发送流式请求到提供商的具体实现
func (c *aiProviderClientImpl) sendStreamRequestToProvider(ctx context.Context, url string, request *AIRequest, headers map[string]string, streamChan chan<- *StreamChunk) error {
	// 序列化请求体
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 设置请求头
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 处理流式响应
	return c.processStreamResponse(ctx, resp.Body, streamChan, request.Model)
}

// processStreamResponse 处理流式响应
func (c *aiProviderClientImpl) processStreamResponse(ctx context.Context, body io.Reader, streamChan chan<- *StreamChunk, model string) error {
	scanner := bufio.NewScanner(body)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line := scanner.Text()

			// 跳过空行和注释行
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			// 保存原始SSE行数据
			rawSSELine := line + "\n"

			// 移除 "data: " 前缀
			data := strings.TrimPrefix(line, "data: ")

			// 检查是否是结束标记
			if data == "[DONE]" {
				// 发送结束标记的原始数据
				chunk := &StreamChunk{
					RawData: []byte(rawSSELine),
				}
				select {
				case streamChan <- chunk:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}

			// 创建基础chunk，无论解析是否成功都包含原始数据
			chunk := &StreamChunk{
				RawData: []byte(rawSSELine),
			}

			// 尝试解析JSON数据以提取元数据（仅用于内部处理，如token统计）
			var sseData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &sseData); err == nil {
				// 提取内容（用于内部处理，如token统计）
				content := ""
				reasoningContent := ""
				contentType := ""

				if choices, ok := sseData["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if delta, ok := choice["delta"].(map[string]interface{}); ok {
							// 提取普通内容
							if deltaContent, ok := delta["content"].(string); ok {
								content = deltaContent
							}

							// 提取推理内容（Claude thinking模型）
							if deltaReasoningContent, ok := delta["reasoning_content"].(string); ok {
								reasoningContent = deltaReasoningContent
								contentType = "thinking" // 标记为thinking内容
							}

							// 提取content_type（如果有）
							if deltaContentType, ok := delta["content_type"].(string); ok {
								contentType = deltaContentType
							}
						}
					}
				}

				// 填充解析出的元数据到chunk中
				chunk.ID = fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
				chunk.Object = "chat.completion.chunk"
				chunk.Created = time.Now().Unix()
				chunk.Model = model
				chunk.Content = content
				chunk.ReasoningContent = reasoningContent
				chunk.ContentType = contentType
			}
			// 如果解析失败，chunk只包含原始数据，这也是可以的

			// 发送数据块（无论解析是否成功都发送）
			select {
			case streamChan <- chunk:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}

// ===== Anthropic Messages API 结构定义 =====

// AnthropicMessage Anthropic 消息格式
type AnthropicMessage struct {
	Role    string      `json:"role" example:"user" enum:"user,assistant" description:"消息角色：user(用户) 或 assistant(助手)"`
	Content interface{} `json:"content" swaggertype:"string" example:"Hello, how are you?" description:"消息内容，可以是字符串或内容块数组"`
}

// GetTextContent 获取文本内容
func (am *AnthropicMessage) GetTextContent() string {
	switch content := am.Content.(type) {
	case string:
		return content
	case []interface{}:
		for _, block := range content {
			if blockMap, ok := block.(map[string]interface{}); ok {
				if blockType, ok := blockMap["type"].(string); ok && blockType == "text" {
					if text, ok := blockMap["text"].(string); ok {
						return text
					}
				}
			}
		}
	case []AnthropicContentBlock:
		for _, block := range content {
			if block.Type == "text" {
				return block.Text
			}
		}
	}
	return ""
}

// AnthropicContentBlock Anthropic 内容块
type AnthropicContentBlock struct {
	Type   string `json:"type" example:"text" enum:"text,image,tool_use,tool_result,thinking" description:"内容块类型"`
	Text   string `json:"text,omitempty" example:"Hello, how can I help you?" description:"文本内容（当type为text或thinking时）"`
	Source *struct {
		Type      string `json:"type" example:"base64" description:"图片数据类型"`
		MediaType string `json:"media_type" example:"image/jpeg" description:"图片MIME类型"`
		Data      string `json:"data" description:"base64编码的图片数据"`
	} `json:"source,omitempty" description:"图片源（当type为image时）"`
	ID        string      `json:"id,omitempty" example:"toolu_01A09q90qw90lq917835lq9" description:"工具使用的唯一ID（当type为tool_use时）"`
	Name      string      `json:"name,omitempty" example:"get_weather" description:"工具名称（当type为tool_use时）"`
	Input     interface{} `json:"input,omitempty" description:"工具输入参数（当type为tool_use时）"`
	ToolUseID string      `json:"tool_use_id,omitempty" description:"对应的tool_use ID（当type为tool_result时）"`
	Content   interface{} `json:"content,omitempty" description:"工具结果内容（当type为tool_result时）"`
	IsError   bool        `json:"is_error,omitempty" description:"是否为错误结果（当type为tool_result时）"`
}

// AnthropicTool Anthropic 工具定义
type AnthropicTool struct {
	Name         string                 `json:"name" binding:"required"`
	Description  string                 `json:"description,omitempty"`
	InputSchema  map[string]interface{} `json:"input_schema" binding:"required"`
	Type         string                 `json:"type,omitempty"` // "custom" 等
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicCacheControl 缓存控制
type AnthropicCacheControl struct {
	Type string `json:"type"` // "ephemeral"
	TTL  string `json:"ttl"`  // "5m", "1h"
}

// AnthropicMetadata Anthropic 元数据
type AnthropicMetadata struct {
	UserID string `json:"user_id,omitempty"` // 外部用户标识符
}

// AnthropicMCPServer MCP 服务器配置
type AnthropicMCPServer struct {
	Name               string                      `json:"name" binding:"required"`
	Type               string                      `json:"type" binding:"required"` // "url"
	URL                string                      `json:"url" binding:"required"`
	AuthorizationToken *string                     `json:"authorization_token,omitempty"`
	ToolConfiguration  *AnthropicToolConfiguration `json:"tool_configuration,omitempty"`
}

// AnthropicToolConfiguration 工具配置
type AnthropicToolConfiguration struct {
	AllowedTools []string `json:"allowed_tools,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
}

// AnthropicThinkingConfig 思考配置
type AnthropicThinkingConfig struct {
	Type         string `json:"type" binding:"required"` // "enabled"
	BudgetTokens int    `json:"budget_tokens" binding:"required,min=1024"`
}

// ThinkingConfig 通用思考配置
type ThinkingConfig struct {
	Enabled        bool   `json:"enabled" example:"true"`              // 是否启用深度思考
	ShowProcess    bool   `json:"show_process" example:"true"`         // 是否显示思考过程
	MaxTokens      int    `json:"max_tokens,omitempty" example:"2048"` // 思考部分最大token数
	ThinkingPrompt string `json:"thinking_prompt,omitempty"`           // 自定义思考提示词
	Language       string `json:"language,omitempty" example:"zh"`     // 思考语言（zh/en）
}

// AnthropicMessageResponse Anthropic 消息响应
type AnthropicMessageResponse struct {
	ID           string                  `json:"id" example:"msg_013Zva2CMHLNnXjNJJKqJ2EF" description:"消息的唯一标识符"`
	Type         string                  `json:"type" example:"message" description:"响应类型，固定为 message"`
	Role         string                  `json:"role" example:"assistant" description:"响应角色，固定为 assistant"`
	Content      []AnthropicContentBlock `json:"content" description:"响应内容块数组"`
	Model        string                  `json:"model" example:"claude-3-sonnet-20240229" description:"使用的模型名称"`
	StopReason   string                  `json:"stop_reason" example:"end_turn" enum:"end_turn,max_tokens,stop_sequence,tool_use,pause_turn,refusal" description:"停止原因"`
	StopSequence *string                 `json:"stop_sequence,omitempty" description:"触发停止的序列（如果适用）"`
	Usage        AnthropicUsage          `json:"usage" description:"token使用情况统计"`
	Container    *AnthropicContainer     `json:"container,omitempty" description:"容器信息（如果适用）"`
}

// AnthropicUsage Anthropic 使用情况
type AnthropicUsage struct {
	InputTokens              int                     `json:"input_tokens" example:"10" description:"输入token数量"`
	OutputTokens             int                     `json:"output_tokens" example:"25" description:"输出token数量"`
	CacheCreationInputTokens *int                    `json:"cache_creation_input_tokens,omitempty" description:"缓存创建时的输入token数"`
	CacheReadInputTokens     *int                    `json:"cache_read_input_tokens,omitempty" description:"缓存读取时的输入token数"`
	CacheCreation            *AnthropicCacheCreation `json:"cache_creation,omitempty" description:"缓存创建统计"`
	ServerToolUse            *AnthropicServerToolUse `json:"server_tool_use,omitempty" description:"服务器工具使用统计"`
	ServiceTier              *string                 `json:"service_tier,omitempty" example:"standard" enum:"standard,priority,batch" description:"使用的服务层级"`
}

// AnthropicCacheCreation 缓存创建统计
type AnthropicCacheCreation struct {
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
}

// AnthropicServerToolUse 服务器工具使用统计
type AnthropicServerToolUse struct {
	WebSearchRequests int `json:"web_search_requests"`
}

// AnthropicContainer 容器信息
type AnthropicContainer struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
}

// isAnthropicModel 判断是否为Anthropic模型
func (c *aiProviderClientImpl) isAnthropicModel(model string) bool {
	anthropicModels := []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-3-5-sonnet-20240620",
		"claude-3-5-haiku-20241022",
		"claude-sonnet-4-20250514-thinking",
		"claude-2.1",
		"claude-2.0",
		"claude-instant-1.2",
	}

	for _, anthropicModel := range anthropicModels {
		if model == anthropicModel {
			return true
		}
	}

	// 也可以通过前缀判断
	return strings.HasPrefix(model, "claude-")
}

// isThinkingModel 判断是否为thinking模型
func (c *aiProviderClientImpl) isThinkingModel(model string) bool {
	// Claude thinking模型通常包含"thinking"关键字
	return strings.Contains(model, "thinking")
}
