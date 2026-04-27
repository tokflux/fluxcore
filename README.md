# fluxcore ⚡

**LLM API Client Library**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v0.9.0-blue?style=flat)]()
[![中文](https://img.shields.io/badge/README-中文-red?style=flat)](README_CN.md)

Simple LLM API client with routing and health management.

---

## Quick Start

```go
import (
    "github.com/tokzone/fluxcore/endpoint"
    "github.com/tokzone/fluxcore/flux"
    "github.com/tokzone/fluxcore/provider"
)

// 1. Define providers (baseURL only, protocol-agnostic)
openai := provider.NewProvider(1, "https://api.openai.com")
anthropic := provider.NewProvider(2, "https://api.anthropic.com")

// 2. Register endpoints to global registry (with protocol capabilities)
endpoint.RegisterEndpoint(1, openai, "", []provider.Protocol{provider.ProtocolOpenAI})
endpoint.RegisterEndpoint(2, anthropic, "", []provider.Protocol{provider.ProtocolAnthropic})

// 3. Create APIKeys (Provider + Secret)
key1, _ := flux.NewAPIKey(openai, "sk-xxx")
key2, _ := flux.NewAPIKey(anthropic, "sk-ant-xxx")

// 4. Create UserEndpoints (Endpoint + APIKey + Priority)
ue1, _ := flux.NewUserEndpoint("", key1, 1000)
ue2, _ := flux.NewUserEndpoint("", key2, 800)

// 5. Create client
client := flux.NewClient([]*flux.UserEndpoint{ue1, ue2}, flux.WithRetryMax(3))

// 6. Generate a pre-prepared DoFunc (input protocol baked in, zero overhead on hot path)
doFunc := flux.DoFuncGen(client, provider.ProtocolOpenAI)

// 7. Send request
resp, usage, providerURL, err := doFunc(ctx, rawReq)

// 8. Streaming
streamDoFunc := flux.StreamDoFuncGen(client, provider.ProtocolAnthropic)
result, model, providerURL, err := streamDoFunc(ctx, rawReq)
defer result.Close()
for chunk := range result.Ch {
    // process chunk
}
```

---

## Core Concepts

### Provider (protocol-agnostic)

A provider is identified solely by its BaseURL. Protocol is **not** a property of the provider — a provider like DeepSeek or OpenRouter supports multiple protocols (OpenAI + Anthropic).

```go
prov := provider.NewProvider(id, "https://api.deepseek.com")
```

### Endpoint = (Provider, Model) + Protocol Capabilities

An endpoint declares which protocols it **supports** via `Protocols []Protocol`. Protocol is a capability, not identity.

```go
ep, _ := endpoint.NewEndpoint(1, prov, "deepseek-chat",
    []provider.Protocol{provider.ProtocolOpenAI, provider.ProtocolAnthropic})

// Check capability
ep.HasProtocol(provider.ProtocolAnthropic) // true

// Select best match (returns matching protocol or fallback Protocols[0])
target := ep.SelectProtocol(provider.ProtocolAnthropic) // ProtocolAnthropic
target := ep.SelectProtocol(provider.ProtocolGemini)    // ProtocolOpenAI (fallback)
```

### DoFunc / DoFuncGen — Prepare/Do Separation

`DoFuncGen(client, inputProtocol)` pre-computes the target protocol mapping for each endpoint and returns a `DoFunc` closure. The hot path has **zero protocol decision overhead**.

```go
// Generated once at startup/reload — input protocol baked into closure
openAIDoFunc := flux.DoFuncGen(client, provider.ProtocolOpenAI)

// Hot path — no protocol parameter needed
resp, usage, providerURL, err := openAIDoFunc(ctx, body)
```

`Client.Do()` and `Client.DoStream()` are **deprecated** — prefer `DoFuncGen` / `StreamDoFuncGen`.

---

## Features

- **Simple API** — Provider, Endpoint, APIKey, UserEndpoint, Client. Five concepts.
- **Protocol-Agnostic Provider** — Provider is just a BaseURL. Protocol is declared by Endpoint as a capability list.
- **Multi-Tenant** — Shared health state (Provider/Endpoint), private secrets (APIKey) and priorities (UserEndpoint).
- **Two-Layer Health** — Provider (network) + Endpoint (model) circuit breakers.
- **Protocol Translation** — Anthropic in, Gemini out. Transparent protocol conversion via `SelectProtocol` fallback.
- **Prepare/Do Separation** — `DoFuncGen` bakes input protocol at generation time; hot path is zero-protocol.
- **Custom HTTP Client** — Inject custom client for connection pool tuning.

---

## Options

```go
// Retry configuration
flux.WithRetryMax(5)  // Max retries (default: 3)

// Custom HTTP client
flux.WithHTTPClient(&http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        200,
        MaxIdleConnsPerHost: 20,
        IdleConnTimeout:     120 * time.Second,
    },
})
```

---

## Module Architecture

```
flux (user entry)
  │
  └── DoFuncGen(client, inputProtocol) → DoFunc
      StreamDoFuncGen(client, inputProtocol) → StreamDoFunc
  │
flux (user data)
  │
  ├── APIKey: Provider + Secret (user private)
  └── UserEndpoint: Endpoint + APIKey + Priority (user private)
  │
endpoint (global state)
  │
  └── Endpoint: Provider + Model + Protocols[] + Health (global singleton)
  │
provider (global state)
  │
  └── Provider: BaseURL + Health (global singleton, protocol-agnostic)
```

---

## Two-Layer Circuit Breaker

```
Provider Layer (Network):
  Connection refused → Immediate circuit (threshold=1)
  Recovery: 120s

Endpoint Layer (Model):
  429 Rate Limit → Circuit (threshold=1)
  500 Server Error → Circuit (threshold=3)
  Recovery: 60s
```

---

## Protocol Selection

When a request arrives with input protocol X:
1. `SelectProtocol(X)` checks if the endpoint has X in its `Protocols` list
2. If yes → direct pass-through (no conversion)
3. If no → fallback to `Protocols[0]` (translation applied)

This enables providers like DeepSeek (native OpenAI) to serve Anthropic-formatted requests via protocol translation.

---

## License

MIT
