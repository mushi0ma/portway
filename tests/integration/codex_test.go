package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"portway/internal/codexbridge"
	"portway/internal/codexbridge/fakecli"
	"portway/internal/config"
	"portway/internal/identity"
	"portway/internal/secure"
	"portway/internal/storage"
	"portway/internal/transport"
)

func exerciseCodexBridge(t *testing.T, cfg config.Config, app, root *storage.Store, one identity.LoginResult) {
	t.Helper()
	cfg.CodexBridgeKey = secure.Token()
	ctx := context.Background()
	api := httptest.NewServer(transport.New(cfg, app, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))).Handler())
	defer api.Close()
	call := func(method, path, body, key string) (int, response, []byte) {
		return request(t, api.Client(), api.URL, method, path, body, one.Session.AccessToken, one.DeviceProof, key, nil)
	}
	bridge := func(path string, in any) (int, response, []byte) {
		b, _ := json.Marshal(in)
		return request(t, api.Client(), api.URL, "POST", path, string(b), "", "", "", map[string]string{"X-Portway-Bridge-Key": cfg.CodexBridgeKey})
	}
	code, got, raw := call("GET", "/v1/codex/status", "", "")
	want(t, code, 200, raw)
	var early codexbridge.Status
	if json.Unmarshal(got.Data, &early) != nil || early.Providers == nil || len(early.Providers) != 0 || early.Login != nil || early.Commands == nil || early.StateUpdatedAt != nil {
		t.Fatal("status before any bridge snapshot", string(raw))
	}
	code, _, raw = call("POST", "/v1/codex/runs", `{"workspace":"portway","prompt":"prompt-marker"}`, "codex-offline-key")
	want(t, code, 503, raw)
	code, _, raw = request(t, api.Client(), api.URL, "POST", "/internal/v1/codex/poll", fmt.Sprintf(`{"instance_id":%q}`, secure.ID()), "", "", "", nil)
	want(t, code, 403, raw)
	for _, table := range []string{"codex_runs", "codex_events", "codex_bridge", "codex_commands"} {
		var accessible bool
		if err := root.Pool.QueryRow(ctx, `SELECT has_table_privilege('portway_execution',$1,'SELECT') OR has_table_privilege('portway_relay',$1,'SELECT') OR has_table_privilege('portway_audit',$1,'SELECT')`, "portway."+table).Scan(&accessible); err != nil || accessible {
			t.Fatal("foreign service can read Codex data", table, err)
		}
	}
	instance := secure.ID()
	code, _, raw = bridge("/internal/v1/codex/poll", codexbridge.Poll{Instance: instance})
	want(t, code, 200, raw)
	code, _, raw = bridge("/internal/v1/codex/poll", codexbridge.Poll{Instance: secure.ID()})
	want(t, code, 409, raw)
	code, _, raw = call("POST", "/v1/codex/runs", `{"workspace":"/tmp","prompt":"prompt"}`, "codex-invalid-path")
	want(t, code, 400, raw)
	input := `{"workspace":"portway","prompt":"prompt-marker"}`
	code, created, raw := call("POST", "/v1/codex/runs", input, "codex-create-key")
	want(t, code, 202, raw)
	var run codexbridge.Run
	if json.Unmarshal(created.Data, &run) != nil || !secure.UUID(run.ID) || run.Provider != "codex" || run.Model != nil || run.PromptPreview != nil || bytes.Contains(created.Data, []byte("prompt-marker")) {
		t.Fatal("invalid run", string(created.Data))
	}
	code, replayed, raw := call("POST", "/v1/codex/runs", input, "codex-create-key")
	want(t, code, 202, raw)
	var replayRun codexbridge.Run
	if json.Unmarshal(replayed.Data, &replayRun) != nil || replayRun.ID != run.ID || replayRun.State != run.State {
		t.Fatal("create replay changed")
	}
	code, _, raw = call("POST", "/v1/codex/runs", input, "codex-create-other")
	want(t, code, 409, raw)
	var ct []byte
	if err := root.Pool.QueryRow(ctx, `SELECT prompt_ciphertext FROM portway.codex_runs WHERE id=$1`, run.ID).Scan(&ct); err != nil || bytes.Contains(ct, []byte("prompt-marker")) {
		t.Fatal("unencrypted prompt", err)
	}
	code, polled, raw := bridge("/internal/v1/codex/poll", codexbridge.Poll{Instance: instance})
	want(t, code, 200, raw)
	var result codexbridge.PollResult
	if json.Unmarshal(polled.Data, &result) != nil || result.Job == nil || result.Job.ID != run.ID || result.Job.Prompt != "prompt-marker" || result.Job.Provider != "codex" || result.Commands == nil {
		t.Fatal("job not delivered")
	}
	job := *result.Job
	code, polled, raw = bridge("/internal/v1/codex/poll", codexbridge.Poll{Instance: instance})
	want(t, code, 200, raw)
	if json.Unmarshal(polled.Data, &result) != nil || result.Job != nil {
		t.Fatal("job delivered twice")
	}
	report := codexbridge.Report{RunID: job.ID, Token: job.Token, Event: codexbridge.Event{Sequence: 1, Type: "started"}}
	code, _, raw = bridge("/internal/v1/codex/report", report)
	want(t, code, 200, raw)
	code, _, raw = bridge("/internal/v1/codex/report", report)
	want(t, code, 200, raw)
	report.Event.Text = "changed"
	code, _, raw = bridge("/internal/v1/codex/report", report)
	want(t, code, 409, raw)
	report.Event = codexbridge.Event{Sequence: 3, Type: "completed", Text: "answer-marker"}
	code, _, raw = bridge("/internal/v1/codex/report", report)
	want(t, code, 409, raw)
	report.Event.Sequence = 2
	code, _, raw = bridge("/internal/v1/codex/report", report)
	want(t, code, 200, raw)
	code, _, raw = bridge("/internal/v1/codex/report", report)
	want(t, code, 200, raw)
	code, got, raw = call("GET", "/v1/codex/runs/"+run.ID, "", "")
	want(t, code, 200, raw)
	if json.Unmarshal(got.Data, &run) != nil || run.State != "completed" || run.Answer != "answer-marker" || run.PromptPreview == nil || *run.PromptPreview != "prompt-marker" || run.Tokens != nil {
		t.Fatal("result missing")
	}
	code, got, raw = call("GET", "/v1/codex/runs/"+run.ID+"/events?after=1", "", "")
	want(t, code, 200, raw)
	var ev struct {
		Items []codexbridge.Event `json:"items"`
		Next  int                 `json:"next_after"`
	}
	if json.Unmarshal(got.Data, &ev) != nil || len(ev.Items) != 1 || ev.Next != 2 {
		t.Fatal("event replay wrong")
	}
	newJob := func(key string) codexbridge.Job {
		code, _, raw = call("POST", "/v1/codex/runs", input, key)
		want(t, code, 202, raw)
		code, got, raw = bridge("/internal/v1/codex/poll", codexbridge.Poll{Instance: instance})
		want(t, code, 200, raw)
		var out codexbridge.PollResult
		if json.Unmarshal(got.Data, &out) != nil || out.Job == nil {
			t.Fatal("missing next job")
		}
		return *out.Job
	}
	job = newJob("codex-stop-create")
	code, _, raw = call("POST", "/v1/codex/runs/"+job.ID+"/stop", `{}`, "codex-stop-key")
	want(t, code, 200, raw)
	code, got, raw = bridge("/internal/v1/codex/poll", codexbridge.Poll{Instance: instance, RunID: job.ID, Token: job.Token})
	want(t, code, 200, raw)
	if json.Unmarshal(got.Data, &result) != nil || result.Continue {
		t.Fatal("stop did not cancel lease")
	}
	report = codexbridge.Report{RunID: job.ID, Token: job.Token, Event: codexbridge.Event{Sequence: 1, Type: "completed", Text: "late answer"}}
	code, _, raw = bridge("/internal/v1/codex/report", report)
	want(t, code, 409, raw)
	job = newJob("codex-expiry-create")
	if _, err := root.Pool.Exec(ctx, `UPDATE portway.codex_runs SET lease_until=now()-interval '1 second' WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	code, got, raw = bridge("/internal/v1/codex/poll", codexbridge.Poll{Instance: instance})
	want(t, code, 200, raw)
	if json.Unmarshal(got.Data, &result) != nil || result.Job != nil {
		t.Fatal("uncertain job rerun")
	}
	code, got, raw = call("GET", "/v1/codex/runs/"+job.ID, "", "")
	want(t, code, 200, raw)
	if json.Unmarshal(got.Data, &run) != nil || run.State != "failed" {
		t.Fatal("uncertain dispatch not failed")
	}
	// Exercise Worker -> API -> durable encrypted events with native process fixtures
	// (Go fakes of Codex CLI and Claude Code), or with the real CLIs when explicitly requested.
	if _, err := root.Pool.Exec(ctx, `UPDATE portway.codex_bridge SET last_seen_at=now()-interval '31 seconds' WHERE installation_id=$1`, app.Installation); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("PORTWAY_TEST_NATIVE_CODEX_BINARY") != "" || os.Getenv("PORTWAY_TEST_NATIVE_CLAUDE_BINARY") != "" {
		exerciseNativeAgents(t, cfg, api.URL, call)
	} else {
		exerciseAgentBridge(t, cfg, api.URL, app, root, call, bridge)
	}
	// A revoked originating device invalidates running output before its next heartbeat.
	if _, err := root.Pool.Exec(ctx, `UPDATE portway.codex_bridge SET instance_id=$2,last_seen_at=now() WHERE installation_id=$1`, app.Installation, instance); err != nil {
		t.Fatal(err)
	}
	job = newJob("codex-revoke-create")
	if _, err := root.Pool.Exec(ctx, `UPDATE portway.device_sessions SET revoked_at=now() WHERE id=$1`, one.DeviceID); err != nil {
		t.Fatal(err)
	}
	report = codexbridge.Report{RunID: job.ID, Token: job.Token, Event: codexbridge.Event{Sequence: 1, Type: "completed", Text: "late answer"}}
	code, _, raw = bridge("/internal/v1/codex/report", report)
	want(t, code, 409, raw)
	code, got, raw = bridge("/internal/v1/codex/poll", codexbridge.Poll{Instance: instance, RunID: job.ID, Token: job.Token})
	want(t, code, 200, raw)
	if json.Unmarshal(got.Data, &result) != nil || result.Continue {
		t.Fatal("revoked device kept authority")
	}
	if _, err := root.Pool.Exec(ctx, `UPDATE portway.device_sessions SET revoked_at=NULL WHERE id=$1`, one.DeviceID); err != nil {
		t.Fatal(err)
	}
}

type caller func(method, path, body, key string) (int, response, []byte)

func codexStatus(t *testing.T, call caller) codexbridge.Status {
	t.Helper()
	code, got, raw := call("GET", "/v1/codex/status", "", "")
	want(t, code, 200, raw)
	var status codexbridge.Status
	if err := json.Unmarshal(got.Data, &status); err != nil {
		t.Fatal(err)
	}
	return status
}
func waitStatus(t *testing.T, call caller, what string, ok func(codexbridge.Status) bool) codexbridge.Status {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if s := codexStatus(t, call); ok(s) {
			return s
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("status never reached:", what)
	return codexbridge.Status{}
}
func waitRun(t *testing.T, call caller, id string) codexbridge.Run {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		code, got, raw := call("GET", "/v1/codex/runs/"+id, "", "")
		want(t, code, 200, raw)
		var run codexbridge.Run
		_ = json.Unmarshal(got.Data, &run)
		if run.State == "completed" || run.State == "failed" {
			return run
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("run did not finish")
	return codexbridge.Run{}
}
func provider(s codexbridge.Status, id string) codexbridge.ProviderState {
	for _, p := range s.Providers {
		if p.ID == id {
			return p
		}
	}
	return codexbridge.ProviderState{}
}
func commandDone(id string) func(codexbridge.Status) bool {
	return func(s codexbridge.Status) bool {
		for _, c := range s.Commands {
			if c.ID == id && c.State == "done" {
				return true
			}
		}
		return false
	}
}
func commandID(t *testing.T, got response) (string, string) {
	t.Helper()
	var out struct {
		Command string `json:"command_id"`
		Login   string `json:"login_id"`
	}
	if json.Unmarshal(got.Data, &out) != nil || !secure.UUID(out.Command) {
		t.Fatal("command id missing", string(got.Data))
	}
	return out.Command, out.Login
}
func startWorker(t *testing.T, cfg config.Config, base string, b *codexbridge.Bridge) func() {
	t.Helper()
	worker, err := codexbridge.NewWorker(base, cfg.CodexBridgeKey, b)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Serve(ctx) }()
	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			if err := <-done; err != nil {
				t.Error(err)
			}
		})
	}
}

func exerciseAgentBridge(t *testing.T, cfg config.Config, base string, app, root *storage.Store, call caller, bridge func(string, any) (int, response, []byte)) {
	t.Helper()
	ctx := context.Background()
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	dir := t.TempDir()
	codexHome, claudeHome := filepath.Join(dir, "codex-home"), filepath.Join(dir, "claude-home")
	codexBin, err := fakecli.Install(dir, "codex", codexHome)
	if err != nil {
		t.Fatal(err)
	}
	claudeBin, err := fakecli.Install(dir, "claude", claudeHome)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(codexHome, "auth.json"), []byte("{}"), 0600)
	_ = os.WriteFile(filepath.Join(claudeHome, ".credentials.json"), []byte(`{"claudeAiOauth":{"accessToken":"fake","expiresAt":1}}`), 0600)
	registry, err := codexbridge.OpenRegistry(filepath.Join(dir, "bridge"))
	if err != nil {
		t.Fatal(err)
	}
	b := &codexbridge.Bridge{Workspace: dir, Accounts: registry, ForceInterval: time.Millisecond, Agents: []codexbridge.Agent{
		&codexbridge.Codex{Binary: codexBin, Workspace: dir},
		&codexbridge.Claude{Binary: claudeBin, Workspace: dir, SystemDir: claudeHome, APIBase: "http://127.0.0.1:1"},
	}}
	stop := startWorker(t, cfg, base, b)
	defer stop()
	status := waitStatus(t, call, "bridge snapshot", func(s codexbridge.Status) bool {
		return s.Connected && s.StateUpdatedAt != nil && len(provider(s, "codex").Models) == 2 && provider(s, "codex").Usage != nil
	})
	cp, cl := provider(status, "codex"), provider(status, "claude")
	if !cp.Installed || cp.CLIVersion == nil || len(cp.Accounts) != 1 || !cp.Accounts[0].LoggedIn || !cp.Accounts[0].System || cp.Accounts[0].Email == nil || cp.Usage.Status != "ok" || len(cp.Usage.Windows) != 2 {
		t.Fatal("codex provider snapshot", cp)
	}
	if !cl.Installed || len(cl.Models) != 4 || cl.ActiveAccountID == nil || *cl.ActiveAccountID != "claude-system" {
		t.Fatal("claude provider snapshot", cl)
	}
	var ct []byte
	if err = root.Pool.QueryRow(ctx, `SELECT state_ciphertext FROM portway.codex_bridge WHERE installation_id=$1`, app.Installation).Scan(&ct); err != nil || bytes.Contains(ct, []byte("owner@example.com")) {
		t.Fatal("bridge snapshot not encrypted", err)
	}
	// Provider/model/effort selection, preflight validation and completion metadata.
	for body, wantCode := range map[string]int{
		`{"workspace":"portway","prompt":"p","provider":"gemini"}`:                   400,
		`{"workspace":"portway","prompt":"p","model":"bad model"}`:                   400,
		`{"workspace":"portway","prompt":"p","effort":"turbo"}`:                      400,
		`{"workspace":"portway","prompt":"p","provider":"codex","model":"gpt-nope"}`: 400,
		`{"workspace":"portway","prompt":"p","model":"gpt-test","effort":"medium"}`:  400,
		`{"workspace":"portway","prompt":"p","provider":"claude","unknown":"field"}`: 400,
	} {
		code, _, raw := call("POST", "/v1/codex/runs", body, "agent-invalid-"+secure.ID())
		want(t, code, wantCode, raw)
	}
	code, got, raw := call("POST", "/v1/codex/runs", `{"workspace":"portway","prompt":"Claude\n  run   marker","provider":"claude","model":"haiku","effort":"low"}`, "agent-claude-run")
	want(t, code, 202, raw)
	var run codexbridge.Run
	if json.Unmarshal(got.Data, &run) != nil || run.Provider != "claude" || run.Model == nil || *run.Model != "haiku" || run.Effort != nil {
		t.Fatal("claude admission (unsupported effort is ignored)", string(raw))
	}
	run = waitRun(t, call, run.ID)
	if run.State != "completed" || run.Answer != "answer model=haiku effort=" || run.ResolvedModel == nil || *run.ResolvedModel != "claude-haiku-fake" || run.AccountLabel == nil || *run.AccountLabel != "Claude" || run.Tokens == nil || run.Tokens.Input != 10 || run.Tokens.Output != 4 || run.PromptPreview == nil || *run.PromptPreview != "Claude run marker" {
		t.Fatal("claude run", run)
	}
	claudeRun := run.ID
	code, got, raw = call("POST", "/v1/codex/runs", `{"workspace":"portway","prompt":"codex run","model":"gpt-mini","effort":"high"}`, "agent-codex-run")
	want(t, code, 202, raw)
	_ = json.Unmarshal(got.Data, &run)
	run = waitRun(t, call, run.ID)
	if run.State != "completed" || run.Provider != "codex" || run.Answer != "answer model=gpt-mini effort=high" || *run.ResolvedModel != "gpt-mini" || *run.AccountLabel != "ChatGPT" || run.Tokens.Input != 11 || run.Tokens.Output != 7 {
		t.Fatal("codex run", run)
	}
	code, got, raw = call("GET", "/v1/codex/runs?provider=claude", "", "")
	want(t, code, 200, raw)
	var list struct {
		Items []codexbridge.Run `json:"items"`
	}
	if json.Unmarshal(got.Data, &list) != nil || len(list.Items) != 1 || list.Items[0].ID != claudeRun || list.Items[0].PromptPreview == nil || list.Items[0].Prompt != "" || list.Items[0].Answer != "" {
		t.Fatal("provider filter / preview", string(raw))
	}
	code, _, raw = call("GET", "/v1/codex/runs?provider=gemini", "", "")
	want(t, code, 400, raw)
	code, got, raw = call("GET", "/v1/codex/runs?state=failed&provider=codex", "", "")
	want(t, code, 200, raw)
	// A limit error during a run is a typed failure; the event carries the code, not CLI text.
	code, got, raw = call("POST", "/v1/codex/runs", `{"workspace":"portway","prompt":"limit","provider":"claude"}`, "agent-claude-limit")
	want(t, code, 202, raw)
	_ = json.Unmarshal(got.Data, &run)
	run = waitRun(t, call, run.ID)
	if run.State != "failed" || run.ErrorCode == nil || *run.ErrorCode != "AI_LIMIT_EXHAUSTED" {
		t.Fatal("limit failure", run)
	}
	code, got, raw = call("GET", "/v1/codex/runs/"+run.ID+"/events", "", "")
	want(t, code, 200, raw)
	if !bytes.Contains(raw, []byte(`"type":"failed","code":"AI_LIMIT_EXHAUSTED"`)) || bytes.Contains(raw, []byte("usage limit reached")) {
		t.Fatal("failed event", string(raw))
	}
	// Usage refresh command: exhausted snapshot blocks admission with reset details.
	if err = os.WriteFile(filepath.Join(codexHome, "limits.json"), []byte(`{"ordinaryUsageAllowed":false,"rateLimits":{"primary":{"usedPercent":100,"windowDurationMins":300,"resetsAt":4102444800},"secondary":{"usedPercent":31,"windowDurationMins":10080,"resetsAt":4102444800},"rateLimitReachedType":"rate_limit_reached"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	code, got, raw = call("POST", "/v1/codex/usage/refresh", `{}`, "agent-usage-refresh")
	want(t, code, 202, raw)
	refresh, _ := commandID(t, got)
	status = waitStatus(t, call, "usage refresh", func(s codexbridge.Status) bool {
		u := provider(s, "codex").Usage
		return commandDone(refresh)(s) && u != nil && u.Status == "exhausted"
	})
	if u := provider(status, "codex").Usage; u.ResetAt == nil || u.ResetAt.Year() != 2100 {
		t.Fatal("reset_at", u)
	}
	code, got, raw = call("POST", "/v1/codex/runs", `{"workspace":"portway","prompt":"p"}`, "agent-codex-exhausted")
	want(t, code, 429, raw)
	if got.Error.Code != "AI_LIMIT_EXHAUSTED" || !bytes.Contains(raw, []byte(`"details":{"provider":"codex","reset_at":"2100-01-01T00:00:00Z"}`)) {
		t.Fatal("limit preflight", string(raw))
	}
	// Claude subscription login from the phone: URL + pasted code, isolated new profile.
	code, got, raw = call("POST", "/v1/codex/accounts/login", `{"provider":"claude","method":"phone"}`, "agent-login-claude")
	want(t, code, 202, raw)
	_, login := commandID(t, got)
	if !secure.UUID(login) {
		t.Fatal("login id")
	}
	code, got, raw = call("POST", "/v1/codex/accounts/login", `{"provider":"codex","method":"phone"}`, "agent-login-second")
	want(t, code, 409, raw)
	if got.Error.Code != "AI_LOGIN_ACTIVE" {
		t.Fatal("concurrent login", string(raw))
	}
	status = waitStatus(t, call, "claude login waiting", func(s codexbridge.Status) bool {
		return s.Login != nil && s.Login.ID == login && s.Login.State == "waiting_user"
	})
	if l := status.Login; l.URL == nil || !strings.HasPrefix(*l.URL, "https://claude.com/cai/oauth/authorize") || !l.NeedsCode || l.Provider != "claude" || l.Method != "phone" || l.ExpiresAt == nil {
		t.Fatal("login view", l)
	}
	code, _, raw = call("POST", "/v1/codex/accounts/login/"+secure.ID()+"/code", `{"code":"x"}`, "agent-code-unknown")
	want(t, code, 404, raw)
	code, _, raw = call("POST", "/v1/codex/accounts/login/"+login+"/code", `{"code":"has inner space"}`, "agent-code-invalid")
	want(t, code, 400, raw)
	code, got, raw = call("POST", "/v1/codex/accounts/login/"+login+"/code", `{"code":" good-code#fake-state\n"}`, "agent-code")
	want(t, code, 202, raw)
	codeCommand, _ := commandID(t, got)
	status = waitStatus(t, call, "claude login completed", func(s codexbridge.Status) bool {
		return s.Login != nil && s.Login.ID == login && s.Login.State == "completed" && len(provider(s, "claude").Accounts) == 2
	})
	newAccount := *status.Login.AccountID
	if cl = provider(status, "claude"); *cl.ActiveAccountID != newAccount || !cl.Accounts[1].LoggedIn || cl.Accounts[1].Label != "Claude 2" || cl.Accounts[1].System {
		t.Fatal("new claude account", cl)
	}
	var payload []byte
	var state string
	if err = root.Pool.QueryRow(ctx, `SELECT state,payload_ciphertext FROM portway.codex_commands WHERE id=$1`, codeCommand).Scan(&state, &payload); err != nil || payload != nil || state != "done" {
		t.Fatal("login code kept after delivery", state, err)
	}
	var leaked bool
	if err = root.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM portway.request_keys WHERE response_body_safe::text LIKE '%good-code%')`).Scan(&leaked); err != nil || leaked {
		t.Fatal("login code persisted in replay store", err)
	}
	code, got, raw = call("POST", "/v1/codex/accounts/claude-system/activate", `{"provider":"claude"}`, "agent-activate")
	want(t, code, 202, raw)
	activate, _ := commandID(t, got)
	waitStatus(t, call, "activate system", func(s codexbridge.Status) bool {
		return commandDone(activate)(s) && *provider(s, "claude").ActiveAccountID == "claude-system"
	})
	code, got, raw = call("DELETE", "/v1/codex/accounts/claude-system?provider=claude", "", "agent-remove-system")
	want(t, code, 409, raw)
	if got.Error.Code != "AI_ACCOUNT_SYSTEM" {
		t.Fatal("system account removal", string(raw))
	}
	code, _, raw = call("POST", "/v1/codex/accounts/codex-system/activate", `{"provider":"claude"}`, "agent-activate-mismatch")
	want(t, code, 400, raw)
	code, _, raw = call("DELETE", "/v1/codex/accounts/claude-nobody?provider=claude", "", "agent-remove-missing")
	want(t, code, 404, raw)
	code, got, raw = call("DELETE", "/v1/codex/accounts/"+newAccount+"?provider=claude", "", "agent-remove")
	want(t, code, 202, raw)
	remove, _ := commandID(t, got)
	waitStatus(t, call, "remove account", func(s codexbridge.Status) bool {
		return commandDone(remove)(s) && len(provider(s, "claude").Accounts) == 1
	})
	// ChatGPT device-code login for Codex, then a cancelled Claude login.
	code, got, raw = call("POST", "/v1/codex/accounts/login", `{"provider":"codex","method":"phone"}`, "agent-login-codex")
	want(t, code, 202, raw)
	_, login = commandID(t, got)
	status = waitStatus(t, call, "codex device code", func(s codexbridge.Status) bool {
		return s.Login != nil && s.Login.ID == login && s.Login.State == "waiting_user"
	})
	if l := status.Login; l.UserCode == nil || *l.UserCode != "ABCD-12345" || l.URL == nil || *l.URL != "https://auth.openai.com/codex/device" || l.NeedsCode {
		t.Fatal("device code view", l)
	}
	code, got, raw = call("POST", "/v1/codex/accounts/login/"+login+"/code", `{"code":"x"}`, "agent-code-codex")
	want(t, code, 409, raw)
	pending, _ := os.ReadDir(filepath.Join(registry.Root, "accounts", "codex"))
	if len(pending) != 1 {
		t.Fatal("pending codex profile", len(pending))
	}
	if err = fakecli.Approve(filepath.Join(registry.Root, "accounts", "codex", pending[0].Name())); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, call, "codex login completed", func(s codexbridge.Status) bool {
		u := provider(s, "codex").Usage
		return s.Login != nil && s.Login.ID == login && s.Login.State == "completed" && len(provider(s, "codex").Accounts) == 2 && u != nil && u.Status == "ok"
	})
	code, got, raw = call("POST", "/v1/codex/accounts/login", `{"provider":"claude","method":"phone"}`, "agent-login-cancel")
	want(t, code, 202, raw)
	_, login = commandID(t, got)
	waitStatus(t, call, "login waiting before cancel", func(s codexbridge.Status) bool {
		return s.Login != nil && s.Login.ID == login && s.Login.State == "waiting_user"
	})
	code, _, raw = call("POST", "/v1/codex/accounts/login/"+login+"/cancel", `{}`, "agent-cancel")
	want(t, code, 202, raw)
	waitStatus(t, call, "login cancelled", func(s codexbridge.Status) bool {
		return s.Login != nil && s.Login.ID == login && s.Login.State == "cancelled"
	})
	// Internal endpoints: authentication, strict validation, size limit and ack replay.
	code, _, raw = request(t, http.DefaultClient, base, "POST", "/internal/v1/codex/state", `{"instance_id":"`+secure.ID()+`","providers":[],"login":null}`, "", "", "", map[string]string{"X-Portway-Bridge-Key": "wrong-key-wrong-key-wrong-key-wrong"})
	want(t, code, 403, raw)
	code, _, raw = bridge("/internal/v1/codex/state", map[string]any{"instance_id": secure.ID(), "providers": []any{map[string]any{"id": "gemini"}}, "login": nil})
	want(t, code, 400, raw)
	models := []codexbridge.Model{}
	for i := 0; i < 64; i++ {
		models = append(models, codexbridge.Model{ID: fmt.Sprintf("m%d", i), Name: "m", Description: strings.Repeat("d", 1000), Efforts: []string{}})
	}
	big := codexbridge.Snapshot{Instance: secure.ID(), Providers: []codexbridge.ProviderState{{ID: "codex", Name: "Codex", Accounts: []codexbridge.Account{}, Models: models}, {ID: "claude", Name: "Claude", Accounts: []codexbridge.Account{}, Models: models}}}
	code, _, raw = bridge("/internal/v1/codex/state", big)
	want(t, code, 413, raw)
	code, _, raw = bridge("/internal/v1/codex/commands/"+secure.ID()+"/ack", codexbridge.Ack{State: "done"})
	want(t, code, 404, raw)
	code, _, raw = bridge("/internal/v1/codex/commands/"+remove+"/ack", codexbridge.Ack{State: "done"})
	want(t, code, 200, raw)
	failed := "AI_ACCOUNT_BUSY"
	code, _, raw = bridge("/internal/v1/codex/commands/"+remove+"/ack", codexbridge.Ack{State: "failed", ErrorCode: &failed})
	want(t, code, 409, raw)
	code, _, raw = bridge("/internal/v1/codex/commands/"+remove+"/ack", map[string]any{"state": "done", "error_code": "lower"})
	want(t, code, 400, raw)
	stop()
	if _, err = root.Pool.Exec(ctx, `UPDATE portway.codex_bridge SET last_seen_at=now()-interval '31 seconds' WHERE installation_id=$1`, app.Installation); err != nil {
		t.Fatal(err)
	}
	code, got, raw = call("POST", "/v1/codex/accounts/login", `{"provider":"claude","method":"computer"}`, "agent-login-offline")
	want(t, code, 503, raw)
	if status = codexStatus(t, call); status.Connected || len(status.Providers) != 2 {
		t.Fatal("offline status keeps last snapshot", status.Connected)
	}
	t.Log("Agent Bridge v2 queue -> fake Codex/Claude CLIs -> encrypted state/commands/results PASS")
}

// exerciseNativeAgents runs the real CLIs on this laptop (opt-in, owner-approved usage only).
func exerciseNativeAgents(t *testing.T, cfg config.Config, base string, call caller) {
	t.Helper()
	workspace := os.Getenv("PORTWAY_TEST_NATIVE_CODEX_WORKSPACE")
	if !filepath.IsAbs(workspace) {
		t.Fatal("native test requires explicit absolute workspace")
	}
	registry, err := codexbridge.OpenRegistry(filepath.Join(t.TempDir(), "bridge"))
	if err != nil {
		t.Fatal(err)
	}
	b := &codexbridge.Bridge{Workspace: workspace, Accounts: registry, Agents: []codexbridge.Agent{
		&codexbridge.Codex{Binary: os.Getenv("PORTWAY_TEST_NATIVE_CODEX_BINARY"), Workspace: workspace},
		&codexbridge.Claude{Binary: os.Getenv("PORTWAY_TEST_NATIVE_CLAUDE_BINARY"), Workspace: workspace},
	}}
	stop := startWorker(t, cfg, base, b)
	defer stop()
	status := waitStatus(t, call, "native snapshot", func(s codexbridge.Status) bool { return s.Connected && s.StateUpdatedAt != nil })
	for _, p := range status.Providers {
		usage, windows := "null", 0
		if p.Usage != nil {
			usage, windows = p.Usage.Status, len(p.Usage.Windows)
		}
		logged := false
		for _, a := range p.Accounts {
			logged = logged || (a.Active && a.LoggedIn)
		}
		t.Logf("native provider=%s installed=%t version=%v logged_in=%t models=%d usage=%s windows=%d", p.ID, p.Installed, p.CLIVersion != nil, logged, len(p.Models), usage, windows)
	}
	for _, p := range strings.Split(os.Getenv("PORTWAY_TEST_NATIVE_RUNS"), ",") {
		if p == "" {
			continue
		}
		body := `{"workspace":"portway","prompt":"Ответь одним словом: ок","provider":"` + p + `"}`
		if p == "claude" {
			body = `{"workspace":"portway","prompt":"Ответь одним словом: ок","provider":"claude","model":"haiku"}`
		}
		code, got, raw := call("POST", "/v1/codex/runs", body, "native-run-"+p)
		if code != 202 {
			t.Logf("native admission provider=%s status=%d code=%s", p, code, got.Error.Code)
			continue
		}
		var run codexbridge.Run
		_ = json.Unmarshal(got.Data, &run)
		run = waitRun(t, call, run.ID)
		t.Logf("native run provider=%s state=%s error=%v resolved_model=%v tokens=%v answer_len=%d", p, run.State, deref(run.ErrorCode), deref(run.ResolvedModel), run.Tokens, len(run.Answer))
		_ = raw
	}
}
func deref(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}
