package flux

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/tokzone/fluxcore/errors"
	"github.com/tokzone/fluxcore/internal/translate"
	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/provider"
)

// Do sends a chat completion request with automatic retry and failover.
// Deprecated: Use DoFuncGen to generate a pre-prepared DoFunc instead.
func (c *Client) Do(ctx context.Context, rawReq []byte, inputProtocol provider.Protocol) ([]byte, *message.Usage, error) {
	resp, usage, _, err := DoFuncGen(c, inputProtocol)(ctx, rawReq)
	return resp, usage, err
}

func doWithParsedRequest(ctx context.Context, ue *UserEndpoint, req *message.MessageRequest, targetProtocol provider.Protocol, inputProtocol provider.Protocol, httpClient *http.Client) ([]byte, *message.Usage, error) {
	reqBody, err := translateRequest(req, targetProtocol)
	if err != nil {
		return nil, nil, fmt.Errorf("convert request: %w", err)
	}

	respBody, err := transport(ctx, ue, reqBody, targetProtocol, httpClient)
	if err != nil {
		return nil, nil, err
	}

	resp, err := translateResponse(respBody, targetProtocol)
	if err != nil {
		return nil, nil, fmt.Errorf("convert response: %w", err)
	}

	usage := &message.Usage{
		IsAccurate:   resp.Usage != nil && resp.Usage.IsAccurate,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}

	output, err := translateOutput(resp, inputProtocol)
	if err != nil {
		return nil, nil, fmt.Errorf("convert output: %w", err)
	}

	return output, usage, nil
}

const defaultWrappedChannelBuffer = 100

// atomicUsage provides thread-safe access to usage statistics.
type atomicUsage struct {
	inputTokens  atomic.Int64
	outputTokens atomic.Int64
	latencyMs    atomic.Int64
	isAccurate   atomic.Bool
}

// Get returns a snapshot of the current usage values.
func (u *atomicUsage) Get() *message.Usage {
	return &message.Usage{
		InputTokens:  int(u.inputTokens.Load()),
		OutputTokens: int(u.outputTokens.Load()),
		LatencyMs:    int(u.latencyMs.Load()),
		IsAccurate:   u.isAccurate.Load(),
	}
}

// Set atomically updates all usage fields.
func (u *atomicUsage) Set(usage *message.Usage) {
	u.inputTokens.Store(int64(usage.InputTokens))
	u.outputTokens.Store(int64(usage.OutputTokens))
	u.latencyMs.Store(int64(usage.LatencyMs))
	u.isAccurate.Store(usage.IsAccurate)
}

// StreamResult holds the result of a streaming request.
// Thread-safe: Ch can be read concurrently, Usage() and Error() and Close() are safe to call from any goroutine.
type StreamResult struct {
	Ch     chan []byte
	Usage  func() *message.Usage
	Error  func() error // Returns the first error encountered during streaming
	cancel context.CancelFunc
}

// DoStream sends a streaming chat completion request with automatic retry and failover.
// Deprecated: Use StreamDoFuncGen to generate a pre-prepared streaming closure instead.
func (c *Client) DoStream(ctx context.Context, rawReq []byte, inputProtocol provider.Protocol) (*StreamResult, error) {
	result, _, _, err := StreamDoFuncGen(c, inputProtocol)(ctx, rawReq)
	return result, err
}

func doStreamWithParsedRequest(ctx context.Context, ue *UserEndpoint, req *message.MessageRequest, targetProtocol provider.Protocol, inputProtocol provider.Protocol, httpClient *http.Client) (*StreamResult, error) {
	start := time.Now()

	reqBody, err := translateRequest(req, targetProtocol)
	if err != nil {
		return nil, fmt.Errorf("convert request: %w", err)
	}

	respBody, cancel, err := streamTransport(ctx, ue, reqBody, targetProtocol, httpClient)
	if err != nil {
		return nil, err
	}

	ch := make(chan []byte, translate.GetSSEConfig().ChannelBuffer)
	usageData := &atomicUsage{}
	var firstError atomic.Pointer[error]

	eventCh := translate.ParseSSEStream(ctx, respBody, targetProtocol.String(), start)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[fluxcore] SSE processor panic recovered: %v", r)
			}
		}()
		defer close(ch)
		for result := range eventCh {
			if result.Error != nil {
				// Store first error only (atomic CAS)
				firstError.CompareAndSwap(nil, &result.Error)
				continue
			}

			if result.Usage != nil {
				usageData.Set(result.Usage)
			}

			if result.Event.Type == translate.SSETypeDone {
				continue
			}

			if targetProtocol != inputProtocol {
				converted := translate.ConvertSSEEvent(result.Event, targetProtocol.String(), inputProtocol.String())
				if converted != nil {
					// Non-blocking send to avoid goroutine leak
					select {
					case ch <- converted:
					case <-ctx.Done():
						return
					}
				}
			} else {
				output := translate.FormatSSEOutput(result.Event, inputProtocol.String())
				if output != nil {
					// Non-blocking send to avoid goroutine leak
					select {
					case ch <- output:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return &StreamResult{
		Ch:    ch,
		Usage: usageData.Get,
		Error: func() error {
			if p := firstError.Load(); p != nil {
				return *p
			}
			return nil
		},
		cancel: cancel,
	}, nil
}

func streamTransport(ctx context.Context, ue *UserEndpoint, body []byte, targetProtocol provider.Protocol, client *http.Client) (io.ReadCloser, context.CancelFunc, error) {
	var cancel context.CancelFunc = func() {} // no-op default

	// Ensure requests have a deadline for timeout control
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", buildURL(ue, targetProtocol, true), bytes.NewReader(body))
	if err != nil {
		cancel()
		return nil, nil, err
	}
	setHeaders(req, ue, targetProtocol, true)

	resp, err := client.Do(req)
	if err != nil {
		cancel()
		return nil, nil, errors.ClassifyNetError(err)
	}

	if resp.StatusCode >= 400 {
		cancel()
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, defaultErrorBodyLimit))
		resp.Body.Close()
		return nil, nil, errors.ClassifyHTTPError(resp.StatusCode, string(respBody))
	}

	return resp.Body, cancel, nil
}

// Close releases resources for the StreamResult.
func (s *StreamResult) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}
