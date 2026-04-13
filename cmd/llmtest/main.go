package main

import (
	"context"
	"fmt"
	"log"

	"github.com/newbeehao/csgo_skinquant/internal/config"
	"github.com/newbeehao/csgo_skinquant/internal/llm"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. 创建 LLM 客户端
	client := llm.NewClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel)

	// 3. 发一条简单消息
	messages := []llm.Message{
		{Role: "system", Content: "你是一个 CS2 饰品交易专家, 用简洁的中文回答问题。"},
		{Role: "user", Content: "AK-47 红线皮肤属于什么品质? 一句话回答。"},
	}

	fmt.Println("📤 发送请求中...")
	resp, err := client.Chat(context.Background(), messages, nil)
	if err != nil {
		log.Fatalf("❌ 请求失败: %v", err)
	}

	// 4. 打印结果
	fmt.Println("📥 模型回复:")
	fmt.Println(resp.Choices[0].Message.Content)
	fmt.Println()
	fmt.Printf("🔢 Token 消耗: prompt=%d, completion=%d, total=%d\n",
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		resp.Usage.TotalTokens)
	fmt.Printf("💰 (本次大约花费 %.6f 元)\n",
		float64(resp.Usage.TotalTokens)*0.000002, // 粗略估算, DeepSeek 现价很便宜
	)
}
