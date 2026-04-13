package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/newbeehao/csgo_skinquant/internal/agent"
	"github.com/newbeehao/csgo_skinquant/internal/api"
	"github.com/newbeehao/csgo_skinquant/internal/config"
	"github.com/newbeehao/csgo_skinquant/internal/datasource"
	"github.com/newbeehao/csgo_skinquant/internal/llm"
	"github.com/newbeehao/csgo_skinquant/internal/orchestrator"
	"github.com/newbeehao/csgo_skinquant/internal/tools"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}
	log.Printf("✅ 配置加载成功")

	// === 初始化客户端 ===
	llmClient := llm.NewClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel)
	csqaq := datasource.NewCSQAQClient(cfg.CSQAQToken, cfg.CSQAQBaseURL)

	var tavily *datasource.TavilyClient
	if cfg.TavilyAPIKey != "" {
		tavily = datasource.NewTavilyClient(cfg.TavilyAPIKey, cfg.TavilyBaseURL)
		log.Printf("✅ Tavily 已启用 (情报 Agent 可用)")
	} else {
		log.Printf("⚠️  未设置 TAVILY_API_KEY, 情报 Agent 将不可用")
	}

	// === 构造 Agent 团队 (和 teamtest 里一样) ===
	orch := buildOrchestrator(cfg, llmClient, csqaq, tavily)

	// === 启动 Gin 服务 ===
	handler := api.NewHandler(orch)

	r := gin.Default()
	r.GET("/health", handler.Health)
	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/analyze", handler.Analyze)
		apiGroup.POST("/analyze/stream", handler.AnalyzeStream)
	}
	// 静态文件 (阶段 10 前端用)
	r.Static("/web", "./web")

	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("🚀 SkinQuant 启动, 监听 %s", addr)
	log.Printf("   POST %s/api/analyze         — 非流式")
	log.Printf("   POST %s/api/analyze/stream  — SSE 流式")
	if err := r.Run(addr); err != nil {
		log.Fatalf("❌ 服务启动失败: %v", err)
	}
}

// buildOrchestrator 组装完整的 5-Agent 团队
// 这段代码本质和 cmd/teamtest/main.go 相同, 抽出来复用
func buildOrchestrator(
	cfg *config.Config,
	llmClient *llm.Client,
	csqaq *datasource.CSQAQClient,
	tavily *datasource.TavilyClient,
) *orchestrator.Orchestrator {
	// 共享工具
	searchTool := tools.NewSearchSkinTool(csqaq)
	detailTool := tools.NewGetSkinDetailTool(csqaq)
	arbTool := tools.NewCalculateArbitrageTool(csqaq)
	riskTool := tools.NewRiskScoreTool(csqaq)

	today := time.Now().Format("2006年01月02日")

	// Alex
	alexReg := tools.NewRegistry()
	alexReg.Register(searchTool)
	alexReg.Register(detailTool)
	alex := agent.New(agent.Config{
		Name:         "Alex",
		SystemPrompt: `你是市场数据分析师 Alex。先 search_skin_id 再 get_skin_detail, 基于真实三平台数据给出简洁估值分析 (价格单位 RMB)。`,
		LLMClient:    llmClient, Registry: alexReg, MaxTurns: 8,
	})

	// Hunter
	hunterReg := tools.NewRegistry()
	hunterReg.Register(searchTool)
	hunterReg.Register(arbTool)
	hunter := agent.New(agent.Config{
		Name:         "Hunter",
		SystemPrompt: `你是套利猎手 Hunter。search_skin_id → calculate_arbitrage, 挑 ROI 最高的路径, 提醒 Steam 余额不可提现。`,
		LLMClient:    llmClient, Registry: hunterReg, MaxTurns: 8,
	})

	// Iris (没有 tavily 就给它一个空工具集, 它会直接基于常识回答)
	irisReg := tools.NewRegistry()
	if tavily != nil {
		irisReg.Register(tools.NewWebSearchTool(tavily))
		irisReg.Register(tools.NewFetchWebpageTool(tavily))
	}
	iris := agent.New(agent.Config{
		Name: "Iris",
		SystemPrompt: fmt.Sprintf(`你是情报研究员 Iris。当前日期: %s。
最多搜索 3 次, 关注版本更新/职业赛事/社区热点。标注来源和时间, 给出看涨/看跌判断。`, today),
		LLMClient: llmClient, Registry: irisReg, MaxTurns: 10,
	})

	// Risa
	risaReg := tools.NewRegistry()
	risaReg.Register(searchTool)
	risaReg.Register(riskTool)
	risa := agent.New(agent.Config{
		Name:         "Risa",
		SystemPrompt: `你是风控分析师 Risa。search_skin_id → calculate_risk_score。综合四维度评分+上游报告给出评估。总分>60 必须警告。`,
		LLMClient:    llmClient, Registry: risaReg, MaxTurns: 8,
	})

	// Chief
	chief := agent.New(agent.Config{
		Name: "Chief",
		SystemPrompt: `你是 SkinQuant 首席策略官。综合团队四位分析师的报告生成最终研报。
结构: ## 投资研报 / 核心结论 / 当前估值(Alex) / 操作机会(Hunter) / 市场情报(Iris) / 风险提示(Risa) / 最终建议。
专业简洁, 数据驱动, 不引入原始报告之外的信息。`,
		LLMClient: llmClient, Registry: tools.NewRegistry(), MaxTurns: 3,
	})

	return orchestrator.New(orchestrator.Config{
		MarketAnalyst: alex,
		ArbHunter:     hunter,
		IntelResearch: iris,
		RiskAnalyst:   risa,
		Chief:         chief,
	})
}
