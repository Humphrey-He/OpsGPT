package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPConfig HTTP 工具配置
type HTTPConfig struct {
	Timeout    time.Duration
	MaxRetries int
}

// DefaultHTTPConfig 默认配置
func DefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
}

// HTTPGetTool HTTP GET 请求工具
type HTTPGetTool struct {
	config HTTPConfig
}

// NewHTTPGetTool 创建 HTTP GET 工具
func NewHTTPGetTool() *HTTPGetTool {
	return &HTTPGetTool{config: DefaultHTTPConfig()}
}

// Name 返回工具名称
func (t *HTTPGetTool) Name() string {
	return "http_get"
}

// Description 返回工具描述
func (t *HTTPGetTool) Description() string {
	return "发送 HTTP GET 请求，支持查询参数、请求头"
}

// Schema 返回参数模式
func (t *HTTPGetTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "请求 URL",
				"required":    true,
			},
			"headers": map[string]interface{}{
				"type":        "object",
				"description": "HTTP 请求头，键值对形式",
				"required":    false,
			},
			"params": map[string]interface{}{
				"type":        "object",
				"description": "URL 查询参数",
				"required":    false,
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "请求超时时间（秒）",
				"required":    false,
			},
		},
		"required": []string{"url"},
	}
}

// IsDangerous 返回是否危险操作
func (t *HTTPGetTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *HTTPGetTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	targetURL := ExtractStringParam(params, "url", "")
	if targetURL == "" {
		return nil, fmt.Errorf("url parameter is required")
	}

	// 构建 URL（添加查询参数）
	if params["params"] != nil {
		if queryParams, ok := params["params"].(map[string]interface{}); ok {
			v := url.Values{}
			for key, val := range queryParams {
				v.Set(key, fmt.Sprintf("%v", val))
			}
			if strings.Contains(targetURL, "?") {
				targetURL += "&" + v.Encode()
			} else {
				targetURL += "?" + v.Encode()
			}
		}
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加请求头
	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if str, ok := v.(string); ok {
				req.Header.Set(k, str)
			}
		}
	}

	// 设置超时
	timeout := t.config.Timeout
	if params["timeout"] != nil {
		if t, ok := params["timeout"].(int); ok {
			timeout = time.Duration(t) * time.Second
		}
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析响应
	contentType := resp.Header.Get("Content-Type")
	var data interface{}
	if strings.Contains(contentType, "application/json") {
		if err := json.Unmarshal(body, &data); err != nil {
			data = string(body)
		}
	} else {
		data = string(body)
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        data,
		"size":        len(body),
	}, nil
}

// HTTPPostTool HTTP POST 请求工具
type HTTPPostTool struct {
	config HTTPConfig
}

// NewHTTPPostTool 创建 HTTP POST 工具
func NewHTTPPostTool() *HTTPPostTool {
	return &HTTPPostTool{config: DefaultHTTPConfig()}
}

// Name 返回工具名称
func (t *HTTPPostTool) Name() string {
	return "http_post"
}

// Description 返回工具描述
func (t *HTTPPostTool) Description() string {
	return "发送 HTTP POST 请求，支持 JSON body"
}

// Schema 返回参数模式
func (t *HTTPPostTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "请求 URL",
				"required":    true,
			},
			"body": map[string]interface{}{
				"type":        "object",
				"description": "POST 请求体（JSON 格式）",
				"required":    false,
			},
			"headers": map[string]interface{}{
				"type":        "object",
				"description": "HTTP 请求头",
				"required":    false,
			},
			"content_type": map[string]interface{}{
				"type":        "string",
				"description": "Content-Type，如 'application/json'",
				"required":    false,
				"default":     "application/json",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "请求超时时间（秒）",
				"required":    false,
			},
		},
		"required": []string{"url"},
	}
}

// IsDangerous 返回是否危险操作
func (t *HTTPPostTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *HTTPPostTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	targetURL := ExtractStringParam(params, "url", "")
	if targetURL == "" {
		return nil, fmt.Errorf("url parameter is required")
	}

	contentType := ExtractStringParam(params, "content_type", "application/json")

	// 构建请求体
	var body []byte
	var err error
	if params["body"] != nil {
		body, err = json.Marshal(params["body"])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	// 添加自定义请求头
	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if str, ok := v.(string); ok {
				req.Header.Set(k, str)
			}
		}
	}

	// 设置超时
	timeout := t.config.Timeout
	if params["timeout"] != nil {
		if t, ok := params["timeout"].(int); ok {
			timeout = time.Duration(t) * time.Second
		}
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析响应
	contentType = resp.Header.Get("Content-Type")
	var data interface{}
	if strings.Contains(contentType, "application/json") {
		if err := json.Unmarshal(respBody, &data); err != nil {
			data = string(respBody)
		}
	} else {
		data = string(respBody)
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        data,
		"size":        len(respBody),
	}, nil
}

// HTTPHealthCheckTool HTTP 健康检查工具
type HTTPHealthCheckTool struct{}

// NewHTTPHealthCheckTool 创建健康检查工具
func NewHTTPHealthCheckTool() *HTTPHealthCheckTool {
	return &HTTPHealthCheckTool{}
}

// Name 返回工具名称
func (t *HTTPHealthCheckTool) Name() string {
	return "http_health_check"
}

// Description 返回工具描述
func (t *HTTPHealthCheckTool) Description() string {
	return "检查 HTTP 服务的健康状态，支持 GET /health 端点"
}

// Schema 返回参数模式
func (t *HTTPHealthCheckTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "服务 URL",
				"required":    true,
			},
			"endpoint": map[string]interface{}{
				"type":        "string",
				"description": "健康检查端点路径",
				"required":    false,
				"default":     "/health",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "超时时间（秒）",
				"required":    false,
				"default":     5,
			},
		},
		"required": []string{"url"},
	}
}

// IsDangerous 返回是否危险操作
func (t *HTTPHealthCheckTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *HTTPHealthCheckTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	targetURL := ExtractStringParam(params, "url", "")
	endpoint := ExtractStringParam(params, "endpoint", "/health")
	timeout := ExtractIntParam(params, "timeout", 5)

	if targetURL == "" {
		return nil, fmt.Errorf("url parameter is required")
	}

	// 构建完整 URL
	healthURL := strings.TrimSuffix(targetURL, "/") + endpoint

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	start := time.Now()

	var statusCode int
	var responseBody string
	var respErr error

	resp, err := client.Do(req)
	if err != nil {
		respErr = err
		statusCode = 0
	} else {
		statusCode = resp.StatusCode
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		responseBody = string(body)
	}

	duration := time.Since(start)

	// 判断健康状态
	healthy := statusCode >= 200 && statusCode < 300

	return map[string]interface{}{
		"healthy":     healthy,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"response":    responseBody,
		"error":       respErr != nil,
		"error_msg":   respErr,
	}, nil
}

// HealthCheckTool 健康检查工具（别名，用于便捷调用）
type HealthCheckTool struct{}

// NewHealthCheckTool 创建健康检查工具
func NewHealthCheckTool() *HealthCheckTool {
	return &HealthCheckTool{}
}

// Name 返回工具名称
func (t *HealthCheckTool) Name() string {
	return "health_check"
}

// Description 返回工具描述
func (t *HealthCheckTool) Description() string {
	return "检查服务健康状态"
}

// Schema 返回参数模式
func (t *HealthCheckTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"service": map[string]interface{}{
				"type":        "string",
				"description": "服务名称或 URL",
				"required":    true,
			},
		},
		"required": []string{"service"},
	}
}

// IsDangerous 返回是否危险操作
func (t *HealthCheckTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *HealthCheckTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	service := ExtractStringParam(params, "service", "")
	if service == "" {
		return nil, fmt.Errorf("service parameter is required")
	}

	// 判断是 URL 还是服务名
	var targetURL string
	if strings.HasPrefix(service, "http://") || strings.HasPrefix(service, "https://") {
		targetURL = service
	} else {
		// 假设是服务名，尝试拼接
		targetURL = fmt.Sprintf("http://%s:8080/health", service)
	}

	// 使用 HTTP 健康检查
	httpTool := NewHTTPHealthCheckTool()
	result, err := httpTool.Execute(ctx, map[string]interface{}{
		"url":      targetURL,
		"endpoint": "/health",
		"timeout":  5,
	})

	if err != nil {
		return map[string]interface{}{
			"service":  service,
			"healthy":  false,
			"error":    err.Error(),
		}, nil
	}

	if resp, ok := result.(map[string]interface{}); ok {
		resp["service"] = service
		return resp, nil
	}

	return result, nil
}
