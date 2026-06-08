package tools

import (
	"context"
	"fmt"
)

// PrometheusTool Prometheus 查询工具
type PrometheusTool struct {
	baseURL string
}

func NewPrometheusTool(baseURL string) *PrometheusTool {
	return &PrometheusTool{baseURL: baseURL}
}

func (t *PrometheusTool) Name() string {
	return "prometheus_query"
}

func (t *PrometheusTool) Description() string {
	return "Query Prometheus metrics. Supports CPU, memory, request rate, etc."
}

func (t *PrometheusTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"name": "prometheus_query",
		"description": t.Description(),
		"parameters": map[string]interface{}{
			"service": map[string]interface{}{
				"type":        "string",
				"description": "Service name",
				"required":    true,
				"enum":        []string{"order", "payment", "user", "inventory"},
			},
			"metric": map[string]interface{}{
				"type":        "string",
				"description": "Metric name",
				"required":    true,
				"enum":        []string{"cpu", "memory", "request_count", "error_rate"},
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Time range, e.g. '1h', '24h', '7d'",
				"required":    false,
			},
		},
	}
}

func (t *PrometheusTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	service, _ := params["service"].(string)
	metric, _ := params["metric"].(string)

	// 模拟 Prometheus 查询
	// 实际项目中需要调用 Prometheus HTTP API
	result := map[string]interface{}{
		"service": service,
		"metric":  metric,
		"value":   87.5,
		"unit":    "%",
	}

	return fmt.Sprintf("Prometheus query result for %s.%s: %.2f%%", service, metric, result["value"].(float64)), nil
}

// LokiTool Loki 日志查询工具
type LokiTool struct {
	baseURL string
}

func NewLokiTool(baseURL string) *LokiTool {
	return &LokiTool{baseURL: baseURL}
}

func (t *LokiTool) Name() string {
	return "loki_query"
}

func (t *LokiTool) Description() string {
	return "Query Loki logs. Supports filtering by service, level, and keywords."
}

func (t *LokiTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"name": "loki_query",
		"description": t.Description(),
		"parameters": map[string]interface{}{
			"service": map[string]interface{}{
				"type":        "string",
				"description": "Service name",
				"required":    true,
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Log query expression",
				"required":    true,
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of logs to return",
				"required":    false,
				"default":     100,
			},
		},
	}
}

func (t *LokiTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	service, _ := params["service"].(string)
	query, _ := params["query"].(string)

	// 模拟 Loki 查询
	return fmt.Sprintf("Loki logs for service=%s, query='%s':\n"+
		"[2024-01-15 10:30:01] INFO: Request received\n"+
		"[2024-01-15 10:30:02] DEBUG: Processing...", service, query), nil
}

// HealthCheckTool 健康检查工具
type HealthCheckTool struct{}

func NewHealthCheckTool() *HealthCheckTool {
	return &HealthCheckTool{}
}

func (t *HealthCheckTool) Name() string {
	return "health_check"
}

func (t *HealthCheckTool) Description() string {
	return "Check service health status"
}

func (t *HealthCheckTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"name":        "health_check",
		"description": t.Description(),
		"parameters": map[string]interface{}{
			"service": map[string]interface{}{
				"type":        "string",
				"description": "Service name",
				"required":    true,
			},
		},
	}
}

func (t *HealthCheckTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	service, _ := params["service"].(string)

	return fmt.Sprintf("Health check for %s: OK\n"+
		"- Status: healthy\n"+
		"- Uptime: 99.9%%\n"+
		"- Last check: 1 minute ago", service), nil
}
