package tools

import (
	"context"

	"github.com/newbeehao/csgo_skinquant/internal/llm"
)

// Tool 是所有工具必须实现的接口
// Agent 执行循环时只认这个接口, 不关心底层是 mock 还是真实 HTTP 调用
type Tool interface {
	// Name 返回工具名, 必须和 Definition().Function.Name 一致
	Name() string

	// Definition 返回工具的 LLM 可读定义 (JSON Schema)
	// 这是传给 DeepSeek 的 tools 字段内容
	Definition() llm.Tool

	// Execute 执行工具, 接收 LLM 解析好的参数, 返回字符串结果
	// 返回的字符串会被塞回 messages 让 LLM 继续看
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// Registry 是工具注册表, 方便按名字查找工具
type Registry struct {
	tools map[string]Tool
}

// NewRegistry 创建一个空的工具注册表
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 注册一个工具
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get 按名字查找工具
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Definitions 返回所有已注册工具的 LLM 定义列表
// Agent 会把这个列表传给 LLM 的 tools 参数
func (r *Registry) Definitions() []llm.Tool {
	defs := make([]llm.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	return defs
}
