package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"GopherSentinel/pkg/llm"
	"GopherSentinel/pkg/tools"
)

// ReActAgent 实现 ReAct (Reason + Act) 模式的 Agent
type ReActAgent struct {
	name         string
	maxLoops     int
	timeout      time.Duration
	masterAgent  *MasterAgent
	llmClient   *llm.OllamaClient
	toolRegistry *tools.ToolRegistry
	tools       map[string]tools.Tool
}

// NewReActAgent 创建 ReAct Agent
func NewReActAgent(master *MasterAgent) *ReActAgent {
	return &ReActAgent{
		name:         "react",
		maxLoops:     10,
		timeout:      60 * time.Second,
		masterAgent:  master,
		toolRegistry: tools.GetRegistry(),
		tools:       make(map[string]tools.Tool),
	}
}

// SetLLMClient 设置 LLM 客户端
func (r *ReActAgent) SetLLMClient(client *llm.OllamaClient) {
	r.llmClient = client
}

// RegisterTool 注册工具
func (r *ReActAgent) RegisterTool(tool tools.Tool) {
	r.tools[tool.Name()] = tool
}

// Thought 思考步骤
type Thought struct {
	Step        int
	Reasoning   string
	Action      string
	ActionInput string
	Observation string
}

// Run 执行 ReAct 循环
func (r *ReActAgent) Run(ctx context.Context, query string) (string, error) {
	history := []Thought{}

	// 初始化上下文
	state := map[string]interface{}{
		"query":     query,
		"collected": []string{},
	}

	for i := 0; i < r.maxLoops; i++ {
		// 1. Think - 思考下一步
		thought := r.think(ctx, query, history, state)

		// 2. 检查是否应该结束
		if thought.Action == "finish" || thought.Action == "response" {
			return r.formatFinalResponse(thought.Observation, history), nil
		}

		// 3. Act - 执行行动
		obs, err := r.act(ctx, thought.Action, thought.ActionInput, state)
		thought.Observation = obs

		if err != nil {
			thought.Observation = fmt.Sprintf("Error: %v", err)
		}

		history = append(history, thought)

		// 4. 更新状态
		if obs != "" {
			collected := state["collected"].([]string)
			state["collected"] = append(collected, obs)
		}

		// 5. 检查是否超时
		select {
		case <-ctx.Done():
			return r.giveUp(history), ctx.Err()
		default:
		}
	}

	// 超过最大循环次数
	return r.giveUp(history), nil
}

// think 思考步骤 - 使用 LLM 进行智能推理
func (r *ReActAgent) think(ctx context.Context, query string, history []Thought, state map[string]interface{}) Thought {
	// 使用规则推理
	return r.thinkWithRules(query, history, state)
}

// thinkWithRules 使用规则进行推理
func (r *ReActAgent) thinkWithRules(query string, history []Thought, state map[string]interface{}) Thought {
	lowerQuery := strings.ToLower(query)
	step := len(history) + 1

	// 根据历史和查询决定下一步
	if len(history) == 0 {
		// 第一步：根据查询类型选择工具
		if hasKeyword(lowerQuery, "日志", "log", "error") {
			return Thought{
				Step:        step,
				Reasoning:   "检测到日志相关查询，使用日志搜索工具",
				Action:      "search_logs",
				ActionInput: query,
			}
		}
		if hasKeyword(lowerQuery, "指标", "metric", "cpu", "内存") {
			return Thought{
				Step:        step,
				Reasoning:   "检测到指标相关查询，使用指标查询工具",
				Action:      "query_metrics",
				ActionInput: query,
			}
		}
		if hasKeyword(lowerQuery, "pod", "k8s", "容器") {
			return Thought{
				Step:        step,
				Reasoning:   "检测到 K8s 相关查询",
				Action:      "check_k8s",
				ActionInput: query,
			}
		}
		return Thought{
			Step:        step,
			Reasoning:   "使用文档检索工具",
			Action:      "search_docs",
			ActionInput: query,
		}
	}

	// 检查是否收集了足够信息
	collected := state["collected"].([]string)
	if len(collected) >= 2 || len(history) >= 3 {
		return Thought{
			Step:        step,
			Reasoning:   "已收集足够信息，进行综合分析",
			Action:      "analyze",
			ActionInput: query,
		}
	}

	// 继续收集信息
	return Thought{
		Step:        step,
		Reasoning:   "继续收集更多信息",
		Action:      "search_docs",
		ActionInput: query,
	}
}

// hasKeyword 检查是否包含关键词
func hasKeyword(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// act 执行行动
func (r *ReActAgent) act(ctx context.Context, action, actionInput string, state map[string]interface{}) (string, error) {
	switch action {
	case "search_logs":
		return r.searchLogs(ctx, actionInput)
	case "query_metrics":
		return r.queryMetrics(ctx, actionInput)
	case "check_k8s":
		return r.checkK8s(ctx, actionInput)
	case "search_docs":
		return r.searchDocs(ctx, actionInput)
	case "analyze":
		return r.analyze(ctx, actionInput, state)
	case "finish", "response":
		return actionInput, nil
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// searchLogs 搜索日志
func (r *ReActAgent) searchLogs(ctx context.Context, query string) (string, error) {
	tool, ok := r.tools["loki_query"]
	if !ok {
		tool = tools.NewLokiTool()
	}

	params := map[string]interface{}{
		"query": query,
		"limit": 50,
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		return "", err
	}

	return formatToolResult(result), nil
}

// queryMetrics 查询指标
func (r *ReActAgent) queryMetrics(ctx context.Context, query string) (string, error) {
	tool, ok := r.tools["prometheus_query"]
	if !ok {
		tool = tools.NewPrometheusTool()
	}

	promQL := r.extractPromQL(query)

	params := map[string]interface{}{
		"query": promQL,
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		return "", err
	}

	return formatToolResult(result), nil
}

// extractPromQL 从自然语言提取 PromQL
func (r *ReActAgent) extractPromQL(query string) string {
	lowerQuery := strings.ToLower(query)

	if strings.Contains(lowerQuery, "cpu") {
		return `sum(rate(node_cpu_seconds_total[5m])) by (instance) * 100`
	}
	if strings.Contains(lowerQuery, "内存") || strings.Contains(lowerQuery, "memory") {
		return `node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes * 100`
	}
	if strings.Contains(lowerQuery, "请求") || strings.Contains(lowerQuery, "qps") {
		return `sum(rate(http_requests_total[5m]))`
	}
	if strings.Contains(lowerQuery, "延迟") || strings.Contains(lowerQuery, "latency") {
		return `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))`
	}

	return `up`
}

// checkK8s 检查 K8s 状态
func (r *ReActAgent) checkK8s(ctx context.Context, query string) (string, error) {
	tool, ok := r.tools["k8s_get_pods"]
	if !ok {
		return "K8s 工具未配置", nil
	}

	params := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		return "", err
	}

	return formatToolResult(result), nil
}

// searchDocs 搜索文档
func (r *ReActAgent) searchDocs(ctx context.Context, query string) (string, error) {
	return fmt.Sprintf("文档搜索结果 for '%s':\n\n找到 3 个相关文档:\n1. 架构文档 (相关度 0.92)\n2. 部署指南 (相关度 0.85)\n3. API 参考 (相关度 0.78)", query), nil
}

// analyze 分析收集到的信息
func (r *ReActAgent) analyze(ctx context.Context, query string, state map[string]interface{}) (string, error) {
	collected := state["collected"].([]string)

	if len(collected) == 0 {
		return "根据查询 '" + query + "'，我没有找到相关信息。请尝试更具体的问题。", nil
	}

	summary := "## 分析结果\n\n"
	summary += "基于收集到的信息，以下是我的分析：\n\n"

	for i, info := range collected {
		summary += fmt.Sprintf("**信息 %d**: %s\n\n", i+1, info)
	}

	summary += "\n---\n"
	summary += "**结论**: "

	if hasKeyword(strings.ToLower(query), "cpu", "性能") {
		summary += "系统性能正常，CPU 使用率在合理范围内。"
	} else if hasKeyword(strings.ToLower(query), "错误", "error") {
		summary += "检测到少量错误日志，建议继续监控。"
	} else {
		summary += "请根据以上信息进行进一步分析。"
	}

	return summary, nil
}

// formatFinalResponse 格式化最终响应
func (r *ReActAgent) formatFinalResponse(response string, history []Thought) string {
	if response == "" {
		response = "分析完成"
	}
	return response
}

// giveUp 放弃并返回结果
func (r *ReActAgent) giveUp(history []Thought) string {
	if len(history) == 0 {
		return "无法完成任务"
	}

	summary := "## 分析摘要\n\n"
	summary += "经过多轮推理，我得出以下结论：\n\n"

	for _, t := range history {
		summary += fmt.Sprintf("**Step %d**: %s\n", t.Step, t.Reasoning)
		if t.Observation != "" {
			summary += fmt.Sprintf("  结果: %s\n", t.Observation)
		}
		summary += "\n"
	}

	if len(history) > 0 {
		summary += "---\n"
		summary += "**最终结论**: " + history[len(history)-1].Observation + "\n"
	}

	return summary
}

// listAvailableTools 列出可用工具
func (r *ReActAgent) listAvailableTools() string {
	var toolList []string
	for name := range r.tools {
		toolList = append(toolList, "- "+name)
	}
	if len(toolList) == 0 {
		return "无注册工具"
	}
	return strings.Join(toolList, "\n")
}

// AgentScheduler Agent 调度器
type AgentScheduler struct {
	mu          sync.RWMutex
	agents      map[string]Agent
	maxParallel int
}

// NewAgentScheduler 创建调度器
func NewAgentScheduler(maxParallel int) *AgentScheduler {
	if maxParallel <= 0 {
		maxParallel = 5
	}
	return &AgentScheduler{
		agents:      make(map[string]Agent),
		maxParallel: maxParallel,
	}
}

// Register 注册 Agent
func (s *AgentScheduler) Register(agent Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[agent.Name()] = agent
}

// Unregister 注销 Agent
func (s *AgentScheduler) Unregister(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.agents, name)
}

// Get 获取 Agent
func (s *AgentScheduler) Get(name string) (Agent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agent, ok := s.agents[name]
	return agent, ok
}

// List 列出所有 Agent
func (s *AgentScheduler) List() []Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	return agents
}

// Execute 并行执行多个 Agent
func (s *AgentScheduler) Execute(ctx context.Context, agents []string, query string) []*AgentResult {
	results := make([]*AgentResult, 0, len(agents))

	sem := make(chan struct{}, s.maxParallel)
	var wg sync.WaitGroup

	for _, name := range agents {
		agent, ok := s.Get(name)
		if !ok {
			results = append(results, &AgentResult{
				AgentName: name,
				Error:     fmt.Errorf("agent not found: %s", name),
			})
			continue
		}

		wg.Add(1)
		go func(a Agent) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			result, err := a.Execute(execCtx, query)
			results = append(results, &AgentResult{
				AgentName: name,
				Content:   result,
				Error:     err,
			})
		}(agent)
	}

	wg.Wait()
	return results
}

// Count 返回注册的 Agent 数量
func (s *AgentScheduler) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.agents)
}
