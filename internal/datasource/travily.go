package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TavilyClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewTavilyClient(apiKey, baseURL string) *TavilyClient {
	return &TavilyClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// SearchResult 是单条搜索结果
type SearchResult struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`        // 摘要
	Score         float64 `json:"score"`          // 相关性分数 0-1
	PublishedDate string  `json:"published_date"` // 可能为空
}

// SearchResponse 是搜索接口的返回
type SearchResponse struct {
	Query   string         `json:"query"`
	Answer  string         `json:"answer"` // Tavily 自动生成的综合答案
	Results []SearchResult `json:"results"`
}

// Search 调用 Tavily /search 接口
// query: 搜索词
// maxResults: 返回多少条结果 (建议 5-10)
// days: 限定最近 N 天内的结果 (0 表示不限)
func (c *TavilyClient) Search(ctx context.Context, query string, maxResults, days int) (*SearchResponse, error) {
	reqBody := map[string]any{
		"api_key":        c.apiKey,
		"query":          query,
		"max_results":    maxResults,
		"include_answer": true,       // 让 Tavily 自动生成摘要答案
		"search_depth":   "advanced", // advanced 比 basic 质量高很多, 但多耗 1 次配额
	}
	if days > 0 {
		reqBody["days"] = days
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/search", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tavily 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tavily API 异常 (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var result SearchResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return &result, nil
}

// ExtractResult 是单条 URL 提取结果
type ExtractResult struct {
	URL        string `json:"url"`
	RawContent string `json:"raw_content"` // 页面正文
}

// Extract 调用 Tavily /extract 接口, 抓取指定 URL 的正文
func (c *TavilyClient) Extract(ctx context.Context, url string) (*ExtractResult, error) {
	reqBody := map[string]any{
		"api_key": c.apiKey,
		"urls":    []string{url}, // 接口支持批量, 我们一次传一个
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/extract", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tavily Extract 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tavily Extract 异常 (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var result struct {
		Results []ExtractResult `json:"results"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, err
	}
	if len(result.Results) == 0 {
		return nil, fmt.Errorf("Tavily 未返回提取结果")
	}
	return &result.Results[0], nil
}
