package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"
)

// ThinkingService 深度思考服务接口
type ThinkingService interface {
	// ProcessThinkingRequest 处理带思考的请求
	ProcessThinkingRequest(ctx context.Context, request *clients.AIRequest) (*clients.AIRequest, error)
	
	// ParseThinkingResponse 解析包含思考过程的响应
	ParseThinkingResponse(response string) (*ThinkingResult, error)
	
	// IsThinkingEnabled 检查是否启用思考模式
	IsThinkingEnabled(request *clients.AIRequest) bool
}

// ThinkingResult 思考解析结果
type ThinkingResult struct {
	ThinkingProcess string `json:"thinking_process"` // 思考过程
	FinalAnswer     string `json:"final_answer"`     // 最终答案
	HasThinking     bool   `json:"has_thinking"`     // 是否包含思考过程
}

// thinkingServiceImpl 思考服务实现
type thinkingServiceImpl struct {
	logger logger.Logger
}

// NewThinkingService 创建思考服务
func NewThinkingService(logger logger.Logger) ThinkingService {
	return &thinkingServiceImpl{
		logger: logger,
	}
}

// ProcessThinkingRequest 处理带思考的请求
func (s *thinkingServiceImpl) ProcessThinkingRequest(ctx context.Context, request *clients.AIRequest) (*clients.AIRequest, error) {
	if !s.IsThinkingEnabled(request) {
		return request, nil
	}

	s.logger.WithField("model", request.Model).Info("Processing thinking request")

	// 创建请求副本
	thinkingRequest := *request
	
	// 构造思考提示词
	thinkingPrompt := s.buildThinkingPrompt(request.Thinking)
	
	// 修改消息，添加思考指令
	if len(thinkingRequest.Messages) > 0 {
		lastMessage := &thinkingRequest.Messages[len(thinkingRequest.Messages)-1]
		if lastMessage.Role == "user" {
			// 在用户消息前添加思考指令
			lastMessage.Content = thinkingPrompt + "\n\n用户问题：\n" + lastMessage.Content
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"original_messages": len(request.Messages),
		"modified_messages": len(thinkingRequest.Messages),
		"thinking_enabled":  request.Thinking.Enabled,
		"show_process":     request.Thinking.ShowProcess,
	}).Info("Thinking request processed")

	return &thinkingRequest, nil
}

// ParseThinkingResponse 解析包含思考过程的响应
func (s *thinkingServiceImpl) ParseThinkingResponse(response string) (*ThinkingResult, error) {
	result := &ThinkingResult{
		HasThinking: false,
	}

	// 使用正则表达式匹配思考过程
	// 匹配 <thinking>...</thinking> 或 【思考】...【/思考】格式
	thinkingRegex1 := regexp.MustCompile(`(?s)<thinking>(.*?)</thinking>`)
	thinkingRegex2 := regexp.MustCompile(`(?s)【思考】(.*?)【/思考】`)
	thinkingRegex3 := regexp.MustCompile(`(?s)## 思考过程\n(.*?)## 最终答案`)

	var thinkingContent string
	var finalAnswer string

	// 尝试匹配不同的格式
	if matches := thinkingRegex1.FindStringSubmatch(response); len(matches) > 1 {
		thinkingContent = strings.TrimSpace(matches[1])
		finalAnswer = strings.TrimSpace(thinkingRegex1.ReplaceAllString(response, ""))
		result.HasThinking = true
	} else if matches := thinkingRegex2.FindStringSubmatch(response); len(matches) > 1 {
		thinkingContent = strings.TrimSpace(matches[1])
		finalAnswer = strings.TrimSpace(thinkingRegex2.ReplaceAllString(response, ""))
		result.HasThinking = true
	} else if matches := thinkingRegex3.FindStringSubmatch(response); len(matches) > 1 {
		thinkingContent = strings.TrimSpace(matches[1])
		// 提取最终答案部分
		finalAnswerRegex := regexp.MustCompile(`(?s)## 最终答案\n(.*)`)
		if finalMatches := finalAnswerRegex.FindStringSubmatch(response); len(finalMatches) > 1 {
			finalAnswer = strings.TrimSpace(finalMatches[1])
		}
		result.HasThinking = true
	}

	if result.HasThinking {
		result.ThinkingProcess = thinkingContent
		result.FinalAnswer = finalAnswer
	} else {
		// 没有明确的思考标记，整个响应作为最终答案
		result.FinalAnswer = response
	}

	s.logger.WithFields(map[string]interface{}{
		"has_thinking":      result.HasThinking,
		"thinking_length":   len(result.ThinkingProcess),
		"answer_length":     len(result.FinalAnswer),
	}).Debug("Thinking response parsed")

	return result, nil
}

// IsThinkingEnabled 检查是否启用思考模式
func (s *thinkingServiceImpl) IsThinkingEnabled(request *clients.AIRequest) bool {
	return request.Thinking != nil && request.Thinking.Enabled
}

// buildThinkingPrompt 构造思考提示词
func (s *thinkingServiceImpl) buildThinkingPrompt(config *clients.ThinkingConfig) string {
	if config.ThinkingPrompt != "" {
		return config.ThinkingPrompt
	}

	// 根据语言设置选择提示词
	language := config.Language
	if language == "" {
		language = "zh" // 默认中文
	}

	var basePrompt string
	if language == "en" {
		basePrompt = `Please think step by step before providing your final answer. Put your thinking process between <thinking> and </thinking> tags.

Your response format should be:
<thinking>
Your detailed thinking process here...
- Analyze the question
- Consider different aspects
- Work through the logic
- Come to a conclusion
</thinking>

Your final answer here (without thinking tags).`
	} else {
		basePrompt = `请在给出最终答案之前进行深度思考。将你的思考过程放在 <thinking> 和 </thinking> 标签之间。

你的回答格式应该是：
<thinking>
在这里详细描述你的思考过程...
- 分析问题的关键点
- 考虑不同的角度和方面  
- 逐步推理和分析
- 得出结论
</thinking>

在这里给出你的最终答案（不包含思考标签）。`
	}

	// 如果设置了最大token数，添加限制说明
	if config.MaxTokens > 0 {
		if language == "en" {
			basePrompt += fmt.Sprintf("\n\nPlease limit your thinking process to approximately %d tokens.", config.MaxTokens)
		} else {
			basePrompt += fmt.Sprintf("\n\n请将思考过程控制在大约%d个token以内。", config.MaxTokens)
		}
	}

	return basePrompt
}

// StreamThinkingProcessor 流式思考处理器
type StreamThinkingProcessor struct {
	logger        logger.Logger
	buffer        strings.Builder
	inThinking    bool
	thinkingDone  bool
	sendThinking  bool
}

// NewStreamThinkingProcessor 创建流式思考处理器
func NewStreamThinkingProcessor(logger logger.Logger, showProcess bool) *StreamThinkingProcessor {
	return &StreamThinkingProcessor{
		logger:       logger,
		sendThinking: showProcess,
	}
}

// ProcessChunk 处理流式数据块
func (p *StreamThinkingProcessor) ProcessChunk(chunk string) ([]*clients.StreamChunk, error) {
	p.buffer.WriteString(chunk)
	currentBuffer := p.buffer.String()

	var results []*clients.StreamChunk

	// 检查是否开始思考
	if !p.inThinking && strings.Contains(currentBuffer, "<thinking>") {
		p.inThinking = true
		// 发送思考开始前的内容
		beforeThinking := strings.Split(currentBuffer, "<thinking>")[0]
		if beforeThinking != "" {
			results = append(results, &clients.StreamChunk{
				Content:     beforeThinking,
				ContentType: "response",
			})
		}
		// 清空缓冲区，保留thinking之后的内容
		afterThinking := strings.Join(strings.Split(currentBuffer, "<thinking>")[1:], "<thinking>")
		p.buffer.Reset()
		p.buffer.WriteString(afterThinking)
		currentBuffer = afterThinking
	}

	// 如果在思考阶段
	if p.inThinking && !p.thinkingDone {
		if strings.Contains(currentBuffer, "</thinking>") {
			// 思考结束
			parts := strings.Split(currentBuffer, "</thinking>")
			thinkingContent := parts[0]
			afterThinking := strings.Join(parts[1:], "</thinking>")

			if p.sendThinking && thinkingContent != "" {
				results = append(results, &clients.StreamChunk{
					Content:     thinkingContent,
					ContentType: "thinking",
				})
			}

			p.thinkingDone = true
			p.inThinking = false

			// 开始发送最终答案
			if afterThinking != "" {
				results = append(results, &clients.StreamChunk{
					Content:     afterThinking,
					ContentType: "response",
				})
			}

			// 重置缓冲区
			p.buffer.Reset()
		} else if p.sendThinking {
			// 还在思考中，发送思考内容
			results = append(results, &clients.StreamChunk{
				Content:     chunk,
				ContentType: "thinking",
			})
		}
	} else if p.thinkingDone || !p.inThinking {
		// 思考完成后的内容，或者没有思考标签的内容
		results = append(results, &clients.StreamChunk{
			Content:     chunk,
			ContentType: "response",
		})
	}

	return results, nil
}

// IsThinkingComplete 检查思考是否完成
func (p *StreamThinkingProcessor) IsThinkingComplete() bool {
	return p.thinkingDone
}