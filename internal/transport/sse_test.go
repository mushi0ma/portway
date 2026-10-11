package transport

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

	"portway/internal/execution"
	"portway/internal/problem"
	"portway/internal/secure"
)

func TestEventCursorRejectsAmbiguityAndCredentials(t *testing.T) {
	run := secure.ID()
	for _, tc := range []struct {
		query, header string
		valid         bool
	}{
		{"", "", true}, {"cursor=" + run + ":0", "", true}, {"", run + ":12", true},
		{"cursor=" + run + ":1", run + ":1", false}, {"cursor=" + run + ":1&cursor=" + run + ":2", "", false},
		{"access_token=secret", "", false}, {"cursor=", "", false}, {"", secure.ID() + ":2", false},
		{"", run + ":-1", false}, {"", run + ":01", false}, {"", run + ":+1", false},
		{"", run + ":9223372036854775808", false}, {"cursor=%zz", "", false},
	} {
		r := httptest.NewRequest("GET", "/v1/runs/"+run+"/events?"+tc.query, nil)
		if tc.header != "" {
			r.Header.Set("Last-Event-ID", tc.header)
		}
		_, err := eventCursor(r, run)
		if (err == nil) != tc.valid {
			t.Fatalf("cursor %q %q: %v", tc.query, tc.header, err)
		}
	}
}

func TestHistoryRejectsGapsAndSSEInjection(t *testing.T) {
	run := secure.ID()
	e := execution.Event{SchemaVersion: 1, ID: secure.ID(), RunID: run, CorrelationID: secure.ID(), Seq: 1, Type: "run.created", Payload: json.RawMessage(`{"state":"queued"}`)}
	p := execution.HistoryPage{Items: []execution.Event{e}, SnapshotSeq: 1}
	if !validHistory(p, run, 0) {
		t.Fatal("valid page rejected")
	}
	p.Items[0].Type = "run.created\nid: injected"
	if validHistory(p, run, 0) {
		t.Fatal("SSE field injection accepted")
	}
	p.Items[0] = e
	p.SnapshotSeq = 2
	if validHistory(p, run, 0) {
		t.Fatal("missing suffix accepted")
	}
	p.SnapshotSeq = 1
	p.PrunedThrough = 1
	if validHistory(p, run, 0) {
		t.Fatal("expired prefix accepted")
	}
}

func TestStreamOverridesWriteTimeoutAndClosesOnAuthorityFailure(t *testing.T) {
	var polls atomic.Int32
	var revoked atomic.Bool
	run := secure.ID()
	h := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		load := func(ctx context.Context, seq int64) (execution.HistoryPage, error) {
			polls.Add(1)
			if revoked.Load() {
				return execution.HistoryPage{}, problem.New(401, "AUTH_REQUIRED", "Revoked")
			}
			return execution.HistoryPage{Items: []execution.Event{}, SnapshotSeq: seq}, nil
		}
		page := execution.HistoryPage{Items: []execution.Event{{SchemaVersion: 1, ID: secure.ID(), RunID: run, CorrelationID: secure.ID(), Seq: 1, Type: "run.created", Payload: json.RawMessage(`{"state":"queued"}`)}}, SnapshotSeq: 1}
		streamHistory(w, r, 0, page, load)
	}))
	h.Config.WriteTimeout = 100 * time.Millisecond
	h.Start()
	defer h.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	r, err := client.Get(h.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	done := make(chan []byte, 1)
	go func() { b, _ := io.ReadAll(r.Body); done <- b }()
	deadline := time.Now().Add(3 * time.Second)
	for polls.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if polls.Load() < 2 {
		t.Fatal("stream did not survive WriteTimeout")
	}
	revoked.Store(true)
	select {
	case b := <-done:
		if !strings.Contains(string(b), "id: "+run+":1\n") || strings.Contains(string(b), `"error"`) {
			t.Fatalf("invalid SSE: %s", b)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not close on failed authorization")
	}
}
