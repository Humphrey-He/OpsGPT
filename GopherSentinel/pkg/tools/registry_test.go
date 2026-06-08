package tools

import (
	"context"
	"testing"
)

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	// 注册一个工具
	tool := &MockTool{name: "test_tool"}
	err := registry.Register(tool)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// 重复注册应该报错
	err = registry.Register(tool)
	if err == nil {
		t.Error("Register() expected error for duplicate registration")
	}
}

func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()

	// 获取不存在的工具
	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for nonexistent tool")
	}

	// 注册并获取
	tool := &MockTool{name: "test_tool"}
	registry.Register(tool)

	got, ok := registry.Get("test_tool")
	if !ok {
		t.Error("Get() should return true for existing tool")
	}
	if got.Name() != "test_tool" {
		t.Errorf("Get() name = %v, want %v", got.Name(), "test_tool")
	}
}

func TestToolRegistry_List(t *testing.T) {
	registry := NewToolRegistry()

	// 列表应该为空
	tools := registry.List()
	if len(tools) != 0 {
		t.Errorf("List() length = %d, want 0", len(tools))
	}

	// 注册工具
	registry.Register(&MockTool{name: "tool1"})
	registry.Register(&MockTool{name: "tool2"})

	tools = registry.List()
	if len(tools) != 2 {
		t.Errorf("List() length = %d, want 2", len(tools))
	}
}

func TestToolRegistry_GetSchema(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(&MockTool{name: "test_tool"})

	schemas := registry.GetSchema()
	if len(schemas) != 1 {
		t.Errorf("GetSchema() length = %d, want 1", len(schemas))
	}
}

func TestSecurityChecker_Check(t *testing.T) {
	checker := NewSecurityChecker()

	tests := []struct {
		name    string
		call    ToolCall
		wantErr bool
	}{
		{
			name: "normal call",
			call: ToolCall{
				ToolName: "prometheus_query",
				Params:   map[string]interface{}{"service": "order"},
			},
			wantErr: false,
		},
		{
			name: "dangerous tool name - delete pod",
			call: ToolCall{
				ToolName: "k8s_delete_pod",
				Params:   map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "dangerous param - path traversal",
			call: ToolCall{
				ToolName: "http_get",
				Params:   map[string]interface{}{"url": "../etc/passwd"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.Check(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToolExecutor_Execute(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(&MockTool{name: "test_tool"})

	executor := NewToolExecutor(registry)

	result, err := executor.Execute(context.Background(), ToolCall{
		ToolName: "test_tool",
		Params:   map[string]interface{}{},
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result == nil {
		t.Error("Execute() returned nil result")
	}
}

// MockTool 测试用工具
type MockTool struct {
	name      string
	dangerous bool
}

func (t *MockTool) Name() string {
	return t.name
}

func (t *MockTool) Description() string {
	return "A mock tool for testing"
}

func (t *MockTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"name":        t.name,
		"description": t.Description(),
	}
}

func (t *MockTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return "mock result", nil
}

func (t *MockTool) IsDangerous() bool {
	return t.dangerous
}
