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
// 基于GPT tokenizer的经验规律进行估算
func (t *SimpleTokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// 简化但相对准确的估算方法：
	// 1. 先按照 1 token ≈ 0.75 words (英文) 或 1 token ≈ 2-3 characters (中文) 估算
	// 2. 考虑特殊字符和格式
	
	// 统计不同类型字符
	var charCount, wordCount, chineseCount int
	inWord := false
	
	for _, r := range text {
		charCount++
		
		if unicode.Is(unicode.Han, r) {
			// 中文字符
			chineseCount++
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				wordCount++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	
	// 基于经验公式估算
	var tokens float64
	
	if chineseCount > charCount/2 {
		// 主要是中文内容：每个中文字符约等于 1.2 个token
		tokens = float64(chineseCount) * 1.2
		// 英文部分：每个单词约等于 1.3 个token
		englishChars := charCount - chineseCount
		estimatedEnglishWords := float64(englishChars) / 5.0 // 平均单词长度
		tokens += estimatedEnglishWords * 1.3
	} else {
		// 主要是英文内容：每个单词约等于 1.3 个token
		tokens = float64(wordCount) * 1.3
		// 中文部分
		tokens += float64(chineseCount) * 1.2
	}
	
	// 考虑标点和格式字符 (约增加5%)
	tokens *= 1.05
	
	result := int(tokens + 0.5) // 四舍五入
	
	// 保证最小值
	if result < 1 {
		result = 1
	}
	
	// 对于很短的文本，使用字符数的保守估算
	if charCount < 10 {
		conservative := (charCount + 3) / 4 // 每4个字符约1个token
		if conservative > result {
			result = conservative
		}
	}
	
	return result
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
