package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// SecurityChecker 安全检查器
type SecurityChecker struct {
	dangerousPatterns []string
	dangerousTools    map[string]bool
	rateLimiter      *RateLimiter
}

// NewSecurityChecker 创建安全检查器
func NewSecurityChecker() *SecurityChecker {
	return &SecurityChecker{
		dangerousPatterns: []string{
			"rm -rf",
			"rm -r /",
			"drop table",
			"truncate table",
			"shutdown",
			"halt",
			"poweroff",
			"--all", // dangerous flags
		},
		dangerousTools: map[string]bool{
			"k8s_delete_pod":           true,
			"k8s_scale_deployment":      true,
			"kubectl_exec":              true,
			"database_drop":             true,
			"file_delete":               true,
			"system_shutdown":           true,
		},
		rateLimiter: NewRateLimiter(100, 60), // 100 requests per minute
	}
}

// Check 检查工具调用是否安全
func (c *SecurityChecker) Check(call ToolCall) error {
	// 1. 检查危险工具
	if c.dangerousTools[call.ToolName] {
		return fmt.Errorf("dangerous tool detected: %s", call.ToolName)
	}

	// 2. 检查危险模式
	for _, pattern := range c.dangerousPatterns {
		if strings.Contains(strings.ToLower(call.ToolName), pattern) {
			return fmt.Errorf("dangerous pattern in tool name: %s", pattern)
		}
	}

	// 3. 检查参数
	for key, value := range call.Params {
		if v, ok := value.(string); ok {
			// 检查命令注入
			if containsCommandInjection(v) {
				return fmt.Errorf("command injection detected in param %s", key)
			}
			// 检查 SQL 注入
			if containsSQLInjection(v) {
				return fmt.Errorf("SQL injection detected in param %s", key)
			}
			// 检查路径遍历
			if containsPathTraversal(v) {
				return fmt.Errorf("path traversal detected in param %s", key)
			}
		}
	}

	// 4. 速率限制检查
	if !c.rateLimiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

// containsCommandInjection 检测命令注入
func containsCommandInjection(s string) bool {
	patterns := []string{
		";", "|", "&", "`", "$(", "${",
		"\n", "\r", "&&", "||",
		"curl |", "wget |", "bash -c",
		"eval ", "exec ", "system ",
	}
	s = strings.ToLower(s)
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// containsSQLInjection 检测 SQL 注入
func containsSQLInjection(s string) bool {
	patterns := []string{
		"'", "\"", "--", "/*", "*/",
		"union", "select", "insert", "update", "delete",
		"drop", "create", "alter", "exec", "execute",
		"information_schema", "sys.tables", "sysobjects",
	}
	s = strings.ToLower(s)
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// containsPathTraversal 检测路径遍历
func containsPathTraversal(s string) bool {
	patterns := []string{
		"../", "..\\", "%2e%2e", "%252e",
		"etc/passwd", "c:\\windows", "c:\\boot",
	}
	s = strings.ToLower(s)
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// RateLimiter 速率限制器
type RateLimiter struct {
	requests int
	window   int
	counters map[string]int
}

const rateLimitWindow = 60 // 1 minute window

// NewRateLimiter 创建速率限制器
func NewRateLimiter(requests, window int) *RateLimiter {
	return &RateLimiter{
		requests: requests,
		window:   window,
		counters: make(map[string]int),
	}
}

// Allow 检查是否允许请求
func (r *RateLimiter) Allow() bool {
	// 简化实现：不做真实的时间窗口限制
	return true
}

// BlacklistChecker 黑名单检查器
type BlacklistChecker struct {
	blockedTools    map[string]bool
	blockedPatterns []string
}

// NewBlacklistChecker 创建黑名单检查器
func NewBlacklistChecker() *BlacklistChecker {
	return &BlacklistChecker{
		blockedTools: map[string]bool{
			"system_shutdown": true,
			"system_reboot":    true,
			"database_drop":    true,
		},
		blockedPatterns: []string{
			"rm -rf /*",
			"drop database",
			":(){:|:&};:",
		},
	}
}

// IsBlocked 检查是否在黑名单中
func (c *BlacklistChecker) IsBlocked(toolName string, params map[string]interface{}) bool {
	// 检查工具名
	if c.blockedTools[toolName] {
		return true
	}

	// 检查参数
	for _, value := range params {
		if v, ok := value.(string); ok {
			for _, pattern := range c.blockedPatterns {
				if strings.Contains(v, pattern) {
					return true
				}
			}
		}
	}

	return false
}

// ConfirmRequest 确认请求
type ConfirmRequest struct {
	Tool     string
	Params   map[string]interface{}
	Reason   string
	UserID   string
	SessionID string
}

// ConfirmResponse 确认响应
type ConfirmResponse struct {
	Approved bool
	Message  string
}

// ConfirmService 确认服务
type ConfirmService struct {
	blacklistChecker *BlacklistChecker
}

// NewConfirmService 创建确认服务
func NewConfirmService() *ConfirmService {
	return &ConfirmService{
		blacklistChecker: NewBlacklistChecker(),
	}
}

// RequireConfirmation 检查是否需要确认
func (s *ConfirmService) RequireConfirmation(call ToolCall) bool {
	// 检查是否在黑名单中
	if s.blacklistChecker.IsBlocked(call.ToolName, call.Params) {
		return true
	}

	// 危险工具总是需要确认
	dangerousTools := map[string]bool{
		"k8s_delete_pod":       true,
		"k8s_scale_deployment":  true,
		"database_delete":       true,
		"file_delete":           true,
	}

	return dangerousTools[call.ToolName]
}

// RequestConfirm 请求用户确认
func (s *ConfirmService) RequestConfirm(ctx context.Context, req ConfirmRequest) (bool, error) {
	// 实际实现中：
	// 1. 发送通知到钉钉/飞书/邮件
	// 2. 等待用户确认或超时
	// 3. 返回结果

	// 简化实现：总是返回 true
	return true, nil
}

// AuditLog 审计日志
type AuditLog struct {
	ToolName    string
	Params      map[string]interface{}
	Result      *ToolResult
	UserID      string
	SessionID   string
	Timestamp   int64
	Approved    bool
	ApprovedBy  string
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	logs []*AuditLog
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		logs: make([]*AuditLog, 0),
	}
}

// Log 记录审计日志
func (l *AuditLogger) Log(log *AuditLog) {
	l.logs = append(l.logs, log)
}

// GetLogs 获取审计日志
func (l *AuditLogger) GetLogs() []*AuditLog {
	return l.logs
}

// FilterByUser 过滤指定用户的日志
func (l *AuditLogger) FilterByUser(userID string) []*AuditLog {
	var result []*AuditLog
	for _, log := range l.logs {
		if log.UserID == userID {
			result = append(result, log)
		}
	}
	return result
}

// FilterByTool 过滤指定工具的日志
func (l *AuditLogger) FilterByTool(toolName string) []*AuditLog {
	var result []*AuditLog
	for _, log := range l.logs {
		if log.ToolName == toolName {
			result = append(result, log)
		}
	}
	return result
}

// DangerousPatternDetector 危险模式检测器
type DangerousPatternDetector struct {
	patterns []*regexp.Regexp
}

// NewDangerousPatternDetector 创建危险模式检测器
func NewDangerousPatternDetector() *DangerousPatternDetector {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`rm\s+-rf\s+/\s*\*`),
		regexp.MustCompile(`drop\s+(database|table)\s+\w+`),
		regexp.MustCompile(`delete\s+from\s+\w+\s+where\s+1\s*=\s*1`),
		regexp.MustCompile(`;\s*shutdown`),
		regexp.MustCompile(`eval\s*\(\s*\$`),
	}
	return &DangerousPatternDetector{patterns: patterns}
}

// Detect 检测危险模式
func (d *DangerousPatternDetector) Detect(s string) bool {
	for _, pattern := range d.patterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}
