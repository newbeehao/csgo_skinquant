package llm

import "encoding/json"

// ====================== 请求相关 ======================

// ChatRequest 是发给 DeepSeek /chat/completions 的请求体
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`       // 可选: Function Calling 工具列表
	Temperature float64   `json:"temperature,omitempty"` // 可选: 0-2, 越高越随机
	Stream      bool      `json:"stream,omitempty"`      // 可选: 是否流式返回
}

// Message 是对话中的一条消息
// Role 可以是: "system" / "user" / "assistant" / "tool"
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`                // 注意: assistant 触发 tool_calls 时 content 可能为空
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 仅 assistant 消息可能有
	ToolCallID string     `json:"tool_call_id,omitempty"` // 仅 role=tool 时必填
	Name       string     `json:"name,omitempty"`         // 仅 role=tool 时可选, 工具名
}

// Tool 描述一个可用的工具 (函数)
type Tool struct {
	Type     string       `json:"type"` // 目前固定为 "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction 是工具的具体定义
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema 格式
}

// ToolCall 表示模型决定要调用一个工具
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function ToolCallFunc `json:"function"`
}

// ToolCallFunc 是模型要调用的具体函数和参数
// Arguments 是 JSON 字符串 (不是对象), 由我们自己解析
type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ====================== 响应相关 ======================

// ChatResponse 是 DeepSeek 返回的响应体
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 是模型的一条回复候选
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"` // "stop" / "tool_calls" / "length" 等
}

// Usage 记录 token 消耗 (后面做成本统计会用到)
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ====================== 错误响应 ======================

// APIError 是 DeepSeek 返回的错误结构
type APIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// ParseArguments 是个小帮手: 把 ToolCall.Function.Arguments (JSON 字符串) 解析成 map
// 在 Agent 执行工具时会用到
func (tc *ToolCall) ParseArguments() (map[string]any, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return nil, err
	}
	return args, nil
}
