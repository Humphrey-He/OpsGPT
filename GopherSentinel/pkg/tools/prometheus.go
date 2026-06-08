package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PrometheusConfig Prometheus 配置
type PrometheusConfig struct {
	BaseURL string
}

// DefaultPrometheusConfig 默认配置
func DefaultPrometheusConfig() PrometheusConfig {
	return PrometheusConfig{
		BaseURL: "http://localhost:9090",
	}
}

// PrometheusData Prometheus 查询结果数据
type PrometheusData struct {
	ResultType string `json:"resultType"`
	Result     []struct {
		Metric map[string]string `json:"metric"`
		Value  []interface{}     `json:"value"`
	} `json:"result"`
}

// PrometheusResponse Prometheus API 响应
type PrometheusResponse struct {
	Status string          `json:"status"`
	Data   PrometheusData  `json:"data"`
	Error  string          `json:"error"`
}

// PrometheusTool Prometheus 查询工具
type PrometheusTool struct {
	baseURL string
}

// NewPrometheusTool 创建 Prometheus 工具
func NewPrometheusTool() *PrometheusTool {
	return &PrometheusTool{
		baseURL: DefaultPrometheusConfig().BaseURL,
	}
}

// Name 返回工具名称
func (t *PrometheusTool) Name() string {
	return "prometheus_query"
}

// Description 返回工具描述
func (t *PrometheusTool) Description() string {
	return "查询 Prometheus 监控指标，支持 CPU、内存、请求率、错误率等"
}

// Schema 返回参数模式
func (t *PrometheusTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "PromQL 查询表达式，如 'sum(rate(http_requests_total[5m]))'",
				"required":    true,
			},
			"time": map[string]interface{}{
				"type":        "string",
				"description": "查询时间点，RFC3339 格式或 Unix 时间戳，默认当前时间",
				"required":    false,
			},
		},
		"required": []string{"query"},
	}
}

// IsDangerous 返回是否危险操作
func (t *PrometheusTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *PrometheusTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query := ExtractStringParam(params, "query", "")
	if query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	// 构建 Prometheus API URL
	apiURL := fmt.Sprintf("%s/api/v1/query?query=%s", t.baseURL, query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// 如果 Prometheus 不可用，返回模拟数据
		return t.mockQueryResult(query), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Prometheus API error: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result PrometheusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Prometheus query failed: %s", result.Error)
	}

	return t.formatResult(result.Data), nil
}

// formatResult 格式化结果
func (t *PrometheusTool) formatResult(data PrometheusData) string {
	var results []string

	for _, r := range data.Result {
		labels := fmt.Sprintf("%v", r.Metric)
		var value string
		if len(r.Value) >= 2 {
			value = fmt.Sprintf("%v", r.Value[1])
		}
		results = append(results, fmt.Sprintf("Metric: %s\nValue: %s", labels, value))
	}

	if len(results) == 0 {
		return "No results found"
	}

	return fmt.Sprintf("Found %d results:\n%s", len(results), joinStrings(results, "\n---\n"))
}

// mockQueryResult 返回模拟查询结果
func (t *PrometheusTool) mockQueryResult(query string) interface{} {
	return map[string]interface{}{
		"status":   "success",
		"query":    query,
		"results": []map[string]interface{}{
			{
				"metric": map[string]string{
					"service": "example",
				},
				"value": 87.5,
			},
		},
		"_note": "Using mock data (Prometheus not available)",
	}
}

// PrometheusRangeTool Prometheus 范围查询工具
type PrometheusRangeTool struct {
	baseURL string
}

// NewPrometheusRangeTool 创建 Prometheus 范围查询工具
func NewPrometheusRangeTool() *PrometheusRangeTool {
	return &PrometheusRangeTool{
		baseURL: DefaultPrometheusConfig().BaseURL,
	}
}

// Name 返回工具名称
func (t *PrometheusRangeTool) Name() string {
	return "prometheus_range_query"
}

// Description 返回工具描述
func (t *PrometheusRangeTool) Description() string {
	return "查询 Prometheus 指标的时间范围数据，用于趋势分析"
}

// Schema 返回参数模式
func (t *PrometheusRangeTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "PromQL 查询表达式",
				"required":    true,
			},
			"start": map[string]interface{}{
				"type":        "string",
				"description": "开始时间，RFC3339 格式或 Unix 时间戳",
				"required":    true,
			},
			"end": map[string]interface{}{
				"type":        "string",
				"description": "结束时间，RFC3339 格式或 Unix 时间戳",
				"required":    true,
			},
			"step": map[string]interface{}{
				"type":        "string",
				"description": "查询步长，如 '15s', '1m', '5m'",
				"required":    false,
				"default":     "1m",
			},
		},
		"required": []string{"query", "start", "end"},
	}
}

// IsDangerous 返回是否危险操作
func (t *PrometheusRangeTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *PrometheusRangeTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query := ExtractStringParam(params, "query", "")
	start := ExtractStringParam(params, "start", "")
	end := ExtractStringParam(params, "end", "")
	step := ExtractStringParam(params, "step", "1m")

	if query == "" || start == "" || end == "" {
		return nil, fmt.Errorf("query, start, and end are required")
	}

	apiURL := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%s&end=%s&step=%s",
		t.baseURL, query, start, end, step)

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
			"data": []map[string]interface{}{
				{"timestamp": time.Now().Unix() - 3600, "value": 75.5},
				{"timestamp": time.Now().Unix() - 1800, "value": 82.3},
				{"timestamp": time.Now().Unix(), "value": 87.5},
			},
		}, nil
	}
	defer resp.Body.Close()

	var result struct {
		Status string         `json:"status"`
		Data   PrometheusData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// PrometheusInstantQuery 执行即时查询
func PrometheusInstantQuery(ctx context.Context, baseURL, query string) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/api/v1/query?query=%s", baseURL, query)

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// PrometheusRangeQuery 执行范围查询
func PrometheusRangeQuery(ctx context.Context, baseURL, query, start, end, step string) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%s&end=%s&step=%s",
		baseURL, query, start, end, step)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetTargets 获取 Prometheus targets
func GetTargets(ctx context.Context, baseURL string) ([]map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/api/v1/targets", baseURL)

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			ActiveTargets []map[string]interface{} `json:"activeTargets"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Data.ActiveTargets, nil
}

// GetAlerts 获取当前告警
func GetAlerts(ctx context.Context, baseURL string) ([]map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/api/v1/alerts", baseURL)

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Alerts []map[string]interface{} `json:"alerts"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Data.Alerts, nil
}

// helper functions
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString(strs[0])
	for _, s := range strs[1:] {
		buf.WriteString(sep)
		buf.WriteString(s)
	}
	return buf.String()
}
