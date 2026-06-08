package tools

import (
	"context"
	"fmt"
	"strings"
)

// SecurityChecker 安全检查器
type SecurityChecker struct {
	dangerousPatterns []string
}

// NewSecurityChecker 创建安全检查器
func NewSecurityChecker() *SecurityChecker {
	return &SecurityChecker{
		dangerousPatterns: []string{
			"rm -rf",
			"kubectl delete",
			"drop table",
			"shutdown",
			"exit(",
			"system(",
			"exec(",
			"eval(",
		},
	}
}

// Check 检查是否危险操作
func (c *SecurityChecker) Check(call *ToolCall) error {
	// 检查工具名
	for _, pattern := range c.dangerousPatterns {
		if strings.Contains(strings.ToLower(call.ToolName), pattern) {
			return fmt.Errorf("dangerous tool: %s", call.ToolName)
		}
	}

	// 检查参数
	for key, value := range call.Params {
		if v, ok := value.(string); ok {
			for _, pattern := range c.dangerousPatterns {
				if strings.Contains(strings.ToLower(v), pattern) {
					return fmt.Errorf("dangerous pattern detected in param %s", key)
				}
			}
		}
	}

	return nil
}

// ConfirmService 确认服务
type ConfirmService struct {
	// 实际项目中可以接入 Webhook/邮件/钉钉等通知
}

// RequestConfirm 请求确认
func (s *ConfirmService) RequestConfirm(ctx context.Context, req *ConfirmRequest) (bool, error) {
	// 模拟确认流程
	// 实际项目中需要用户确认后才能继续
	return true, nil
}

// ConfirmRequest 确认请求
type ConfirmRequest struct {
	Tool     string
	Params   map[string]interface{}
	Reason   string
	UserID   string
}
