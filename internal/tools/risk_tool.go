package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/newbeehao/csgo_skinquant/internal/datasource"
	"github.com/newbeehao/csgo_skinquant/internal/llm"
)

type RiskScoreTool struct {
	client *datasource.CSQAQClient
}

func NewRiskScoreTool(client *datasource.CSQAQClient) *RiskScoreTool {
	return &RiskScoreTool{client: client}
}

func (t *RiskScoreTool) Name() string { return "calculate_risk_score" }

func (t *RiskScoreTool) Definition() llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name: "calculate_risk_score",
			Description: "基于饰品的流动性、波动性、长期趋势、集中度四个维度, 计算 0-100 的综合风险评分。" +
				"评分越高风险越大。输入饰品 ID, 返回各维度得分、总分、风险等级、以及每个维度的解释。" +
				"用于评估某饰品作为投资标的的风险水平。",
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

// ScoreDimension 单个维度的评分详情
type ScoreDimension struct {
	Name        string  `json:"name"`
	Score       int     `json:"score"`       // 本维度得分
	MaxScore    int     `json:"max_score"`   // 本维度满分
	RawValue    float64 `json:"raw_value"`   // 原始数据值 (成交量/涨跌幅等)
	Explanation string  `json:"explanation"` // 文字解释
}

func (t *RiskScoreTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	skinID, ok := args["skin_id"].(string)
	if !ok {
		return "", fmt.Errorf("参数 skin_id 缺失")
	}

	info, err := t.client.GetSkinDetail(ctx, skinID)
	if err != nil {
		return "", err
	}

	// === 维度 1: 流动性风险 (看近期成交量) ===
	liquidity := ScoreDimension{Name: "流动性风险", MaxScore: 25, RawValue: float64(info.TurnoverNumber)}
	switch {
	case info.TurnoverNumber < 10:
		liquidity.Score = 25
		liquidity.Explanation = fmt.Sprintf("近期成交仅 %d 件, 流动性极差, 买入后可能难以出手", info.TurnoverNumber)
	case info.TurnoverNumber < 50:
		liquidity.Score = 15
		liquidity.Explanation = fmt.Sprintf("近期成交 %d 件, 流动性一般, 出手需要等待", info.TurnoverNumber)
	case info.TurnoverNumber < 200:
		liquidity.Score = 8
		liquidity.Explanation = fmt.Sprintf("近期成交 %d 件, 流动性尚可", info.TurnoverNumber)
	default:
		liquidity.Score = 0
		liquidity.Explanation = fmt.Sprintf("近期成交 %d 件, 流动性优秀, 容易出手", info.TurnoverNumber)
	}

	// === 维度 2: 波动性风险 (看 30 天涨跌幅绝对值) ===
	vol := math.Abs(info.SellPriceRate30)
	volatility := ScoreDimension{Name: "波动性风险", MaxScore: 25, RawValue: info.SellPriceRate30}
	switch {
	case vol > 30:
		volatility.Score = 25
		volatility.Explanation = fmt.Sprintf("30 天涨跌幅 %.2f%%, 价格剧烈波动, 风险极高", info.SellPriceRate30)
	case vol > 15:
		volatility.Score = 15
		volatility.Explanation = fmt.Sprintf("30 天涨跌幅 %.2f%%, 波动较大", info.SellPriceRate30)
	case vol > 5:
		volatility.Score = 8
		volatility.Explanation = fmt.Sprintf("30 天涨跌幅 %.2f%%, 波动适中", info.SellPriceRate30)
	default:
		volatility.Score = 0
		volatility.Explanation = fmt.Sprintf("30 天涨跌幅 %.2f%%, 价格稳定", info.SellPriceRate30)
	}

	// === 维度 3: 趋势风险 (看 365 天涨跌幅) ===
	trend := ScoreDimension{Name: "长期趋势风险", MaxScore: 25, RawValue: info.SellPriceRate365}
	switch {
	case info.SellPriceRate365 < -20:
		trend.Score = 25
		trend.Explanation = fmt.Sprintf("365 天下跌 %.2f%%, 长期熊市, 抄底需谨慎", info.SellPriceRate365)
	case info.SellPriceRate365 < -10:
		trend.Score = 15
		trend.Explanation = fmt.Sprintf("365 天下跌 %.2f%%, 趋势偏空", info.SellPriceRate365)
	case info.SellPriceRate365 < 0:
		trend.Score = 8
		trend.Explanation = fmt.Sprintf("365 天下跌 %.2f%%, 略有下跌", info.SellPriceRate365)
	default:
		trend.Score = 0
		trend.Explanation = fmt.Sprintf("365 天上涨 %.2f%%, 长期趋势向好", info.SellPriceRate365)
	}

	// === 维度 4: 集中度风险 (看存世量) ===
	concentration := ScoreDimension{Name: "集中度风险", MaxScore: 25, RawValue: float64(info.Statistic)}
	switch {
	case info.Statistic < 500:
		concentration.Score = 10
		concentration.Explanation = fmt.Sprintf("存世量仅 %d 件, 可能是稀有品或流动性陷阱", info.Statistic)
	case info.Statistic > 500000:
		concentration.Score = 15
		concentration.Explanation = fmt.Sprintf("存世量 %d 件, 过于常见, 缺乏稀缺性溢价", info.Statistic)
	case info.Statistic > 100000:
		concentration.Score = 8
		concentration.Explanation = fmt.Sprintf("存世量 %d 件, 较为常见", info.Statistic)
	default:
		concentration.Score = 0
		concentration.Explanation = fmt.Sprintf("存世量 %d 件, 稀缺性适中", info.Statistic)
	}

	// 汇总
	totalScore := liquidity.Score + volatility.Score + trend.Score + concentration.Score
	level := riskLevel(totalScore)

	result := map[string]any{
		"skin_id":       info.ID,
		"skin_name":     info.Name,
		"total_score":   totalScore,
		"risk_level":    level,
		"dimensions":    []ScoreDimension{liquidity, volatility, trend, concentration},
		"scoring_rules": "评分 0-100, 越高越危险。0-20 低风险 / 20-40 中低 / 40-60 中 / 60-80 中高 / 80-100 高",
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}

func riskLevel(score int) string {
	switch {
	case score < 20:
		return "低风险"
	case score < 40:
		return "中低风险"
	case score < 60:
		return "中等风险"
	case score < 80:
		return "中高风险"
	default:
		return "高风险"
	}
}
