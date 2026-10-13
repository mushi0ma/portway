package modelgateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"portway/internal/openrouter"
	"portway/internal/problem"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestFreeOnlyRequestAndCost(t *testing.T) {
	r, step := fixture(t)
	c, err := New(Config{ProviderID: r.Snapshot.ProviderID, Endpoint: openrouter.Endpoint, AllowedEndpoints: []string{openrouter.Endpoint}, Model: "vendor/model:free", APIKey: "private-key-marker", FreeOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	checks, calls := 0, 0
	c.checkFree = func(context.Context, string) error { checks++; return nil }
	var counted []byte
	c.count = func(_ context.Context, _ string, body []byte) (int, error) {
		counted = append([]byte{}, body...)
		return 100, nil
	}
	c.http = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		raw, _ := io.ReadAll(req.Body)
		if string(raw) != string(counted) || strings.Contains(string(raw), "private-key-marker") {
			t.Error("count/send differs or secret leak")
		}
		var body map[string]json.RawMessage
		json.Unmarshal(raw, &body)
		var provider struct {
			Fallback bool              `json:"allow_fallbacks"`
			Require  bool              `json:"require_parameters"`
			Price    map[string]string `json:"max_price"`
		}
		json.Unmarshal(body["provider"], &provider)
		if provider.Fallback || !provider.Require || provider.Price["prompt"] != "0" || provider.Price["request"] != "0" || body["plugins"] != nil || body["models"] != nil {
			t.Error("missing zero-price policy")
		}
		stream := strings.Replace(finalStream(), `"total_tokens":60`, `"total_tokens":60,"cost":0`, 1)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(stream))}, nil
	})}
	if _, err = c.InputTokens(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || checks != 0 {
		t.Fatal("count generated or fetched model")
	}
	if reply, err := c.Complete(context.Background(), r, step); err != nil || reply.Finish != "final" {
		t.Fatal(reply, err)
	}
	c.checkFree = func(context.Context, string) error {
		return problem.New(403, "FREE_MODEL_REQUIRED", "Model pricing changed")
	}
	if _, err := c.Complete(context.Background(), r, step); problem.Public(err).Code != "FREE_MODEL_REQUIRED" || calls != 1 {
		t.Fatal("paid price dispatched", calls, err)
	}
}
func TestFreeOnlyRejectsCostAndFallback(t *testing.T) {
	r, step := fixture(t)
	for _, cost := range []string{"", `,"cost":0.000000001`, `,"cost":null`} {
		stream := strings.Replace(finalStream(), `"total_tokens":60`, `"total_tokens":60`+cost, 1)
		if _, err := readStreamPolicy(strings.NewReader(stream), r, step, true); problem.Public(err).Code != "MODEL_COST_UNVERIFIED" {
			t.Fatal("unknown/nonzero cost accepted", err)
		}
	}
	for _, body := range []string{`{"model":"vendor/paid"}`, `{"model":"openrouter/free"}`, `{"model":"vendor/model:free","plugins":[{"id":"web"}]}`, `{"model":"vendor/model:free","models":["paid"]}`} {
		if _, err := freeBody([]byte(body)); err == nil {
			t.Fatal("paid/automatic route accepted", body)
		}
	}
	if _, err := New(Config{ProviderID: r.Snapshot.ProviderID, Endpoint: openrouter.Endpoint, AllowedEndpoints: []string{openrouter.Endpoint}, Model: "vendor/paid", FreeOnly: true}); err == nil {
		t.Fatal("paid client constructed")
	}
	if _, err := New(Config{ProviderID: r.Snapshot.ProviderID, Endpoint: "https://api.groq.com/openai/v1", AllowedEndpoints: []string{"https://api.groq.com/openai/v1"}, Model: "vendor/model:free", FreeOnly: true}); err == nil {
		t.Fatal("unverified cloud plan accepted")
	}
}

func TestOpenRouterUsageAttachedToFinish(t *testing.T) {
	r, step := fixture(t)
	stream := event(`{"id":"free-model","choices":[{"index":0,"delta":{"content":"done"},"finish_reason":"stop"}],"usage":{"prompt_tokens":50,"completion_tokens":10,"total_tokens":60,"cost":0}}`) + event("[DONE]")
	reply, err := readStreamPolicy(strings.NewReader(stream), r, step, true)
	if err != nil || reply.Usage != 60 || reply.UsageSource != "actual" || reply.Finish != "final" {
		t.Fatal(reply, err)
	}
	if _, err := readStream(strings.NewReader(stream), r, step); err == nil {
		t.Fatal("generic strict profile changed")
	}
	if _, err := readStreamPolicy(strings.NewReader(strings.Replace(stream, `"cost":0`, `"cost":0.1`, 1)), r, step, true); err == nil {
		t.Fatal("nonzero attached cost accepted")
	}
	if _, err := readStreamPolicy(strings.NewReader(strings.Replace(stream, `"finish_reason":"stop"`, `"finish_reason":null`, 1)), r, step, true); err == nil {
		t.Fatal("usage before finish accepted")
	}
}

func TestOpenRouterRepeatsFinishInAccountingFrame(t *testing.T) {
	r, step := fixture(t)
	accounting := `{"id":"completion-fixture","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":"stop"}],"usage":{"prompt_tokens":50,"completion_tokens":10,"total_tokens":60,"cost":0}}`
	prefix := choice(`{"content":"done"}`, `"stop"`)
	stream := prefix + event(accounting) + event("[DONE]")
	if reply, err := readStreamPolicy(strings.NewReader(stream), r, step, true); err != nil || reply.Finish != "final" || reply.Usage != 60 {
		t.Fatal(reply, err)
	}
	for _, bad := range []string{
		strings.Replace(accounting, `"content":""`, `"content":"late"`, 1),
		strings.Replace(accounting, `"finish_reason":"stop"`, `"finish_reason":"length"`, 1),
		strings.Replace(accounting, `"cost":0`, `"cost":0.01`, 1),
	} {
		if _, err := readStreamPolicy(strings.NewReader(prefix+event(bad)+event("[DONE]")), r, step, true); err == nil {
			t.Fatal("bad accounting frame accepted")
		}
	}
	if _, err := readStreamPolicy(strings.NewReader(prefix+event(accounting)+event(accounting)+event("[DONE]")), r, step, true); err == nil {
		t.Fatal("duplicate usage accepted")
	}
}

func TestFreeFollowupHasNoAvailableTools(t *testing.T) {
	raw, err := freeBody([]byte(`{"model":"vendor/model:free","tool_choice":"none","tools":[{}],"messages":[],"n":1,"parallel_tool_calls":false,"stream_options":{"include_usage":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	json.Unmarshal(raw, &body)
	for _, name := range []string{"tools", "tool_choice", "n", "parallel_tool_calls", "stream_options"} {
		if body[name] != nil {
			t.Fatal("unsupported optional parameter retained", name)
		}
	}
}
