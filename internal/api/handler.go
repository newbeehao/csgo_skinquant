package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/newbeehao/csgo_skinquant/internal/orchestrator"
)

// Handler 是所有 HTTP 接口的处理器集合
type Handler struct {
	orch *orchestrator.Orchestrator
}

func NewHandler(orch *orchestrator.Orchestrator) *Handler {
	return &Handler{orch: orch}
}

// AnalyzeRequest 是 /analyze 的请求体
type AnalyzeRequest struct {
	Question string `json:"question" binding:"required"`
}

// AnalyzeStream 是 SSE 流式分析接口
// 同时推送: 进度事件 (agent_start/done 等) + Chief 流式 token
func (h *Handler) AnalyzeStream(c *gin.Context) {
	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	// 1. SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// 2. 两个 channel: 进度事件 + token 流
	progressChan := make(chan orchestrator.ProgressEvent, 100)
	tokenChan := make(chan string, 1000) // token 量大, buffer 开大点

	var wg sync.WaitGroup
	var finalReport *orchestrator.FinalReport
	var finalErr error

	// 3. 启动 orchestrator 任务 (单独 goroutine)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(progressChan)
		defer close(tokenChan)

		// 注册两个回调: 进度事件 + token 流
		h.orch.SetProgressCallback(func(e orchestrator.ProgressEvent) {
			select {
			case progressChan <- e:
			default:
				log.Printf("⚠️ 进度事件被丢弃 (channel 满)")
			}
		})
		h.orch.SetChiefTokenCallback(func(token string) {
			select {
			case tokenChan <- token:
			default:
				// token 通道满极不可能, 丢少量 token 不致命
			}
		})

		report, err := h.orch.Run(c.Request.Context(), req.Question)
		finalReport = report
		finalErr = err
	}()

	// 4. 主线程用 select 同时消费两个 channel
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-progressChan:
			if !ok {
				// progressChan 关了, 先把 tokenChan 剩余的 token 推完
				for token := range tokenChan {
					c.SSEvent("token", gin.H{"text": token})
				}
				// 再推送最终结果
				if finalErr != nil {
					c.SSEvent("error", gin.H{"error": finalErr.Error()})
				} else if finalReport != nil {
					c.SSEvent("final_report", gin.H{
						"question":       finalReport.UserQuestion,
						"chief_summary":  finalReport.ChiefSummary,
						"sub_reports":    finalReport.SubReports,
						"total_duration": finalReport.TotalDuration.Seconds(),
					})
				}
				return false
			}
			c.SSEvent(event.Phase, gin.H{
				"agent_name": event.AgentName,
				"message":    event.Message,
				"timestamp":  event.Timestamp.Format("15:04:05"),
			})
			return true

		case token, ok := <-tokenChan:
			if !ok {
				// tokenChan 关了但 progressChan 可能还没关, 继续等
				return true
			}
			c.SSEvent("token", gin.H{"text": token})
			return true
		}
	})

	wg.Wait()
}

// Health 是健康检查接口
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "skinquant",
	})
}

// 一个简单的非流式接口, 方便 curl 测试不用处理 SSE
// POST /api/analyze
func (h *Handler) Analyze(c *gin.Context) {
	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 清空进度回调 (这个接口不需要流式)
	h.orch.SetProgressCallback(nil)

	report, err := h.orch.Run(c.Request.Context(), req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"question":       report.UserQuestion,
		"chief_summary":  report.ChiefSummary,
		"sub_reports":    report.SubReports,
		"total_duration": report.TotalDuration.Seconds(),
	})
	_ = fmt.Sprintf // 避免 unused import 提示
	_ = context.Background
}
