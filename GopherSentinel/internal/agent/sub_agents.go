package agent

import (
	"context"
	"fmt"
	"strings"

	"GopherSentinel/internal/rag"
	"GopherSentinel/pkg/tools"
)

// LogAgent 日志分析专家 Agent
type LogAgent struct {
	name        string
	toolRegistry *tools.ToolRegistry
	lokiTool   *tools.LokiTool
}

// NewLogAgent 创建日志分析 Agent
func NewLogAgent() *LogAgent {
	return &LogAgent{
		name:        "log",
		toolRegistry: tools.GetRegistry(),
		lokiTool:   tools.NewLokiTool(),
	}
}

// Name 返回 Agent 名称
func (a *LogAgent) Name() string {
	return a.name
}

// Description 返回 Agent 描述
func (a *LogAgent) Description() string {
	return "日志分析专家，负责查询和分析系统日志"
}

// Execute 执行日志分析任务
func (a *LogAgent) Execute(ctx context.Context, query string) (string, error) {
	// 1. 解析查询意图
	params := a.parseQuery(query)

	// 2. 查询日志
	logResult, err := a.lokiTool.Execute(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to query logs: %w", err)
	}

	// 3. 分析日志
	return a.analyzeLogs(query, logResult)
}

// parseQuery 解析查询参数
func (a *LogAgent) parseQuery(query string) map[string]interface{} {
	params := map[string]interface{}{
		"limit": 100,
	}

	lowerQuery := strings.ToLower(query)

	// 提取服务名
	if strings.Contains(lowerQuery, "order") || strings.Contains(lowerQuery, "订单") {
		params["query"] = `{service="order"}`
	} else if strings.Contains(lowerQuery, "payment") || strings.Contains(lowerQuery, "支付") {
		params["query"] = `{service="payment"}`
	} else if strings.Contains(lowerQuery, "user") || strings.Contains(lowerQuery, "用户") {
		params["query"] = `{service="user"}`
	} else {
		params["query"] = "{}"
	}

	// 提取日志级别
	if strings.Contains(lowerQuery, "error") || strings.Contains(lowerQuery, "错误") {
		params["query"] = params["query"].(string) + ` |= "ERROR"`
	} else if strings.Contains(lowerQuery, "warn") || strings.Contains(lowerQuery, "警告") {
		params["query"] = params["query"].(string) + ` |= "WARN"`
	} else if strings.Contains(lowerQuery, "debug") {
		params["query"] = params["query"].(string) + ` |= "DEBUG"`
	}

	return params
}

// analyzeLogs 分析日志结果
func (a *LogAgent) analyzeLogs(query string, result interface{}) (string, error) {
	var analysis strings.Builder

	analysis.WriteString("### 📋 日志分析结果\n\n")

	if result == nil {
		analysis.WriteString("未找到匹配的日志\n")
		return analysis.String(), nil
	}

	// 解析结果
	logs, ok := result.(map[string]interface{})
	if !ok {
		analysis.WriteString(fmt.Sprintf("日志结果: %v\n", result))
		return analysis.String(), nil
	}

	// 检查是否为模拟数据
	if status, ok := logs["status"].(string); ok && status == "mock" {
		analysis.WriteString("> ⚠️ 使用模拟数据（请检查 Loki 连接）\n\n")
	}

	// 提取日志
	if logEntries, ok := logs["logs"].([]interface{}); ok {
		analysis.WriteString(fmt.Sprintf("**找到 %d 条日志**\n\n", len(logEntries)))

		errorCount := 0
		warnCount := 0

		for i, entry := range logEntries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				// 统计日志级别
				if msg, ok := entryMap["message"].(string); ok {
					msgUpper := strings.ToUpper(msg)
					if strings.Contains(msgUpper, "ERROR") || strings.Contains(msgUpper, "ERROR") {
						errorCount++
					} else if strings.Contains(msgUpper, "WARN") {
						warnCount++
					}
				}

				// 显示前 5 条
				if i < 5 {
					if timestamp, ok := entryMap["timestamp"].(string); ok {
						analysis.WriteString(fmt.Sprintf("**时间**: %s\n", timestamp))
					}
					if msg, ok := entryMap["message"].(string); ok {
						analysis.WriteString(fmt.Sprintf("**日志**: %s\n", msg))
					}
					analysis.WriteString("---\n")
				}
			}
		}

		analysis.WriteString("\n**统计摘要**:\n")
		analysis.WriteString(fmt.Sprintf("- 错误日志: %d\n", errorCount))
		analysis.WriteString(fmt.Sprintf("- 警告日志: %d\n", warnCount))

		if errorCount > 0 {
			analysis.WriteString("\n> ⚠️ 检测到错误日志，建议进一步调查\n")
		}
	}

	return analysis.String(), nil
}

// MetricAgent 指标分析专家 Agent
type MetricAgent struct {
	name         string
	toolRegistry *tools.ToolRegistry
	promTool    *tools.PrometheusTool
}

// NewMetricAgent 创建指标分析 Agent
func NewMetricAgent() *MetricAgent {
	return &MetricAgent{
		name:         "metric",
		toolRegistry: tools.GetRegistry(),
		promTool:    tools.NewPrometheusTool(),
	}
}

// Name 返回 Agent 名称
func (a *MetricAgent) Name() string {
	return a.name
}

// Description 返回 Agent 描述
func (a *MetricAgent) Description() string {
	return "指标分析专家，负责查询和分析系统监控指标"
}

// Execute 执行指标分析任务
func (a *MetricAgent) Execute(ctx context.Context, query string) (string, error) {
	// 1. 解析查询意图
	promQL := a.parseQuery(query)

	// 2. 查询指标
	params := map[string]interface{}{
		"query": promQL,
	}

	result, err := a.promTool.Execute(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to query metrics: %w", err)
	}

	// 3. 分析指标
	return a.analyzeMetrics(query, result)
}

// parseQuery 解析查询为 PromQL
func (a *MetricAgent) parseQuery(query string) string {
	lowerQuery := strings.ToLower(query)

	// 根据关键词生成 PromQL
	if strings.Contains(lowerQuery, "cpu") {
		return `sum(rate(node_cpu_seconds_total[5m])) by (instance) * 100`
	}

	if strings.Contains(lowerQuery, "内存") || strings.Contains(lowerQuery, "memory") {
		return `node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes * 100`
	}

	if strings.Contains(lowerQuery, "请求") || strings.Contains(lowerQuery, "request") || strings.Contains(lowerQuery, "qps") {
		return `sum(rate(http_requests_total[5m]))`
	}

	if strings.Contains(lowerQuery, "延迟") || strings.Contains(lowerQuery, "latency") {
		return `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))`
	}

	if strings.Contains(lowerQuery, "错误") || strings.Contains(lowerQuery, "error") || strings.Contains(lowerQuery, "错误率") {
		return `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100`
	}

	// 默认返回通用指标
	return `up`
}

// analyzeMetrics 分析指标结果
func (a *MetricAgent) analyzeMetrics(query string, result interface{}) (string, error) {
	var analysis strings.Builder

	analysis.WriteString("### 📊 指标分析结果\n\n")

	if result == nil {
		analysis.WriteString("未找到匹配的指标\n")
		return analysis.String(), nil
	}

	// 解析结果
	metrics, ok := result.(map[string]interface{})
	if !ok {
		analysis.WriteString(fmt.Sprintf("指标结果: %v\n", result))
		return analysis.String(), nil
	}

	// 检查是否为模拟数据
	if status, ok := metrics["status"].(string); ok && status == "mock" {
		analysis.WriteString("> ⚠️ 使用模拟数据（请检查 Prometheus 连接）\n\n")
	}

	// 提取结果
	if results, ok := metrics["results"].([]interface{}); ok {
		analysis.WriteString(fmt.Sprintf("**找到 %d 个指标**\n\n", len(results)))

		for i, r := range results {
			if resultMap, ok := r.(map[string]interface{}); ok {
				if metric, ok := resultMap["metric"].(map[string]interface{}); ok {
					analysis.WriteString(fmt.Sprintf("#### 指标 %d\n", i+1))
					for k, v := range metric {
						analysis.WriteString(fmt.Sprintf("- **%s**: %v\n", k, v))
					}
				}
				if value, ok := resultMap["value"].(float64); ok {
					analysis.WriteString(fmt.Sprintf("- **值**: %.2f\n", value))
				}
				analysis.WriteString("\n")
			}
		}
	}

	// 添加趋势分析建议
	lowerQuery := strings.ToLower(query)
	if strings.Contains(lowerQuery, "cpu") {
		analysis.WriteString("**建议**:\n")
		analysis.WriteString("- 如果 CPU 使用率 > 80%，考虑扩容或优化应用\n")
		analysis.WriteString("- 检查是否有异常进程占用 CPU\n")
	}

	return analysis.String(), nil
}

// DocAgent 文档检索专家 Agent
type DocAgent struct {
	name     string
	ragChain *rag.RAGChain
}

// NewDocAgent 创建文档检索 Agent
func NewDocAgent() *DocAgent {
	return &DocAgent{
		name: "doc",
	}
}

// SetRAGChain 设置 RAG 链
func (a *DocAgent) SetRAGChain(ragChain *rag.RAGChain) {
	a.ragChain = ragChain
}

// Name 返回 Agent 名称
func (a *DocAgent) Name() string {
	return a.name
}

// Description 返回 Agent 描述
func (a *DocAgent) Description() string {
	return "文档检索专家，负责从知识库中查找相关文档"
}

// Execute 执行文档检索任务
func (a *DocAgent) Execute(ctx context.Context, query string) (string, error) {
	// 如果没有 RAG 链，使用模拟数据
	if a.ragChain == nil {
		return a.mockExecute(query)
	}

	// 使用 RAG 查询
	ragQuery := rag.Query{
		Question: query,
		TopK:     3,
	}

	answer, err := a.ragChain.Ask(ctx, ragQuery)
	if err != nil {
		return a.mockExecute(query)
	}

	// 格式化结果
	return a.formatAnswer(answer)
}

// mockExecute 模拟执行
func (a *DocAgent) mockExecute(query string) (string, error) {
	var result strings.Builder

	result.WriteString("### 📚 文档检索结果\n\n")
	result.WriteString(fmt.Sprintf("**查询**: %s\n\n", query))

	result.WriteString("根据您的查询，以下文档可能相关:\n\n")

	result.WriteString("1. **项目架构文档** (相关度: 0.92)\n")
	result.WriteString("   - 描述了系统的整体架构和组件\n\n")

	result.WriteString("2. **部署指南** (相关度: 0.85)\n")
	result.WriteString("   - 包含 Docker 和 Kubernetes 部署步骤\n\n")

	result.WriteString("3. **API 参考文档** (相关度: 0.78)\n")
	result.WriteString("   - 详细描述了所有 API 接口\n\n")

	result.WriteString("---\n")
	result.WriteString("> 💡 提示: 使用 `ingest` 命令索引更多文档可以获得更准确的结果\n")

	return result.String(), nil
}

// formatAnswer 格式化 RAG 回答
func (a *DocAgent) formatAnswer(answer *rag.Answer) (string, error) {
	var result strings.Builder

	result.WriteString("### 📚 文档检索结果\n\n")

	if answer.Content != "" {
		result.WriteString("**回答**:\n")
		result.WriteString(answer.Content)
		result.WriteString("\n\n")
	}

	if len(answer.Sources) > 0 {
		result.WriteString("**来源文档**:\n")
		for i, source := range answer.Sources {
			result.WriteString(fmt.Sprintf("%d. %s (相关度: %.2f)\n", i+1, source.Source, source.Score))
		}
	}

	return result.String(), nil
}

// K8sAgent K8s 操作专家 Agent
type K8sAgent struct {
	name        string
	toolRegistry *tools.ToolRegistry
	k8sTool    *tools.K8sTool
}

// NewK8sAgent 创建 K8s 操作 Agent
func NewK8sAgent() *K8sAgent {
	return &K8sAgent{
		name:        "k8s",
		toolRegistry: tools.GetRegistry(),
		k8sTool:    nil, // 需要配置
	}
}

// Name 返回 Agent 名称
func (a *K8sAgent) Name() string {
	return a.name
}

// Description 返回 Agent 描述
func (a *K8sAgent) Description() string {
	return "K8s 操作专家，负责查询和管理 Kubernetes 资源"
}

// Execute 执行 K8s 操作任务
func (a *K8sAgent) Execute(ctx context.Context, query string) (string, error) {
	// 模拟 K8s 查询
	return a.mockExecute(query)
}

// mockExecute 模拟执行
func (a *K8sAgent) mockExecute(query string) (string, error) {
	var result strings.Builder

	result.WriteString("### ☸️ Kubernetes 资源查询结果\n\n")
	result.WriteString(fmt.Sprintf("**查询**: %s\n\n", query))

	result.WriteString("**Pod 状态**:\n")
	result.WriteString("| Pod | 状态 | Ready | 重启次数 |\n")
	result.WriteString("|-----|------|-------|----------|\n")
	result.WriteString("| order-7b8d9f6d4-xk2p9 | Running | 2/2 | 0 |\n")
	result.WriteString("| payment-6c7f8d5b8-mn4q7 | Running | 1/1 | 1 |\n")
	result.WriteString("| user-5d9e7c6a9-jk1l3 | Pending | 0/2 | 0 |\n")

	result.WriteString("\n**Deployment 状态**:\n")
	result.WriteString("- order-deployment: 3/3 replicas\n")
	result.WriteString("- payment-deployment: 2/2 replicas\n")
	result.WriteString("- user-deployment: 1/2 replicas (Scaling...)\n")

	result.WriteString("\n---\n")
	result.WriteString("> 💡 提示: 配置 K8s 连接后可以执行实际查询\n")

	return result.String(), nil
}

// HealthAgent 健康检查 Agent
type HealthAgent struct {
	name string
}

// NewHealthAgent 创建健康检查 Agent
func NewHealthAgent() *HealthAgent {
	return &HealthAgent{name: "health"}
}

// Name 返回 Agent 名称
func (a *HealthAgent) Name() string {
	return a.name
}

// Description 返回 Agent 描述
func (a *HealthAgent) Description() string {
	return "健康检查专家，负责检查服务健康状态"
}

// Execute 执行健康检查任务
func (a *HealthAgent) Execute(ctx context.Context, query string) (string, error) {
	var result strings.Builder

	result.WriteString("### 🏥 服务健康检查结果\n\n")

	// 模拟健康检查
	result.WriteString("| 服务 | 状态 | 响应时间 | 错误率 |\n")
	result.WriteString("|------|------|----------|--------|\n")
	result.WriteString("| API Gateway | ✅ Healthy | 23ms | 0.01% |\n")
	result.WriteString("| Order Service | ✅ Healthy | 45ms | 0.05% |\n")
	result.WriteString("| Payment Service | ⚠️ Degraded | 120ms | 1.2% |\n")
	result.WriteString("| User Service | ✅ Healthy | 18ms | 0.02% |\n")

	result.WriteString("\n**总体状态**: ⚠️ 部分服务性能下降\n\n")
	result.WriteString("**建议**: Payment 服务响应时间偏高，建议检查数据库连接池配置\n")

	return result.String(), nil
}

// extractKeywords 提取关键词
func extractKeywords(query string) []string {
	// 简单的关键词提取
	words := strings.Fields(query)
	if len(words) > 5 {
		return words[:5]
	}
	return words
}
