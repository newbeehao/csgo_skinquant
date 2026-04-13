package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/newbeehao/csgo_skinquant/internal/agent"
	"github.com/newbeehao/csgo_skinquant/internal/config"
	"github.com/newbeehao/csgo_skinquant/internal/datasource"
	"github.com/newbeehao/csgo_skinquant/internal/llm"
	"github.com/newbeehao/csgo_skinquant/internal/orchestrator"
	"github.com/newbeehao/csgo_skinquant/internal/tools"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	llmClient := llm.NewClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel)
	csqaqClient := datasource.NewCSQAQClient(cfg.CSQAQToken, cfg.CSQAQBaseURL)
	tavily := datasource.NewTavilyClient(cfg.TavilyAPIKey, cfg.TavilyBaseURL)

	// === 共享工具 ===
	searchTool := tools.NewSearchSkinTool(csqaqClient)
	detailTool := tools.NewGetSkinDetailTool(csqaqClient)
	arbTool := tools.NewCalculateArbitrageTool(csqaqClient)
	riskTool := tools.NewRiskScoreTool(csqaqClient)
	webSearchTool := tools.NewWebSearchTool(tavily)
	fetchTool := tools.NewFetchWebpageTool(tavily)

	today := time.Now().Format("2006年01月02日")

	// === 市场数据分析师 Alex ===
	alexReg := tools.NewRegistry()
	alexReg.Register(searchTool)
	alexReg.Register(detailTool)
	alex := agent.New(agent.Config{
		Name: "Alex",
		SystemPrompt: `你是市场数据分析师 Alex。专注 CS2 饰品的三平台价格、涨跌幅、流动性数据。
先 search_skin_id 再 get_skin_detail, 基于真实数据给出简洁的估值分析 (价格单位 RMB)。`,
		LLMClient: llmClient, Registry: alexReg, MaxTurns: 8,
	})

	// === 套利猎手 Hunter ===
	hunterReg := tools.NewRegistry()
	hunterReg.Register(searchTool)
	hunterReg.Register(arbTool)
	hunter := agent.New(agent.Config{
		Name: "Hunter",
		SystemPrompt: `你是套利猎手 Hunter。工作流程: search_skin_id → calculate_arbitrage。
挑出 ROI 最高的路径, 扣除手续费后给出明确建议。提醒 Steam 余额不可提现。`,
		LLMClient: llmClient, Registry: hunterReg, MaxTurns: 8,
	})

	// === 情报研究员 Iris ===
	irisReg := tools.NewRegistry()
	irisReg.Register(webSearchTool)
	irisReg.Register(fetchTool)
	iris := agent.New(agent.Config{
		Name: "Iris",
		SystemPrompt: fmt.Sprintf(`你是情报研究员 Iris。当前日期: %s。
最多搜索 3 次, 关注版本更新/职业赛事/社区热点三维度。
每条情报标注来源和时间, 给出看涨/看跌判断。`, today),
		LLMClient: llmClient, Registry: irisReg, MaxTurns: 10,
	})

	// === 风控分析师 Risa ===
	risaReg := tools.NewRegistry()
	risaReg.Register(searchTool)
	risaReg.Register(riskTool)
	risa := agent.New(agent.Config{
		Name: "Risa",
		SystemPrompt: `你是风控分析师 Risa。先 search_skin_id → calculate_risk_score。
基于四维度评分给出风险评估。你可能会收到其他分析师的报告作为参考, 请综合他们的发现。
如果总分 > 60 必须明确警告。`,
		LLMClient: llmClient, Registry: risaReg, MaxTurns: 8,
	})

	// === 首席策略官 Chief (无工具, 纯综合) ===
	chiefReg := tools.NewRegistry() // 空注册表
	chief := agent.New(agent.Config{
		Name: "Chief",
		SystemPrompt: `你是 SkinQuant 首席策略官。你会收到团队四位分析师(Alex估值/Hunter套利/Iris情报/Risa风控)
对同一问题的独立报告, 请综合生成一份给投资者的最终研报。

结构要求:
## 投资研报: [饰品名]
### 核心结论 (一句话)
### 当前估值 (引用 Alex)
### 操作机会 (引用 Hunter, 如有)
### 市场情报 (引用 Iris)
### 风险提示 (引用 Risa)
### 最终建议: 买入/观望/卖出, 附理由

风格: 专业、简洁、数据驱动。绝不引入原始报告之外的信息。`,
		LLMClient: llmClient, Registry: chiefReg, MaxTurns: 3,
	})

	// === 组装编排器 ===
	orch := orchestrator.New(orchestrator.Config{
		MarketAnalyst: alex,
		ArbHunter:     hunter,
		IntelResearch: iris,
		RiskAnalyst:   risa,
		Chief:         chief,
		OnProgress: func(e orchestrator.ProgressEvent) {
			fmt.Printf("   📡 [%s] %s\n", e.Phase, e.Message)
		},
	})

	// === 执行 ===
	ctx := context.Background()
	question := "帮我深度分析 AK-47 红线(久经沙场), 现在值不值得入手?"

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("👤 用户: %s\n", question)
	fmt.Println("═══════════════════════════════════════════════════════")

	report, err := orch.Run(ctx, question)
	if err != nil {
		log.Fatalf("❌ 团队执行失败: %v", err)
	}

	fmt.Println("\n═══════════════════════════════════════════════════════")
	fmt.Println("📋 最终投资研报")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println(report.ChiefSummary)
	fmt.Println("\n═══════════════════════════════════════════════════════")
	fmt.Printf("⏱️  总耗时: %v | 子报告数: %d\n", report.TotalDuration.Round(time.Second), len(report.SubReports))
	for _, sr := range report.SubReports {
		status := "✅"
		if sr.Error != "" {
			status = "❌"
		}
		fmt.Printf("   %s %s: %v\n", status, sr.AgentName, sr.Duration.Round(time.Second))
	}
}
