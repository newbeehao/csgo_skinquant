# 🎭 SkinQuant — Multi-Agent AI 饰品量化研究团队

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go" />
  <img src="https://img.shields.io/badge/LLM-DeepSeek-blue" />
  <img src="https://img.shields.io/badge/Framework-Gin-00BFA5" />
  <img src="https://img.shields.io/badge/Streaming-SSE-ff6b6b" />
  <img src="https://img.shields.io/badge/License-MIT-green" />
</p>

> 一个基于 Go 原生并发模型的多 Agent 协作系统。5 位 AI 分析师角色各司其职,并发调度真实市场数据与网络情报,通过 SSE 流式接口实时推送协作过程,为 CS2 饰品投资决策生成**专业级量化研报**。

## ✨ 项目亮点

- 🧠 **5 个异构 Agent 团队协作**: 市场数据分析师、套利猎手、情报研究员、风控分析师、首席策略官,各拥独立工具集与 system prompt,由编排器按 DAG 依赖关系调度
- ⚡ **Go 原生并发编排**: 前三位分析师通过 goroutine 并行工作 (耗时从 165s → 61s, 提速 63%),后置 Agent 等待上游结果后接力,任一 Agent 失败不影响整体输出
- 📡 **SSE 流式实时推送**: 基于 Server-Sent Events 的流式接口,浏览器实时看到"Alex 开始工作 / Hunter 完成 / 首席综合中..."的全过程,2 分钟任务的等待体验从白屏变成可视化进度条
- 🎨 **零依赖浏览器 UI**: 纯原生 HTML/CSS/JS 单文件前端,实时时间线 + Markdown 研报渲染,无需任何前端构建
- 🔧 **完整 Function Calling 循环**: 自研 Agent 核心循环支持多轮工具调用、错误容错、轮次保护,兼容 OpenAI 协议可无缝切换任意 LLM
- 📊 **真实数据源接入**: 对接 CSQAQ API (Steam/BUFF163/悠悠有品三平台实时价格、历史 K 线、存世量) + Tavily AI 搜索 (版本更新/赛事热点情报)
- 🎯 **精确金融计算**: 跨平台套利自动扣除真实手续费 (Steam 13.04% / BUFF 2.5% / 悠悠 1.5%),风险评分覆盖流动性/波动性/趋势/集中度四维度

## 🏗️ 系统架构

```
                        用户浏览器
                            │
                    (SSE 流式推送)
                            │
                            ▼
                      Gin HTTP Server
                            │
                            ▼
                   ┌──── 首席策略官 Chief ────┐
                   │   (任务拆解 / 结果综合)   │
                   └───────────┬──────────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                      ▼
  市场数据分析师 Alex     套利猎手 Hunter      情报研究员 Iris
  [CSQAQ 三平台数据]     [跨平台套利计算]      [Tavily 网络搜索]
        └──────────────────────┬──────────────────────┘
                               │ (goroutine 并行)
                               ▼
                       风控分析师 Risa
                    [四维度风险评分 + 综合前三报告]
                               │
                               ▼
                         📋 最终投资研报
```

### 各 Agent 职责

| Agent | 角色定位 | 专属工具 | 数据源 |
|-------|---------|---------|--------|
| **Alex** 市场数据分析师 | 深度估值分析 | `search_skin_id` / `get_skin_detail` | CSQAQ |
| **Hunter** 套利猎手 | 跨平台挂刀计算 | `search_skin_id` / `calculate_arbitrage` | CSQAQ |
| **Iris** 情报研究员 | 软性情报挖掘 | `web_search` / `fetch_webpage` | Tavily |
| **Risa** 风控分析师 | 综合风险评估 | `search_skin_id` / `calculate_risk_score` | CSQAQ + 上游报告 |
| **Chief** 首席策略官 | 任务编排 + 研报综合 | (无工具,纯综合) | 所有 Agent 输出 |

## 🎬 使用体验

### Web UI 实时体验

打开浏览器访问 `http://localhost:8080/web/index.html`:

1. **输入问题** — 例如"帮我分析 AK-47 红线(久经沙场),值不值得入手?"
2. **瞬间启动** — 时间线上三个黄色圆点同时亮起并脉冲闪烁: `🟡 Alex / 🟡 Hunter / 🟡 Iris 开始工作`(**这是并发的视觉证据**)
3. **陆续完成** — 20-60 秒内三个圆点陆续变绿 ✅,每个 Agent 完成立即反馈
4. **风控接力** — 前三人全部完成后,风控圆点自动亮起
5. **首席综合** — 最后首席策略官综合生成研报
6. **研报呈现** — 下方区域淡入渲染的 Markdown 研报,4 个可折叠按钮查看各分析师原始报告

### 命令行体验

```bash
# 非流式接口 (适合 API 集成)
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"question":"帮我分析 AK-47 红线(久经沙场)"}'

# 流式接口 (SSE, 实时看到进度)
curl -N -X POST http://localhost:8080/api/analyze/stream \
  -H "Content-Type: application/json" \
  -d '{"question":"帮我分析 AK-47 红线(久经沙场)"}'
```

流式输出示例:

```
event: agent_start
data: {"agent_name":"情报研究员","message":"情报研究员 开始工作","timestamp":"15:11:28"}

event: agent_start
data: {"agent_name":"市场数据分析师","message":"市场数据分析师 开始工作","timestamp":"15:11:28"}

event: agent_start
data: {"agent_name":"套利猎手","message":"套利猎手 开始工作","timestamp":"15:11:28"}

event: agent_done
data: {"agent_name":"套利猎手","message":"套利猎手 完成, 耗时 26s","timestamp":"15:11:54"}

... (陆续推送)

event: final_report
data: {"chief_summary":"## 投资研报...","sub_reports":[...]}
```

## 🛠️ 技术栈

| 层 | 技术 |
|---|---|
| 语言 | Go 1.23 |
| Web 框架 | Gin |
| 流式推送 | Server-Sent Events (SSE) |
| 前端 | 原生 HTML + CSS + JS (marked.js 渲染 Markdown) |
| LLM | DeepSeek (`deepseek-chat`,兼容 OpenAI 协议) |
| 饰品数据源 | [CSQAQ API](https://csqaq.com/) — 三平台聚合价格 |
| 搜索数据源 | [Tavily](https://tavily.com/) — AI 原生搜索 |
| 并发原语 | `goroutine` / `sync.WaitGroup` / `channel` |

## 🚀 快速开始

### 前置准备

1. **Go 1.23** 或更高版本
2. 注册 [DeepSeek](https://platform.deepseek.com/) 拿 API Key
3. 注册 [CSQAQ](https://csqaq.com/) 拿 ApiToken,并绑定开发机公网 IP 白名单
4. (可选) 注册 [Tavily](https://tavily.com/) 拿 API Key — 情报 Agent 需要

### 安装与启动

```bash
# 克隆仓库
git clone https://github.com/newbeehao/cs_goskinquant.git
cd skinquant

# 安装依赖
go mod download

# 设置环境变量
export DEEPSEEK_API_KEY="sk-xxxxx"
export CSQAQ_TOKEN="your_csqaq_token"
export TAVILY_API_KEY="tvly-xxxxx"   # 可选

# 启动完整服务 (Gin + Web UI + SSE)
go run ./cmd/server
```

启动成功后看到:

```
✅ 配置加载成功
✅ Tavily 已启用 (情报 Agent 可用)
🚀 SkinQuant 启动, 监听 :8080
   POST /api/analyze         — 非流式
   POST /api/analyze/stream  — SSE 流式
```

**浏览器打开**: `http://localhost:8080/web/index.html`

### 独立测试每个 Agent

```bash
go run ./cmd/teamtest        # 完整 5-Agent 团队协作 (终端输出)
go run ./cmd/agenttest       # 市场数据分析师
go run ./cmd/arbitragetest   # 套利猎手
go run ./cmd/inteltest       # 情报研究员
go run ./cmd/risktest        # 风控分析师
go run ./cmd/csqaqtest       # CSQAQ 数据源联通性测试
```

## 📁 项目结构

```
skinquant/
├── cmd/                    # 各场景入口程序
│   ├── server/             # 完整 HTTP 服务 (主入口)
│   ├── teamtest/           # 完整团队协作终端演示
│   ├── agenttest/          # 市场数据分析师独立测试
│   ├── arbitragetest/      # 套利猎手独立测试
│   ├── inteltest/          # 情报研究员独立测试
│   ├── risktest/           # 风控分析师独立测试
│   └── csqaqtest/          # CSQAQ 数据源联通性测试
├── internal/
│   ├── config/             # 环境变量与配置管理
│   ├── llm/                # DeepSeek 客户端 + Function Calling 协议
│   ├── tools/              # 工具接口与具体工具实现
│   ├── datasource/         # CSQAQ / Tavily HTTP 客户端
│   ├── agent/              # Agent 核心循环
│   ├── orchestrator/       # 多 Agent DAG 调度器
│   └── api/                # Gin HTTP Handler + SSE 推送
├── web/
│   └── index.html          # 零依赖前端页面
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

## 🔬 Agent 核心循环伪代码

```go
func (a *Agent) Run(ctx, userMessage) (string, error) {
    messages := [系统提示, 用户消息]
    tools := 注册的工具定义列表

    for turn := 1; turn <= maxTurns; turn++ {
        resp := llmClient.Chat(ctx, messages, tools)
        assistantMsg := resp.Choices[0].Message
        messages = append(messages, assistantMsg)

        // 没有工具调用 → 最终答案
        if len(assistantMsg.ToolCalls) == 0 {
            return assistantMsg.Content, nil
        }

        // 执行每个工具调用并把结果回传
        for _, toolCall := range assistantMsg.ToolCalls {
            result := registry.Get(toolCall.Name).Execute(args)
            messages = append(messages, Message{
                Role: "tool", ToolCallID: toolCall.ID, Content: result,
            })
        }
    }
    return "", 超过最大轮数
}
```

## 🌐 HTTP API 接口

### `POST /api/analyze` — 非流式

**Request**:
```json
{ "question": "帮我分析AWP克拉考（略有磨损）自用玩家最近能入手吗" }
```

**Response** (2 分钟后一次性返回):
```json
{
  "question": "...",
  "chief_summary": "## 投资研报...",
  "sub_reports": [
    { "agent_name": "市场数据分析师", "content": "...", "duration": 28754019074 },
    { "agent_name": "套利猎手", "content": "...", "duration": 30886069819 }
  ],
  "total_duration": 158.5
}
```

### `POST /api/analyze/stream` — SSE 流式

同样的请求体,返回 `Content-Type: text/event-stream`,事件类型包括:

| Event | 含义 |
|-------|------|
| `start` | 任务启动 |
| `agent_start` | 某个 Agent 开始工作 (三个分析师同时触发) |
| `agent_done` | 某个 Agent 完成 |
| `agent_error` | 某个 Agent 失败 (其他 Agent 继续) |
| `chief_start` | 首席策略官开始综合 |
| `done` | 全部完成 |
| `final_report` | 最终研报数据 |
| `error` | 致命错误 |

### `GET /health` — 健康检查



## 🎯 适用场景

- **CS2 饰品投资者**: 自动化生成研报辅助决策
- **AI Agent 学习者**: 完整可运行的 Multi-Agent 参考实现
- **Go 并发学习者**: 真实场景下的 goroutine + DAG 编排范例
- **LLM 应用开发者**: Function Calling / SSE / Prompt Engineering 最佳实践

## ⚠️ 免责声明

本项目仅用于技术学习与演示,输出的投资研报**不构成任何投资建议**。CSGO 饰品市场存在显著价格波动风险,入市需谨慎。

本项目接入的第三方数据源 (CSQAQ / Tavily / DeepSeek) 均需遵守其各自的服务条款。使用者需自行申请相应 API Key 并承担使用成本与合规责任。

## 📝 License

MIT License — 详见 [LICENSE](./LICENSE) 文件。

你可以自由地使用、修改、分发本项目,包括商业用途,只需保留原作者的版权声明。

## 🙋 致谢

- 数据源: [CSQAQ](https://csqaq.com/) — 国内最大的 CS2 饰品量化分析平台
- 搜索: [Tavily](https://tavily.com/) — AI 原生搜索 API
- LLM: [DeepSeek](https://platform.deepseek.com/) — 性价比极高的国产大模型
- 框架: [Gin](https://github.com/gin-gonic/gin) — 高性能 Go Web 框架
- Markdown: [marked.js](https://github.com/markedjs/marked) — 轻量的 Markdown 渲染器

---

<p align="center">
  如果这个项目对你有帮助,欢迎点个 ⭐ Star!
</p>