package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LokiConfig Loki 配置
type LokiConfig struct {
	BaseURL string
}

// DefaultLokiConfig 默认配置
func DefaultLokiConfig() LokiConfig {
	return LokiConfig{
		BaseURL: "http://localhost:3100",
	}
}

// LokiLogData Loki 日志数据结构
type LokiLogData struct {
	ResultType string `json:"resultType"`
	Result     []struct {
		Stream map[string]string `json:"stream"`
		Values [][]interface{}   `json:"values"`
	} `json:"result"`
}

// LokiResponse Loki API 响应
type LokiResponse struct {
	Status string      `json:"status"`
	Data   LokiLogData `json:"data"`
	Error  string      `json:"error"`
}

// LokiTool Loki 日志查询工具
type LokiTool struct {
	baseURL string
}

// NewLokiTool 创建 Loki 工具
func NewLokiTool() *LokiTool {
	return &LokiTool{
		baseURL: DefaultLokiConfig().BaseURL,
	}
}

// Name 返回工具名称
func (t *LokiTool) Name() string {
	return "loki_query"
}

// Description 返回工具描述
func (t *LokiTool) Description() string {
	return "查询 Loki 日志，支持按服务、日志级别、关键词过滤"
}

// Schema 返回参数模式
func (t *LokiTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "LogQL 查询表达式，如 '{service=\"order\"} |= \"error\"'",
				"required":    true,
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "返回日志条数上限",
				"required":    false,
				"default":     100,
			},
			"start": map[string]interface{}{
				"type":        "string",
				"description": "开始时间，RFC3339 格式或 Unix 时间戳 (纳秒)",
				"required":    false,
			},
			"end": map[string]interface{}{
				"type":        "string",
				"description": "结束时间，RFC3339 格式或 Unix 时间戳 (纳秒)",
				"required":    false,
			},
			"direction": map[string]interface{}{
				"type":        "string",
				"description": "日志排序方向: 'forward' 或 'backward'",
				"required":    false,
				"default":     "backward",
			},
		},
		"required": []string{"query"},
	}
}

// IsDangerous 返回是否危险操作
func (t *LokiTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *LokiTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query := ExtractStringParam(params, "query", "")
	if query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	limit := ExtractIntParam(params, "limit", 100)
	direction := ExtractStringParam(params, "direction", "backward")

	// 构建查询参数
	v := url.Values{}
	v.Set("query", query)
	v.Set("limit", fmt.Sprintf("%d", limit))
	v.Set("direction", direction)

	// 添加时间范围
	if start, ok := params["start"].(string); ok && start != "" {
		v.Set("start", t.parseTime(start))
	}
	if end, ok := params["end"].(string); ok && end != "" {
		v.Set("end", t.parseTime(end))
	}

	apiURL := fmt.Sprintf("%s/loki/api/v1/query?%s", t.baseURL, v.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// 如果 Loki 不可用，返回模拟数据
		return t.mockQueryResult(query, limit), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Loki API error: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result LokiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Loki query failed: %s", result.Error)
	}

	return t.formatLogResult(result.Data, limit), nil
}

// parseTime 解析时间字符串
func (t *LokiTool) parseTime(timeStr string) string {
	// 如果已经是数字字符串，直接返回
	if _, err := fmt.Sscan(timeStr, new(float64)); err == nil {
		return timeStr
	}

	// 尝试解析为时间
	timestamp, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}

	// 返回纳秒时间戳
	return fmt.Sprintf("%d", timestamp.UnixNano())
}

// formatLogResult 格式化日志结果
func (t *LokiTool) formatLogResult(data LokiLogData, limit int) map[string]interface{} {
	var logs []map[string]interface{}
	count := 0

	for _, stream := range data.Result {
		labels := stream.Stream
		for _, value := range stream.Values {
			if count >= limit {
				break
			}
			if len(value) >= 2 {
				timestamp, _ := value[0].(string)
				line, _ := value[1].(string)

				// 解析时间戳
				var ts time.Time
				var ns int64
				if _, err := fmt.Sscanf(timestamp, "%d", &ns); err == nil {
					ts = time.Unix(0, ns)
				}

				logs = append(logs, map[string]interface{}{
					"timestamp": ts.Format(time.RFC3339),
					"labels":    labels,
					"message":   line,
				})
				count++
			}
		}
		if count >= limit {
			break
		}
	}

	return map[string]interface{}{
		"status": "success",
		"count":  len(logs),
		"logs":   logs,
	}
}

// mockQueryResult 返回模拟查询结果
func (t *LokiTool) mockQueryResult(query string, limit int) map[string]interface{} {
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
			"labels":    map[string]string{"service": "order", "level": "info"},
			"message":   "[INFO] Request received: GET /api/orders/123",
		},
		{
			"timestamp": time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
			"labels":    map[string]string{"service": "order", "level": "debug"},
			"message":   "[DEBUG] Processing order: 123",
		},
		{
			"timestamp": time.Now().Add(-3 * time.Minute).Format(time.RFC3339),
			"labels":    map[string]string{"service": "order", "level": "info"},
			"message":   "[INFO] Order processed successfully: 123",
		},
	}

	if limit < len(logs) {
		logs = logs[:limit]
	}

	return map[string]interface{}{
		"status": "mock",
		"query":  query,
		"count":  len(logs),
		"logs":   logs,
		"_note":  "Using mock data (Loki not available)",
	}
}

// LokiRangeTool Loki 范围查询工具（返回更多日志）
type LokiRangeTool struct {
	baseURL string
}

// NewLokiRangeTool 创建 Loki 范围查询工具
func NewLokiRangeTool() *LokiRangeTool {
	return &LokiRangeTool{
		baseURL: DefaultLokiConfig().BaseURL,
	}
}

// Name 返回工具名称
func (t *LokiRangeTool) Name() string {
	return "loki_range_query"
}

// Description 返回工具描述
func (t *LokiRangeTool) Description() string {
	return "查询 Loki 日志的时间范围数据，支持更多过滤条件"
}

// Schema 返回参数模式
func (t *LokiRangeTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "LogQL 查询表达式",
				"required":    true,
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "返回日志条数上限",
				"required":    false,
				"default":     500,
			},
			"start": map[string]interface{}{
				"type":        "string",
				"description": "开始时间，RFC3339 格式",
				"required":    true,
			},
			"end": map[string]interface{}{
				"type":        "string",
				"description": "结束时间，RFC3339 格式",
				"required":    true,
			},
		},
		"required": []string{"query", "start", "end"},
	}
}

// IsDangerous 返回是否危险操作
func (t *LokiRangeTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *LokiRangeTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query := ExtractStringParam(params, "query", "")
	start := ExtractStringParam(params, "start", "")
	end := ExtractStringParam(params, "end", "")
	limit := ExtractIntParam(params, "limit", 500)

	if query == "" || start == "" || end == "" {
		return nil, fmt.Errorf("query, start, and end are required")
	}

	// 解析时间
	startNano := parseTimeToNanos(start)
	endNano := parseTimeToNanos(end)

	v := url.Values{}
	v.Set("query", query)
	v.Set("start", startNano)
	v.Set("end", endNano)
	v.Set("limit", fmt.Sprintf("%d", limit))

	apiURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", t.baseURL, v.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"status": "mock",
			"query":  query,
			"range":  fmt.Sprintf("%s to %s", start, end),
			"logs":   []map[string]interface{}{},
		}, nil
	}
	defer resp.Body.Close()

	var result struct {
		Status string      `json:"status"`
		Data   LokiLogData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return t.formatRangeLogResult(result.Data, limit), nil
}

// formatRangeLogResult 格式化范围日志结果
func (t *LokiRangeTool) formatRangeLogResult(data LokiLogData, limit int) map[string]interface{} {
	return t.formatLogResult(data, limit)
}

// formatLogResult 格式化日志结果
func (t *LokiRangeTool) formatLogResult(data LokiLogData, limit int) map[string]interface{} {
	var logs []map[string]interface{}
	count := 0

	for _, stream := range data.Result {
		labels := stream.Stream
		for _, value := range stream.Values {
			if count >= limit {
				break
			}
			if len(value) >= 2 {
				timestamp, _ := value[0].(string)
				line, _ := value[1].(string)

				var ts time.Time
				var ns int64
				if _, err := fmt.Sscanf(timestamp, "%d", &ns); err == nil {
					ts = time.Unix(0, ns)
				}

				logs = append(logs, map[string]interface{}{
					"timestamp": ts.Format(time.RFC3339),
					"labels":   labels,
					"message":  line,
				})
				count++
			}
		}
		if count >= limit {
			break
		}
	}

	return map[string]interface{}{
		"status": "success",
		"count":  len(logs),
		"logs":   logs,
	}
}

// parseTimeToNanos 解析时间为纳秒时间戳
func parseTimeToNanos(timeStr string) string {
	if _, err := fmt.Sscan(timeStr, new(float64)); err == nil {
		return timeStr
	}

	timestamp, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	return fmt.Sprintf("%d", timestamp.UnixNano())
}

// LokiServiceLogsTool 查询特定服务的日志
type LokiServiceLogsTool struct {
	baseURL string
}

// NewLokiServiceLogsTool 创建服务日志查询工具
func NewLokiServiceLogsTool() *LokiServiceLogsTool {
	return &LokiServiceLogsTool{
		baseURL: DefaultLokiConfig().BaseURL,
	}
}

// Name 返回工具名称
func (t *LokiServiceLogsTool) Name() string {
	return "loki_service_logs"
}

// Description 返回工具描述
func (t *LokiServiceLogsTool) Description() string {
	return "查询特定服务的最近日志，按日志级别过滤"
}

// Schema 返回参数模式
func (t *LokiServiceLogsTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"service": map[string]interface{}{
				"type":        "string",
				"description": "服务名称",
				"required":    true,
			},
			"level": map[string]interface{}{
				"type":        "string",
				"description": "日志级别: info, warn, error, debug",
				"required":    false,
				"default":     "info",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "返回日志条数",
				"required":    false,
				"default":     50,
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "时间范围，如 '1h', '30m', '24h'",
				"required":    false,
				"default":     "1h",
			},
		},
		"required": []string{"service"},
	}
}

// IsDangerous 返回是否危险操作
func (t *LokiServiceLogsTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *LokiServiceLogsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	service := ExtractStringParam(params, "service", "")
	level := ExtractStringParam(params, "level", "info")
	limit := ExtractIntParam(params, "limit", 50)
	timeRange := ExtractStringParam(params, "time_range", "1h")

	if service == "" {
		return nil, fmt.Errorf("service parameter is required")
	}

	// 构建 LogQL 查询
	query := fmt.Sprintf(`{service="%s"}`, service)
	if level != "" && level != "all" {
		query += fmt.Sprintf(` |= "%s"`, strings.ToUpper(level))
	}

	// 计算时间范围
	duration, err := parseDuration(timeRange)
	if err != nil {
		duration = time.Hour
	}

	end := time.Now()
	start := end.Add(-duration)

	tool := NewLokiRangeTool()
	params["query"] = query
	params["start"] = start.Format(time.RFC3339)
	params["end"] = end.Format(time.RFC3339)
	params["limit"] = limit

	return tool.Execute(ctx, params)
}

// parseDuration 解析时间范围字符串
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	switch {
	case strings.HasSuffix(s, "h"):
		var hours int
		fmt.Sscanf(s, "%dh", &hours)
		return time.Duration(hours) * time.Hour, nil
	case strings.HasSuffix(s, "m"):
		var mins int
		fmt.Sscanf(s, "%dm", &mins)
		return time.Duration(mins) * time.Minute, nil
	case strings.HasSuffix(s, "d"):
		var days int
		fmt.Sscanf(s, "%dd", &days)
		return time.Duration(days) * 24 * time.Hour, nil
	default:
		return time.Hour, nil
	}
}

// GetLokiLabels 获取所有标签
func GetLokiLabels(ctx context.Context, baseURL string) ([]string, error) {
	apiURL := fmt.Sprintf("%s/loki/api/v1/label", baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetLokiLabelValues 获取标签值
func GetLokiLabelValues(ctx context.Context, baseURL, labelName string) ([]string, error) {
	apiURL := fmt.Sprintf("%s/loki/api/v1/label/%s/values", baseURL, labelName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
