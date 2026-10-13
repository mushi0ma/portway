package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"portway/internal/agents"
	"portway/internal/execution"
	"portway/internal/identity"
	"portway/internal/modelgateway"
	"portway/internal/problem"
	"portway/internal/secure"
	"portway/internal/storage"
)

type harnessGateway struct {
	count    func(context.Context, execution.Run) (int, error)
	complete func(context.Context, execution.Run, execution.Step) (execution.ModelReply, error)
	calls    atomic.Int32
}

func (g *harnessGateway) InputTokens(ctx context.Context, r execution.Run) (int, error) {
	if g.count != nil {
		return g.count(ctx, r)
	}
	return 100, nil
}
func (g *harnessGateway) Complete(ctx context.Context, r execution.Run, s execution.Step) (execution.ModelReply, error) {
	g.calls.Add(1)
	return g.complete(ctx, r, s)
}
func TestHarnessTransactions(t *testing.T) {
	dsn, workerDSN := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_EXECUTION_DATABASE_URL")
	if dsn == "" || workerDSN == "" {
		if os.Getenv("PORTWAY_REQUIRE_INTEGRATION") == "1" {
			t.Fatal("execution database required")
		}
		t.Skip("use make docker-test")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Path != "/portway_test" {
		t.Fatal("disposable database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	installation, owner := secure.ID(), secure.ID()
	root, err := storage.Open(ctx, dsn, installation)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err = root.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	worker, err := storage.OpenWithLimit(ctx, workerDSN, installation, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	store := execution.Store{Pool: worker.Pool, Installation: installation}
	create := func(name string) execution.Chain {
		t.Helper()
		provider := secure.ID()
		cfg, err := agents.Validate(agents.Config{Instructions: "harness instructions", RoleCode: "coder", RoleCatalogVersion: 1, ProviderRef: provider, Tools: []agents.Tool{{Alias: "shell", Kind: "shell.exec"}}})
		if err != nil {
			t.Fatal(err)
		}
		c, err := store.Create(ctx, owner, "harness-"+name, execution.CreateInput{Task: "private harness task", Snapshot: execution.Snapshot{AgentVersionID: secure.ID(), ProviderID: provider, Config: cfg, ConfigHash: agents.Hash(cfg), RepositoryRef: "fixture", Revision: strings.Repeat("a", 40)}})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	finish := func() execution.ModelReply {
		return execution.ModelReply{Finish: "final", Text: "private harness result", UsageSource: "actual", Usage: 200}
	}
	t.Run("tool-to-result", func(t *testing.T) {
		c := create("roundtrip")
		run := c.Runs[0].ID
		var calls int
		g := &harnessGateway{complete: func(ctx context.Context, r execution.Run, s execution.Step) (execution.ModelReply, error) {
			// The reservation is observable from another DB connection before dispatch.
			saved, err := store.Lookup(ctx, owner, run, true)
			if err != nil || saved.Runs[0].Budget.Reserved != s.Reserved || saved.Runs[0].State != "waiting_model" {
				t.Error("dispatch before reservation commit", err)
			}
			calls++
			if calls == 1 {
				return execution.ModelReply{Finish: "tool", Usage: 123, UsageSource: "actual", Tool: &execution.ShellProposal{Alias: "shell", Command: execution.ShellCommand{Argv: []string{"echo", "private arguments"}, WorkingDir: ".", TimeoutSeconds: 60}}}, nil
			}
			if len(r.Turns) != 1 || r.Turns[0].Observation == nil || r.Turns[0].Observation.Code != "completed" {
				t.Error("tool outcome missing from next request")
			}
			return finish(), nil
		}}
		h := execution.Harness{Store: store, Gateway: g}
		c, err = h.Advance(ctx, owner, run)
		if err != nil || c.Runs[0].State != "awaiting_approval" {
			t.Fatal(err, c.State)
		}
		if _, err = h.Advance(ctx, owner, run); err != nil || g.calls.Load() != 1 {
			t.Fatal("approval did not pause harness", err)
		}
		a := c.Runs[0].Approvals[0]
		if _, err = store.DecideApproval(ctx, owner, secure.ID(), a.ID, "harness-decision", execution.Decision{Decision: "approve", ExpectedVersion: 1, ArgsHash: a.ArgsHash}); err != nil {
			t.Fatal(err)
		}
		if _, err = h.Advance(ctx, owner, run); err != nil || g.calls.Load() != 1 {
			t.Fatal("approval bypassed tool result", err)
		}
		dispatch, step := secure.ID(), c.Runs[0].Turns[0].StepID
		// Only a fixture calls the trusted Runtime adapter boundary. No host command.
		if _, err = store.Change(ctx, owner, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
			return c.BeginToolDispatch(run, step, dispatch, now)
		}); err != nil {
			t.Fatal(err)
		}
		if _, err = store.Change(ctx, owner, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
			return c.CompleteTool(run, step, dispatch, "completed", "private tool output", now)
		}); err != nil {
			t.Fatal(err)
		}
		c, err = h.Advance(ctx, owner, run)
		if err != nil || c.State != "completed" || c.Runs[0].Result == nil {
			t.Fatal("no durable result", err)
		}
		if c.Budget.Spent != 323 || c.Budget.Reserved != 0 || g.calls.Load() != 2 {
			t.Fatal("incorrect budget/calls")
		}
		seq := c.Runs[0].Seq
		if _, err = h.Advance(ctx, owner, run); err != nil || g.calls.Load() != 2 {
			t.Fatal("terminal dispatched", err)
		}
		if err = store.Reconcile(ctx); err != nil {
			t.Fatal(err)
		}
		saved, err := store.Lookup(ctx, owner, run, true)
		if err != nil || saved.Runs[0].Result.ID != c.Runs[0].Result.ID || saved.Runs[0].Seq != seq {
			t.Fatal("restart lost result", err)
		}
		events, err := store.History(ctx, owner, run, 0)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(events)
		if strings.Contains(string(raw), "private") {
			t.Fatal("private data leaked to history")
		}
		completed := 0
		for _, e := range events {
			if e.Type == "run.completed" {
				completed++
			}
		}
		if completed != 1 {
			t.Fatal("completion event not unique")
		}
		for {
			sent, err := store.RelayOne(ctx, func(_ context.Context, e execution.Envelope) error { return e.Validate(time.Now()) })
			if err != nil {
				t.Fatal("invalid outbox", err)
			}
			if !sent {
				break
			}
		}
		key := strings.Repeat("k", 32)
		var who atomic.Value
		who.Store(owner)
		server := httptest.NewServer(execution.HTTPServer{Store: store, ServiceKey: key, Authorize: func(context.Context, string, string) (execution.Authority, error) {
			return execution.Authority{OwnerID: who.Load().(string), DeviceID: secure.ID(), InstallationID: installation}, nil
		}}.Handler())
		defer server.Close()
		get := func(status int) {
			t.Helper()
			req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/internal/v1/runs/"+run+"/result", nil)
			req.Header.Set("X-Portway-Service-Key", key)
			res, err := server.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != status {
				t.Fatal("result HTTP", res.StatusCode)
			}
			if status == 200 {
				var body struct {
					Data execution.Result `json:"data"`
				}
				if json.NewDecoder(res.Body).Decode(&body) != nil || body.Data.ID != c.Runs[0].Result.ID {
					t.Fatal("wrong result body")
				}
			}
		}
		get(200)
		who.Store(secure.ID())
		get(404)
	})
	t.Run("http-model-adapter", func(t *testing.T) {
		c := create("http-model")
		run := c.Runs[0].ID
		var calls atomic.Int32
		var counted []byte
		model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if string(body) != string(counted) {
				t.Error("count/send body mismatch")
			}
			n := calls.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			var delta any
			finish := "stop"
			if n == 1 {
				finish = "tool_calls"
				delta = map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "model-call", "type": "function", "function": map[string]string{"name": "shell", "arguments": `{"argv":["echo","fixture"],"working_dir":".","timeout_seconds":60}`}}}}
			} else {
				if !strings.Contains(string(body), `"role":"tool"`) || !strings.Contains(string(body), "adapter observation") {
					t.Error("next HTTP prompt missing tool outcome")
				}
				delta = map[string]string{"content": "HTTP model final"}
			}
			chunk, _ := json.Marshal(map[string]any{"id": "model-response", "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}})
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			io.WriteString(w, `data: {"id":"model-response","choices":[],"usage":{"prompt_tokens":50,"completion_tokens":20,"total_tokens":70}}`+"\n\ndata: [DONE]\n\n")
		}))
		defer model.Close()
		gateway, err := modelgateway.New(modelgateway.Config{ProviderID: c.Runs[0].Snapshot.ProviderID, Endpoint: model.URL + "/v1", AllowedEndpoints: []string{model.URL + "/v1"}, Model: "fixture-model", Count: func(_ context.Context, _ string, raw []byte) (int, error) {
			counted = append([]byte{}, raw...)
			return 100, nil
		}})
		if err != nil {
			t.Fatal(err)
		}
		defer gateway.Close()
		h := execution.Harness{Store: store, Gateway: gateway}
		c, err = h.Advance(ctx, owner, run)
		if err != nil || c.Runs[0].State != "awaiting_approval" {
			t.Fatal("HTTP tool not accepted", err, c.Runs[0].Reason)
		}
		a := c.Runs[0].Approvals[0]
		if _, err = store.DecideApproval(ctx, owner, secure.ID(), a.ID, "http-model-approve", execution.Decision{Decision: "approve", ExpectedVersion: 1, ArgsHash: a.ArgsHash}); err != nil {
			t.Fatal(err)
		}
		dispatch, step := secure.ID(), c.Runs[0].Turns[0].StepID
		if _, err = store.Change(ctx, owner, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
			return c.BeginToolDispatch(run, step, dispatch, now)
		}); err != nil {
			t.Fatal(err)
		}
		if _, err = store.Change(ctx, owner, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
			return c.CompleteTool(run, step, dispatch, "completed", "adapter observation", now)
		}); err != nil {
			t.Fatal(err)
		}
		c, err = h.Advance(ctx, owner, run)
		if err != nil || c.State != "completed" || c.Runs[0].Result == nil || c.Runs[0].Result.Summary != "HTTP model final" || c.Budget.Spent != 140 || calls.Load() != 2 {
			t.Fatal("HTTP adapter cycle failed", err, c.State, c.Budget)
		}
	})

	t.Run("concurrent-and-stop", func(t *testing.T) {
		c := create("concurrency")
		run := c.Runs[0].ID
		started := make(chan struct{})
		g := &harnessGateway{complete: func(ctx context.Context, _ execution.Run, _ execution.Step) (execution.ModelReply, error) {
			close(started)
			<-ctx.Done()
			return execution.ModelReply{}, errors.New("private provider failure")
		}}
		h := execution.Harness{Store: store, Gateway: g}
		done := make(chan error, 1)
		go func() { _, err := h.Advance(ctx, owner, run); done <- err }()
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		if _, err := h.Advance(ctx, owner, run); err != nil {
			t.Fatal(err)
		}
		if g.calls.Load() != 1 {
			t.Fatal("double dispatch")
		}
		if _, err = store.Stop(ctx, owner, run, "harness-stop", true); err != nil {
			t.Fatal(err)
		}
		select {
		case err = <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("model context not cancelled after stop")
		}
		c, err = store.Lookup(ctx, owner, run, true)
		if err != nil || c.Runs[0].Reason != "budget_uncertain" || c.Runs[0].Result != nil || c.Budget.Reserved != 0 || c.Budget.Spent != 2148 {
			t.Fatal("unknown usage lost", err, c.Budget)
		}
	})

	t.Run("competing-reservations", func(t *testing.T) {
		c := create("preflight-race")
		run := c.Runs[0].ID
		counted := make(chan struct{}, 2)
		release := make(chan struct{})
		g := &harnessGateway{count: func(context.Context, execution.Run) (int, error) { counted <- struct{}{}; <-release; return 100, nil }, complete: func(context.Context, execution.Run, execution.Step) (execution.ModelReply, error) {
			return finish(), nil
		}}
		h := execution.Harness{Store: store, Gateway: g}
		done := make(chan error, 2)
		for i := 0; i < 2; i++ {
			go func() { _, err := h.Advance(ctx, owner, run); done <- err }()
		}
		for i := 0; i < 2; i++ {
			select {
			case <-counted:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}
		close(release)
		wins, conflicts := 0, 0
		for i := 0; i < 2; i++ {
			err := <-done
			if err == nil {
				wins++
			} else if problem.Public(err).Code == "RUN_CHANGED" {
				conflicts++
			} else {
				t.Fatal(err)
			}
		}
		if wins != 1 || conflicts != 1 || g.calls.Load() != 1 {
			t.Fatal("concurrent reservations dispatched twice", wins, conflicts, g.calls.Load())
		}
	})
	t.Run("stop-before-reservation", func(t *testing.T) {
		c := create("preflight-stop")
		run := c.Runs[0].ID
		g := &harnessGateway{count: func(context.Context, execution.Run) (int, error) {
			_, err := store.Stop(ctx, owner, run, "preflight-stop", true)
			return 100, err
		}, complete: func(context.Context, execution.Run, execution.Step) (execution.ModelReply, error) {
			t.Error("dispatch after preflight stop")
			return finish(), nil
		}}
		_, err := (execution.Harness{Store: store, Gateway: g}).Advance(ctx, owner, run)
		if problem.Public(err).Code != "RUN_CHANGED" || g.calls.Load() != 0 {
			t.Fatal("preflight stop ignored", err)
		}
		saved, err := store.Lookup(ctx, owner, run, true)
		if err != nil || saved.Budget.Reserved != 0 || saved.Budget.Steps != 0 {
			t.Fatal("preflight stop reserved budget", err)
		}
		if err = store.Reconcile(ctx); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("rollback-and-reply-replay", func(t *testing.T) {
		c := create("rollback")
		run, step := c.Runs[0].ID, secure.ID()
		c, err = store.Change(ctx, owner, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
			e, err := c.Start(run, now)
			if err != nil {
				return nil, err
			}
			ev, err := c.BeginTurn(run, step, 100, 1000, now)
			return append([]execution.Event{e}, ev...), err
		})
		if err != nil {
			t.Fatal(err)
		}
		before := c.Runs[0].Seq
		_, err = store.Change(ctx, owner, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
			ev, err := c.AcceptReply(run, step, finish(), now)
			if err != nil {
				return nil, err
			}
			return append(ev, ev[0]), nil
		})
		if err == nil {
			t.Fatal("duplicate event should roll back transaction")
		}
		saved, err := store.Lookup(ctx, owner, run, true)
		if err != nil || saved.Runs[0].Result != nil || saved.Runs[0].Seq != before || saved.Budget.Reserved != 1100 {
			t.Fatal("result/usage escaped rollback", err)
		}
		for i := 0; i < 2; i++ {
			c, err = store.Change(ctx, owner, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
				return c.AcceptReply(run, step, finish(), now)
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if c.Runs[0].Seq != before+3 || c.Budget.Spent != 200 {
			t.Fatal("reply duplicate changed result")
		}
		if _, err = store.Lookup(ctx, secure.ID(), run, true); problem.Public(err).Status != 404 {
			t.Fatal("foreign owner access")
		}
	})
}

func exerciseHarnessResult(t *testing.T, ctx context.Context, store execution.Store, client *http.Client, base string, login identity.LoginResult) {
	t.Helper()
	provider := secure.ID()
	cfg, err := agents.Validate(agents.Config{Instructions: "final answer fixture", RoleCode: "researcher", RoleCatalogVersion: 1, ProviderRef: provider})
	if err != nil {
		t.Fatal(err)
	}
	c, err := store.Create(ctx, login.Owner.ID, "public-result-fixture", execution.CreateInput{Task: "fixture task", Snapshot: execution.Snapshot{AgentVersionID: secure.ID(), ProviderID: provider, Config: cfg, ConfigHash: agents.Hash(cfg), RepositoryRef: "fixture", Revision: strings.Repeat("a", 40)}})
	if err != nil {
		t.Fatal(err)
	}
	run := c.Runs[0].ID
	path := "/v1/runs/" + run + "/result"
	code, _, raw := request(t, client, base, "GET", path, "", login.Session.AccessToken, login.DeviceProof, "", nil)
	want(t, code, 404, raw)
	g := &harnessGateway{complete: func(context.Context, execution.Run, execution.Step) (execution.ModelReply, error) {
		return execution.ModelReply{Finish: "final", Text: "persisted answer through public API", Usage: 50, UsageSource: "actual"}, nil
	}}
	c, err = (execution.Harness{Store: store, Gateway: g}).Advance(ctx, login.Owner.ID, run)
	if err != nil {
		t.Fatal(err)
	}
	code, body, raw := request(t, client, base, "GET", path, "", login.Session.AccessToken, login.DeviceProof, "", nil)
	want(t, code, 200, raw)
	var result execution.Result
	if json.Unmarshal(body.Data, &result) != nil || result.ID != c.Runs[0].Result.ID || result.Summary != "persisted answer through public API" {
		t.Fatal("public result contract mismatch")
	}
	code, _, raw = request(t, client, base, "GET", path, "", "", "", "", nil)
	want(t, code, 401, raw)
}
