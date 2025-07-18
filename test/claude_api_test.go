package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"ai-api-gateway/internal/infrastructure/clients"
)

// TestClaudeMessagesAPI 测试Claude消息API
func TestClaudeMessagesAPI(t *testing.T) {
	// 测试配置
	baseURL := "http://localhost:8080"
	apiKey := "ak_test_key_here" // 需要替换为实际的API密钥

	// 构造Claude请求
	claudeRequest := clients.ClaudeMessageRequest{
		Model:     "claude-3-sonnet-20240229",
		MaxTokens: 100,
		Messages: []clients.AIMessage{
			{
				Role:    "user",
				Content: "Hello, Claude! Please respond with a simple greeting.",
			},
		},
		Temperature: 0.7,
		Stream:      false,
	}

	// 序列化请求
	requestBody, err := json.Marshal(claudeRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("x-api-key", apiKey) // Claude风格的API密钥头
	req.Header.Set("anthropic-version", "2023-06-01")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 解析响应
	var claudeResponse clients.ClaudeMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// 验证响应结构
	if claudeResponse.ID == "" {
		t.Error("Response ID is empty")
	}

	if claudeResponse.Type != "message" {
		t.Errorf("Expected type 'message', got '%s'", claudeResponse.Type)
	}

	if claudeResponse.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got '%s'", claudeResponse.Role)
	}

	if len(claudeResponse.Content) == 0 {
		t.Error("Response content is empty")
	}

	// 验证内容块
	for i, content := range claudeResponse.Content {
		if content.Type == "" {
			t.Errorf("Content block %d has empty type", i)
		}
		if content.Type == "text" && content.Text == "" {
			t.Errorf("Text content block %d has empty text", i)
		}
	}

	// 验证使用情况
	if claudeResponse.Usage.InputTokens <= 0 {
		t.Error("Input tokens should be greater than 0")
	}

	if claudeResponse.Usage.OutputTokens <= 0 {
		t.Error("Output tokens should be greater than 0")
	}

	// 打印响应用于调试
	fmt.Printf("Claude API Response:\n")
	fmt.Printf("ID: %s\n", claudeResponse.ID)
	fmt.Printf("Model: %s\n", claudeResponse.Model)
	fmt.Printf("Stop Reason: %s\n", claudeResponse.StopReason)
	fmt.Printf("Usage: Input=%d, Output=%d\n", claudeResponse.Usage.InputTokens, claudeResponse.Usage.OutputTokens)
	fmt.Printf("Content blocks: %d\n", len(claudeResponse.Content))
	for i, content := range claudeResponse.Content {
		fmt.Printf("  Block %d: Type=%s, Text=%s\n", i, content.Type, content.Text)
	}
}

// TestClaudeMessagesAPIWithSystem 测试带系统消息的Claude API
func TestClaudeMessagesAPIWithSystem(t *testing.T) {
	baseURL := "http://localhost:8080"
	apiKey := "ak_test_key_here" // 需要替换为实际的API密钥

	claudeRequest := clients.ClaudeMessageRequest{
		Model:     "claude-3-sonnet-20240229",
		MaxTokens: 50,
		System:    "You are a helpful assistant that responds in a very concise manner.",
		Messages: []clients.AIMessage{
			{
				Role:    "user",
				Content: "What is the capital of France?",
			},
		},
		Temperature: 0.1,
		Stream:      false,
	}

	requestBody, _ := json.Marshal(claudeRequest)
	req, _ := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var claudeResponse clients.ClaudeMessageResponse
	json.NewDecoder(resp.Body).Decode(&claudeResponse)

	// 验证响应包含预期内容
	if len(claudeResponse.Content) == 0 {
		t.Error("Response should contain content")
	}

	fmt.Printf("System message test response: %+v\n", claudeResponse.Content[0].Text)
}

// TestClaudeMessagesAPIValidation 测试Claude API参数验证
func TestClaudeMessagesAPIValidation(t *testing.T) {
	baseURL := "http://localhost:8080"
	apiKey := "ak_test_key_here"

	// 测试缺少model参数
	invalidRequest := map[string]interface{}{
		"max_tokens": 100,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	requestBody, _ := json.Marshal(invalidRequest)
	req, _ := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 应该返回400错误
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing model, got %d", resp.StatusCode)
	}

	// 测试缺少max_tokens参数
	invalidRequest2 := map[string]interface{}{
		"model": "claude-3-sonnet-20240229",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	requestBody2, _ := json.Marshal(invalidRequest2)
	req2, _ := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewBuffer(requestBody2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+apiKey)

	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp2.Body.Close()

	// 应该返回400错误
	if resp2.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing max_tokens, got %d", resp2.StatusCode)
	}
}
