package tokenizer

import (
	"unicode"
)

// SimpleTokenizer 简单的token计算器
type SimpleTokenizer struct{}

// NewSimpleTokenizer 创建简单的token计算器
func NewSimpleTokenizer() *SimpleTokenizer {
	return &SimpleTokenizer{}
}

// CountTokens 计算文本的token数量
// 参考Claude tokenizer的计算方式
func (t *SimpleTokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Claude tokenizer特点：
	// 1. 英文：约3.5-4个字符 = 1个token
	// 2. 中文：约1个汉字 = 1个token
	// 3. 标点符号通常单独计算
	// 4. 空格和换行符有特殊处理
	
	var tokens float64
	var i int
	runes := []rune(text)
	
	for i < len(runes) {
		r := runes[i]
		
		if unicode.Is(unicode.Han, r) {
			// 中文字符，GPT对中文的token化比预期要多
			// 经验值：每个中文字符约1.38个token（基于实际测试调优）
			tokens += 1.38
			i++
		} else if unicode.IsLetter(r) {
			// 英文单词，统计连续字母
			wordStart := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i])) {
				i++
			}
			wordLen := i - wordStart
			// Claude tokenizer: 约3.8个字符 = 1个token
			tokens += float64(wordLen) / 3.8
		} else if unicode.IsDigit(r) {
			// 数字，统计连续数字
			numStart := i
			for i < len(runes) && unicode.IsDigit(runes[i]) {
				i++
			}
			numLen := i - numStart
			// 数字token化：根据长度调整，短数字权重稍高
			if numLen <= 3 {
				tokens += 1.0 // 1-3位数字通常是1个token
			} else {
				tokens += float64(numLen) / 3.5 // 长数字按3.5字符/token
			}
		} else if unicode.IsSpace(r) {
			// 空格和换行符
			if r == '\n' {
				tokens += 0.5 // 换行符权重较高
			} else {
				tokens += 0.2 // 空格权重较低
			}
			i++
		} else if unicode.IsPunct(r) {
			// 标点符号，大部分单独成token
			if r == '.' || r == ',' || r == ';' || r == ':' || r == '!' || r == '?' {
				tokens += 1.0 // 主要标点符号
			} else {
				tokens += 0.8 // 其他标点符号
			}
			i++
		} else {
			// 其他字符
			tokens += 0.5
			i++
		}
	}
	
	result := int(tokens + 0.5) // 四舍五入
	
	// 保证最小值
	if result < 1 {
		result = 1
	}
	
	// 对于特别短的文本，保守估算
	if len(runes) <= 3 {
		result = 1
	} else if len(runes) <= 8 {
		result = max(result, 2)
	}
	
	return result
}

// max 辅助函数
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CountTokensFromMessages 从消息数组计算token数量
func (t *SimpleTokenizer) CountTokensFromMessages(messages []map[string]interface{}) int {
	var totalTokens int

	for _, message := range messages {
		// 内容token
		if content, ok := message["content"]; ok {
			if contentStr, ok := content.(string); ok {
				totalTokens += t.CountTokens(contentStr)
			}
		}
		
		// 消息结构开销：
		// - role标识: ~2 tokens
		// - 消息边界标记: ~3 tokens (稍微增加以匹配GPT)
		totalTokens += 5
		
		// 角色特定开销
		if role, ok := message["role"]; ok {
			if roleStr, ok := role.(string); ok {
				switch roleStr {
				case "system":
					totalTokens += 3 // system消息有额外开销
				case "assistant":
					totalTokens += 2 // assistant消息有中等开销
				case "user":
					totalTokens += 1 // user消息开销最小
				}
			}
		}
	}

	// 对话级别的开销（开始和结束标记）
	if len(messages) > 0 {
		totalTokens += 3
	}

	return totalTokens
}

// EstimateOutputTokensFromContent 从流式内容估算输出token数量
func (t *SimpleTokenizer) EstimateOutputTokensFromContent(content string) int {
	return t.CountTokens(content)
}

// DebugTokenCount 调试用的token计数，返回详细信息
func (t *SimpleTokenizer) DebugTokenCount(text string) map[string]interface{} {
	if text == "" {
		return map[string]interface{}{
			"total_tokens": 0,
			"char_count":   0,
		}
	}

	var chineseCount, englishCount, digitCount, punctCount, spaceCount, otherCount int
	var tokens float64
	
	runes := []rune(text)
	
	for _, r := range runes {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
			tokens += 1.0
		} else if unicode.IsLetter(r) {
			englishCount++
		} else if unicode.IsDigit(r) {
			digitCount++
		} else if unicode.IsSpace(r) {
			spaceCount++
			if r == '\n' {
				tokens += 0.5
			} else {
				tokens += 0.2
			}
		} else if unicode.IsPunct(r) {
			punctCount++
			if r == '.' || r == ',' || r == ';' || r == ':' || r == '!' || r == '?' {
				tokens += 1.0
			} else {
				tokens += 0.8
			}
		} else {
			otherCount++
			tokens += 0.5
		}
	}
	
	// 英文按单词处理已在CountTokens中处理，这里简化计算
	tokens += float64(englishCount) / 3.8
	tokens += float64(digitCount) / 4.0
	
	return map[string]interface{}{
		"total_tokens":   int(tokens + 0.5),
		"char_count":     len(runes),
		"chinese_count":  chineseCount,
		"english_count":  englishCount,
		"digit_count":    digitCount,
		"punct_count":    punctCount,
		"space_count":    spaceCount,
		"other_count":    otherCount,
	}
}
