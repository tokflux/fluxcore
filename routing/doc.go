// Package routing provides endpoint management, selection, and health tracking.
//
// The routing package handles:
//   - Endpoint definition (Key + Model + priority attribute)
//   - Endpoint selection with priority-based optimization
//   - Circuit breaker pattern for health management
//   - Thread-safe endpoint pool with atomic updates
//
// The Priority field is a generic sorting value - lower values are preferred.
// The semantic meaning (price, latency, custom combination) is defined by the caller.
//
// Core types:
//   - Key: Connection credentials (BaseURL, APIKey, Protocol)
//   - Endpoint: Routing unit with health state
//   - EndpointPool: Collection of endpoints with selection logic
//   - Protocol: LLM provider protocol (OpenAI, Anthropic, Gemini, Cohere)
//
// Example usage:
//
//	key := &routing.Key{
//	    BaseURL:  "https://api.openai.com/v1",
//	    APIKey:   "sk-xxx",
//	    Protocol: routing.ProtocolOpenAI,
//	}
//	ep := routing.NewEndpoint(1, key, "", 100) // priority=100 (lower is better)
//	pool := routing.NewEndpointPool([]*routing.Endpoint{ep}, 3)
//	selected := pool.Select()
//
// Health management:
//
//	ep.MarkSuccess()  // Reset failure count
//	ep.MarkFail()     // Increment failure count
//	if ep.IsCircuitBreakerOpen() { /* skip endpoint */ }
package routing