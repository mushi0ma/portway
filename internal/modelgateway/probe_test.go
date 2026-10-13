package modelgateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"portway/internal/problem"
)

func probeToolStream() string {
	return choice(`{"tool_calls":[{"index":0,"id":"probe","type":"function","function":{"name":"portway_probe","arguments":"{\"argv\":[\"echo\",\"PORTWAY_PROBE_OK\"],\"working_dir\":\".\",\"timeout_seconds\":1}"}}]}`, `"tool_calls"`) + usageEvent(100, 30) + event("[DONE]")
}
func probeResultStream() string {
	return choice(`{"content":"PORTWAY_PROBE_OK"}`, `"stop"`) + usageEvent(140, 10) + event("[DONE]")
}

func TestProbeBoundedProtocol(t *testing.T) {
	for _, tc := range []struct {
		name, first, second, code string
		status                    int
		requests                  int
	}{
		{"compatible", probeToolStream(), probeResultStream(), "", 200, 2},
		{"auth", "private-key-marker", "", "MODEL_AUTH_FAILED", 401, 1},
		{"no-tool", probeResultStream(), "", "MODEL_PROBE_TOOL_MISMATCH", 200, 1},
		{"wrong-args", strings.Replace(probeToolStream(), "PORTWAY_PROBE_OK", "untrusted-command", 1), "", "MODEL_PROBE_TOOL_MISMATCH", 200, 1},
		{"no-usage", strings.Replace(probeToolStream(), usageEvent(100, 30), "", 1), probeResultStream(), "MODEL_USAGE_UNAVAILABLE", 200, 2},
		{"too-many-output-tokens", strings.Replace(probeToolStream(), usageEvent(100, 30), usageEvent(100, 129), 1), probeResultStream(), "MODEL_USAGE_UNAVAILABLE", 200, 2},
		{"wrong-result", probeToolStream(), finalStream(), "MODEL_PROBE_RESULT_MISMATCH", 200, 2},
		{"incomplete-result", probeToolStream(), strings.Replace(probeResultStream(), event("[DONE]"), "", 1), "MODEL_STREAM_INCOMPLETE", 200, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				raw, _ := io.ReadAll(req.Body)
				var body map[string]json.RawMessage
				if json.Unmarshal(raw, &body) != nil || string(body["max_tokens"]) != "128" || string(body["parallel_tool_calls"]) != "false" || len(raw) > 4096 {
					t.Error("unbounded probe body", string(raw))
				}
				if req.Header.Get("Authorization") != "Bearer private-key-marker" || strings.Contains(string(raw), "private-key-marker") || strings.Contains(string(raw), "fixture task") {
					t.Error("probe credential/task isolation")
				}
				if n == 2 && (!strings.Contains(string(body["messages"]), `"role":"tool"`) || string(body["tool_choice"]) != `"none"`) {
					t.Error("missing simulated roundtrip")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(tc.status)
				if n == 1 {
					io.WriteString(w, tc.first)
				} else {
					io.WriteString(w, tc.second)
				}
			}))
			defer s.Close()
			r, _ := fixture(t)
			c := newClient(t, s.URL+"/v1", r, nil)
			got := c.Probe(context.Background(), nil)
			if got.ErrorCode != tc.code || got.Requests != tc.requests || int(calls.Load()) != tc.requests || got.Compatible() != (tc.code == "") {
				t.Fatal(got, calls.Load())
			}
		})
	}
}

func TestProbeCancelStopsRequests(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, req.Body)
		<-req.Context().Done()
	}))
	defer s.Close()
	r, _ := fixture(t)
	c := newClient(t, s.URL, r, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	got := c.Probe(ctx, nil)
	if got.ErrorCode != "MODEL_CANCELLED" || got.Requests != 1 || calls.Load() != 1 {
		t.Fatal(got, calls.Load())
	}
	got = c.Probe(ctx, nil)
	if got.Requests != 0 || calls.Load() != 1 {
		t.Fatal("cancelled probe dispatched", got)
	}
}

func TestProbeAuthorityRecheckedBeforeEachRequest(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, probeToolStream())
	}))
	defer s.Close()
	r, _ := fixture(t)
	c := newClient(t, s.URL, r, nil)
	checks := 0
	got := c.Probe(context.Background(), func(context.Context) error {
		checks++
		if checks == 2 {
			return problem.New(409, "CAPABILITY_CHECK_INVALIDATED", "Check is no longer current")
		}
		return nil
	})
	if got.Requests != 1 || got.ErrorCode != "CAPABILITY_CHECK_INVALIDATED" || calls.Load() != 1 || checks != 2 {
		t.Fatal("invalidated authority reached second request", got, calls.Load(), checks)
	}
}
