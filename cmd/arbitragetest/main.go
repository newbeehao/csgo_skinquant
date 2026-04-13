package main

import (
	"context"
	"fmt"
	"log"

	"github.com/newbeehao/csgo_skinquant/internal/agent"
	"github.com/newbeehao/csgo_skinquant/internal/config"
	"github.com/newbeehao/csgo_skinquant/internal/datasource"
	"github.com/newbeehao/csgo_skinquant/internal/llm"
	"github.com/newbeehao/csgo_skinquant/internal/tools"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	llmClient := llm.NewClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel)
	csqaqClient := datasource.NewCSQAQClient(cfg.CSQAQToken, cfg.CSQAQBaseURL)

	// 套利猎手的工具集: 搜索 + 套利计算
	// 注意: 它不需要 get_skin_detail, 因为 calculate_arbitrage 内部会查
	registry := tools.NewRegistry()
	registry.Register(tools.NewSearchSkinTool(csqaqClient))
	registry.Register(tools.NewCalculateArbitrageTool(csqaqClient))

	hunter := agent.New(agent.Config{
		Name: "套利猎手",
		SystemPrompt: `你是 SkinQuant 的套利猎手 Hunter, 专门挖掘 CS2 饰品跨平台挂刀(套利)机会。

工作流程:
1. 用户提饰品 → 先用 search_skin_id 拿到 ID
2. 用 calculate_arbitrage 计算所有套利路径的真实净利润(已扣手续费)
3. 从 opportunities 列表中挑出 ROI 最高的 1-2 条, 给用户明确的操作建议

输出要求:
- 简洁直接, 用表格或清单呈现最佳机会
- 必须提醒手续费已扣除、Steam 余额不可提现等风险
- 给出明确的"买入价上限"和"卖出价下限"建议
- ROI < 5% 的机会明确标注"利润微薄, 不建议操作"

绝不凭感觉, 一切以 calculate_arbitrage 返回的数据为准。`,
		LLMClient: llmClient,
		Registry:  registry,
	})

	ctx := context.Background()
	question := "帮我看看 AK-47 红线(久经沙场)现在有没有挂刀机会?"

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("👤 用户: %s\n", question)
	fmt.Println("═══════════════════════════════════════")

	answer, err := hunter.Run(ctx, question)
	if err != nil {
		log.Fatalf("❌ Agent 执行失败: %v", err)
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("🤖 Hunter:\n%s\n", answer)
	fmt.Println("═══════════════════════════════════════")
}
