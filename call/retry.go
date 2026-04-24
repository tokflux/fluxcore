package call

import (
	"math/rand"
	"sync"
	"time"
)

// retryConfig holds retry configuration (internal).
type retryConfig struct {
	BaseBackoff time.Duration // Base backoff duration (default: 100ms)
	MaxBackoff  time.Duration // Maximum backoff duration (default: 5s)
}

var defaultRetryConfig = retryConfig{
	BaseBackoff: 100 * time.Millisecond,
	MaxBackoff:  5 * time.Second,
}

var retryConfigMu sync.RWMutex

// setRetryConfig updates the retry configuration (for tests).
func setRetryConfig(cfg *retryConfig) {
	if cfg == nil {
		return
	}
	retryConfigMu.Lock()
	defer retryConfigMu.Unlock()
	if cfg.BaseBackoff > 0 {
		defaultRetryConfig.BaseBackoff = cfg.BaseBackoff
	}
	if cfg.MaxBackoff > 0 {
		defaultRetryConfig.MaxBackoff = cfg.MaxBackoff
	}
}

// getRetryConfig returns the current retry configuration (for tests).
func getRetryConfig() retryConfig {
	retryConfigMu.RLock()
	defer retryConfigMu.RUnlock()
	return defaultRetryConfig
}

// backoffWithJitter calculates exponential backoff with full jitter.
// Go 1.20+ rand.Float64 is concurrent-safe.
func backoffWithJitter(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	retryConfigMu.RLock()
	base := defaultRetryConfig.BaseBackoff
	maxBackoff := defaultRetryConfig.MaxBackoff
	retryConfigMu.RUnlock()

	// Exponential backoff: base * 2^(attempt-1)
	backoff := base << uint(attempt-1)
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	// Full jitter: random value between 0 and backoff
	// Avoids thundering herd problem when multiple clients retry simultaneously
	return time.Duration(rand.Float64() * float64(backoff))
}