package tools

import (
	"context"
	"fmt"
	"sync"
)

// ToolRegistry 工具注册中心
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewToolRegistry 创建工具注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name()]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name())
	}

	r.tools[tool.Name()] = tool
	return nil
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// List 获取所有工具
func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetSchema 获取工具 Schema（供 LLM 使用）
func (r *ToolRegistry) GetSchema() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]map[string]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, tool.Schema())
	}
	return schemas
}

// Tool 接口
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]interface{}
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ToolExecutor 工具执行器
type ToolExecutor struct {
	registry  *ToolRegistry
	security *SecurityChecker
}

// NewToolExecutor 创建工具执行器
func NewToolExecutor(registry *ToolRegistry) *ToolExecutor {
	return &ToolExecutor{
		registry:  registry,
		security: NewSecurityChecker(),
	}
}

// ToolCall 工具调用请求
type ToolCall struct {
	ToolName string
	Params   map[string]interface{}
}

// ToolResult 工具执行结果
type ToolResult struct {
	Success bool
	Data    interface{}
	Error   error
}

// Execute 执行工具
func (e *ToolExecutor) Execute(ctx context.Context, call *ToolCall) (*ToolResult, error) {
	// 1. 获取工具
	tool, ok := e.registry.Get(call.ToolName)
	if !ok {
		return nil, fmt.Errorf("tool %s not found", call.ToolName)
	}

	// 2. 安全检查
	if err := e.security.Check(call); err != nil {
		return nil, err
	}

	// 3. 执行工具
	data, err := tool.Execute(ctx, call.Params)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err,
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    data,
	}, nil
}
