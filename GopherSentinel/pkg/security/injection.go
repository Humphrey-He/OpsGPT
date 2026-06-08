package security

import (
	"regexp"
	"strings"
)

// PromptInjectionDetector Prompt 注入检测器
type PromptInjectionDetector struct {
	patterns []*regexp.Regexp
}

// NewPromptInjectionDetector 创建检测器
func NewPromptInjectionDetector() *PromptInjectionDetector {
	patterns := []*regexp.Regexp{
		// 忽略系统提示词的模式
		regexp.MustCompile(`(?i)(ignore|disregard|forget)\s+(previous|all|system|instructions)`),
		// 尝试注入新指令的模式
		regexp.MustCompile(`(?i)(instead|rather|new\s+instruction)`),
		// 角色扮演/劫持模式
		regexp.MustCompile(`(?i)(you\s+are\s+now|pretend|act\s+as|simulate)`),
		// Base64/编码注入
		regexp.MustCompile(`[A-Za-z0-9+/]{50,}={0,2}`),
	}

	return &PromptInjectionDetector{patterns: patterns}
}

// Detect 检测 Prompt 注入
func (d *PromptInjectionDetector) Detect(text string) (bool, string) {
	for _, pattern := range d.patterns {
		if pattern.MatchString(text) {
			return true, pattern.String()
		}
	}

	// 检查异常长度
	if len(text) > 10000 {
		return true, "text too long"
	}

	return false, ""
}

// Sanitize 清理危险内容
func (d *PromptInjectionDetector) Sanitize(text string) string {
	// 移除可能的注入标记
	text = strings.ReplaceAll(text, "```system", "```")
	text = strings.ReplaceAll(text, "```xml", "```")

	return text
}

// SensitiveWordFilter 敏感词过滤器
type SensitiveWordFilter struct {
	words []string
}

// NewSensitiveWordFilter 创建过滤器
func NewSensitiveWordFilter() *SensitiveWordFilter {
	return &SensitiveWordFilter{
		words: []string{
			// 根据实际需求添加敏感词
		},
	}
}

// Filter 过滤敏感词
func (f *SensitiveWordFilter) Filter(text string) (bool, []string) {
	var found []string

	lowerText := strings.ToLower(text)
	for _, word := range f.words {
		if strings.Contains(lowerText, strings.ToLower(word)) {
			found = append(found, word)
		}
	}

	return len(found) > 0, found
}

// Mask 屏蔽敏感词
func (f *SensitiveWordFilter) Mask(text string, mask string) string {
	result := text
	for _, word := range f.words {
		result = strings.ReplaceAll(strings.ToLower(result), strings.ToLower(word), mask)
	}
	return result
}
