package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"GopherSentinel/pkg/llm"
	"GopherSentinel/pkg/tools"
)

// MasterAgent 负责意图识别和任务分发
type MasterAgent struct {
	name         string
	subAgents    map[string]Agent
	toolRegistry *tools.ToolRegistry
	llmClient   *llm.OllamaClient
	maxParallel  int
	timeout     time.Duration
	tools       map[string]tools.Tool
}

// NewMasterAgent 创建 Master Agent
func NewMasterAgent() *MasterAgent {
	return &MasterAgent{
		name:         "master",
		subAgents:    make(map[string]Agent),
		toolRegistry: tools.GetRegistry(),
		maxParallel:  5,
		timeout:      120 * time.Second,
		tools:        make(map[string]tools.Tool),
	}
}

// RegisterAgent 注册子 Agent
func (m *MasterAgent) RegisterAgent(name string, agent Agent) {
	m.subAgents[name] = agent
}

// SetLLMClient 设置 LLM 客户端（用于智能意图识别）
func (m *MasterAgent) SetLLMClient(client *llm.OllamaClient) {
	m.llmClient = client
}

// RegisterTool 注册工具
func (m *MasterAgent) RegisterTool(tool tools.Tool) {
	m.tools[tool.Name()] = tool
}

// IntentResult 意图识别结果
type IntentResult struct {
	RequiredAgents []string   `json:"required_agents"`
	RequiredTools []string   `json:"required_tools"`
	Confidence    float64    `json:"confidence"`
	Reasoning     string     `json:"reasoning"`
	QueryType    QueryType  `json:"query_type"`
}

// QueryType 查询类型
type QueryType string

const (
	QueryTypeMetric QueryType = "metric"
	QueryTypeLog   QueryType = "log"
	QueryTypeDoc   QueryType = "doc"
	QueryTypeK8s  QueryType = "k8s"
	QueryTypeMixed QueryType = "mixed"
	QueryTypeOther QueryType = "other"
)

// RecognizeIntent 识别用户意图
func (m *MasterAgent) RecognizeIntent(ctx context.Context, query string) (*IntentResult, error) {
	return m.recognizeWithKeywords(query), nil
}

// recognizeWithKeywords 使用关键词进行意图识别
func (m *MasterAgent) recognizeWithKeywords(query string) *IntentResult {
	result := &IntentResult{
		RequiredAgents: []string{},
		RequiredTools: []string{},
		Confidence:    0.7,
		Reasoning:     "Based on keyword analysis",
	}

	lowerQuery := strings.ToLower(query)

	// 检测日志查询
	if containsAny(lowerQuery, "日志", "log", "error", "exception", "trace", "debug", "warn") {
		result.RequiredAgents = append(result.RequiredAgents, "log")
		result.RequiredTools = append(result.RequiredTools, "loki_query")
		result.QueryType = QueryTypeLog
	}

	// 检测指标查询
	if containsAny(lowerQuery, "指标", "metric", "cpu", "内存", "memory", "使用率", "qps", "latency", "load") {
		result.RequiredAgents = append(result.RequiredAgents, "metric")
		result.RequiredTools = append(result.RequiredTools, "prometheus_query")
		result.QueryType = QueryTypeMetric
	}

	// 检测 K8s 操作
	if containsAny(lowerQuery, "pod", "deployment", "service", "namespace", "k8s", "kubernetes", "容器", "集群") {
		result.RequiredAgents = append(result.RequiredAgents, "k8s")
		result.RequiredTools = append(result.RequiredTools, "k8s_get_pods")
		result.QueryType = QueryTypeK8s
	}

	// 检测文档查询
	if containsAny(lowerQuery, "文档", "doc", "怎么", "如何", "部署", "配置", "使用", "说明") {
		result.RequiredAgents = append(result.RequiredAgents, "doc")
		result.QueryType = QueryTypeDoc
	}

	// 混合查询
	if len(result.RequiredAgents) > 1 {
		result.QueryType = QueryTypeMixed
	}

	// 没有匹配时的默认值
	if len(result.RequiredAgents) == 0 {
		result.RequiredAgents = append(result.RequiredAgents, "doc")
		result.QueryType = QueryTypeOther
		result.Confidence = 0.5
	}

	return result
}

// Handle 处理用户请求
func (m *MasterAgent) Handle(ctx context.Context, query string) (string, error) {
	// 1. 意图识别
	intent, _ := m.RecognizeIntent(ctx, query)

	// 2. 决定处理方式
	var results []*AgentResult

	if len(intent.RequiredAgents) > 0 {
		results = m.dispatchParallel(ctx, intent.RequiredAgents, query)
	} else if len(intent.RequiredTools) > 0 {
		results = m.executeTools(ctx, intent.RequiredTools, query)
	}

	return m.aggregateResults(results), nil
}

// executeTools 直接执行工具
func (m *MasterAgent) executeTools(ctx context.Context, toolNames []string, query string) []*AgentResult {
	var wg sync.WaitGroup
	results := make(chan *AgentResult, len(toolNames))

	for _, toolName := range toolNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			tool, ok := m.tools[name]
			if !ok {
				results <- &AgentResult{
					AgentName: name,
					Content:  "",
					Error:    fmt.Errorf("tool not found: %s", name),
				}
				return
			}

			params := m.extractToolParams(query, name)
			result, err := tool.Execute(execCtx, params)
			results <- &AgentResult{
				AgentName: name,
				Content:   formatToolResult(result),
				Error:     err,
			}
		}(toolName)
	}

	wg.Wait()
	close(results)

	var agentResults []*AgentResult
	for r := range results {
		agentResults = append(agentResults, r)
	}

	return agentResults
}

// extractToolParams 从查询中提取工具参数
func (m *MasterAgent) extractToolParams(query, toolName string) map[string]interface{} {
	params := make(map[string]interface{})

	switch {
	case toolName == "loki_query":
		params["query"] = query
		params["limit"] = 100
	case toolName == "prometheus_query":
		params["query"] = query
	case toolName == "k8s_get_pods":
		if ns := extractNamespace(query); ns != "" {
			params["namespace"] = ns
		}
	}

	return params
}

// extractNamespace 从查询中提取命名空间
func extractNamespace(query string) string {
	lowerQuery := strings.ToLower(query)
	if strings.Contains(lowerQuery, "default") {
		return "default"
	}
	if strings.Contains(lowerQuery, "kube-system") {
		return "kube-system"
	}
	if strings.Contains(lowerQuery, "production") || strings.Contains(lowerQuery, "prod") {
		return "production"
	}
	return ""
}

// formatToolResult 格式化工具结果
func formatToolResult(result interface{}) string {
	if result == nil {
		return "No result"
	}

	switch v := result.(type) {
	case string:
		return v
	case map[string]interface{}:
		data, _ := json.MarshalIndent(v, "", "  ")
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// dispatchParallel 并行分发任务
func (m *MasterAgent) dispatchParallel(ctx context.Context, agents []string, query string) []*AgentResult {
	var wg sync.WaitGroup
	results := make(chan *AgentResult, len(agents))

	semaphore := make(chan struct{}, m.maxParallel)

	for _, agentName := range agents {
		agent, ok := m.subAgents[agentName]
		if !ok {
			continue
		}

		wg.Add(1)
		go func(name string, a Agent) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

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

	var agentResults []*AgentResult
	for r := range results {
		agentResults = append(agentResults, r)
	}

	return agentResults
}

// aggregateResults 汇总结果
func (m *MasterAgent) aggregateResults(results []*AgentResult) string {
	if len(results) == 0 {
		return "No results available"
	}

	var summary strings.Builder
	summary.WriteString("## 分析结果\n\n")

	for _, r := range results {
		if r.Error != nil {
			summary.WriteString(fmt.Sprintf("### [%s] 错误\n", r.AgentName))
			summary.WriteString(fmt.Sprintf("> %v\n\n", r.Error))
		} else {
			summary.WriteString(fmt.Sprintf("### [%s]\n", r.AgentName))
			summary.WriteString(r.Content)
			summary.WriteString("\n\n")
		}
	}

	return summary.String()
}

// AgentResult Agent 执行结果
type AgentResult struct {
	AgentName string
	Content  string
	Error    error
	Metadata map[string]interface{}
}

// Agent 接口
type Agent interface {
	Name() string
	Execute(ctx context.Context, query string) (string, error)
}

// GetSubAgents 获取所有注册的子 Agent
func (m *MasterAgent) GetSubAgents() map[string]Agent {
	return m.subAgents
}

// GetTools 获取所有注册的工具
func (m *MasterAgent) GetTools() map[string]tools.Tool {
	return m.tools
}

// ListCapabilities 列出所有能力
func (m *MasterAgent) ListCapabilities() map[string]interface{} {
	capabilities := map[string]interface{}{
		"agents": []string{},
		"tools":  []string{},
	}

	for name := range m.subAgents {
		capabilities["agents"] = append(capabilities["agents"].([]string), name)
	}

	for name := range m.tools {
		capabilities["tools"] = append(capabilities["tools"].([]string), name)
	}

	return capabilities
}

// Reset 重置 Agent 状态
func (m *MasterAgent) Reset() {
	// 保留 subAgents 和 tools，只清理临时状态
}

// 辅助函数
func containsAny(s string, keywords ...string) bool {
	lowerS := strings.ToLower(s)
	for _, kw := range keywords {
		if strings.Contains(lowerS, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}
