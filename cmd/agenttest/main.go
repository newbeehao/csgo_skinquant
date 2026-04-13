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

	// 创建 LLM 客户端
	llmClient := llm.NewClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel)

	// 创建 CSQAQ 客户端 (数据源)
	csqaqClient := datasource.NewCSQAQClient(cfg.CSQAQToken, cfg.CSQAQBaseURL)

	// 注册真实工具
	registry := tools.NewRegistry()
	registry.Register(tools.NewSearchSkinTool(csqaqClient))
	registry.Register(tools.NewGetSkinDetailTool(csqaqClient))

	myAgent := agent.New(agent.Config{
		Name: "市场数据分析师",
		SystemPrompt: `你是 SkinQuant 的 CS2 饰品市场数据分析师 Alex。
你通过工具查询真实的三大平台(Steam/BUFF163/悠悠有品)市场数据, 并给出专业分析。

工作流程:
1. 用户提到饰品时, 先用 search_skin_id 搜索, 如果返回多个结果(不同磨损), 挑最合理的一个
2. 拿到 ID 后用 get_skin_detail 查完整数据
3. 基于真实数据给出简洁专业的分析, 所有价格单位是人民币(RMB)

分析重点: 三大平台价差(套利空间)、多周期涨跌趋势、流动性(成交量/存世量)。
绝对不要凭记忆回答价格, 必须查询真实数据。`,
		LLMClient: llmClient,
		Registry:  registry,
		MaxTurns:  8,
	})

	ctx := context.Background()
	question := "帮我分析一下 AK-47 红线(久经沙场)这个饰品, 现在值不值得入手?"

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("👤 用户: %s\n", question)
	fmt.Println("═══════════════════════════════════════")

	answer, err := myAgent.Run(ctx, question)
	if err != nil {
		log.Fatalf("❌ Agent 执行失败: %v", err)
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("🤖 Alex:\n%s\n", answer)
	fmt.Println("═══════════════════════════════════════")
}
