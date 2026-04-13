package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/newbeehao/csgo_skinquant/internal/llm"
	"github.com/newbeehao/csgo_skinquant/internal/tools"
)

// Agent 是一个带工具能力的 LLM 代理
type Agent struct {
	name         string          // Agent 名字 (用于日志)
	systemPrompt string          // 系统提示, 定义 Agent 的角色和行为
	llmClient    *llm.Client     // LLM 客户端
	registry     *tools.Registry // 可用工具注册表
	maxTurns     int             // 最大循环轮数 (防死循环)
}

// Config 是创建 Agent 的参数
type Config struct {
	Name         string
	SystemPrompt string
	LLMClient    *llm.Client
	Registry     *tools.Registry
	MaxTurns     int // 0 表示使用默认值 10
}

// New 创建一个新的 Agent
func New(cfg Config) *Agent {
	maxTurns := cfg.MaxTurns
	if maxTurns == 0 {
		maxTurns = 10 // 默认最多 10 轮循环, 足够应对绝大多数场景
	}
	return &Agent{
		name:         cfg.Name,
		systemPrompt: cfg.SystemPrompt,
		llmClient:    cfg.LLMClient,
		registry:     cfg.Registry,
		maxTurns:     maxTurns,
	}
}

// Run 执行一次完整的 Agent 任务
// userMessage: 用户的问题
// 返回: 最终的文本回答, 以及错误
func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	// 1. 初始化对话历史
	messages := []llm.Message{
		{Role: "system", Content: a.systemPrompt},
		{Role: "user", Content: userMessage},
	}

	// 2. 准备工具定义 (只在第一轮需要传, 但每轮传也无妨)
	toolDefs := a.registry.Definitions()

	log.Printf("🤖 [%s] 开始处理任务: %s", a.name, userMessage)

	// 3. 进入主循环
	for turn := 1; turn <= a.maxTurns; turn++ {
		log.Printf("🔄 [%s] 第 %d 轮: 调用 LLM", a.name, turn)

		// 3.1 调用 LLM
		resp, err := a.llmClient.Chat(ctx, messages, toolDefs)
		if err != nil {
			return "", fmt.Errorf("[%s] 第%d轮 LLM 调用失败: %w", a.name, turn, err)
		}

		assistantMsg := resp.Choices[0].Message
		finishReason := resp.Choices[0].FinishReason

		// 3.2 把 assistant 的回复追加到 messages (无论是否有工具调用)
		messages = append(messages, assistantMsg)

		// 3.3 判断 LLM 是否完成了任务
		// 情况 A: 没有工具调用 -> 这是最终回答
		if len(assistantMsg.ToolCalls) == 0 {
			log.Printf("✅ [%s] 第 %d 轮: 给出最终回答 (finish_reason=%s)", a.name, turn, finishReason)
			return assistantMsg.Content, nil
		}

		// 情况 B: LLM 想调用工具
		log.Printf("🔧 [%s] 第 %d 轮: LLM 请求调用 %d 个工具", a.name, turn, len(assistantMsg.ToolCalls))

		// 3.4 依次执行每个工具调用
		for _, toolCall := range assistantMsg.ToolCalls {
			toolResult := a.executeToolCall(ctx, toolCall)

			// 把工具执行结果以 role=tool 的消息追加到历史
			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
				Content:    toolResult,
			})
		}

		// 进入下一轮循环, 让 LLM 看到工具结果后继续思考
	}

	// 4. 超过最大轮数仍未完成
	return "", fmt.Errorf("[%s] 超过最大轮数 %d 仍未给出最终回答", a.name, a.maxTurns)
}

// executeToolCall 执行一次工具调用, 返回结果字符串 (失败时也返回描述错误的字符串而不是 error)
// 为什么失败也返回字符串? 因为我们希望 LLM 看到"工具报错了"然后自己决定怎么办, 而不是直接中断整个 Agent
func (a *Agent) executeToolCall(ctx context.Context, tc llm.ToolCall) string {
	toolName := tc.Function.Name
	log.Printf("   ⚙️  [%s] 执行工具: %s(%s)", a.name, toolName, tc.Function.Arguments)

	// 1. 从注册表找工具
	tool, ok := a.registry.Get(toolName)
	if !ok {
		errMsg := fmt.Sprintf(`{"error": "工具 %s 不存在"}`, toolName)
		log.Printf("   ❌ %s", errMsg)
		return errMsg
	}

	// 2. 解析参数
	args, err := tc.ParseArguments()
	if err != nil {
		errMsg := fmt.Sprintf(`{"error": "参数解析失败: %s"}`, err.Error())
		log.Printf("   ❌ %s", errMsg)
		return errMsg
	}

	// 3. 执行工具
	result, err := tool.Execute(ctx, args)
	if err != nil {
		errMsg := fmt.Sprintf(`{"error": "工具执行失败: %s"}`, err.Error())
		log.Printf("   ❌ %s", errMsg)
		return errMsg
	}

	log.Printf("   ✅ 工具返回: %s", truncate(result, 200))
	return result
}

// truncate 截断字符串用于日志显示
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
