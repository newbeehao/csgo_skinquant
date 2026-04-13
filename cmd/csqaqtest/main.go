package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/newbeehao/csgo_skinquant/internal/config"
	"github.com/newbeehao/csgo_skinquant/internal/datasource"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	client := datasource.NewCSQAQClient(cfg.CSQAQToken, cfg.CSQAQBaseURL)
	ctx := context.Background()

	// 测试 1: 联想查询
	fmt.Println("🔍 测试联想查询: '红线'")
	suggestions, err := client.SuggestSkin(ctx, "红线")
	if err != nil {
		log.Fatalf("联想查询失败: %v", err)
	}
	pretty, _ := json.MarshalIndent(suggestions, "", "  ")
	fmt.Printf("返回 %d 个结果:\n%s\n\n", len(suggestions), string(pretty))

	if len(suggestions) == 0 {
		return
	}

	// 测试 2: 查详情 (用第一个结果的 ID)
	goodID := suggestions[0].GoodID
	fmt.Printf("📊 测试查详情: good_id=%s (%s)\n", goodID, suggestions[0].Value)
	detail, err := client.GetSkinDetail(ctx, goodID)
	if err != nil {
		log.Fatalf("查详情失败: %v", err)
	}
	pretty2, _ := json.MarshalIndent(detail, "", "  ")
	fmt.Printf("详情数据:\n%s\n", string(pretty2))
}
