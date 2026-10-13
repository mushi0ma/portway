package modelgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"portway/internal/agents"
	"portway/internal/execution"
	"portway/internal/problem"
	"portway/internal/secure"
)

func fixture(t *testing.T) (execution.Run, execution.Step) {
	t.Helper()
	provider := secure.ID()
	cfg, err := agents.Validate(agents.Config{Instructions: "Agent instructions", RoleCode: "coder", RoleCatalogVersion: 1, ProviderRef: provider, Tools: []agents.Tool{{Alias: "shell", Kind: "shell.exec"}}})
	if err != nil {
		t.Fatal(err)
	}
	c, _, err := execution.NewChain(secure.ID(), secure.ID(), execution.CreateInput{Task: "fixture task", Snapshot: execution.Snapshot{AgentVersionID: secure.ID(), ProviderID: provider, Config: cfg, ConfigHash: agents.Hash(cfg), RepositoryRef: "fixture", Revision: strings.Repeat("a", 40)}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r := c.Runs[0]
	r.State = "running"
	return r, execution.Step{ID: secure.ID(), Input: 100, Output: 2048, Reserved: 2148}
}
func event(v string) string { return "data: " + v + "\n\n" }
func choice(delta, finish string) string {
	return event(`{"id":"completion-fixture","choices":[{"index":0,"delta":` + delta + `,"finish_reason":` + finish + `}]}`)
}
func usageEvent(input, output int) string {
	return event(fmt.Sprintf(`{"id":"completion-fixture","choices":[],"usage":{"prompt_tokens":%d,"completion_tokens":%d,"total_tokens":%d}}`, input, output, input+output))
}
func finalStream() string {
	return choice(`{"content":"fixture answer"}`, `"stop"`) + usageEvent(50, 10) + event("[DONE]")
}
func newClient(t *testing.T, endpoint string, r execution.Run, count TokenCounter) *Client {
	t.Helper()
	c, err := New(Config{ProviderID: r.Snapshot.ProviderID, Endpoint: endpoint, AllowedEndpoints: []string{endpoint}, Model: "fixture-model", APIKey: "private-key-marker", Count: count})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}
func TestRequestCountMatchesWireAndStream(t *testing.T) {
	r, step := fixture(t)
	var counted []byte
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests.Add(1)
		raw, _ := io.ReadAll(req.Body)
		if string(raw) != string(counted) {
			t.Error("counted body differs from sent body")
		}
		if req.URL.Path != "/v1/chat/completions" || req.Method != "POST" || req.Header.Get("Authorization") != "Bearer private-key-marker" {
			t.Error("wrong endpoint/credentials")
		}
		var in struct {
			Stream   bool             `json:"stream"`
			Parallel bool             `json:"parallel_tool_calls"`
			Max      int              `json:"max_tokens"`
			Messages []message        `json:"messages"`
			Tools    []toolDefinition `json:"tools"`
		}
		if json.Unmarshal(raw, &in) != nil || !in.Stream || in.Parallel || in.Max != step.Output || len(in.Tools) != 1 || in.Tools[0].Function.Name != "shell" {
			t.Error("wrong model request")
		}
		if strings.Contains(string(raw), "private-key-marker") {
			t.Error("key leaked into model input")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, finalStream())
	}))
	defer server.Close()
	c := newClient(t, server.URL+"/v1", r, func(_ context.Context, model string, body []byte) (int, error) {
		counted = append([]byte{}, body...)
		if model != "fixture-model" {
			t.Error("wrong counted model")
		}
		return 100, nil
	})
	if n, err := c.InputTokens(context.Background(), r); err != nil || n != 100 || requests.Load() != 0 {
		t.Fatal("count invoked generation", n, err)
	}
	reply, err := c.Complete(context.Background(), r, step)
	if err != nil || reply.Text != "fixture answer" || reply.Usage != 60 || reply.UsageSource != "actual" || requests.Load() != 1 {
		t.Fatal(reply, err)
	}
}
func TestFragmentedToolAndObservationRequest(t *testing.T) {
	r, step := fixture(t)
	first := `{"tool_calls":[{"index":0,"id":"provider-call","type":"function","function":{"name":"sh","arguments":"{\"argv\":[\"echo\",\"hi\"],"}}]}`
	last := `{"tool_calls":[{"index":0,"function":{"name":"ell","arguments":"\"working_dir\":\".\",\"timeout_seconds\":60}"}}]}`
	stream := choice(`{"content":"I will propose a command"}`, "null") + choice(first, "null") + choice(last, `"tool_calls"`) + usageEvent(50, 20) + event("[DONE]")
	reply, err := readStream(strings.NewReader(stream), r, step)
	if err != nil || reply.Finish != "tool" || reply.Tool.Alias != "shell" || reply.Tool.Command.Argv[1] != "hi" || reply.Text != "" {
		t.Fatal(reply, err)
	}
	r.Turns = []execution.Turn{{StepID: step.ID, State: "done", Proposal: reply.Tool, Observation: &execution.Observation{Code: "APPROVAL_REJECTED", ToolCallID: secure.ID(), ArgsHash: strings.Repeat("a", 64), Summary: "untrusted observation"}}}
	c := newClient(t, "http://fixture.invalid/v1", r, nil)
	raw, err := c.request(r)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Messages []message `json:"messages"`
	}
	if json.Unmarshal(raw, &body) != nil {
		t.Fatal("request JSON")
	}
	a, b := body.Messages[len(body.Messages)-2], body.Messages[len(body.Messages)-1]
	if a.Role != "assistant" || len(a.ToolCalls) != 1 || b.Role != "tool" || b.ToolCallID != a.ToolCalls[0].ID || !strings.Contains(b.Content, "APPROVAL_REJECTED") {
		t.Fatal("invalid tool result roundtrip")
	}
}
func TestStreamRejectsIncompleteAndAmbiguous(t *testing.T) {
	r, step := fixture(t)
	tool := func(args string) string {
		b, _ := json.Marshal(map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "call", "type": "function", "function": map[string]string{"name": "shell", "arguments": args}}}})
		return choice(string(b), `"tool_calls"`) + event("[DONE]")
	}
	cases := map[string]string{
		"empty":                 "",
		"no-done":               choice(`{"content":"answer"}`, `"stop"`),
		"no-finish":             choice(`{"content":"partial"}`, "null") + event("[DONE]"),
		"bad-json":              event(`{"id":`) + event("[DONE]"),
		"changed-id":            choice(`{"content":"partial"}`, "null") + strings.Replace(finalStream(), "completion-fixture", "other", -1),
		"two-choices":           event(`{"id":"x","choices":[{"index":0,"delta":{}},{"index":1,"delta":{}}]}`),
		"second-tool":           choice(`{"tool_calls":[{"index":1,"id":"x","function":{"name":"shell","arguments":"{}"}}]}`, `"tool_calls"`) + event("[DONE]"),
		"late-content":          choice(`{"content":"answer"}`, `"stop"`) + choice(`{"content":"injected"}`, "null") + event("[DONE]"),
		"duplicate-usage":       choice(`{"content":"answer"}`, `"stop"`) + usageEvent(2, 2) + usageEvent(2, 2) + event("[DONE]"),
		"missing-usage-fields":  choice(`{"content":"answer"}`, `"stop"`) + event(`{"id":"completion-fixture","choices":[],"usage":{"total_tokens":0}}`) + event("[DONE]"),
		"bad-usage-total":       choice(`{"content":"answer"}`, `"stop"`) + event(`{"id":"completion-fixture","choices":[],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":0}}`) + event("[DONE]"),
		"unknown-tool-fields":   tool(`{"argv":["echo"],"working_dir":".","timeout_seconds":60,"approved":true}`),
		"duplicate-tool-fields": tool(`{"argv":["echo"],"argv":["bad"],"working_dir":".","timeout_seconds":60}`),
		"invalid-tool-json":     tool(`{"argv":[`),
		"tool-finish-as-text":   choice(`{"tool_calls":[{"index":0,"id":"x","function":{"name":"shell","arguments":"{}"}}]}`, `"stop"`) + event("[DONE]"),
		"oversized-line":        event(strings.Repeat("x", 140000)),
		"oversized-text":        choice(`{"content":"`+strings.Repeat("x", 65537)+`"}`, `"stop"`) + event("[DONE]"),
		"oversized-stream":      strings.Repeat(": ignored\n", 110000) + finalStream(),
		"refusal":               choice(`{"refusal":"no"}`, `"stop"`) + event("[DONE]"),
	}
	for name, stream := range cases {
		t.Run(name, func(t *testing.T) {
			reply, err := readStream(strings.NewReader(stream), r, step)
			if err == nil || reply.Finish != "" {
				t.Fatal("bad stream accepted", reply, err)
			}
		})
	}
}
func TestUnknownUsageAndTruncatedFinish(t *testing.T) {
	r, step := fixture(t)
	for _, stream := range []string{choice(`{"content":"answer"}`, `"stop"`) + event("[DONE]"), choice(`{"content":"answer"}`, `"stop"`) + usageEvent(101, 1) + event("[DONE]"), choice(`{"content":"answer"}`, `"stop"`) + usageEvent(1, 2049) + event("[DONE]")} {
		reply, err := readStream(strings.NewReader(stream), r, step)
		if err != nil || reply.UsageSource != "unknown" {
			t.Fatal("missing/overrun usage counted as actual", reply, err)
		}
	}
	reply, err := readStream(strings.NewReader(choice(`{"content":"truncated"}`, `"length"`)+usageEvent(50, 10)+event("[DONE]")), r, step)
	if err != nil || reply.Finish != "error" || reply.Usage != 60 {
		t.Fatal("truncation became success or lost actual usage", reply, err)
	}
}
func TestNoRedirectRetryOrErrorLeaks(t *testing.T) {
	r, step := fixture(t)
	var destination atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { destination.Add(1) }))
	defer target.Close()
	for _, status := range []int{302, 401, 403, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Location", target.URL)
				w.WriteHeader(status)
				io.WriteString(w, "private-key-marker provider-body-secret")
			}))
			defer server.Close()
			c := newClient(t, server.URL, r, nil)
			_, err := c.Complete(context.Background(), r, step)
			if err == nil || calls.Load() != 1 || destination.Load() != 0 || strings.Contains(err.Error(), "marker") || strings.Contains(err.Error(), "secret") {
				t.Fatal("redirect/retry/secret leak", err)
			}
		})
	}
}
func TestCancellationAndBinding(t *testing.T) {
	r, step := fixture(t)
	started, ended := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		io.Copy(io.Discard, req.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		close(started)
		<-req.Context().Done()
		close(ended)
	}))
	defer server.Close()
	c := newClient(t, server.URL, r, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := c.Complete(ctx, r, step); done <- err }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request never started")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancel succeeded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancel stalled")
	}
	select {
	case <-ended:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream not cancelled")
	}
	if _, err := c.InputTokens(context.Background(), r); problem.Public(err).Code != "TOKEN_COUNT_UNAVAILABLE" {
		t.Fatal("guessed tokens", err)
	}
	r.Snapshot.ProviderID = secure.ID()
	if _, err := c.Complete(context.Background(), r, step); err == nil {
		t.Fatal("provider mismatch sent")
	}
	if _, err := New(Config{ProviderID: secure.ID(), Endpoint: server.URL, Model: "x", AllowedEndpoints: []string{"https://different.test"}}); err == nil {
		t.Fatal("unlisted endpoint accepted")
	}
}
