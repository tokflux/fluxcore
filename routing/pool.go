package routing

import (
	"sync"
	"sync/atomic"
)

type EndpointPool struct {
	currentEp atomic.Pointer[Endpoint] // Cached best endpoint (lock-free read)
	mu         sync.RWMutex
	endpoints []*Endpoint // Sequential access (routing)
	retryMax  int
}

func NewEndpointPool(endpoints []*Endpoint, retryMax int) *EndpointPool {
	if retryMax <= 0 {
		retryMax = 2
	}

	pool := &EndpointPool{
		endpoints: endpoints,
		retryMax:  retryMax,
	}

	// Set initial currentEp to best endpoint
	if best := pool.SelectBest(); best != nil {
		pool.currentEp.Store(best)
	}

	return pool
}

// CurrentEp returns the cached best endpoint.
func (p *EndpointPool) CurrentEp() *Endpoint {
	return p.currentEp.Load()
}

// RetryMax returns the maximum retry count.
func (p *EndpointPool) RetryMax() int {
	return p.retryMax
}

// MarkFail marks an endpoint as failed and switches to another.
func (p *EndpointPool) MarkFail(ep *Endpoint) {
	ep.MarkFail()

	// Use CAS loop to ensure atomic switching
	for {
		current := p.currentEp.Load()
		if current != ep {
			break
		}

		newEp := p.selectBestExcluding(ep)
		if newEp == nil {
			break
		}

		if p.currentEp.CompareAndSwap(current, newEp) {
			break
		}
	}
}

// selectBestExcluding selects best endpoint, excluding the specified one.
// Zero-allocation: iterates directly without creating a filtered slice.
func (p *EndpointPool) selectBestExcluding(exclude *Endpoint) *Endpoint {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var best *Endpoint
	for _, ep := range p.endpoints {
		if ep == exclude || ep.IsCircuitBreakerOpen() {
			continue
		}
		if best == nil || compareEndpoint(ep, best) < 0 {
			best = ep
		}
	}
	return best
}

// SelectBest selects the best endpoint (skipping unhealthy ones).
// Zero-allocation: iterates directly without creating a filtered slice.
func (p *EndpointPool) SelectBest() *Endpoint {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var best *Endpoint
	for _, ep := range p.endpoints {
		if ep.IsCircuitBreakerOpen() {
			continue
		}
		if best == nil || compareEndpoint(ep, best) < 0 {
			best = ep
		}
	}
	return best
}