package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 是 DeepSeek API 的客户端封装
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient 创建一个新的 DeepSeek 客户端
func NewClient(apiKey, baseURL, model string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // LLM 响应可能较慢, 给 2 分钟
		},
	}
}

// Chat 发送一次对话请求, 返回模型响应
// messages: 完整的对话历史
// tools: 可用工具列表 (可为 nil, 表示不启用 Function Calling)
func (c *Client) Chat(ctx context.Context, messages []Message, tools []Tool) (*ChatResponse, error) {
	// 1. 构造请求体
	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       tools,
		Temperature: 0.7, // 对 Agent 场景, 稍微降低随机性比默认 1.0 更稳
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 2. 构造 HTTP 请求
	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 3. 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 4. 读取响应体
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 5. 处理错误响应 (非 200 状态码)
	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(respBytes, &apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("DeepSeek API 错误 (HTTP %d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		// 如果连错误结构都解析不出来, 把原始响应返回
		return nil, fmt.Errorf("DeepSeek API 异常 (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	// 6. 解析成功响应
	var chatResp ChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w (原始: %s)", err, string(respBytes))
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("DeepSeek 返回空 choices")
	}

	return &chatResp, nil
}
