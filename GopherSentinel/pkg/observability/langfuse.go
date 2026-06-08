package observability

import (
	"context"
	"time"
)

// LangfuseTracer Langfuse 追踪器
type LangfuseTracer struct {
	publicKey string
	secretKey string
	host     string
}

// NewLangfuseTracer 创建 Langfuse 追踪器
func NewLangfuseTracer(publicKey, secretKey, host string) *LangfuseTracer {
	return &LangfuseTracer{
		publicKey: publicKey,
		secretKey: secretKey,
		host:     host,
	}
}

// Span 追踪跨度
type Span struct {
	traceID   string
	spanID    string
	name     string
	startTime time.Time
	endTime   time.Time
	metadata map[string]string
	input    string
	output   string
}

// StartSpan 开始追踪
func (t *LangfuseTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	span := &Span{
		traceID:   generateID(),
		spanID:    generateID(),
		name:     name,
		startTime: time.Now(),
		metadata: make(map[string]string),
	}

	for _, opt := range opts {
		opt(span)
	}

	return ctx, span
}

// SpanOption Span 配置选项
type SpanOption func(*Span)

// WithMetadata 添加元数据
func WithMetadata(metadata map[string]string) SpanOption {
	return func(s *Span) {
		for k, v := range metadata {
			s.metadata[k] = v
		}
	}
}

// RecordInput 记录输入
func (s *Span) RecordInput(input string) {
	s.input = input
}

// RecordOutput 记录输出
func (s *Span) RecordOutput(output string) {
	s.output = output
}

// RecordToolCall 记录工具调用
func (s *Span) RecordToolCall(toolName string, params map[string]interface{}) {
	s.metadata["tool_call"] = toolName
}

// RecordToolResult 记录工具结果
func (s *Span) RecordToolResult(result interface{}) {
	s.metadata["tool_result"] = "executed"
}

// Score 记录评分
func (s *Span) Score(score float64) {
	s.metadata["quality_score"] = string(rune(int(score * 100)))
}

// End 结束追踪
func (s *Span) End() {
	s.endTime = time.Now()
}

// Log 记录日志
func (s *Span) Log(message string) {
	// 实际实现中会发送到 Langfuse
}

// 辅助函数
func generateID() string {
	return time.Now().Format("20060102150405.000000")
}
