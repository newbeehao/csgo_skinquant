# 🎭 SkinQuant — Multi-Agent AI 饰品量化研究团队

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go" />
  <img src="https://img.shields.io/badge/LLM-DeepSeek-blue" />
  <img src="https://img.shields.io/badge/Framework-Gin-00BFA5" />
  <img src="https://img.shields.io/badge/License-MIT-green" />
</p>

> 一个基于 Go 原生并发模型的多 Agent 协作系统。5 位 AI 分析师角色各司其职，并发调度真实市场数据与网络情报，为 CS2 饰品投资决策生成**专业级量化研报**。

## ✨ 项目亮点

- 🧠 **5 个异构 Agent 团队协作**: 市场数据分析师、套利猎手、情报研究员、风控分析师、首席策略官,各拥独立工具集与 system prompt,由编排器按 DAG 依赖关系调度
- ⚡ **Go 原生并发编排**: 前三位分析师通过 goroutine 并行工作 (耗时从 165s → 61s, 提速 63%),后置 Agent 等待上游结果后接力,任一 Agent 失败不影响整体输出
- 🔧 **完整 Function Calling 循环**: 自研 Agent 核心循环支持多轮工具调用、错误容错、轮次保护,兼容 OpenAI 协议可无缝切换任意 LLM
- 📊 **真实数据源接入**: 对接 CSQAQ API (Steam/BUFF163/悠悠有品三平台实时价格、历史 K 线、存世量) + Tavily AI 搜索 (版本更新/赛事热点情报)
- 🎯 **精确金融计算**: 跨平台套利自动扣除真实手续费 (Steam 13.04% / BUFF 2.5% / 悠悠 1.5%),风险评分覆盖流动性/波动性/趋势/集中度四维度

## 🏗️ 系统架构

```
                        用户问题
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

## 🛠️ 技术栈

| 层 | 技术 |
|---|---|
| 语言 | Go 1.23 |
| Web 框架 | Gin |
| LLM | DeepSeek (`deepseek-chat`,兼容 OpenAI 协议) |
| 饰品数据源 | [CSQAQ API](https://csqaq.com/) — 三平台聚合价格 |
| 搜索数据源 | [Tavily](https://tavily.com/) — AI 原生搜索 |
| 并发原语 | `goroutine` / `sync.WaitGroup` / `channel` |
| 持久化 | SQLite (会话存储) |

## 🚀 快速开始

### 前置准备

1. Go 1.23 或更高版本
2. 注册 [DeepSeek](https://platform.deepseek.com/) 拿 API Key
3. 注册 [CSQAQ](https://csqaq.com/) 拿 ApiToken,并绑定开发机公网 IP 白名单
4. (可选,情报 Agent 需要) 注册 [Tavily](https://tavily.com/) 拿 API Key

### 安装运行

```bash
# 克隆仓库
git clone https://github.com/newbeehao/csgo_skinquant.git
cd csgo_skinquant

# 安装依赖
go mod download

# 设置环境变量
export DEEPSEEK_API_KEY="sk-xxxxx"
export CSQAQ_TOKEN="your_csqaq_token"
export TAVILY_API_KEY="tvly-xxxxx"  # 可选

# 运行完整团队协作 (演示)
go run ./cmd/teamtest

# 或运行单个 Agent 验证
go run ./cmd/agenttest      # 市场数据分析师
go run ./cmd/arbitragetest  # 套利猎手
go run ./cmd/inteltest      # 情报研究员
go run ./cmd/risktest       # 风控分析师
```

### 示例输出

```
👤 用户: 帮我深度分析 AK-47 红线(久经沙场), 现在值不值得入手?

📡 [agent_start] 情报研究员 开始工作
📡 [agent_start] 市场数据分析师 开始工作    ← 三人同时启动
📡 [agent_start] 套利猎手 开始工作
📡 [agent_done] 套利猎手 完成, 耗时 26s
📡 [agent_done] 市场数据分析师 完成, 耗时 39s
📡 [agent_done] 情报研究员 完成, 耗时 1m1s
📡 [agent_start] 风控分析师开始综合风险评估   ← 等前三人完成后启动
📡 [chief_start] 首席策略官综合生成最终研报

📋 最终投资研报
## 投资研报: AK-47 红线(久经沙场)
### 核心结论
作为一款经典皮肤, 普通版当前价格处于相对低位, 流动性优秀...
### 最终建议: 观望 (纯投资) / 买入 (自用或 Steam 消费需求)
...
⏱️  总耗时: 2m0s (串行预估 3m10s, 节省 37%)
```

## 📁 项目结构

```
skinquant/
├── cmd/                    # 各场景入口程序
│   ├── server/             # Gin HTTP 服务
│   ├── teamtest/           # 完整团队协作演示
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
│   └── api/                # Gin HTTP / SSE 接口
├── web/                    # 前端页面
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

## 🎯 适用场景

- **CS2 饰品投资者**: 自动化生成研报辅助决策
- **AI Agent 学习者**: 完整可运行的 Multi-Agent 参考实现
- **Go 并发学习者**: 真实场景下的 goroutine + DAG 编排范例
- **LLM 应用开发者**: Function Calling / Prompt Engineering 最佳实践

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

---

<p align="center">
  如果这个项目对你有帮助,欢迎点个 ⭐ Star!
</p>
