package routing

import (
	"testing"
	"time"
)

func TestSelectBest(t *testing.T) {
	t.Run("selects_lowest_priority", func(t *testing.T) {
		endpoints := []*Endpoint{
			{ID: 1, Key: &Key{Protocol: ProtocolOpenAI}, Priority: 50, state: &endpointState{}},
			{ID: 2, Key: &Key{Protocol: ProtocolOpenAI}, Priority: 20, state: &endpointState{}}, // Lowest priority
			{ID: 3, Key: &Key{Protocol: ProtocolOpenAI}, Priority: 100, state: &endpointState{}},
		}
		for _, ep := range endpoints {
			ep.setHealthy(true)
		}

		selected := selectBest(endpoints)
		if selected.ID != 2 {
			t.Errorf("expected lowest priority endpoint ID 2, got %d", selected.ID)
		}
	})

	t.Run("same_priority_selects_lower_latency", func(t *testing.T) {
		endpoints := []*Endpoint{
			{ID: 1, Key: &Key{Protocol: ProtocolOpenAI}, Priority: 20, LatencyMs: 100, state: &endpointState{}},
			{ID: 2, Key: &Key{Protocol: ProtocolOpenAI}, Priority: 20, LatencyMs: 50, state: &endpointState{}}, // Lower latency
		}
		for _, ep := range endpoints {
			ep.setHealthy(true)
		}

		selected := selectBest(endpoints)
		if selected.ID != 2 {
			t.Errorf("expected lower latency endpoint ID 2, got %d", selected.ID)
		}
	})

	t.Run("empty_endpoints", func(t *testing.T) {
		selected := selectBest([]*Endpoint{})
		if selected != nil {
			t.Errorf("expected nil for empty endpoints, got %v", selected)
		}
	})
}

func TestPoolSelectBestMethod(t *testing.T) {
	t.Run("skips_unhealthy", func(t *testing.T) {
		endpoints := []*Endpoint{
			{ID: 1, Key: &Key{Protocol: ProtocolOpenAI}, Priority: 10, state: &endpointState{}},
			{ID: 2, Key: &Key{Protocol: ProtocolOpenAI}, Priority: 20, state: &endpointState{}},
		}
		endpoints[0].setHealthy(false)
		endpoints[0].setLastFail(time.Now()) // Recent failure - should skip
		endpoints[1].setHealthy(true)

		pool := NewEndpointPool(endpoints, 2)
		selected := pool.SelectBest()
		if selected.ID != 2 {
			t.Errorf("expected healthy endpoint ID 2, got %d", selected.ID)
		}
	})

	t.Run("no_healthy_endpoints", func(t *testing.T) {
		endpoints := []*Endpoint{
			{ID: 1, Key: &Key{Protocol: ProtocolOpenAI}, state: &endpointState{}},
		}
		endpoints[0].setHealthy(false)

		pool := NewEndpointPool(endpoints, 2)
		selected := pool.SelectBest()
		if selected != nil {
			t.Errorf("expected nil for no healthy endpoints, got %v", selected)
		}
	})
}