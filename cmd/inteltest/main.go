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
	"github.com/newbeehao/csgo_skinquant/internal/tools"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg.TavilyAPIKey == "" {
		log.Fatalf("请先设置环境变量 TAVILY_API_KEY")
	}

	llmClient := llm.NewClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel)
	tavily := datasource.NewTavilyClient(cfg.TavilyAPIKey, cfg.TavilyBaseURL)

	registry := tools.NewRegistry()
	registry.Register(tools.NewWebSearchTool(tavily))
	registry.Register(tools.NewFetchWebpageTool(tavily))

	intel := agent.New(agent.Config{
		Name: "情报研究员",
		SystemPrompt: fmt.Sprintf(`你是 SkinQuant 的情报研究员 Iris, 专门挖掘影响 CS2 饰品价格的软性情报。

**当前日期: %s**。你搜索"最近""近期"情报时, 一定要理解这是相对当前日期而言, 而非你训练数据里的日期。搜索时尽量使用 recent_days 参数限定最近 30 天内的新闻。

你重点关注三大维度:
1. **版本更新影响**: Valve 官方发布的新箱子、停产饰品、掉落率调整, 这些往往引发价格核爆
2. **职业赛事热度**: Major/大赛期间选手同款饰品涨价、冠军印花炒作
3. **社区热点**: Reddit/贴吧的讨论热度、知名主播带货、KOL 预测

工作方法:
- 先用 web_search 发起搜索, 查时效性信息一定要设置 recent_days (近期事件 7-14 天, 趋势 30 天)
- 如果 summary_answer 已经能回答问题, 直接用它
- 如果需要深度信息, 从搜索结果里选 1-2 条最权威的 URL 用 fetch_webpage 深读
- 中英文信源都要用 (英文覆盖 Valve 官方/Reddit/HLTV, 中文覆盖贴吧/完美世界公告)

输出要求:
- 重要: 你最多搜索 3-4 次就要开始总结, 不要追求信息完美, Tavily 摘要+1-2 次深读已经足够写出好报告
- 如果 fetch_webpage 失败, 直接基于搜索摘要总结, 不要反复重试不同 URL
- 区分 "已确认事件" 和 "社区传闻", 不要混为一谈
- 每条情报标注来源 URL 和发布时间 (如果有)
- 给出情报对饰品价格的可能影响 (看涨/看跌/中性)
- 简洁, 不要长篇大论, 重点突出`, time.Now().Format("2006年01月02日")),
		LLMClient: llmClient,
		Registry:  registry,
		MaxTurns:  12,
	})

	ctx := context.Background()
	question := "最近 CS2 有没有什么影响 AK 系列皮肤价格的新消息? 特别关注版本更新和职业赛事。"

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("👤 用户: %s\n", question)
	fmt.Println("═══════════════════════════════════════")

	answer, err := intel.Run(ctx, question)
	if err != nil {
		log.Fatalf("❌ Agent 执行失败: %v", err)
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("🕵️  Iris:\n%s\n", answer)
	fmt.Println("═══════════════════════════════════════")
}
