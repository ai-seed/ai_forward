package tokenizer

import (
	"strings"
	"unicode"
)

// SimpleTokenizer 简单的token计算器
type SimpleTokenizer struct{}

// NewSimpleTokenizer 创建简单的token计算器
func NewSimpleTokenizer() *SimpleTokenizer {
	return &SimpleTokenizer{}
}

// CountTokens 计算文本的token数量
// 使用简化的计算方法：大约4个字符 = 1个token（英文），中文字符按2个token计算
func (t *SimpleTokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}
	
	var tokenCount int
	
	// 遍历每个字符
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			// 中文字符，计为2个token
			tokenCount += 2
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// 英文字母和数字，按4个字符1个token计算
			tokenCount += 1
		} else if unicode.IsPunct(r) || unicode.IsSpace(r) {
			// 标点符号和空格，按较少的权重计算
			tokenCount += 1
		}
	}
	
	// 最少返回1个token，最后按4字符1token的比例调整
	if tokenCount == 0 {
		return 1
	}
	
	// 简化计算：总字符数除以4，但至少为1
	estimatedTokens := len(text) / 4
	if estimatedTokens < 1 {
		estimatedTokens = 1
	}
	
	// 取两种方法的平均值
	finalTokens := (tokenCount + estimatedTokens) / 2
	if finalTokens < 1 {
		finalTokens = 1
	}
	
	return finalTokens
}

// CountTokensFromMessages 从消息数组计算token数量
func (t *SimpleTokenizer) CountTokensFromMessages(messages []map[string]interface{}) int {
	var totalTokens int
	
	for _, message := range messages {
		if content, ok := message["content"]; ok {
			if contentStr, ok := content.(string); ok {
				totalTokens += t.CountTokens(contentStr)
			}
		}
		// 添加消息结构的额外token开销（角色标识等）
		totalTokens += 4
	}
	
	return totalTokens
}

// EstimateOutputTokensFromContent 从流式内容估算输出token数量
func (t *SimpleTokenizer) EstimateOutputTokensFromContent(content string) int {
	return t.CountTokens(content)
}