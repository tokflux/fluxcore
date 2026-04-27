# fluxcore ⚡

**LLM API 客户端库**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v0.9.0-blue?style=flat)]()
[![English](https://img.shields.io/badge/README-English-blue?style=flat)](README.md)

简洁的 LLM API 客户端，带路由和健康管理。

---

## 快速开始

```go
import (
    "github.com/tokzone/fluxcore/endpoint"
    "github.com/tokzone/fluxcore/flux"
    "github.com/tokzone/fluxcore/provider"
)

// 1. 定义 Provider（仅 BaseURL，不绑协议）
openai := provider.NewProvider(1, "https://api.openai.com")
anthropic := provider.NewProvider(2, "https://api.anthropic.com")

// 2. 注册 Endpoint 到全局 Registry（带协议能力列表）
endpoint.RegisterEndpoint(1, openai, "", []provider.Protocol{provider.ProtocolOpenAI})
endpoint.RegisterEndpoint(2, anthropic, "", []provider.Protocol{provider.ProtocolAnthropic})

// 3. 创建 APIKey（Provider + Secret）
key1, _ := flux.NewAPIKey(openai, "sk-xxx")
key2, _ := flux.NewAPIKey(anthropic, "sk-ant-xxx")

// 4. 创建 UserEndpoint（Endpoint + APIKey + Priority）
ue1, _ := flux.NewUserEndpoint("", key1, 1000)
ue2, _ := flux.NewUserEndpoint("", key2, 800)

// 5. 创建 Client
client := flux.NewClient([]*flux.UserEndpoint{ue1, ue2}, flux.WithRetryMax(3))

// 6. 生成预编译 DoFunc（输入端协议闭包固化，热路径零开销）
doFunc := flux.DoFuncGen(client, provider.ProtocolOpenAI)

// 7. 发送请求
resp, usage, providerURL, err := doFunc(ctx, rawReq)

// 8. 流式请求
streamDoFunc := flux.StreamDoFuncGen(client, provider.ProtocolAnthropic)
result, model, providerURL, err := streamDoFunc(ctx, rawReq)
defer result.Close()
for chunk := range result.Ch {
    // 处理 chunk
}
```

---

## 核心概念

### Provider（协议无关）

Provider 仅由 BaseURL 标识。协议**不是** Provider 的属性 — 像 DeepSeek 或 OpenRouter 这样的 Provider 同时支持多种协议（OpenAI + Anthropic）。

```go
prov := provider.NewProvider(id, "https://api.deepseek.com")
```

### Endpoint = (Provider, Model) + 协议能力列表

Endpoint 通过 `Protocols []Protocol` 声明所支持的协议。协议是**能力**而非身份。

```go
ep, _ := endpoint.NewEndpoint(1, prov, "deepseek-chat",
    []provider.Protocol{provider.ProtocolOpenAI, provider.ProtocolAnthropic})

// 检测能力
ep.HasProtocol(provider.ProtocolAnthropic) // true

// 最佳匹配（命中则直传，未命中则回退到 Protocols[0]）
target := ep.SelectProtocol(provider.ProtocolAnthropic) // ProtocolAnthropic
target := ep.SelectProtocol(provider.ProtocolGemini)    // ProtocolOpenAI（回退）
```

### DoFunc / DoFuncGen — Prepare/Do 分离

`DoFuncGen(client, inputProtocol)` 预计算每个 endpoint 的目标协议映射，返回 `DoFunc` 闭包。热路径**零协议判断开销**。

```go
// 启动/reload 时生成一次 — 输入端协议固化在闭包中
openAIDoFunc := flux.DoFuncGen(client, provider.ProtocolOpenAI)

// 热路径 — 无需传 protocol 参数
resp, usage, providerURL, err := openAIDoFunc(ctx, body)
```

`Client.Do()` 和 `Client.DoStream()` 已**废弃** — 推荐使用 `DoFuncGen` / `StreamDoFuncGen`。

---

## 特性

- **简洁 API** — Provider、Endpoint、APIKey、UserEndpoint、Client。五个概念。
- **协议无关 Provider** — Provider 仅为 BaseURL；协议由 Endpoint 以能力列表声明。
- **多租户** — 共享健康状态（Provider/Endpoint），私有密钥（APIKey）和优先级（UserEndpoint）。
- **双层健康** — Provider（网络）+ Endpoint（模型）熔断器。
- **协议转换** — Anthropic 入，Gemini 出。通过 `SelectProtocol` 回退实现透明协议转换。
- **Prepare/Do 分离** — `DoFuncGen` 在生成时固化输入端协议；热路径零协议开销。
- **自定义 HTTP Client** — 注入自定义 Client 调整连接池参数。

---

## Options 配置

```go
// 重试配置
flux.WithRetryMax(5)  // 最大重试次数（默认：3）

// 自定义 HTTP Client
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

## 模块架构

```
flux（用户入口）
  │
  └── DoFuncGen(client, inputProtocol) → DoFunc
      StreamDoFuncGen(client, inputProtocol) → StreamDoFunc
  │
flux（用户数据）
  │
  ├── APIKey: Provider + Secret（用户私有）
  └── UserEndpoint: Endpoint + APIKey + Priority（用户私有）
  │
endpoint（全局状态）
  │
  └── Endpoint: Provider + Model + Protocols[] + Health（全局单例）
  │
provider（全局状态）
  │
  └── Provider: BaseURL + Health（全局单例，协议无关）
```

---

## 双层熔断器

```
Provider 层（网络）：
  连接拒绝 → 立即熔断（阈值=1）
  恢复：120s

Endpoint 层（模型）：
  429 Rate Limit → 熔断（阈值=1）
  500 Server Error → 熔断（阈值=3）
  恢复：60s
```

---

## 协议选择

当请求以输入协议 X 到达时：
1. `SelectProtocol(X)` 检查 endpoint 的 `Protocols` 列表是否包含 X
2. 命中 → 直传（无转换）
3. 未命中 → 回退到 `Protocols[0]`（应用协议转换）

这使得 DeepSeek（原生 OpenAI）等 Provider 可以通过协议转换为 Anthropic 格式请求服务。

---

## 许可证

MIT
