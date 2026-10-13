// Package modelgateway implements the bounded Chat Completions wire adapter.
// It probes wire compatibility, without authorizing runs or owning stored secrets.
package modelgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"portway/internal/execution"
	"portway/internal/openrouter"
	"portway/internal/problem"
	"portway/internal/secure"
)

// TokenCounter must use a verified tokenizer/chat template for this exact model,
// or a proven conservative upper bound including framing and tool definitions.
// No default heuristic is supplied. Body contains no credential values. Count
// must not invoke generation. Actual provider usage is still checked afterwards.
type TokenCounter func(ctx context.Context, model string, body []byte) (int, error)
type Config struct {
	ProviderID       string
	Endpoint         string // Exact allowlisted base URL, normally ending in /v1.
	Model            string
	APIKey           string
	AllowedEndpoints []string
	Count            TokenCounter
	FreeOnly         bool
}
type Client struct {
	provider, endpoint, model, key string
	count                          TokenCounter
	http                           *http.Client
	freeOnly                       bool
	checkFree                      func(context.Context, string) error
	closeCatalog                   func()
}

var _ execution.Gateway = (*Client)(nil)

func failure(code string) error {
	return problem.New(502, code, "Model gateway request could not be completed")
}
func New(cfg Config) (*Client, error) {
	base := strings.TrimRight(cfg.Endpoint, "/")
	u, err := url.Parse(base)
	allowed := false
	for _, entry := range cfg.AllowedEndpoints {
		allowed = allowed || base == strings.TrimRight(entry, "/")
	}
	if err != nil || !allowed || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") || !secure.UUID(cfg.ProviderID) || strings.TrimSpace(cfg.Model) == "" || len(cfg.Model) > 256 || strings.IndexFunc(cfg.Model, unicode.IsControl) >= 0 || len(cfg.APIKey) > 8192 || strings.IndexFunc(cfg.APIKey, unicode.IsControl) >= 0 {
		return nil, problem.Invalid("Invalid allowlisted model connection")
	}
	if cfg.FreeOnly && (base != openrouter.Endpoint || !openrouter.FreeID(cfg.Model)) {
		return nil, problem.New(403, "FREE_MODEL_REQUIRED", "Only explicit OpenRouter :free models are enabled")
	}
	// Local/private HTTP requires an explicit trusted operator allowlist entry.
	// No environment proxy, redirect or retry may redirect prompts/credentials.
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 15 * time.Second, MaxIdleConns: 4, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second, MaxResponseHeaderBytes: 16384}
	client := &Client{provider: cfg.ProviderID, endpoint: base + "/chat/completions", model: cfg.Model, key: cfg.APIKey, count: cfg.Count, http: &http.Client{Transport: transport, Timeout: 120 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	if cfg.FreeOnly {
		catalog := openrouter.New()
		client.freeOnly = true
		client.checkFree = catalog.Verify
		client.closeCatalog = catalog.Close
	}
	return client, nil
}
func (c *Client) Close() {
	c.http.CloseIdleConnections()
	if c.closeCatalog != nil {
		c.closeCatalog()
	}
}
func (c *Client) InputTokens(ctx context.Context, r execution.Run) (int, error) {
	body, err := c.request(r)
	if err != nil {
		return 0, err
	}
	if c.count == nil {
		return 0, failure("TOKEN_COUNT_UNAVAILABLE")
	}
	n, err := c.count(ctx, c.model, body)
	if err != nil || n < 0 || n > 40000 {
		return 0, failure("TOKEN_COUNT_UNAVAILABLE")
	}
	return n, nil
}
func (c *Client) Complete(ctx context.Context, r execution.Run, step execution.Step) (execution.ModelReply, error) {
	if !secure.UUID(step.ID) || step.Settled || step.Epoch != r.StopEpoch || step.Input < 0 || step.Input > 40000 || step.Output != r.Limits.OutputTokens || step.Reserved != step.Input+step.Output {
		return execution.ModelReply{}, problem.Invalid("Invalid model reservation")
	}
	body, err := c.request(r)
	if err != nil {
		return execution.ModelReply{}, err
	}
	return c.completeBody(ctx, body, r, step)
}

func (c *Client) completeBody(ctx context.Context, body []byte, r execution.Run, step execution.Step) (execution.ModelReply, error) {
	if c.freeOnly {
		if err := c.checkFree(ctx, c.model); err != nil {
			return execution.ModelReply{}, err
		}
		guarded, err := freeBody(body)
		if err != nil {
			return execution.ModelReply{}, err
		}
		body = guarded
	}
	// A non-rewindable POST body prevents net/http from replaying a request on a
	// reused connection after a transport failure. No idempotency/retry header.
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return execution.ModelReply{}, failure("MODEL_REQUEST_INVALID")
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	res, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return execution.ModelReply{}, failure("MODEL_CANCELLED")
		}
		return execution.ModelReply{}, failure("MODEL_UNAVAILABLE")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		code := "MODEL_HTTP_ERROR"
		if res.StatusCode == 400 || res.StatusCode == 422 {
			code = "MODEL_REQUEST_REJECTED"
		}
		if res.StatusCode == 404 {
			code = "MODEL_NOT_AVAILABLE"
		}
		if res.StatusCode == 503 {
			code = "MODEL_CAPACITY_UNAVAILABLE"
		}
		// Classify a few documented upstream conditions, never expose the body.
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
		var upstream struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(raw, &upstream) == nil {
			message := strings.ToLower(upstream.Error.Message)
			if strings.Contains(message, "data policy") {
				code = "MODEL_DATA_POLICY_RESTRICTED"
			}
			if strings.Contains(message, "support") && strings.Contains(message, "parameter") {
				code = "MODEL_PARAMETERS_UNSUPPORTED"
			}
		}
		if res.StatusCode == 401 || res.StatusCode == 403 {
			code = "MODEL_AUTH_FAILED"
		}
		if res.StatusCode == 402 {
			code = "MODEL_FREE_QUOTA_UNAVAILABLE"
		}
		if res.StatusCode == 429 {
			code = "MODEL_RATE_LIMITED"
		}
		if res.StatusCode >= 300 && res.StatusCode < 400 {
			code = "MODEL_REDIRECT_DENIED"
		}
		// Upstream bodies and URL errors may contain API keys. Never expose/log them.
		return execution.ModelReply{}, failure(code)
	}
	media, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil || media != "text/event-stream" {
		return execution.ModelReply{}, failure("MODEL_PROTOCOL_ERROR")
	}
	return readStreamPolicy(res.Body, r, step, c.freeOnly)
}
