package codexbridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"portway/internal/secure"
)

func TestWorkerCancelsOnLostAuthority(t *testing.T) {
	runner := fakeRunner(t, `case "$1" in login) echo 'Logged in using ChatGPT' >&2; exit;; app-server|--version) exit 0;; esac
sleep 30 &
wait
`)
	reg, err := OpenRegistry(filepath.Join(t.TempDir(), "bridge"))
	if err != nil {
		t.Fatal(err)
	}
	job := Job{ID: secure.ID(), Token: secure.Token(), Prompt: "blocked"}
	key := secure.Token()
	var claimed atomic.Bool
	var terminal atomic.Bool
	released := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Portway-Bridge-Key") != key {
			t.Error("missing dedicated key")
		}
		if r.URL.Path == "/internal/v1/codex/report" {
			var in Report
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in.Event.Type == "completed" || in.Event.Type == "failed" {
				terminal.Store(true)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"accepted": true}})
			return
		}
		if r.URL.Path != "/internal/v1/codex/poll" {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"accepted": true}})
			return
		}
		var in Poll
		_ = json.NewDecoder(r.Body).Decode(&in)
		result := PollResult{}
		if in.RunID != "" {
			select {
			case released <- struct{}{}:
			default:
			}
		} else if claimed.CompareAndSwap(false, true) {
			result.Job = &job
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": result})
	}))
	defer server.Close()
	worker, err := NewWorker(server.URL, key, &Bridge{Workspace: runner.Workspace, Accounts: reg, Agents: []Agent{runner, &Claude{}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Serve(ctx) }()
	select {
	case <-released:
	case <-time.After(5 * time.Second):
		t.Fatal("authority not rechecked")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not cancel native process")
	}
	if terminal.Load() {
		t.Fatal("output published after loss of authority")
	}
}

// The worker acks delivered commands, pushes validated snapshots and reports run metadata.
func TestWorkerCommandsStateAndRunMeta(t *testing.T) {
	b, _ := testBridge(t)
	key := secure.Token()
	command := Command{ID: secure.ID(), Kind: "usage_refresh"}
	job := Job{ID: secure.ID(), Token: secure.Token(), Prompt: "hi", Provider: "claude", Model: "sonnet", Effort: "high"}
	var mu sync.Mutex
	var acks []Ack
	var states []Snapshot
	var reports []Report
	var polls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		data := any(map[string]any{"accepted": true})
		switch {
		case strings.HasSuffix(r.URL.Path, "/ack"):
			var a Ack
			_ = json.NewDecoder(r.Body).Decode(&a)
			if r.URL.Path == "/internal/v1/codex/commands/"+command.ID+"/ack" {
				acks = append(acks, a)
			}
		case r.URL.Path == "/internal/v1/codex/state":
			var s Snapshot
			_ = json.NewDecoder(r.Body).Decode(&s)
			if err := s.Validate(); err != nil {
				t.Error("bridge pushed invalid state", err)
			}
			states = append(states, s)
		case r.URL.Path == "/internal/v1/codex/report":
			var in Report
			_ = json.NewDecoder(r.Body).Decode(&in)
			reports = append(reports, in)
		case r.URL.Path == "/internal/v1/codex/poll":
			result := PollResult{Commands: []Command{}}
			switch polls.Add(1) {
			case 1:
				result.Commands = []Command{command}
			case 2:
				result.Job = &job
			}
			var in Poll
			_ = json.NewDecoder(r.Body).Decode(&in)
			result.Continue = in.RunID != ""
			data = result
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer server.Close()
	worker, err := NewWorker(server.URL, key, b)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Serve(ctx) }()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ready := len(acks) > 0 && len(reports) > 0 && reports[len(reports)-1].Event.Type == "completed" && len(states) > 1
		mu.Unlock()
		if ready {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	<-done
	mu.Lock()
	defer mu.Unlock()
	if len(acks) != 1 || acks[0].State != "done" {
		t.Fatal("command not acked", acks)
	}
	if len(states) < 2 || len(states[len(states)-1].Providers) != 2 {
		t.Fatal("state not pushed", len(states))
	}
	last := reports[len(reports)-1].Event
	if last.Type != "completed" || last.Text != "answer model=sonnet effort=high" || last.Meta == nil || last.Meta.AccountLabel == nil || *last.Meta.AccountLabel != "Claude" || *last.Meta.ResolvedModel != "claude-sonnet-fake" || last.Meta.Tokens.Input != 10 {
		t.Fatal("completion metadata", last)
	}
}
