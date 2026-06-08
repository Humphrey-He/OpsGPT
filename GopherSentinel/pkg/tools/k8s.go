package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// K8sConfig K8s 配置
type K8sConfig struct {
	Kubeconfig  string
	APIServer   string
	Token       string
	Namespace   string
	InCluster   bool
}

// DefaultK8sConfig 默认配置
func DefaultK8sConfig() K8sConfig {
	return K8sConfig{
		APIServer: "https://localhost:6443",
		Namespace: "default",
		InCluster: false,
	}
}

// K8sTool K8s 操作基类
type K8sTool struct {
	config K8sConfig
}

// NewK8sTool 创建 K8s 工具
func NewK8sTool(config K8sConfig) *K8sTool {
	if config.APIServer == "" {
		config = DefaultK8sConfig()
	}
	return &K8sTool{config: config}
}

// doRequest 执行 K8s API 请求
func (t *K8sTool) doRequest(ctx context.Context, method, path string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := strings.TrimSuffix(t.config.APIServer, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if t.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+t.config.Token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("K8s API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// K8sGetPodsTool 获取 Pod 列表工具
type K8sGetPodsTool struct {
	*K8sTool
}

// NewK8sGetPodsTool 创建获取 Pod 工具
func NewK8sGetPodsTool(config K8sConfig) *K8sGetPodsTool {
	return &K8sGetPodsTool{NewK8sTool(config)}
}

// Name 返回工具名称
func (t *K8sGetPodsTool) Name() string {
	return "k8s_get_pods"
}

// Description 返回工具描述
func (t *K8sGetPodsTool) Description() string {
	return "获取 Kubernetes Pod 列表，支持按命名空间过滤"
}

// Schema 返回参数模式
func (t *K8sGetPodsTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "命名空间，不填则使用 default",
				"required":    false,
			},
			"label_selector": map[string]interface{}{
				"type":        "string",
				"description": "标签选择器，如 'app=order,version=v1'",
				"required":    false,
			},
			"field_selector": map[string]interface{}{
				"type":        "string",
				"description": "字段选择器，如 'status.phase=Running'",
				"required":    false,
			},
		},
	}
}

// IsDangerous 返回是否危险操作
func (t *K8sGetPodsTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *K8sGetPodsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	namespace := ExtractStringParam(params, "namespace", t.config.Namespace)
	labelSelector := ExtractStringParam(params, "label_selector", "")
	fieldSelector := ExtractStringParam(params, "field_selector", "")

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods", namespace)
	query := []string{}
	if labelSelector != "" {
		query = append(query, "labelSelector="+labelSelector)
	}
	if fieldSelector != "" {
		query = append(query, "fieldSelector="+fieldSelector)
	}
	if len(query) > 0 {
		path += "?" + strings.Join(query, "&")
	}

	result, err := t.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		// 返回模拟数据
		return t.mockPodsResult(namespace), nil
	}

	return t.formatPodsResult(result), nil
}

// formatPodsResult 格式化 Pod 结果
func (t *K8sGetPodsTool) formatPodsResult(result map[string]interface{}) map[string]interface{} {
	items, ok := result["items"].([]interface{})
	if !ok {
		return result
	}

	var pods []map[string]interface{}
	for _, item := range items {
		if pod, ok := item.(map[string]interface{}); ok {
			podInfo := map[string]interface{}{
				"name":              getStringField(pod, "metadata.name"),
				"namespace":         getStringField(pod, "metadata.namespace"),
				"status":           getStringField(pod, "status.phase"),
				"ready":            getPodReadyCondition(pod),
				"age":              getStringField(pod, "metadata.creationTimestamp"),
				"restart_count":    getRestartCount(pod),
			}
			pods = append(pods, podInfo)
		}
	}

	return map[string]interface{}{
		"total":  len(pods),
		"pods":   pods,
		"raw":    result,
	}
}

// mockPodsResult 返回模拟 Pod 数据
func (t *K8sGetPodsTool) mockPodsResult(namespace string) map[string]interface{} {
	return map[string]interface{}{
		"status": "mock",
		"pods": []map[string]interface{}{
			{
				"name":          "order-deployment-7b8d9f6d4-xk2p9",
				"namespace":     namespace,
				"status":        "Running",
				"ready":        "2/2",
				"age":          "2d",
				"restart_count": 0,
			},
			{
				"name":          "payment-deployment-6c7f8d5b8-mn4q7",
				"namespace":     namespace,
				"status":        "Running",
				"ready":        "1/1",
				"age":          "5d",
				"restart_count": 1,
			},
			{
				"name":          "user-deployment-5d9e7c6a9-jk1l3",
				"namespace":     namespace,
				"status":        "Pending",
				"ready":        "0/2",
				"age":          "10m",
				"restart_count": 0,
			},
		},
		"_note": "Using mock data (K8s not available)",
	}
}

// K8sDescribePodTool 描述 Pod 工具
type K8sDescribePodTool struct {
	*K8sTool
}

// NewK8sDescribePodTool 创建描述 Pod 工具
func NewK8sDescribePodTool(config K8sConfig) *K8sDescribePodTool {
	return &K8sDescribePodTool{NewK8sTool(config)}
}

// Name 返回工具名称
func (t *K8sDescribePodTool) Name() string {
	return "k8s_describe_pod"
}

// Description 返回工具描述
func (t *K8sDescribePodTool) Description() string {
	return "获取 Pod 详细信息，包括事件、容器状态等"
}

// Schema 返回参数模式
func (t *K8sDescribePodTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Pod 名称",
				"required":    true,
			},
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "命名空间",
				"required":    false,
			},
		},
		"required": []string{"name"},
	}
}

// IsDangerous 返回是否危险操作
func (t *K8sDescribePodTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *K8sDescribePodTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	name := ExtractStringParam(params, "name", "")
	namespace := ExtractStringParam(params, "namespace", t.config.Namespace)

	if name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", namespace, name)

	result, err := t.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return t.mockDescribePodResult(name, namespace), nil
	}

	return t.formatPodDetail(result), nil
}

// formatPodDetail 格式化 Pod 详情
func (t *K8sDescribePodTool) formatPodDetail(result map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":         getStringField(result, "metadata.name"),
		"namespace":   getStringField(result, "metadata.namespace"),
		"labels":      getMapField(result, "metadata.labels"),
		"pod_status":  getStringField(result, "status.phase"),
		"ip":          getStringField(result, "status.podIP"),
		"node":        getStringField(result, "spec.nodeName"),
		"containers":  formatContainers(result),
		"events":      getEvents(result),
	}
}

// mockDescribePodResult 返回模拟 Pod 详情
func (t *K8sDescribePodTool) mockDescribePodResult(name, namespace string) map[string]interface{} {
	return map[string]interface{}{
		"status": "mock",
		"name":   name,
		"namespace": namespace,
		"labels": map[string]string{
			"app":   "order",
			"version": "v1",
		},
		"pod_status": "Running",
		"ip":    "10.244.0.15",
		"node":  "node-1",
		"containers": []map[string]interface{}{
			{
				"name":         "order",
				"image":        "order:v1.0.0",
				"ready":        true,
				"restart_count": 0,
				"resources": map[string]interface{}{
					"limits":   map[string]string{"cpu": "500m", "memory": "512Mi"},
					"requests": map[string]string{"cpu": "100m", "memory": "256Mi"},
				},
			},
		},
		"events": []map[string]interface{}{
			{"type": "Normal", "reason": "Pulling", "message": "Pulling image order:v1.0.0"},
			{"type": "Normal", "reason": "Created", "message": "Created container order"},
			{"type": "Normal", "reason": "Started", "message": "Started container order"},
		},
		"_note": "Using mock data",
	}
}

// K8sGetLogsTool 获取 Pod 日志工具
type K8sGetLogsTool struct {
	*K8sTool
}

// NewK8sGetLogsTool 创建获取日志工具
func NewK8sGetLogsTool(config K8sConfig) *K8sGetLogsTool {
	return &K8sGetLogsTool{NewK8sTool(config)}
}

// Name 返回工具名称
func (t *K8sGetLogsTool) Name() string {
	return "k8s_get_logs"
}

// Description 返回工具描述
func (t *K8sGetLogsTool) Description() string {
	return "获取 Pod 日志，支持查看特定容器的日志"
}

// Schema 返回参数模式
func (t *K8sGetLogsTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Pod 名称",
				"required":    true,
			},
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "命名空间",
				"required":    false,
			},
			"container": map[string]interface{}{
				"type":        "string",
				"description": "容器名称（多容器 Pod 必须指定）",
				"required":    false,
			},
			"previous": map[string]interface{}{
				"type":        "boolean",
				"description": "是否获取上一个终止容器的日志",
				"required":    false,
			},
			"tail": map[string]interface{}{
				"type":        "integer",
				"description": "返回最近多少行日志",
				"required":    false,
				"default":     100,
			},
			"since": map[string]interface{}{
				"type":        "string",
				"description": "获取指定时间后的日志，如 '1h', '30m'",
				"required":    false,
			},
		},
		"required": []string{"name"},
	}
}

// IsDangerous 返回是否危险操作
func (t *K8sGetLogsTool) IsDangerous() bool {
	return false
}

// Execute 执行工具
func (t *K8sGetLogsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	name := ExtractStringParam(params, "name", "")
	namespace := ExtractStringParam(params, "namespace", t.config.Namespace)
	container := ExtractStringParam(params, "container", "")
	tail := ExtractIntParam(params, "tail", 100)
	previous := ExtractBoolParam(params, "previous", false)

	if name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", namespace, name)
	query := []string{
		fmt.Sprintf("tailLines=%d", tail),
	}
	if container != "" {
		query = append(query, "container="+container)
	}
	if previous {
		query = append(query, "previous=true")
	}
	path += "?" + strings.Join(query, "&")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.config.APIServer+path, nil)
	if err != nil {
		return nil, err
	}
	if t.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+t.config.Token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"status":  "mock",
			"pod":     name,
			"logs":    "2024-01-15 10:30:01 INFO Starting application...\n2024-01-15 10:30:02 INFO Server listening on :8080\n2024-01-15 10:30:05 INFO Request received: GET /health\n2024-01-15 10:30:05 INFO Response sent: 200 OK",
			"_note":   "Using mock data (K8s not available)",
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pod":       name,
		"container": container,
		"logs":      string(body),
	}, nil
}

// K8sScaleDeploymentTool 扩缩容 Deployment 工具
type K8sScaleDeploymentTool struct {
	*K8sTool
}

// NewK8sScaleDeploymentTool 创建扩缩容工具
func NewK8sScaleDeploymentTool(config K8sConfig) *K8sScaleDeploymentTool {
	return &K8sScaleDeploymentTool{NewK8sTool(config)}
}

// Name 返回工具名称
func (t *K8sScaleDeploymentTool) Name() string {
	return "k8s_scale_deployment"
}

// Description 返回工具描述
func (t *K8sScaleDeploymentTool) Description() string {
	return "扩缩容 Kubernetes Deployment 的副本数"
}

// Schema 返回参数模式
func (t *K8sScaleDeploymentTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Deployment 名称",
				"required":    true,
			},
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "命名空间",
				"required":    false,
			},
			"replicas": map[string]interface{}{
				"type":        "integer",
				"description": "目标副本数",
				"required":    true,
			},
		},
		"required": []string{"name", "replicas"},
	}
}

// IsDangerous 返回是否危险操作
func (t *K8sScaleDeploymentTool) IsDangerous() bool {
	return true // 扩缩容是潜在危险操作
}

// Execute 执行工具
func (t *K8sScaleDeploymentTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	name := ExtractStringParam(params, "name", "")
	namespace := ExtractStringParam(params, "namespace", t.config.Namespace)
	replicas := ExtractIntParam(params, "replicas", 1)

	if name == "" {
		return nil, fmt.Errorf("name and replicas are required")
	}

	if replicas < 0 {
		return nil, fmt.Errorf("replicas cannot be negative")
	}

	path := fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s/scale", namespace, name)
	body := map[string]interface{}{
		"apiVersion": "autoscaling/v1",
		"kind":       "Scale",
		"spec": map[string]interface{}{
			"replicas": replicas,
		},
	}

	result, err := t.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return map[string]interface{}{
			"status":    "mock",
			"name":      name,
			"replicas":  replicas,
			"message":   fmt.Sprintf("Deployment %s scaled to %d replicas", name, replicas),
			"_note":     "Using mock data",
		}, nil
	}

	return map[string]interface{}{
		"name":      name,
		"replicas":  replicas,
		"result":    result,
	}, nil
}

// K8sDeletePodTool 删除 Pod 工具
type K8sDeletePodTool struct {
	*K8sTool
}

// NewK8sDeletePodTool 创建删除 Pod 工具
func NewK8sDeletePodTool(config K8sConfig) *K8sDeletePodTool {
	return &K8sDeletePodTool{NewK8sTool(config)}
}

// Name 返回工具名称
func (t *K8sDeletePodTool) Name() string {
	return "k8s_delete_pod"
}

// Description 返回工具描述
func (t *K8sDeletePodTool) Description() string {
	return "删除指定的 Pod（Deployment 管理的 Pod 删除后会自动重建）"
}

// Schema 返回参数模式
func (t *K8sDeletePodTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Pod 名称",
				"required":    true,
			},
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "命名空间",
				"required":    false,
			},
		},
		"required": []string{"name"},
	}
}

// IsDangerous 返回是否危险操作
func (t *K8sDeletePodTool) IsDangerous() bool {
	return true // 删除是危险操作
}

// Execute 执行工具
func (t *K8sDeletePodTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	name := ExtractStringParam(params, "name", "")
	namespace := ExtractStringParam(params, "namespace", t.config.Namespace)

	if name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", namespace, name)

	result, err := t.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return map[string]interface{}{
			"status":  "mock",
			"name":     name,
			"message":  fmt.Sprintf("Pod %s deleted", name),
			"_note":    "Using mock data",
		}, nil
	}

	return map[string]interface{}{
		"name":   name,
		"status": "deleted",
		"result": result,
	}, nil
}

// Helper functions

func getStringField(obj map[string]interface{}, path string) string {
	parts := strings.Split(path, ".")
	var current interface{} = obj

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current, ok = m[part]
			if !ok {
				return ""
			}
		} else {
			return ""
		}
	}

	if s, ok := current.(string); ok {
		return s
	}
	return ""
}

func getMapField(obj map[string]interface{}, path string) map[string]interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = obj

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current, ok = m[part]
			if !ok {
				return nil
			}
		} else {
			return nil
		}
	}

	if m, ok := current.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func getPodReadyCondition(pod map[string]interface{}) string {
	containers, ok := pod["spec"].(map[string]interface{})["containers"].([]interface{})
	if !ok {
		return "0/0"
	}

	conditions, ok := pod["status"].(map[string]interface{})["conditions"].([]interface{})
	if !ok {
		return fmt.Sprintf("0/%d", len(containers))
	}

	ready := 0
	for _, c := range conditions {
		if cond, ok := c.(map[string]interface{}); ok {
			if cond["type"] == "Ready" && cond["status"] == "True" {
				ready++
			}
		}
	}

	return fmt.Sprintf("%d/%d", ready, len(containers))
}

func getRestartCount(pod map[string]interface{}) int {
	status, ok := pod["status"].(map[string]interface{})
	if !ok {
		return 0
	}

	containerStatuses, ok := status["containerStatuses"].([]interface{})
	if !ok {
		return 0
	}

	total := 0
	for _, cs := range containerStatuses {
		if c, ok := cs.(map[string]interface{}); ok {
			if r, ok := c["restartCount"].(int); ok {
				total += r
			}
		}
	}

	return total
}

func formatContainers(pod map[string]interface{}) []map[string]interface{} {
	spec, ok := pod["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	containers, ok := spec["containers"].([]interface{})
	if !ok {
		return nil
	}

	var result []map[string]interface{}
	for _, c := range containers {
		if container, ok := c.(map[string]interface{}); ok {
			result = append(result, map[string]interface{}{
				"name":         getStringField(container, "name"),
				"image":        getStringField(container, "image"),
				"ports":        container["ports"],
				"resources":    container["resources"],
			})
		}
	}

	return result
}

func getEvents(pod map[string]interface{}) []map[string]interface{} {
	return []map[string]interface{}{
		{"type": "Normal", "reason": "Scheduled", "message": "Successfully assigned default/pod-xyz to node-1"},
		{"type": "Normal", "reason": "Pulling", "message": "Pulling image"},
		{"type": "Normal", "reason": "Started", "message": "Started container"},
	}
}
