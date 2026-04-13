package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/newbeehao/csgo_skinquant/internal/datasource"
	"github.com/newbeehao/csgo_skinquant/internal/llm"
)

// ========== 工具 1: 网络搜索 ==========

type WebSearchTool struct {
	client *datasource.TavilyClient
}

func NewWebSearchTool(client *datasource.TavilyClient) *WebSearchTool {
	return &WebSearchTool{client: client}
}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Definition() llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name: "web_search",
			Description: "在互联网上搜索信息。专用于查询三类情报: " +
				"(1) CS2/CSGO 版本更新 (Valve 官方公告、箱子变动、饰品停产), " +
				"(2) 职业赛事热度 (Major/大赛选手同款饰品、冠军印花), " +
				"(3) 社区热点 (Reddit/贴吧讨论、主播带货、价格预测)。" +
				"如果想看某条结果的完整内容, 再用 fetch_webpage 工具深读。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "搜索关键词, 建议中英文混用以覆盖国内外信源, 例如 'CS2 新箱子 2026' 或 'AK Redline price trend reddit'",
					},
					"recent_days": map[string]any{
						"type":        "integer",
						"description": "限定最近 N 天内的结果, 0 表示不限。查时效性新闻建议 7-30",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, ok := args["query"].(string)
	if !ok {
		return "", fmt.Errorf("参数 query 缺失")
	}
	days := 0
	if v, ok := args["recent_days"].(float64); ok { // JSON 数字解析为 float64
		days = int(v)
	}

	resp, err := t.client.Search(ctx, query, 8, days)
	if err != nil {
		return "", err
	}

	// 精简返回: 只给 LLM 它真正需要的字段, 节省 token
	simplified := map[string]any{
		"summary_answer": resp.Answer,
		"results":        resp.Results,
	}
	resultBytes, _ := json.Marshal(simplified)
	return string(resultBytes), nil
}

// ========== 工具 2: 抓取网页正文 ==========

type FetchWebpageTool struct {
	client *datasource.TavilyClient
}

func NewFetchWebpageTool(client *datasource.TavilyClient) *FetchWebpageTool {
	return &FetchWebpageTool{client: client}
}

func (t *FetchWebpageTool) Name() string { return "fetch_webpage" }

func (t *FetchWebpageTool) Definition() llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name: "fetch_webpage",
			Description: "抓取指定 URL 的完整正文内容, 用于深读 web_search 返回的某条重要文章。" +
				"当搜索摘要信息不够, 需要看完整报道/贴子/公告时使用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "要抓取的完整 URL, 必须从 web_search 的结果里选一条",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

func (t *FetchWebpageTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	url, ok := args["url"].(string)
	if !ok {
		return "", fmt.Errorf("参数 url 缺失")
	}

	result, err := t.client.Extract(ctx, url)
	if err != nil {
		// 关键改动: 抓取失败不返回 error, 而是返回给 LLM 一条可读提示
		// 这样 Agent 可以自己决定"换一个 URL 试试"而不是崩溃
		friendly := map[string]any{
			"url":     url,
			"success": false,
			"hint":    "此 URL 无法抓取 (可能是反爬、需登录、或页面不支持提取)。请尝试 web_search 结果中的其他 URL, 或直接基于已有搜索摘要总结。",
		}
		b, _ := json.Marshal(friendly)
		return string(b), nil
	}

	content := result.RawContent
	truncated := false
	if len(content) > 5000 {
		content = content[:5000]
		truncated = true
	}

	output := map[string]any{
		"url":       result.URL,
		"success":   true,
		"content":   content,
		"truncated": truncated,
	}
	outputBytes, _ := json.Marshal(output)
	return string(outputBytes), nil
}
