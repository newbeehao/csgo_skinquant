package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// CSQAQClient 是 CSQAQ API 的客户端
type CSQAQClient struct {
	token      string
	baseURL    string
	httpClient *http.Client

	// 限流: CSQAQ 限制 1 次/秒, 我们用 mutex + 上次请求时间实现简单限流
	rateMu      sync.Mutex
	lastReqTime time.Time
	minInterval time.Duration
}

func NewCSQAQClient(token, baseURL string) *CSQAQClient {
	return &CSQAQClient{
		token:       token,
		baseURL:     baseURL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		minInterval: 1100 * time.Millisecond, // 稍大于 1 秒, 防止踩线
	}
}

// rateLimit 确保两次请求间隔至少 minInterval
// 这是"阻塞式限流"——调用者会被 sleep 到允许发请求为止
func (c *CSQAQClient) rateLimit() {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()

	elapsed := time.Since(c.lastReqTime)
	if elapsed < c.minInterval {
		time.Sleep(c.minInterval - elapsed)
	}
	c.lastReqTime = time.Now()
}

// doRequest 是所有 CSQAQ 请求的统一出口: 加鉴权头、限流、错误处理
func (c *CSQAQClient) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	c.rateLimit() // 先限流

	// 1. 构造请求体
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 2. 构造 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("ApiToken", c.token)
	req.Header.Set("Content-Type", "application/json")

	// 3. 发送并读取响应
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CSQAQ API 异常 (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}

// ===================== 业务方法 =====================

// SuggestSkin 根据关键词联想查询饰品 (返回最匹配的前几个)
// 对应 CSQAQ 的"联想查询饰品的ID信息"接口
func (c *CSQAQClient) SuggestSkin(ctx context.Context, keyword string) ([]SkinSuggestion, error) {
	path := "/api/v1/search/suggest?text=" + urlEncode(keyword)
	respBytes, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code int              `json:"code"`
		Msg  string           `json:"msg"`
		Data []SkinSuggestion `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w (原始: %s)", err, string(respBytes))
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("CSQAQ 业务错误: %s", result.Msg)
	}
	return result.Data, nil
}

func (c *CSQAQClient) GetSkinDetail(ctx context.Context, goodID string) (*GoodsInfo, error) {
	path := "/api/v1/info/good?id=" + goodID
	respBytes, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code int                `json:"code"`
		Msg  string             `json:"msg"`
		Data SkinDetailResponse `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w (原始: %s)", err, string(respBytes))
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("CSQAQ 业务错误: %s", result.Msg)
	}
	return &result.Data.GoodsInfo, nil
}

// SkinSuggestion 是联想查询返回的饰品简要信息
// SkinSuggestion 是联想查询返回的饰品简要信息
// 注意: CSQAQ 的 id 字段是字符串类型, value 是饰品中文名
type SkinSuggestion struct {
	GoodID string `json:"id"`    // 饰品 ID, 字符串类型
	Value  string `json:"value"` // 饰品完整中文名, 例如 "穿肠刀（★） | 多普勒 (崭新出厂)"
}

// SkinDetail 是饰品详情 (只保留我们 Agent 会用到的核心字段)
// CSQAQ 返回字段非常多, 这里我们按需精简; 等后面发现需要更多字段再加
type SkinDetail = map[string]any

// type SkinDetail struct {
// 	GoodID         int    `json:"id"`
// 	Name           string `json:"name"`
// 	MarketHashName string `json:"market_hash_name"`
// 	// Steam 市场数据
// 	SteamSellPrice float64 `json:"steam_sell_price"`
// 	SteamBuyPrice  float64 `json:"steam_buy_price"`
// 	SteamVolume    int     `json:"steam_volume"`
// 	// BUFF 市场数据
// 	BuffSellPrice float64 `json:"buff_sell_price"`
// 	BuffBuyPrice  float64 `json:"buff_buy_price"`
// 	BuffSellNum   int     `json:"buff_sell_num"`
// 	// 悠悠有品
// 	YoupinSellPrice float64 `json:"youpin_sell_price"`
// 	YoupinSellNum   int     `json:"youpin_sell_num"`
// }

// 把文件顶部的 import 加上 "net/url"
// 然后文件末尾简化为:
func urlEncode(s string) string {
	return url.QueryEscape(s)
}

// func urlEncode(s string) string {
// 	// 最小依赖: 用标准库 net/url 的 QueryEscape
// 	return (&url{}).escape(s)
// }

// // 小技巧: 避免再引入一个 import, 用本地包装一下
// type url struct{}

// func (url) escape(s string) string {
// 	return (&netURL{}).do(s)
// }

// type netURL struct{}

// func (netURL) do(s string) string {
// 	// 为简化起见, 直接复用 net/url
// 	return netURLQueryEscape(s)
// }

// SkinDetailResponse 是 GetSkinDetail 接口的完整响应
// 我们主要关心 GoodsInfo 里的数据
type SkinDetailResponse struct {
	GoodsInfo GoodsInfo `json:"goods_info"`
}

// GoodsInfo 包含了饰品的核心市场数据 (按需精选字段)
// 所有价格单位均为人民币 (RMB)
type GoodsInfo struct {
	// 基本信息
	ID                    int    `json:"id"`
	Name                  string `json:"name"`                    // 中文名, 例如 "AK-47 | 红线 (战痕累累)"
	MarketHashName        string `json:"market_hash_name"`        // Steam 英文名
	ExteriorLocalizedName string `json:"exterior_localized_name"` // 磨损, 例如 "战痕累累"
	RarityLocalizedName   string `json:"rarity_localized_name"`   // 稀有度, 例如 "保密"
	QualityLocalizedName  string `json:"quality_localized_name"`  // 品质, 例如 "普通" / "StatTrak™"
	TypeLocalizedName     string `json:"type_localized_name"`     // 类型, 例如 "步枪"

	// Steam 市场 (价格单位: RMB)
	SteamSellPrice float64 `json:"steam_sell_price"`
	SteamBuyPrice  float64 `json:"steam_buy_price"`
	SteamSellNum   int     `json:"steam_sell_num"`
	SteamBuyNum    int     `json:"steam_buy_num"`

	// BUFF163 市场
	BuffSellPrice float64 `json:"buff_sell_price"`
	BuffBuyPrice  float64 `json:"buff_buy_price"`
	BuffSellNum   int     `json:"buff_sell_num"`
	BuffBuyNum    int     `json:"buff_buy_num"`

	// 悠悠有品市场
	YyypSellPrice float64 `json:"yyyp_sell_price"`
	YyypBuyPrice  float64 `json:"yyyp_buy_price"`
	YyypSellNum   int     `json:"yyyp_sell_num"`
	YyypBuyNum    int     `json:"yyyp_buy_num"`

	// 跨平台套利参考 (BUFF 价 / Steam 价 的比率)
	BuffSteamSellConversion float64 `json:"buff_steam_sell_conversion"`
	SteamBuffSellConversion float64 `json:"steam_buff_sell_conversion"`

	// 多周期涨跌幅 (百分比)
	SellPriceRate1   float64 `json:"sell_price_rate_1"`   // 1 天
	SellPriceRate7   float64 `json:"sell_price_rate_7"`   // 7 天
	SellPriceRate30  float64 `json:"sell_price_rate_30"`  // 30 天
	SellPriceRate90  float64 `json:"sell_price_rate_90"`  // 90 天
	SellPriceRate180 float64 `json:"sell_price_rate_180"` // 180 天
	SellPriceRate365 float64 `json:"sell_price_rate_365"` // 365 天

	// 流动性指标
	Statistic        int     `json:"statistic"`          // 存世量
	TurnoverNumber   int     `json:"turnover_number"`    // 近期成交数
	TurnoverAvgPrice float64 `json:"turnover_avg_price"` // 成交均价
	RankNum          int     `json:"rank_num"`           // 热度排名
	RankNumChange    int     `json:"rank_num_change"`    // 排名变化

	// 时间戳
	UpdatedAt string `json:"updated_at"`
}
