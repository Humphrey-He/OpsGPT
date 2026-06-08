package agent

import (
	"context"
	"fmt"
	"strings"
)

// LogAgent 日志分析 Agent
type LogAgent struct {
	name string
}

func NewLogAgent() *LogAgent {
	return &LogAgent{name: "log"}
}

func (a *LogAgent) Name() string {
	return a.name
}

func (a *LogAgent) Execute(ctx context.Context, query string) (string, error) {
	// 模拟日志分析
	// 实际项目中会调用 Loki/ELK 等日志系统
	return "Log analysis: No critical errors found in recent logs.\n" +
		"- Total log entries: 1,234\n" +
		"- Error count: 5\n" +
		"- Warning count: 23", nil
}

// MetricAgent 指标分析 Agent
type MetricAgent struct {
	name string
}

func NewMetricAgent() *MetricAgent {
	return &MetricAgent{name: "metric"}
}

func (a *MetricAgent) Name() string {
	return a.name
}

func (a *MetricAgent) Execute(ctx context.Context, query string) (string, error) {
	// 模拟指标分析
	// 实际项目中会调用 Prometheus 等监控系统
	return "Metric analysis:\n" +
		"- CPU Usage: 45%\n" +
		"- Memory Usage: 62%\n" +
		"- Request Rate: 1,200 req/s\n" +
		"- Error Rate: 0.1%", nil
}

// DocAgent 文档检索 Agent
type DocAgent struct {
	name string
}

func NewDocAgent() *DocAgent {
	return &DocAgent{name: "doc"}
}

func (a *DocAgent) Name() string {
	return a.name
}

func (a *DocAgent) Execute(ctx context.Context, query string) (string, error) {
	// 模拟文档检索
	// 实际项目中会调用 RAG 系统
	keywords := extractKeywords(query)

	return fmt.Sprintf("Document search results for keywords: %v\n"+
		"Found 3 relevant documents:\n"+
		"1. Architecture Overview (relevance: 0.95)\n"+
		"2. Deployment Guide (relevance: 0.87)\n"+
		"3. API Reference (relevance: 0.72)", keywords), nil
}

func extractKeywords(query string) []string {
	// 简单的关键词提取
	// 实际项目中可以使用 NLP 技术
	words := strings.Fields(query)
	if len(words) > 5 {
		return words[:5]
	}
	return words
}
