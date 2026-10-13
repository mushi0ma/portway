package modelgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"portway/internal/execution"
	"portway/internal/problem"
	"portway/internal/secure"
)

type CountRequest struct {
	RunID string `json:"run_id"`
}
type CountReceipt struct {
	RunID           string `json:"run_id"`
	Input           int    `json:"input_tokens"`
	RequestHash     string `json:"request_hash"`
	ProviderVersion int    `json:"provider_version"`
	Epoch           int64  `json:"epoch"`
}
type CompleteRequest struct {
	CountReceipt
	StepID string `json:"step_id"`
}

// RPC keeps owner credentials only in memory. Execution never receives the
// provider key, endpoint or control SQL credentials. There are no HTTP retries.
type RPC struct {
	base, key, bearer, proof string
	http                     *http.Client
	mu                       sync.Mutex
	receipts                 map[string]CountReceipt
}

var _ execution.Gateway = (*RPC)(nil)

func NewRPC(base, key, bearer, proof string, httpClient *http.Client) (*RPC, error) {
	if base != "http://api:8080" || len(key) < 32 || !strings.HasPrefix(bearer, "Bearer ") || proof == "" {
		return nil, problem.Invalid("Invalid private gateway authority")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 125 * time.Second, Transport: &http.Transport{Proxy: nil, ResponseHeaderTimeout: 122 * time.Second}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &RPC{base: base, key: key, bearer: bearer, proof: proof, http: httpClient, receipts: map[string]CountReceipt{}}, nil
}
func (c *RPC) Close() { c.http.CloseIdleConnections(); c.mu.Lock(); clear(c.receipts); c.mu.Unlock() }
func (c *RPC) call(ctx context.Context, path string, in, out any) error {
	raw, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, "POST", c.base+"/internal/v1/model-gateway/"+path, io.NopCloser(bytes.NewReader(raw)))
	if err != nil {
		return failure("GATEWAY_UNAVAILABLE")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.bearer)
	req.Header.Set("X-Device-Session", c.proof)
	req.Header.Set("X-Portway-Service-Key", c.key)
	res, err := c.http.Do(req)
	if err != nil {
		return failure("GATEWAY_UNAVAILABLE")
	}
	defer res.Body.Close()
	raw, err = io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return failure("GATEWAY_PROTOCOL_ERROR")
	}
	var envelope struct {
		Data  json.RawMessage `json:"data"`
		Error *problem.Error  `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return failure("GATEWAY_PROTOCOL_ERROR")
	}
	if res.StatusCode != 200 {
		if envelope.Error != nil {
			envelope.Error.Status = res.StatusCode
			return envelope.Error
		}
		return failure("GATEWAY_UNAVAILABLE")
	}
	if json.Unmarshal(envelope.Data, out) != nil {
		return failure("GATEWAY_PROTOCOL_ERROR")
	}
	return nil
}
func (c *RPC) InputTokens(ctx context.Context, r execution.Run) (int, error) {
	var receipt CountReceipt
	if err := c.call(ctx, "count", CountRequest{RunID: r.ID}, &receipt); err != nil {
		return 0, err
	}
	if receipt.RunID != r.ID || receipt.Epoch != r.StopEpoch || receipt.ProviderVersion != r.Snapshot.ProviderVersion || receipt.ProviderVersion < 1 || len(receipt.RequestHash) != 64 || receipt.Input < 0 || receipt.Input > 40000 {
		return 0, failure("GATEWAY_PROTOCOL_ERROR")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.receipts) >= 64 {
		if _, ok := c.receipts[r.ID]; !ok {
			return 0, failure("GATEWAY_CAPACITY_LIMIT")
		}
	}
	c.receipts[r.ID] = receipt
	return receipt.Input, nil
}
func (c *RPC) Complete(ctx context.Context, r execution.Run, step execution.Step) (execution.ModelReply, error) {
	c.mu.Lock()
	receipt, ok := c.receipts[r.ID]
	c.mu.Unlock()
	if !ok || !secure.UUID(step.ID) || step.Settled || step.Epoch != receipt.Epoch || step.Input != receipt.Input {
		return execution.ModelReply{}, problem.Invalid("Missing matching model count receipt")
	}
	var reply execution.ModelReply
	err := c.call(ctx, "complete", CompleteRequest{CountReceipt: receipt, StepID: step.ID}, &reply)
	return reply, err
}
