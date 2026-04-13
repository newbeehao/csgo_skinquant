package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// StreamChunk 是流式响应中的一个增量片段
type StreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
			// 注意: 流式工具调用的 delta 格式更复杂, 我们暂不支持
			// 只流式处理纯文本内容; 工具调用仍走非流式 Chat 方法
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// ChatStream 以流式方式调用 DeepSeek
// 每收到一个内容 token, 就调用 onDelta(token)
// 返回最终拼接好的完整文本 (方便追加到 messages 历史)
//
// 重要限制: 此方法只适合"纯文本输出"的场景 (例如 Chief 综合研报)
// 涉及工具调用的对话仍应使用非流式 Chat 方法, 因为流式工具调用的协议更复杂
func (c *Client) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []Tool,
	onDelta func(token string),
) (string, error) {
	// 1. 构造请求 (加上 stream: true)
	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       tools,
		Temperature: 0.7,
		Stream:      true,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 2. 构造 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送流式请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("DeepSeek API 异常 (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	// 3. 按行读取 SSE 流
	// DeepSeek 的 SSE 格式: 每条消息以 "data: {json}\n\n" 形式出现
	// 流结束时会发 "data: [DONE]\n\n"
	scanner := bufio.NewScanner(resp.Body)
	// 默认 buffer 64KB, 单条 chunk 可能更大, 扩容
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var fullContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		// 跳过空行和非 data 行
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		if data == "" {
			continue
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 单条解析失败不致命, 继续
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		token := chunk.Choices[0].Delta.Content
		if token == "" {
			continue
		}

		fullContent.WriteString(token)
		if onDelta != nil {
			onDelta(token)
		}
	}

	if err := scanner.Err(); err != nil {
		return fullContent.String(), fmt.Errorf("读取流失败: %w", err)
	}

	return fullContent.String(), nil
}
