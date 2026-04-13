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

	registry := tools.NewRegistry()
	registry.Register(tools.NewSearchSkinTool(csqaqClient))
	registry.Register(tools.NewRiskScoreTool(csqaqClient))

	risa := agent.New(agent.Config{
		Name: "风控分析师",
		SystemPrompt: `你是 SkinQuant 的风控分析师 Risa, 职责是冷静地泼冷水, 提醒投资风险。

工作流程:
1. 用户提饰品 → search_skin_id 拿 ID → calculate_risk_score 拿评分
2. 基于返回的四维度得分和解释, 给出结构化的风险评估报告

输出要求:
- 开头明确给出总分和风险等级
- 逐项解读四个维度 (流动性/波动性/长期趋势/集中度), 用工具返回的数据说话
- 结尾给出明确的"投资建议": 可持有/观望/不推荐
- 语气冷静客观, 不要过度乐观, 风控分析师的职责就是提醒风险
- 如果总分 > 60 分, 必须明确警告用户谨慎操作`,
		LLMClient: llmClient,
		Registry:  registry,
	})

	ctx := context.Background()
	question := "帮我评估一下 AK-47 红线(久经沙场)作为投资标的的风险。"

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("👤 用户: %s\n", question)
	fmt.Println("═══════════════════════════════════════")

	answer, err := risa.Run(ctx, question)
	if err != nil {
		log.Fatalf("❌ Agent 执行失败: %v", err)
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("⚠️  Risa:\n%s\n", answer)
	fmt.Println("═══════════════════════════════════════")
}
