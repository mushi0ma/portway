// Package openrouter implements a fail-closed, zero-price cloud model policy.
package openrouter

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"portway/internal/problem"
)

const Endpoint = "https://openrouter.ai/api/v1"

var decimalPrice = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)
var freeID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*:free$`)

func FreeID(id string) bool { return len(id) <= 256 && freeID.MatchString(id) }

// Zero accepts a finite, exact nonnegative decimal zero; no floating-point
// rounding can turn a positive price into a free price. Missing/null is unknown.
func Zero(raw json.RawMessage) bool {
	var s string
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	if raw[0] == '"' {
		if json.Unmarshal(raw, &s) != nil {
			return false
		}
	} else {
		s = string(raw)
	}
	if len(s) == 0 || len(s) > 64 || s[0] == '-' {
		return false
	}
	if !decimalPrice.MatchString(s) {
		return false
	}
	if index := strings.IndexAny(s, "eE"); index >= 0 {
		s = s[:index]
	}
	for _, c := range s {
		if c != '0' && c != '.' {
			return false
		}
	}
	return true
}

type Model struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	ContextLength       int      `json:"context_length"`
	SupportedParameters []string `json:"supported_parameters"`
}

func (m Model) Supports(parameter string) bool {
	for _, p := range m.SupportedParameters {
		if p == parameter {
			return true
		}
	}
	return false
}

type Client struct{ http *http.Client }

func New() *Client {
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 16384}
	return &Client{http: &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) Close() { c.http.CloseIdleConnections() }
func unavailable() error {
	return problem.New(503, "FREE_MODEL_CATALOG_UNAVAILABLE", "Cannot verify current free model pricing")
}
func (c *Client) Models(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", Endpoint+"/models", nil)
	if err != nil {
		return nil, unavailable()
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, unavailable()
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, unavailable()
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, (4<<20)+1))
	if err != nil || len(raw) > 4<<20 {
		return nil, unavailable()
	}
	return Parse(raw)
}
func Parse(raw []byte) ([]Model, error) {
	if len(raw) > 4<<20 || !uniqueCatalog(raw) {
		return nil, unavailable()
	}
	var response struct {
		Data []struct {
			Model
			Pricing      map[string]json.RawMessage `json:"pricing"`
			Architecture struct {
				Input  []string `json:"input_modalities"`
				Output []string `json:"output_modalities"`
			} `json:"architecture"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &response) != nil || response.Data == nil {
		return nil, unavailable()
	}
	out := []Model{}
	seen := map[string]bool{}
	for _, entry := range response.Data {
		if seen[entry.ID] {
			return nil, unavailable()
		}
		seen[entry.ID] = true
		if !FreeID(entry.ID) || entry.ContextLength < 1 || entry.ContextLength > 10000000 || len(entry.Name) > 512 || !Zero(entry.Pricing["prompt"]) || !Zero(entry.Pricing["completion"]) {
			continue
		}
		free := true
		// Unknown pricing dimensions fail closed too (cache/reasoning/request/etc.).
		for _, price := range entry.Pricing {
			if !Zero(price) {
				free = false
			}
		}
		textInput, textOutput := false, false
		for _, v := range entry.Architecture.Input {
			textInput = textInput || v == "text"
		}
		for _, v := range entry.Architecture.Output {
			textOutput = textOutput || v == "text"
		}
		if !free || !textInput || !textOutput {
			continue
		}
		out = append(out, entry.Model)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
func (c *Client) Verify(ctx context.Context, id string) error {
	if !FreeID(id) {
		return problem.New(403, "FREE_MODEL_REQUIRED", "Only explicit zero-price :free models are allowed")
	}
	models, err := c.Models(ctx)
	if err != nil {
		return err
	}
	for _, m := range models {
		if m.ID == id {
			return nil
		}
	}
	return problem.New(403, "FREE_MODEL_REQUIRED", "Selected model is unavailable or its current pricing is not zero")
}

// Reject ambiguous duplicate fields, including prices, without restricting
// unrelated upstream metadata. Catalog JSON has its own 4 MiB size limit.
func uniqueCatalog(raw []byte) bool {
	d := json.NewDecoder(strings.NewReader(string(raw)))
	var walk func(int) bool
	walk = func(depth int) bool {
		if depth > 32 {
			return false
		}
		token, err := d.Token()
		if err != nil {
			return false
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return true
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return false
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return false
				}
				seen[name] = true
				if !walk(depth + 1) {
					return false
				}
			}
			end, err := d.Token()
			return err == nil && end == json.Delim('}')
		case '[':
			for d.More() {
				if !walk(depth + 1) {
					return false
				}
			}
			end, err := d.Token()
			return err == nil && end == json.Delim(']')
		}
		return false
	}
	if !walk(0) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}
