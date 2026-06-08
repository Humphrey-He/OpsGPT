package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	URL       string
	Secret    string
	Timeout   time.Duration
	Retry     int
	Enabled   bool
}

// DefaultWebhookConfig 默认配置
func DefaultWebhookConfig() WebhookConfig {
	return WebhookConfig{
		URL:     "",
		Secret:  "",
		Timeout: 30 * time.Second,
		Retry:   3,
		Enabled: false,
	}
}

// WebhookNotifier Webhook 通知器
type WebhookNotifier struct {
	config WebhookConfig
	client *http.Client
}

// NewWebhookNotifier 创建 Webhook 通知器
func NewWebhookNotifier(config WebhookConfig) *WebhookNotifier {
	return &WebhookNotifier{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Message 通知消息
type Message struct {
	Type      string                 `json:"type"`
	Title     string                `json:"title"`
	Content   string                `json:"content"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp int64                 `json:"timestamp"`
}

// SendDangerousOperationAlert 发送危险操作告警
func (n *WebhookNotifier) SendDangerousOperationAlert(ctx context.Context, req *DangerousOperationRequest) error {
	if !n.config.Enabled {
		return nil
	}

	message := Message{
		Type:    "dangerous_operation",
		Title:   fmt.Sprintf("危险操作确认请求: %s", req.ToolName),
		Content: req.FormatMessage(),
		Data: map[string]interface{}{
			"tool_name":     req.ToolName,
			"parameters":    req.Parameters,
			"user_id":       req.UserID,
			"session_id":    req.SessionID,
			"risk_level":    req.RiskLevel,
			"operation_id":   req.OperationID,
		},
		Timestamp: time.Now().Unix(),
	}

	return n.Send(ctx, message)
}

// SendOperationResult 发送操作结果通知
func (n *WebhookNotifier) SendOperationResult(ctx context.Context, req *OperationResultRequest) error {
	if !n.config.Enabled {
		return nil
	}

	message := Message{
		Type:    "operation_result",
		Title:   fmt.Sprintf("操作执行结果: %s", req.ToolName),
		Content: req.FormatMessage(),
		Data: map[string]interface{}{
			"tool_name":     req.ToolName,
			"success":       req.Success,
			"result":        req.Result,
			"user_id":       req.UserID,
			"session_id":    req.SessionID,
			"duration_ms":   req.Duration,
		},
		Timestamp: time.Now().Unix(),
	}

	return n.Send(ctx, message)
}

// SendErrorAlert 发送错误告警
func (n *WebhookNotifier) SendErrorAlert(ctx context.Context, req *ErrorAlertRequest) error {
	if !n.config.Enabled {
		return nil
	}

	message := Message{
		Type:    "error_alert",
		Title:   fmt.Sprintf("系统错误告警: %s", req.ErrorType),
		Content: req.FormatMessage(),
		Data: map[string]interface{}{
			"error_type":    req.ErrorType,
			"error_message": req.ErrorMessage,
			"context":        req.Context,
			"user_id":       req.UserID,
			"session_id":    req.SessionID,
		},
		Timestamp: time.Now().Unix(),
	}

	return n.Send(ctx, message)
}

// Send 发送消息
func (n *WebhookNotifier) Send(ctx context.Context, msg Message) error {
	if n.config.URL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 重试逻辑
	var lastErr error
	for i := 0; i <= n.config.Retry; i++ {
		err := n.doSend(ctx, data)
		if err == nil {
			return nil
		}
		lastErr = err

		// 指数退避
		if i < n.config.Retry {
			time.Sleep(time.Duration(1<<i) * time.Second)
		}
	}

	return fmt.Errorf("failed to send webhook after %d retries: %w", n.config.Retry, lastErr)
}

// doSend 执行 HTTP POST
func (n *WebhookNotifier) doSend(ctx context.Context, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.config.URL, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if n.config.Secret != "" {
		req.Header.Set("X-Webhook-Secret", n.config.Secret)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// DangerousOperationRequest 危险操作确认请求
type DangerousOperationRequest struct {
	OperationID string
	ToolName    string
	Parameters  map[string]interface{}
	UserID      string
	SessionID   string
	RiskLevel   string // "low", "medium", "high", "critical"
	Reason      string
}

// FormatMessage 格式化消息
func (r *DangerousOperationRequest) FormatMessage() string {
	return fmt.Sprintf(`🔴 危险操作确认请求

工具: %s
操作ID: %s
风险等级: %s
原因: %s

参数:
%s

用户: %s
会话: %s

请确认是否执行此操作。`, r.ToolName, r.OperationID, r.RiskLevel, r.Reason,
		formatParams(r.Parameters), r.UserID, r.SessionID)
}

// OperationResultRequest 操作结果请求
type OperationResultRequest struct {
	ToolName  string
	Success   bool
	Result    interface{}
	UserID    string
	SessionID string
	Duration  int64 // milliseconds
}

// FormatMessage 格式化消息
func (r *OperationResultRequest) FormatMessage() string {
	status := "✅ 成功"
	if !r.Success {
		status = "❌ 失败"
	}

	return fmt.Sprintf(`%s 操作完成

工具: %s
耗时: %dms
用户: %s
会话: %s

结果: %v`, status, r.ToolName, r.Duration, r.UserID, r.SessionID, r.Result)
}

// ErrorAlertRequest 错误告警请求
type ErrorAlertRequest struct {
	ErrorType    string
	ErrorMessage string
	Context      map[string]interface{}
	UserID       string
	SessionID    string
}

// FormatMessage 格式化消息
func (r *ErrorAlertRequest) FormatMessage() string {
	return fmt.Sprintf(`⚠️ 系统错误告警

错误类型: %s
错误信息: %s
用户: %s
会话: %s

上下文:
%s`, r.ErrorType, r.ErrorMessage, r.UserID, r.SessionID,
		formatParams(r.Context))
}

// formatParams 格式化参数字典
func formatParams(params map[string]interface{}) string {
	if params == nil {
		return "无"
	}

	var lines []string
	for k, v := range params {
		lines = append(lines, fmt.Sprintf("  %s: %v", k, v))
	}

	if len(lines) == 0 {
		return "无"
	}

	return joinStrings(lines, "\n")
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}

// DingTalkNotifier 钉钉通知器
type DingTalkNotifier struct {
	webhook *WebhookNotifier
	token   string
}

// NewDingTalkNotifier 创建钉钉通知器
func NewDingTalkNotifier(webhookURL, token string) *DingTalkNotifier {
	return &DingTalkNotifier{
		webhook: NewWebhookNotifier(WebhookConfig{
			URL:     webhookURL,
			Enabled: true,
		}),
		token: token,
	}
}

// SendDingTalkMessage 发送钉钉消息
func (n *DingTalkNotifier) SendDingTalkMessage(ctx context.Context, msg *DingTalkMessage) error {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": msg.Title,
			"text":  msg.Text,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", n.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// DingTalkMessage 钉钉消息
type DingTalkMessage struct {
	Title string
	Text  string
}

// ConfirmHandler 确认处理器
type ConfirmHandler struct {
	notifier *WebhookNotifier
	timeout  time.Duration
}

// NewConfirmHandler 创建确认处理器
func NewConfirmHandler(notifier *WebhookNotifier, timeout time.Duration) *ConfirmHandler {
	return &ConfirmHandler{
		notifier: notifier,
		timeout:  timeout,
	}
}

// RequestConfirmation 请求确认
func (h *ConfirmHandler) RequestConfirmation(ctx context.Context, req *DangerousOperationRequest) (*ConfirmResult, error) {
	// 1. 发送确认请求
	if err := h.notifier.SendDangerousOperationAlert(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to send confirmation request: %w", err)
	}

	// 2. 等待确认（实际实现中应该等待 webhook 回调或用户确认）
	// 这里简化处理，超时后返回 false
	select {
	case <-ctx.Done():
		return &ConfirmResult{
			Approved: false,
			Reason:   "timeout",
		}, nil
	case <-time.After(h.timeout):
		return &ConfirmResult{
			Approved: false,
			Reason:   "timeout",
		}, nil
	}
}

// ConfirmResult 确认结果
type ConfirmResult struct {
	Approved bool
	Reason   string
	Message  string
}

// InteractiveConfirmHandler 交互式确认处理器
type InteractiveConfirmHandler struct {
	confirmCh chan *ConfirmRequest
	resultCh  chan *ConfirmResult
}

// NewInteractiveConfirmHandler 创建交互式确认处理器
func NewInteractiveConfirmHandler() *InteractiveConfirmHandler {
	return &InteractiveConfirmHandler{
		confirmCh: make(chan *ConfirmRequest, 10),
		resultCh:  make(chan *ConfirmResult, 10),
	}
}

// RequestConfirmation 请求确认
func (h *InteractiveConfirmHandler) RequestConfirmation(req *ConfirmRequest) {
	h.confirmCh <- req
}

// Confirm 确认
func (h *InteractiveConfirmHandler) Confirm(req *ConfirmRequest, approved bool, message string) {
	h.resultCh <- &ConfirmResult{
		Approved: approved,
		Reason:   message,
		Message:  message,
	}
}

// ConfirmRequest 确认请求
type ConfirmRequest struct {
	ID        string
	ToolName  string
	Params    map[string]interface{}
	RiskLevel string
	Message   string
}
