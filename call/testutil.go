package call

import (
	"github.com/tokzone/fluxcore/routing"
)

// testEndpoint creates an endpoint for testing
func testEndpoint(id uint, baseURL, apiKey string, protocol routing.Protocol) *routing.Endpoint {
	key := &routing.Key{BaseURL: baseURL, APIKey: apiKey, Protocol: protocol}
	ep, _ := routing.NewEndpoint(id, key, "", 0)
	return ep
}

// testEndpointWithPriority creates an endpoint with priority for testing
func testEndpointWithPriority(id uint, baseURL, apiKey string, protocol routing.Protocol, priority int64) *routing.Endpoint {
	key := &routing.Key{BaseURL: baseURL, APIKey: apiKey, Protocol: protocol}
	ep, _ := routing.NewEndpoint(id, key, "", priority)
	return ep
}