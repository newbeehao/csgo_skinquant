package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/newbeehao/csgo_skinquant/internal/datasource"
	"github.com/newbeehao/csgo_skinquant/internal/llm"
)

// ========== 工具 1: 搜索饰品 ID ==========

type SearchSkinTool struct {
	client *datasource.CSQAQClient
}

func NewSearchSkinTool(client *datasource.CSQAQClient) *SearchSkinTool {
	return &SearchSkinTool{client: client}
}

func (t *SearchSkinTool) Name() string { return "search_skin_id" }

func (t *SearchSkinTool) Definition() llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name: "search_skin_id",
			Description: "根据饰品名称关键词搜索饰品 ID。CS2 饰品数据库中每个饰品都有唯一 ID, 后续所有详情查询都需要先通过此工具拿到 ID。" +
				"例如搜索'红线'会返回 AK-47 红线各磨损版本的 ID 列表。" +
				"注意: 用户可能只说'红线'这种简称, 此工具会返回多个匹配项, 你需要根据上下文判断用户想要哪个(不同磨损/StatTrak 等)。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"keyword": map[string]any{
						"type":        "string",
						"description": "搜索关键词, 支持中文, 例如 '红线' / 'AK红线' / '龙狙' / '多普勒'",
					},
				},
				"required": []string{"keyword"},
			},
		},
	}
}

func (t *SearchSkinTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	keyword, ok := args["keyword"].(string)
	if !ok {
		return "", fmt.Errorf("参数 keyword 缺失")
	}

	suggestions, err := t.client.SuggestSkin(ctx, keyword)
	if err != nil {
		return "", err
	}

	// 只返回前 10 条, 避免 LLM 被太多选项淹没
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}

	result := map[string]any{
		"count":   len(suggestions),
		"results": suggestions,
		"hint":    "如需查询详情, 使用 get_skin_detail 工具并传入对应的 id",
	}
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// ========== 工具 2: 查询饰品详情 ==========

type GetSkinDetailTool struct {
	client *datasource.CSQAQClient
}

func NewGetSkinDetailTool(client *datasource.CSQAQClient) *GetSkinDetailTool {
	return &GetSkinDetailTool{client: client}
}

func (t *GetSkinDetailTool) Name() string { return "get_skin_detail" }

func (t *GetSkinDetailTool) Definition() llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name: "get_skin_detail",
			Description: "根据饰品 ID 查询完整市场数据, 包括 Steam/BUFF163/悠悠有品 三大平台的实时价格、求购价、成交量、" +
				"多周期涨跌幅(1/7/30/90/180/365天)、存世量、热度排名等。所有价格单位为人民币(RMB)。" +
				"必须先用 search_skin_id 获取 ID 后才能调用此工具。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"skin_id": map[string]any{
						"type":        "string",
						"description": "饰品 ID, 通过 search_skin_id 工具获取",
					},
				},
				"required": []string{"skin_id"},
			},
		},
	}
}

func (t *GetSkinDetailTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	skinID, ok := args["skin_id"].(string)
	if !ok {
		return "", fmt.Errorf("参数 skin_id 缺失")
	}

	detail, err := t.client.GetSkinDetail(ctx, skinID)
	if err != nil {
		return "", err
	}

	resultBytes, _ := json.Marshal(detail)
	return string(resultBytes), nil
}
