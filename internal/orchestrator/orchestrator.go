package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/newbeehao/csgo_skinquant/internal/agent"
)

// Orchestrator 是多 Agent 团队的总调度器
type Orchestrator struct {
	marketAnalyst *agent.Agent
	arbHunter     *agent.Agent
	intelResearch *agent.Agent
	riskAnalyst   *agent.Agent
	chief         *agent.Agent

	cbMu         sync.Mutex
	onProgress   func(event ProgressEvent)
	onChiefToken func(token string) // 新增: Chief 流式 token 回调
}

// ProgressEvent 是调度过程中的进度事件
type ProgressEvent struct {
	Phase     string    `json:"phase"` // "start" / "agent_start" / "agent_done" / "agent_error" / "chief_start" / "done"
	AgentName string    `json:"agent_name,omitempty"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Config 构造 Orchestrator 需要的所有依赖
type Config struct {
	MarketAnalyst *agent.Agent
	ArbHunter     *agent.Agent
	IntelResearch *agent.Agent
	RiskAnalyst   *agent.Agent
	Chief         *agent.Agent

	// 可选: 进度回调 (nil 表示不需要)
	OnProgress func(event ProgressEvent)
}

func New(cfg Config) *Orchestrator {
	return &Orchestrator{
		marketAnalyst: cfg.MarketAnalyst,
		arbHunter:     cfg.ArbHunter,
		intelResearch: cfg.IntelResearch,
		riskAnalyst:   cfg.RiskAnalyst,
		chief:         cfg.Chief,
		onProgress:    cfg.OnProgress,
	}
}

func (o *Orchestrator) emit(phase, agentName, message string) {
	o.cbMu.Lock()
	cb := o.onProgress
	o.cbMu.Unlock()
	if cb == nil {
		return
	}
	cb(ProgressEvent{
		Phase:     phase,
		AgentName: agentName,
		Message:   message,
		Timestamp: time.Now(),
	})
}

// Run 执行完整的多 Agent 协作流程, 返回最终研报
func (o *Orchestrator) Run(ctx context.Context, userQuestion string) (*FinalReport, error) {
	start := time.Now()
	o.emit("start", "", "首席策略官开始拆解任务")

	// ========== 阶段 1: 三个 Agent 并行工作 ==========
	// 使用 errgroup: 任何一个 Agent 失败可以取消其他, 也可以都不取消 (用 sync.WaitGroup 即可)
	// 我们这里允许个别 Agent 失败, 所以用 WaitGroup + 手动收集错误

	type subResult struct {
		report AgentReport
	}

	// 这三个任务互相独立, 定义成一个列表方便循环启动
	parallelTasks := []struct {
		name  string
		agent *agent.Agent
	}{
		{"市场数据分析师", o.marketAnalyst},
		{"套利猎手", o.arbHunter},
		{"情报研究员", o.intelResearch},
	}

	results := make([]AgentReport, len(parallelTasks))
	var wg sync.WaitGroup

	for i, task := range parallelTasks {
		i, task := i, task // Go 1.22 起其实可以省这行, 但保留兼容性
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.emit("agent_start", task.name, task.name+" 开始工作")
			taskStart := time.Now()

			content, err := task.agent.Run(ctx, userQuestion)
			duration := time.Since(taskStart)

			report := AgentReport{
				AgentName: task.name,
				Duration:  duration,
			}
			if err != nil {
				report.Error = err.Error()
				report.Content = fmt.Sprintf("(%s 执行失败: %s)", task.name, err.Error())
				o.emit("agent_error", task.name, fmt.Sprintf("%s 失败: %v", task.name, err))
				log.Printf("❌ %s 失败: %v", task.name, err)
			} else {
				report.Content = content
				o.emit("agent_done", task.name, fmt.Sprintf("%s 完成, 耗时 %v", task.name, duration.Round(time.Second)))
				log.Printf("✅ %s 完成, 耗时 %v", task.name, duration.Round(time.Second))
			}
			results[i] = report
		}()
	}

	wg.Wait() // 等三人全部完成

	// ========== 阶段 2: 风控分析师登场 (依赖前三人的输出) ==========
	o.emit("agent_start", "风控分析师", "风控分析师开始综合风险评估")
	riskStart := time.Now()

	// 把前三人的报告拼成上下文给风控
	priorContext := "以下是其他分析师已经完成的报告, 请在你的风控评估中参考他们的发现:\n\n"
	for _, r := range results {
		priorContext += fmt.Sprintf("### %s 的报告\n%s\n\n", r.AgentName, r.Content)
	}
	riskInput := priorContext + "\n原始用户问题: " + userQuestion

	riskContent, err := o.riskAnalyst.Run(ctx, riskInput)
	riskDuration := time.Since(riskStart)
	riskReport := AgentReport{
		AgentName: "风控分析师",
		Duration:  riskDuration,
	}
	if err != nil {
		riskReport.Error = err.Error()
		riskReport.Content = fmt.Sprintf("(风控分析师执行失败: %s)", err.Error())
		o.emit("agent_error", "风控分析师", fmt.Sprintf("风控失败: %v", err))
	} else {
		riskReport.Content = riskContent
		o.emit("agent_done", "风控分析师", fmt.Sprintf("风控完成, 耗时 %v", riskDuration.Round(time.Second)))
	}
	results = append(results, riskReport)

	// ========== 阶段 3: 首席策略官综合所有报告 ==========
	o.emit("chief_start", "首席策略官", "首席策略官正在综合所有报告, 生成最终投资研报")

	chiefInput := "以下是你团队四位分析师对同一个问题的独立分析报告, 请综合他们的结论, 生成一份给投资者的最终研报:\n\n"
	for _, r := range results {
		chiefInput += fmt.Sprintf("### %s\n%s\n\n---\n\n", r.AgentName, r.Content)
	}
	chiefInput += "\n原始用户问题: " + userQuestion

	// Chief 使用流式输出 (如果外部注册了回调)
	// 没有回调时降级为非流式, teamtest 这种终端场景不受影响
	o.cbMu.Lock()
	chiefCb := o.onChiefToken
	o.cbMu.Unlock()

	var chiefContent string
	if chiefCb != nil {
		chiefContent, err = o.chief.RunStream(ctx, chiefInput, chiefCb)
	} else {
		chiefContent, err = o.chief.Run(ctx, chiefInput)
	}
	if err != nil {
		return nil, fmt.Errorf("首席策略官执行失败: %w", err)
	}

	// ========== 汇总返回 ==========
	totalDuration := time.Since(start)
	o.emit("done", "", fmt.Sprintf("全部完成, 总耗时 %v", totalDuration.Round(time.Second)))

	return &FinalReport{
		UserQuestion:  userQuestion,
		SubReports:    results,
		ChiefSummary:  chiefContent,
		TotalDuration: totalDuration,
	}, nil
}

// 让编译器相信我们 import 了 errgroup (实际上在简化版里暂时没直接用, 但留在 import 里方便未来扩展)
var _ = errgroup.Group{}

// SetProgressCallback 在运行时设置进度回调
// 每个 HTTP 请求可以设置自己的回调, 之后调用 Run 即可
// 注意: 如果多请求并发, 需要外部保证回调的线程安全 (我们下面会加锁)
func (o *Orchestrator) SetProgressCallback(cb func(ProgressEvent)) {
	o.cbMu.Lock()
	defer o.cbMu.Unlock()
	o.onProgress = cb
}

// SetChiefTokenCallback 设置 Chief 的流式 token 回调
// 每当 Chief 生成一个 token 就触发一次, 用于 SSE 流式推送
func (o *Orchestrator) SetChiefTokenCallback(cb func(token string)) {
	o.cbMu.Lock()
	defer o.cbMu.Unlock()
	o.onChiefToken = cb
}
