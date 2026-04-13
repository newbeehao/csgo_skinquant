package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/newbeehao/csgo_skinquant/internal/datasource"
	"github.com/newbeehao/csgo_skinquant/internal/llm"
)

// 平台手续费常量 (2026 年实际数据, 随时可调)
const (
	SteamSellerFeeRate = 0.1304 // Steam 卖家实际到手比例: 1 - 0.8696
	BuffSellerFeeRate  = 0.025  // BUFF 卖家手续费 2.5%
	YyypSellerFeeRate  = 0.015  // 悠悠有品卖家手续费 1.5%
)

// CalculateArbitrageTool 计算跨平台套利的真实净利润
// LLM 会先用 get_skin_detail 拿价格, 然后调用这个工具做精确计算
type CalculateArbitrageTool struct {
	client *datasource.CSQAQClient
}

func NewCalculateArbitrageTool(client *datasource.CSQAQClient) *CalculateArbitrageTool {
	return &CalculateArbitrageTool{client: client}
}

func (t *CalculateArbitrageTool) Name() string { return "calculate_arbitrage" }

func (t *CalculateArbitrageTool) Definition() llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name: "calculate_arbitrage",
			Description: "计算某饰品跨平台套利的真实净利润, 自动扣除各平台手续费。" +
				"输入饰品 ID, 返回所有可能的套利路径及其净收益率, 例如 'BUFF买入→Steam卖出' 的净利润。" +
				"这是做挂刀决策的核心工具, 比直接对比价格准确得多(因为考虑了手续费)。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"skin_id": map[string]any{
						"type":        "string",
						"description": "饰品 ID, 通过 search_skin_id 获取",
					},
				},
				"required": []string{"skin_id"},
			},
		},
	}
}

// ArbitrageOpportunity 描述一条套利机会
type ArbitrageOpportunity struct {
	Route          string  `json:"route"`           // 路径描述, 例如 "BUFF买入 → Steam卖出"
	BuyPlatform    string  `json:"buy_platform"`    // 买入平台
	BuyPrice       float64 `json:"buy_price"`       // 买入价格
	SellPlatform   string  `json:"sell_platform"`   // 卖出平台
	SellPrice      float64 `json:"sell_price"`      // 挂单价
	SellerReceives float64 `json:"seller_receives"` // 卖家实际到手
	NetProfit      float64 `json:"net_profit"`      // 净利润 (人民币)
	ROI            float64 `json:"roi_percent"`     // 净收益率 (%)
	Note           string  `json:"note,omitempty"`  // 风险提示
}

func (t *CalculateArbitrageTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	skinID, ok := args["skin_id"].(string)
	if !ok {
		return "", fmt.Errorf("参数 skin_id 缺失")
	}

	info, err := t.client.GetSkinDetail(ctx, skinID)
	if err != nil {
		return "", err
	}

	// 计算所有可能的套利路径 (3 个买入平台 × 3 个卖出平台 = 9 种组合, 排除同平台)
	opportunities := []ArbitrageOpportunity{}

	// 平台列表 (名称, 买入价即挂单价, 卖家手续费率)
	platforms := []struct {
		name      string
		sellPrice float64 // 该平台当前最低挂单价 (我们从这买)
		feeRate   float64 // 该平台卖家手续费 (如果从这卖)
	}{
		{"Steam", info.SteamSellPrice, SteamSellerFeeRate},
		{"BUFF163", info.BuffSellPrice, BuffSellerFeeRate},
		{"悠悠有品", info.YyypSellPrice, YyypSellerFeeRate},
	}

	for i, buy := range platforms {
		if buy.sellPrice <= 0 {
			continue // 无售价数据跳过
		}
		for j, sell := range platforms {
			if i == j {
				continue // 同平台不构成套利
			}
			if sell.sellPrice <= 0 {
				continue
			}
			// 卖家实际到手 = 挂单价 × (1 - 手续费率)
			sellerReceives := sell.sellPrice * (1 - sell.feeRate)
			netProfit := sellerReceives - buy.sellPrice
			roi := (netProfit / buy.sellPrice) * 100

			opp := ArbitrageOpportunity{
				Route:          fmt.Sprintf("%s买入 → %s卖出", buy.name, sell.name),
				BuyPlatform:    buy.name,
				BuyPrice:       buy.sellPrice,
				SellPlatform:   sell.name,
				SellPrice:      sell.sellPrice,
				SellerReceives: round2(sellerReceives),
				NetProfit:      round2(netProfit),
				ROI:            round2(roi),
			}
			// Steam 卖出的特殊提示
			if sell.name == "Steam" {
				opp.Note = "Steam 到手为游戏余额, 不可直接提现"
			}
			opportunities = append(opportunities, opp)
		}
	}

	result := map[string]any{
		"skin_name":     info.Name,
		"skin_id":       info.ID,
		"opportunities": opportunities,
		"fee_rates_used": map[string]float64{
			"steam": SteamSellerFeeRate,
			"buff":  BuffSellerFeeRate,
			"yyyp":  YyypSellerFeeRate,
		},
		"updated_at": info.UpdatedAt,
	}
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// round2 保留 2 位小数
func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
