package orchestrator

import "time"

// AgentReport 是每个 Agent 完成任务后的标准报告
type AgentReport struct {
	AgentName string        `json:"agent_name"`      // "市场数据分析师" 等
	Content   string        `json:"content"`         // Agent 的完整文本输出
	Duration  time.Duration `json:"duration"`        // 耗时
	Error     string        `json:"error,omitempty"` // 如果失败, 记录错误
}

// FinalReport 是整个团队产出的最终研报
type FinalReport struct {
	UserQuestion  string        `json:"user_question"`
	SubReports    []AgentReport `json:"sub_reports"`   // 四个分析师的报告
	ChiefSummary  string        `json:"chief_summary"` // 首席策略官的综合研报
	TotalDuration time.Duration `json:"total_duration"`
}
