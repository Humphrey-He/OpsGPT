package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Tool 定义工具接口
type Tool interface {
	// Name 返回工具名称
	Name() string
	// Description 返回工具描述
	Description() string
	// Schema 返回参数模式 (JSON Schema)
	Schema() map[string]interface{}
	// Execute 执行工具
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	// IsDangerous 返回是否危险操作
	IsDangerous() bool
}

// ToolResult 工具执行结果
type ToolResult struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall 工具调用请求
type ToolCall struct {
	ToolName string                 `json:"tool_name"`
	Params   map[string]interface{} `json:"parameters"`
}

// ToolRegistry 工具注册中心
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// 全局工具注册中心
var globalRegistry *ToolRegistry
var once sync.Once

// GetRegistry 获取全局注册中心
func GetRegistry() *ToolRegistry {
	once.Do(func() {
		globalRegistry = NewToolRegistry()
	})
	return globalRegistry
}

// NewToolRegistry 创建新的注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// Unregister 注销工具
func (r *ToolRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(r.tools, name)
	return nil
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List 列出所有工具
func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListByCategory 按类别列出工具
func (r *ToolRegistry) ListByCategory(category string) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []Tool
	for _, tool := range r.tools {
		if category == "" || strings.HasPrefix(tool.Name(), category+"_") {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetSchema 获取所有工具的 schema
func (r *ToolRegistry) GetSchema() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]map[string]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		schema := tool.Schema()
		schema["_dangerous"] = tool.IsDangerous()
		schemas = append(schemas, schema)
	}
	return schemas
}

// Execute 执行工具
func (r *ToolRegistry) Execute(ctx context.Context, call ToolCall) (*ToolResult, error) {
	tool, exists := r.Get(call.ToolName)
	if !exists {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("tool %s not found", call.ToolName),
		}, nil
	}

	data, err := tool.Execute(ctx, call.Params)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    data,
	}, nil
}

// Count 返回注册的工具数量
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// BaseTool 基础工具实现
type BaseTool struct {
	NameValue        string
	DescriptionValue string
	SchemaValue      map[string]interface{}
	Dangerous        bool
	Handler          func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// Name 返回工具名称
func (t *BaseTool) Name() string {
	return t.NameValue
}

// Description 返回工具描述
func (t *BaseTool) Description() string {
	return t.DescriptionValue
}

// Schema 返回参数模式
func (t *BaseTool) Schema() map[string]interface{} {
	return t.SchemaValue
}

// IsDangerous 返回是否危险操作
func (t *BaseTool) IsDangerous() bool {
	return t.Dangerous
}

// Execute 执行工具
func (t *BaseTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if t.Handler == nil {
		return nil, fmt.Errorf("handler not implemented")
	}
	return t.Handler(ctx, params)
}

// NewBaseTool 创建基础工具
func NewBaseTool(name, description string, schema map[string]interface{}, dangerous bool,
	handler func(ctx context.Context, params map[string]interface{}) (interface{}, error)) *BaseTool {
	if schema == nil {
		schema = map[string]interface{}{
			"name":        name,
			"description": description,
			"parameters": map[string]interface{}{},
		}
	}
	return &BaseTool{
		NameValue:        name,
		DescriptionValue: description,
		SchemaValue:      schema,
		Dangerous:        dangerous,
		Handler:          handler,
	}
}

// ToolExecutor 工具执行器
type ToolExecutor struct {
	registry    *ToolRegistry
	security    *SecurityChecker
	confirmation ConfirmationHandler
}

// NewToolExecutor 创建工具执行器
func NewToolExecutor(registry *ToolRegistry) *ToolExecutor {
	return &ToolExecutor{
		registry:    registry,
		security:    NewSecurityChecker(),
		confirmation: NewDefaultConfirmation(),
	}
}

// SetConfirmationHandler 设置确认处理器
func (e *ToolExecutor) SetConfirmationHandler(handler ConfirmationHandler) {
	e.confirmation = handler
}

// Execute 执行工具（带确认）
func (e *ToolExecutor) Execute(ctx context.Context, call ToolCall) (*ToolResult, error) {
	// 1. 获取工具
	tool, exists := e.registry.Get(call.ToolName)
	if !exists {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("tool %s not found", call.ToolName),
		}, nil
	}

	// 2. 安全检查
	if err := e.security.Check(call); err != nil {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("security check failed: %s", err.Error()),
		}, nil
	}

	// 3. 危险操作确认
	if tool.IsDangerous() {
		confirmed, err := e.confirmation.ConfirmDangerous(ctx, call.ToolName, call.Params)
		if err != nil {
			return nil, err
		}
		if !confirmed {
			return &ToolResult{
				Success: false,
				Error:   "operation cancelled by user",
			}, nil
		}
	}

	// 4. 执行工具
	data, err := tool.Execute(ctx, call.Params)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    data,
	}, nil
}

// ExecuteBatch 批量执行工具
func (e *ToolExecutor) ExecuteBatch(ctx context.Context, calls []ToolCall) []*ToolResult {
	results := make([]*ToolResult, len(calls))
	for i, call := range calls {
		result, _ := e.Execute(ctx, call)
		results[i] = result
	}
	return results
}

// ExecuteParallel 并行执行工具
func (e *ToolExecutor) ExecuteParallel(ctx context.Context, calls []ToolCall) []*ToolResult {
	type result struct {
		result *ToolResult
		err    error
	}

	results := make([]*ToolResult, len(calls))
	ch := make(chan result, len(calls))

	for i, call := range calls {
		go func(idx int, c ToolCall) {
			r, err := e.Execute(ctx, c)
			ch <- result{result: r, err: err}
		}(i, call)
	}

	for i := 0; i < len(calls); i++ {
		res := <-ch
		if res.err != nil {
			results[i] = &ToolResult{Success: false, Error: res.err.Error()}
		} else {
			results[i] = res.result
		}
	}

	return results
}

// ConfirmationHandler 确认处理器接口
type ConfirmationHandler interface {
	ConfirmDangerous(ctx context.Context, toolName string, params map[string]interface{}) (bool, error)
	ConfirmAction(ctx context.Context, message string) (bool, error)
}

// DefaultConfirmation 默认确认处理器
type DefaultConfirmation struct{}

// NewDefaultConfirmation 创建默认确认
func NewDefaultConfirmation() *DefaultConfirmation {
	return &DefaultConfirmation{}
}

// ConfirmDangerous 确认危险操作
func (c *DefaultConfirmation) ConfirmDangerous(ctx context.Context, toolName string, params map[string]interface{}) (bool, error) {
	return true, nil // 默认允许
}

// ConfirmAction 确认操作
func (c *DefaultConfirmation) ConfirmAction(ctx context.Context, message string) (bool, error) {
	return true, nil
}

// InteractiveConfirmation 交互式确认
type InteractiveConfirmation struct{}

// NewInteractiveConfirmation 创建交互式确认
func NewInteractiveConfirmation() *InteractiveConfirmation {
	return &InteractiveConfirmation{}
}

// ToOpenAISchema 转换为 OpenAI 函数调用格式
func ToOpenAISchema(registry *ToolRegistry) []map[string]interface{} {
	tools := registry.List()
	schemas := make([]map[string]interface{}, len(tools))

	for i, tool := range tools {
		schema := tool.Schema()
		schemas[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  schema,
			},
		}
	}

	return schemas
}

// ExtractStringParam 提取字符串参数
func ExtractStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// ExtractIntParam 提取整数参数
func ExtractIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// ExtractBoolParam 提取布尔参数
func ExtractBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := params[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// ExtractSliceParam 提取 slice 参数
func ExtractSliceParam(params map[string]interface{}, key string) []interface{} {
	if val, ok := params[key]; ok {
		if s, ok := val.([]interface{}); ok {
			return s
		}
	}
	return nil
}

// SetFieldIfNotNil 设置字段值
func SetFieldIfNotNil(obj interface{}, fieldName string, value interface{}) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		return
	}
	v = v.Elem()

	field := v.FieldByName(fieldName)
	if !field.IsValid() || !field.CanSet() {
		return
	}

	if value == nil {
		return
	}

	switch field.Kind() {
	case reflect.String:
		if s, ok := value.(string); ok {
			field.SetString(s)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int:
			field.SetInt(int64(v))
		case int64:
			field.SetInt(v)
		case float64:
			field.SetInt(int64(v))
		}
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
		}
	}
}

// ToJSON 将工具转换为 JSON 格式
func ToolToJSON(tool Tool) (string, error) {
	data := map[string]interface{}{
		"name":        tool.Name(),
		"description": tool.Description(),
		"schema":      tool.Schema(),
		"dangerous":   tool.IsDangerous(),
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ValidateParameters 验证参数
func ValidateParameters(params map[string]interface{}, schema map[string]interface{}) error {
	if schema == nil {
		return nil
	}

	required, _ := schema["required"].([]interface{})
	if required == nil {
		return nil
	}

	for _, req := range required {
		reqName, ok := req.(string)
		if !ok {
			continue
		}
		if _, exists := params[reqName]; !exists {
			return fmt.Errorf("missing required parameter: %s", reqName)
		}
	}

	return nil
}

// RegisterDefaultTools 注册默认工具集
func RegisterDefaultTools(registry *ToolRegistry) {
	// Prometheus 工具
	registry.Register(NewPrometheusTool())

	// Loki 工具
	registry.Register(NewLokiTool())

	// HTTP 工具
	registry.Register(NewHTTPGetTool())
	registry.Register(NewHTTPPostTool())

	// Health 工具
	registry.Register(NewHealthCheckTool())
}
