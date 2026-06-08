package agent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MasterAgent 负责意图识别和任务分发
type MasterAgent struct {
	name        string
	subAgents   map[string]Agent
	maxParallel int
	timeout     time.Duration
}

// NewMasterAgent 创建 Master Agent
func NewMasterAgent() *MasterAgent {
	return &MasterAgent{
		name:        "master",
		subAgents:   make(map[string]Agent),
		maxParallel: 5,
		timeout:     120 * time.Second,
	}
}

// RegisterAgent 注册子 Agent
func (m *MasterAgent) RegisterAgent(name string, agent Agent) {
	m.subAgents[name] = agent
}

// IntentResult 意图识别结果
type IntentResult struct {
	RequiredAgents []string
	Confidence     float64
	Reasoning      string
}

// RecognizeIntent 识别用户意图
func (m *MasterAgent) RecognizeIntent(ctx context.Context, query string) (*IntentResult, error) {
	// 简单的关键词匹配策略
	// 实际项目中可以使用 LLM 进行更智能的意图识别
	result := &IntentResult{
		RequiredAgents: []string{},
		Confidence:     0.8,
		Reasoning:      "Based on keyword analysis",
	}

	// 根据关键词判断需要的 Agent
	lowerQuery := toLower(query)

	if containsAny(lowerQuery, "日志", "log", "error", "exception") {
		result.RequiredAgents = append(result.RequiredAgents, "log")
	}

	if containsAny(lowerQuery, "指标", "metric", "cpu", "内存", "memory") {
		result.RequiredAgents = append(result.RequiredAgents, "metric")
	}

	if containsAny(lowerQuery, "文档", "doc", "文档", "怎么", "如何") {
		result.RequiredAgents = append(result.RequiredAgents, "doc")
	}

	// 如果没有匹配，使用默认的 doc agent
	if len(result.RequiredAgents) == 0 {
		result.RequiredAgents = append(result.RequiredAgents, "doc")
	}

	return result, nil
}

// Handle 处理用户请求
func (m *MasterAgent) Handle(ctx context.Context, query string) (string, error) {
	// 1. 意图识别
	intent, err := m.RecognizeIntent(ctx, query)
	if err != nil {
		return "", fmt.Errorf("intent recognition failed: %w", err)
	}

	// 2. 并行分发任务
	results := m.dispatchParallel(ctx, intent.RequiredAgents, query)

	// 3. 汇总结果
	return m.aggregateResults(results), nil
}

// dispatchParallel 并行分发任务
func (m *MasterAgent) dispatchParallel(ctx context.Context, agents []string, query string) []*AgentResult {
	var wg sync.WaitGroup
	results := make(chan *AgentResult, len(agents))

	for _, agentName := range agents {
		// 检查是否已注册
		agent, ok := m.subAgents[agentName]
		if !ok {
			continue
		}

		wg.Add(1)
		go func(name string, a Agent) {
			defer wg.Done()

			// 带超时的执行
			execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			result, err := a.Execute(execCtx, query)
			results <- &AgentResult{
				AgentName: name,
				Content:   result,
				Error:     err,
			}
		}(agentName, agent)
	}

	wg.Wait()
	close(results)

	// 收集结果
	var agentResults []*AgentResult
	for r := range results {
		agentResults = append(agentResults, r)
	}

	return agentResults
}

// aggregateResults 汇总结果
func (m *MasterAgent) aggregateResults(results []*AgentResult) string {
	// 简单的结果拼接
	// 实际项目中可以使用 LLM 进行更智能的结果融合
	var summary string
	for _, r := range results {
		if r.Error != nil {
			summary += fmt.Sprintf("[%s] Error: %v\n", r.AgentName, r.Error)
		} else {
			summary += fmt.Sprintf("[%s]\n%s\n\n", r.AgentName, r.Content)
		}
	}
	return summary
}

// AgentResult Agent 执行结果
type AgentResult struct {
	AgentName string
	Content   string
	Error     error
}

// Agent 接口
type Agent interface {
	Name() string
	Execute(ctx context.Context, query string) (string, error)
}

// 辅助函数
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if len(kw) <= len(s) {
			for i := 0; i <= len(s)-len(kw); i++ {
				if s[i:i+len(kw)] == kw {
					return true
				}
			}
		}
	}
	return false
}
