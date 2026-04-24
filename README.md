# fluxcore ⚡

**LLM API Router Library**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v0.5.0-blue?style=flat)]()

30 lines to route LLM APIs.

---

## v0.5.0 Highlights

**Major improvements:**

- **🧹 Cleaner API** — Removed unused exports, internal constants now private. Smaller surface, easier to use.
- **🛡️ Battle-Tested** — New stability tests for circuit breaker recovery, EWMA latency tracking, network resilience.
- **⚡ Zero Dependencies** — 17 files, 0 interfaces, 0 external packages. Pure Go.
- **🔒 Security-First** — SSRF protection moved to application layer (your policy, your control).
- **📊 90%+ Test Coverage** — Routing 94.2%, Call 90.8%, comprehensive edge cases covered.

---

## Features

- **Zero Abstraction** — 17 files, no interface layers. Read the code, understand the flow.
- **Price-First Routing** — Auto-select cheapest available endpoint.
- **Circuit Breaker + Retry** — 3 failures trigger circuit, auto-recovery in 60s.
- **Protocol Conversion** — Anthropic in, Gemini out, transparent translation.
- **EWMA Latency Tracking** — Smooth latency estimation for adaptive timeouts.

---

## 4 Lines

```go
pool := routing.NewEndpointPool(endpoints, 3)
resp, usage, err := call.Request(ctx, pool, rawReq, routing.ProtocolOpenAI)
// Done.
```

---

## Quick Start

```go
import (
    "github.com/tokflux/fluxcore/routing"
    "github.com/tokflux/fluxcore/call"
)

// 1. Define keys (connection credentials)
keys := []*routing.Key{
    {BaseURL: "https://api.openai.com", APIKey: key1, Protocol: routing.ProtocolOpenAI},
    {BaseURL: "https://api.anthropic.com", APIKey: key2, Protocol: routing.ProtocolAnthropic},
    {BaseURL: "https://generativelanguage.googleapis.com", APIKey: key3, Protocol: routing.ProtocolGemini},
}

// 2. Create endpoints (key + model + priority)
// Note: Model is required for Gemini (used in URL), empty for OpenAI/Anthropic/Cohere
// Priority: lower values are preferred (e.g., price in micros)
ep1, _ := routing.NewEndpoint(1, keys[0], "", 1000)   // OpenAI, priority=1000
ep2, _ := routing.NewEndpoint(2, keys[1], "", 800)    // Anthropic, priority=800 (cheaper)
ep3, _ := routing.NewEndpoint(3, keys[2], "gemini-pro", 100) // Gemini, priority=100 (cheapest)

endpoints := []*routing.Endpoint{ep1, ep2, ep3}

// 3. Create pool (auto-select cheapest, auto circuit breaker)
pool := routing.NewEndpointPool(endpoints, 3)

// 4. Non-streaming request
resp, usage, err := call.Request(ctx, pool, rawReq, routing.ProtocolOpenAI)

// 5. Streaming request (auto protocol conversion)
result, err := call.RequestStream(ctx, pool, rawReq, routing.ProtocolAnthropic)
if err != nil { return err }
defer result.Close()
for chunk := range result.Ch { c.Write(chunk) }
```

---

## Protocol Conversion

```go
// Frontend: Anthropic SDK format
// Backend: Gemini provider (cheaper)
// fluxcore: auto-converts

anthropicReq := `{"model": "claude-3", "messages": [...]}`
resp, _, _ := call.Request(ctx, pool, anthropicReq, routing.ProtocolAnthropic)
// Output is Anthropic format, even if pool chose Gemini endpoint
```

---

## Price-First Routing

```go
// Create endpoints with priority (lower = preferred)
ep1, _ := routing.NewEndpoint(1, key1, "", 1000)   // OpenAI: priority 1000
ep2, _ := routing.NewEndpoint(2, key2, "", 100)    // Gemini: priority 100 (cheaper)

pool := routing.NewEndpointPool([]*routing.Endpoint{ep1, ep2}, 3)

// Auto-select lowest priority endpoint
// Gemini fails? Auto-switch to OpenAI
```

---

## Circuit Breaker

```
Healthy → Fail → Fail → Fail → 🔴 Circuit Open
                              ↓
                        60s auto-recovery probe
```

3 failures trigger circuit, auto-switch to other endpoints. 60s auto-recovery.

### Default Config

```go
// Default: 3 failures trigger circuit, 60s recovery timeout
ep, _ := routing.NewEndpoint(id, key, model, priority)

// Check circuit breaker status
if ep.IsCircuitBreakerOpen() {
    // Endpoint unhealthy, skip
}
```

### API Reference

| Method | Description |
|--------|-------------|
| `IsCircuitBreakerOpen()` | Returns true if circuit breaker is open (should skip) |
| `MarkSuccess()` | Mark endpoint as healthy, reset failure count |
| `MarkFail()` | Mark endpoint as failed, increment failure count |

---

## EWMA Latency Tracking

Fluxcore uses EWMA (Exponentially Weighted Moving Average) to track endpoint latency:

```go
// After each request, latency is updated
ep.UpdateLatency(200) // 200ms latency

// Get smoothed latency estimate
latency := ep.LatencyEWMA() // Returns EWMA value in ms
```

**EWMA formula:** `new = 0.1 × current + 0.9 × old`

This provides smooth latency estimates that adapt gradually to changes, avoiding sensitivity to outliers.

---

## Error Classification

| Error Type | Handling |
|------------|----------|
| Network timeout | Auto retry |
| Service unavailable (503) | Auto retry |
| Auth failed (401) | No retry, return error |
| Rate limited (429) | Wait and retry |

---

## Usage Statistics

```go
resp, usage, err := call.Request(ctx, pool, rawReq, routing.ProtocolOpenAI)
if usage != nil {
    fmt.Printf("Input tokens: %d\n", usage.InputTokens)
    fmt.Printf("Output tokens: %d\n", usage.OutputTokens)
    fmt.Printf("Latency: %dms\n", usage.LatencyMs)
    fmt.Printf("Total tokens: %d\n", usage.TotalTokens())

    // Check if usage is accurate (provider reported, not estimated)
    if usage.IsAccurate {
        fmt.Println("Usage is accurate (provider reported)")
    }
}
```

### Usage Fields

| Field | Type | Description |
|-------|------|-------------|
| `InputTokens` | `int` | Input/prompt tokens |
| `OutputTokens` | `int` | Output/completion tokens |
| `LatencyMs` | `int` | Request latency in milliseconds |
| `IsAccurate` | `bool` | True if provider reported accurate usage (not estimated) |

---

## Architecture

```
┌─────────────────────────────────────────────┐
│                 fluxcore                     │
│        LLM API Router Library               │
├─────────────┬───────────────────────────────┤
│  message/   │          routing/             │
│  Types      │     Selection + Circuit Breaker│
│  (IR Layer) │          (Lock-free)          │
├─────────────┴───────────────────────────────┤
│                   call/                      │
│      HTTP Transport + Retry + Streaming     │
│            + Protocol Conversion             │
└─────────────────────────────────────────────┘

4 packages. 17 files. 0 interfaces. 0 dependencies.
```

**Package naming with business semantics:**

| Package | Purpose | Files |
|---------|---------|-------|
| `message` | LLM message data structures | 6 |
| `routing` | Endpoint routing + selection + circuit breaker | 6 |
| `call` | HTTP transport + retry + streaming | 5 |
| `errors` | Error classification + retryability | 2 |

---

## Performance

| Operation | Time | Note |
|-----------|------|------|
| `CurrentEp()` | ~10ns | Lock-free atomic read |
| `MarkFail()` | ~50ns | CAS + O(1) map |
| `SelectBest()` | ~100ns | Priority scan with circuit breaker check |
| Concurrent test | 1000 QPS | No deadlock |

---

## Security (SSRF Protection)

fluxcore validates endpoint format. SSRF protection is application-layer responsibility.

```go
ep := &routing.Endpoint{
    Key: &routing.Key{
        BaseURL:  userProvidedURL,  // User input
        APIKey:   "your-key",
        Protocol: routing.ProtocolOpenAI,
    },
}

// Step 1: Validate format (scheme, APIKey, Protocol)
if err := ep.Validate(); err != nil {
    return err
}

// Step 2: SSRF protection (your policy)
// Implement IP validation in your application layer
```

---

## Who Uses Fluxcore

| User | Use Case |
|------|----------|
| **SaaS Teams** | Multi-tenant LLM features with endpoint isolation |
| **AI Startups** | Cost-optimized routing (cheapest provider first) |
| **Platform Teams** | Unified LLM API for internal services |
| **Indie Developers** | Prototype to production in hours, not weeks |

---

## Protocol Support

| Format | Constant | Endpoint |
|--------|----------|----------|
| **OpenAI** | `ProtocolOpenAI` | `/v1/chat/completions` |
| **Anthropic** | `ProtocolAnthropic` | `/v1/messages` |
| **Gemini** | `ProtocolGemini` | `/v1/models/{model}:generateContent` |
| **Cohere** | `ProtocolCohere` | `/v1/chat` |

### OpenAI-Compatible Providers

| Provider | Base URL |
|----------|----------|
| **Azure OpenAI** | Your Azure endpoint |
| **Mistral AI** | `https://api.mistral.ai` |
| **Groq** | `https://api.groq.com` |
| **DeepSeek** | `https://api.deepseek.com` |
| **Zhipu GLM-4** | `https://open.bigmodel.cn/api/paas/v4/` |

---

## Integration Example

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

## Get Started

```bash
go get github.com/tokflux/fluxcore@v0.5.0
```

**Next steps:**
1. Try the Quick Start above
2. See [Integration Example](#integration-example)
3. ⭐ Star if helpful

---

## 中文说明

### fluxcore ⚡ v0.5.0

**LLM API 路由库**

30行代码，LLM API 路由搞定。

---

### v0.5.0 升级亮点

**重大改进：**

- **🧹 更简洁 API** — 删除未使用导出，内部常量私有化。更小接口，更易使用。
- **🛡️实战测试** — 新增稳定性测试：熔断器恢复流程、EWMA 延迟追踪、网络韧性。
- **⚡ 零依赖** — 17个文件，0接口，0外部包。纯 Go 实现。
- **🔒 安全优先** — SSRF防护移至应用层（你的策略，你的控制）。
- **📊 90%+ 测试覆盖** — Routing 94.2%，Call 90.8%，覆盖全面边界情况。

---

### 特性

- **零抽象** — 17个文件，无接口层。读代码，懂流程。
- **价格优先路由** — 自动选择最便宜可用端点。
- **熔断器 + 重试** — 3次失败触发熔断，60秒自动恢复。
- **协议转换** — Anthropic 输入，Gemini 输出，透明翻译。
- **EWMA 延迟追踪** — 平滑延迟估计，自适应超时。

---

### 4行代码起步

```go
pool := routing.NewEndpointPool(endpoints, 3)
resp, usage, err := call.Request(ctx, pool, rawReq, routing.ProtocolOpenAI)
// 完成。
```

---

### 快速开始

```go
import (
    "github.com/tokflux/fluxcore/routing"
    "github.com/tokflux/fluxcore/call"
)

// 1. 定义密钥（连接凭证）
keys := []*routing.Key{
    {BaseURL: "https://api.openai.com", APIKey: key1, Protocol: routing.ProtocolOpenAI},
    {BaseURL: "https://api.anthropic.com", APIKey: key2, Protocol: routing.ProtocolAnthropic},
}

// 2. 创建端点（密钥 + 模型 + 优先级）
// 注意：Model 参数对 Gemini 必填（用于 URL），OpenAI/Anthropic/Cohere 用空字符串 ""
// Priority: 值越小优先级越高（如价格，单位为微）
ep1, _ := routing.NewEndpoint(1, keys[0], "", 1000)   // OpenAI，优先级 1000
ep2, _ := routing.NewEndpoint(2, keys[1], "", 800)    // Anthropic，优先级 800（更便宜）

endpoints := []*routing.Endpoint{ep1, ep2}

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

### 协议透明转换

```go
// 前端: Anthropic SDK 格式
// 后端: Gemini provider（更便宜）
// fluxcore: 自动转换

anthropicReq := `{"model": "claude-3", "messages": [...]}`
resp, _, _ := call.Request(ctx, pool, anthropicReq, routing.ProtocolAnthropic)
// 输出是 Anthropic 格式，即使 pool 选择了 Gemini endpoint
```

---

### 成本优先路由

```go
// 创建端点，标注优先级（值越小越好）
ep1, _ := routing.NewEndpoint(1, key1, "", 1000)   // OpenAI: 优先级 1000
ep2, _ := routing.NewEndpoint(2, key2, "", 100)    // Gemini: 优先级 100（更便宜）

pool := routing.NewEndpointPool([]*routing.Endpoint{ep1, ep2}, 3)

// 自动选择最低优先级端点
// Gemini 失败？自动切换到 OpenAI
```

---

### 熔断器自愈

```
健康 → 失败 → 失败 → 失败 → 🔴 熔断
                              ↓
                        60秒自动恢复探测
```

3次失败触发熔断，自动切换到其他端点。60秒后自动恢复探测。

**默认配置：**
```go
// 默认：3次失败触发熔断，60秒恢复超时
ep, _ := routing.NewEndpoint(id, key, model, priority)

// 检查熔断器状态
if ep.IsCircuitBreakerOpen() {
    // 端点不健康，跳过
}
```

---

### EWMA 延迟追踪

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

### 智能错误分类

| 错误类型 | 处理方式 |
|----------|----------|
| 网络超时 | 自动重试 |
| 服务不可用 (503) | 自动重试 |
| 认证失败 (401) | 不重试，直接报错 |
| 配额超限 (429) | 等待后重试 |

---

### 使用统计

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

---

### 架构

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

---

### 性能

| 操作 | 时间 | 说明 |
|------|------|------|
| `CurrentEp()` | ~10ns | 无锁原子读取 |
| `MarkFail()` | ~50ns | CAS + O(1) 映射 |
| `SelectBest()` | ~100ns | 优先级扫描 + 熔断检查 |
| 并发测试 | 1000 QPS | 无死锁 |

---

### 安全（SSRF 防护）

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

### 谁在使用

| 用户 | 用途 |
|------|------|
| **SaaS 团队** | 多租户 AI 功能，端点隔离 |
| **AI 创业** | 成本控制，自动选最便宜 |
| **平台团队** | 统一 LLM API，运维友好 |
| **独立开发者** | 快速上线，零学习成本 |

---

### 协议支持

| 格式 | 常量 | 端点 |
|------|------|------|
| **OpenAI** | `ProtocolOpenAI` | `/v1/chat/completions` |
| **Anthropic** | `ProtocolAnthropic` | `/v1/messages` |
| **Gemini** | `ProtocolGemini` | `/v1/models/{model}:generateContent` |
| **Cohere** | `ProtocolCohere` | `/v1/chat` |

**OpenAI 兼容提供商：**

| 提供商 | Base URL |
|--------|----------|
| **Azure OpenAI** | 你的 Azure 端点 |
| **Mistral AI** | `https://api.mistral.ai` |
| **Groq** | `https://api.groq.com` |
| **DeepSeek** | `https://api.deepseek.com` |
| **智谱 GLM-4** | `https://open.bigmodel.cn/api/paas/v4/` |

---

### 开始使用

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