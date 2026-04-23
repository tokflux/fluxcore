package call

import (
	"math/rand"
	"sync"
	"time"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	BaseBackoff time.Duration // Base backoff duration (default: 100ms)
	MaxBackoff  time.Duration // Maximum backoff duration (default: 5s)
}

var retryConfig = RetryConfig{
	BaseBackoff: 100 * time.Millisecond,
	MaxBackoff:  5 * time.Second,
}

var retryConfigMu sync.RWMutex

// SetRetryConfig updates the retry configuration.
// Zero values are ignored (keep current defaults).
func SetRetryConfig(cfg *RetryConfig) {
	if cfg == nil {
		return
	}
	retryConfigMu.Lock()
	defer retryConfigMu.Unlock()
	if cfg.BaseBackoff > 0 {
		retryConfig.BaseBackoff = cfg.BaseBackoff
	}
	if cfg.MaxBackoff > 0 {
		retryConfig.MaxBackoff = cfg.MaxBackoff
	}
}

// GetRetryConfig returns the current retry configuration.
func GetRetryConfig() RetryConfig {
	retryConfigMu.RLock()
	defer retryConfigMu.RUnlock()
	return retryConfig
}

// backoffWithJitter calculates exponential backoff with full jitter.
// Go 1.20+ rand.Float64 is concurrent-safe.
func backoffWithJitter(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	retryConfigMu.RLock()
	base := retryConfig.BaseBackoff
	maxBackoff := retryConfig.MaxBackoff
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