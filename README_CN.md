# fluxcore ⚡

**LLM API 路由库**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v0.5.0-blue?style=flat)]()
[![English](https://img.shields.io/badge/README-English-blue?style=flat)](README.md)

30行代码，LLM API 路由搞定。

---

## v0.5.0 升级亮点

**重大改进：**

- **🧹 更简洁 API** — 删除未使用导出，内部常量私有化。更小接口，更易使用。
- **🛡️ 实战测试** — 新增稳定性测试：熔断器恢复流程、EWMA 延迟追踪、网络韧性。
- **⚡ 零依赖** — 17个文件，0接口，0外部包。纯 Go 实现。
- **🔒 安全优先** — SSRF防护移至应用层（你的策略，你的控制）。
- **📊 90%+ 测试覆盖** — Routing 94.2%，Call 90.8%，覆盖全面边界情况。

---

## 特性

- **零抽象** — 17个文件，无接口层。读代码，懂流程。
- **价格优先路由** — 自动选择最便宜可用端点。
- **熔断器 + 重试** — 3次失败触发熔断，60秒自动恢复。
- **协议转换** — Anthropic 输入，Gemini 输出，透明翻译。
- **EWMA 延迟追踪** — 平滑延迟估计，自适应超时。

---

## 4行代码起步

```go
pool := routing.NewEndpointPool(endpoints, 3)
resp, usage, err := call.Request(ctx, pool, rawReq, routing.ProtocolOpenAI)
// 完成。
```

---

## 快速开始

```go
import (
    "github.com/tokflux/fluxcore/routing"
    "github.com/tokflux/fluxcore/call"
)

// 1. 定义密钥（连接凭证）
keys := []*routing.Key{
    {BaseURL: "https://api.openai.com", APIKey: key1, Protocol: routing.ProtocolOpenAI},
    {BaseURL: "https://api.anthropic.com", APIKey: key2, Protocol: routing.ProtocolAnthropic},
    {BaseURL: "https://generativelanguage.googleapis.com", APIKey: key3, Protocol: routing.ProtocolGemini},
}

// 2. 创建端点（密钥 + 模型 + 优先级）
// 注意：Model 参数对 Gemini 必填（用于 URL），OpenAI/Anthropic/Cohere 用空字符串 ""
// Priority: 值越小优先级越高（如价格，单位为微）
ep1, _ := routing.NewEndpoint(1, keys[0], "", 1000)   // OpenAI，优先级 1000
ep2, _ := routing.NewEndpoint(2, keys[1], "", 800)    // Anthropic，优先级 800（更便宜）
ep3, _ := routing.NewEndpoint(3, keys[2], "gemini-pro", 100) // Gemini，优先级 100（最便宜）

endpoints := []*routing.Endpoint{ep1, ep2, ep3}

// 3. 创建池（自动选择最便宜，自动熔断）
pool := routing.NewEndpointPool(endpoints, 3)

// 4. 非流式请求
resp, usage, err := call.Request(ctx, pool, rawReq, routing.ProtocolOpenAI)

// 5. 流式请求（协议自动转换）
result, err := call.RequestStream(ctx, pool, rawReq, routing.ProtocolAnthropic)
if err != nil { return err }
defer result.Close()
for chunk := range result.Ch { c.Write(chunk) }
```

---

## 协议透明转换

```go
// 前端: Anthropic SDK 格式
// 后端: Gemini provider（更便宜）
// fluxcore: 自动转换

anthropicReq := `{"model": "claude-3", "messages": [...]}`
resp, _, _ := call.Request(ctx, pool, anthropicReq, routing.ProtocolAnthropic)
// 输出是 Anthropic 格式，即使 pool 选择了 Gemini endpoint
```

---

## 成本优先路由

```go
// 创建端点，标注优先级（值越小越好）
ep1, _ := routing.NewEndpoint(1, key1, "", 1000)   // OpenAI: 优先级 1000
ep2, _ := routing.NewEndpoint(2, key2, "", 100)    // Gemini: 优先级 100（更便宜）

pool := routing.NewEndpointPool([]*routing.Endpoint{ep1, ep2}, 3)

// 自动选择最低优先级端点
// Gemini 失败？自动切换到 OpenAI
```

---

## 熔断器自愈

```
健康 → 失败 → 失败 → 失败 → 🔴 熔断
                              ↓
                        60秒自动恢复探测
```

3次失败触发熔断，自动切换到其他端点。60秒后自动恢复探测。

### 默认配置

```go
// 默认：3次失败触发熔断，60秒恢复超时
ep, _ := routing.NewEndpoint(id, key, model, priority)

// 检查熔断器状态
if ep.IsCircuitBreakerOpen() {
    // 端点不健康，跳过
}
```

### API 参考

| 方法 | 描述 |
|------|------|
| `IsCircuitBreakerOpen()` | 返回 true 表示熔断器开启（应跳过） |
| `MarkSuccess()` | 标记端点健康，重置失败计数 |
| `MarkFail()` | 标记端点失败，增加失败计数 |

---

## EWMA 延迟追踪

Fluxcore 使用 EWMA（指数加权移动平均）追踪端点延迟：

```go
// 每次请求后更新延迟
ep.UpdateLatency(200) // 200ms 延迟

// 获取平滑延迟估计
latency := ep.LatencyEWMA() // 返回 EWMA 值（毫秒）
```

**EWMA 公式：** `新值 = 0.1 × 当前 + 0.9 × 旧值`

提供平滑延迟估计，逐步适应变化，避免对异常值敏感。

---

## 智能错误分类

| 错误类型 | 处理方式 |
|----------|----------|
| 网络超时 | 自动重试 |
| 服务不可用 (503) | 自动重试 |
| 认证失败 (401) | 不重试，直接报错 |
| 配额超限 (429) | 等待后重试 |

---

## 使用统计

```go
resp, usage, err := call.Request(ctx, pool, rawReq, routing.ProtocolOpenAI)
if usage != nil {
    fmt.Printf("输入 tokens: %d\n", usage.InputTokens)
    fmt.Printf("输出 tokens: %d\n", usage.OutputTokens)
    fmt.Printf("延迟: %dms\n", usage.LatencyMs)
    fmt.Printf("总 tokens: %d\n", usage.TotalTokens())

    // 检查使用量是否精确（Provider 报告，非估算）
    if usage.IsAccurate {
        fmt.Println("使用量精确（Provider 报告）")
    }
}
```

### Usage 字段

| 字段 | 类型 | 描述 |
|------|------|------|
| `InputTokens` | `int` | 输入/prompt tokens |
| `OutputTokens` | `int` | 输出/completion tokens |
| `LatencyMs` | `int` | 请求延迟（毫秒） |
| `IsAccurate` | `bool` | true 表示 Provider 报告的精确值 |

---

## 架构

```
┌─────────────────────────────────────────────┐
│                 fluxcore                     │
│        LLM API 路由库                        │
├─────────────┬───────────────────────────────┤
│  message/   │          routing/             │
│  数据类型    │     路由选择 + 熔断器         │
│  (IR层)     │          (无锁)               │
├─────────────┴───────────────────────────────┤
│                   call/                      │
│      HTTP传输 + 重试 + 流式                  │
│            + 协议转换                        │
└─────────────────────────────────────────────┘

4个包。17个文件。0个接口。0个依赖。
```

**包职责：**

| 包 | 用途 | 文件数 |
|----|------|--------|
| `message` | LLM 消息数据结构 | 6 |
| `routing` | 端点路由 + 选择 + 熔断器 | 6 |
| `call` | HTTP传输 + 重试 + 流式 | 5 |
| `errors` | 错误分类 + 重试判断 | 2 |

---

## 性能

| 操作 | 时间 | 说明 |
|------|------|------|
| `CurrentEp()` | ~10ns | 无锁原子读取 |
| `MarkFail()` | ~50ns | CAS + O(1) 映射 |
| `SelectBest()` | ~100ns | 优先级扫描 + 熔断检查 |
| 并发测试 | 1000 QPS | 无死锁 |

---

## 安全（SSRF 防护）

fluxcore 验证端点格式。SSRF 防护是应用层责任。

```go
ep := &routing.Endpoint{
    Key: &routing.Key{
        BaseURL:  userProvidedURL,  // 用户输入
        APIKey:   "your-key",
        Protocol: routing.ProtocolOpenAI,
    },
}

// Step 1: 验证格式
if err := ep.Validate(); err != nil {
    return err
}

// Step 2: SSRF 防护（你的策略）
// 在应用层实现 IP 验证
```

---

## 谁在使用

| 用户 | 用途 |
|------|------|
| **SaaS 团队** | 多租户 AI 功能，端点隔离 |
| **AI 创业** | 成本控制，自动选最便宜 |
| **平台团队** | 统一 LLM API，运维友好 |
| **独立开发者** | 快速上线，零学习成本 |

---

## 协议支持

| 格式 | 常量 | 端点 |
|------|------|------|
| **OpenAI** | `ProtocolOpenAI` | `/v1/chat/completions` |
| **Anthropic** | `ProtocolAnthropic` | `/v1/messages` |
| **Gemini** | `ProtocolGemini` | `/v1/models/{model}:generateContent` |
| **Cohere** | `ProtocolCohere` | `/v1/chat` |

### OpenAI 兼容提供商

| 提供商 | Base URL |
|--------|----------|
| **Azure OpenAI** | 你的 Azure 端点 |
| **Mistral AI** | `https://api.mistral.ai` |
| **Groq** | `https://api.groq.com` |
| **DeepSeek** | `https://api.deepseek.com` |
| **智谱 GLM-4** | `https://open.bigmodel.cn/api/paas/v4/` |

---

## 集成示例

```go
package main

import (
    "io"
    "github.com/tokflux/fluxcore/routing"
    "github.com/tokflux/fluxcore/call"
    "github.com/gin-gonic/gin"
)

func main() {
    keys := []*routing.Key{
        {BaseURL: "https://api.openai.com", APIKey: "sk-xxx", Protocol: routing.ProtocolOpenAI},
        {BaseURL: "https://api.anthropic.com", APIKey: "sk-yyy", Protocol: routing.ProtocolAnthropic},
    }

    ep1, _ := routing.NewEndpoint(1, keys[0], "", 1000)
    ep2, _ := routing.NewEndpoint(2, keys[1], "", 800)

    endpoints := []*routing.Endpoint{ep1, ep2}
    pool := routing.NewEndpointPool(endpoints, 3)

    r := gin.Default()
    r.POST("/v1/chat/completions", func(c *gin.Context) {
        rawReq, _ := io.ReadAll(c.Request.Body)
        resp, usage, err := call.Request(c.Request.Context(), pool, rawReq, routing.ProtocolOpenAI)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.Data(200, "application/json", resp)
    })

    r.Run(":8080")
}
```

---

## 开始使用

```bash
go get github.com/tokflux/fluxcore@v0.5.0
```

**下一步：**
1. 试试上面的快速开始
2. 查看集成示例
3. ⭐ Star 如果有帮助

---

## License

MIT. Free forever.

---

**fluxcore v0.5.0 - LLM API Router Library. 30行代码，路由搞定。**