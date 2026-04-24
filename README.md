# fluxcore ⚡

**LLM API Router Library**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v0.5.0-blue?style=flat)]()
[![中文](https://img.shields.io/badge/README-中文-red?style=flat)](README_CN.md)

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

## License

MIT. Free forever.

---

**fluxcore v0.5.0 - LLM API Router Library. Route LLM APIs in 30 lines.**

[中文文档](README_CN.md)