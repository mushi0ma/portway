package codexbridge

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"portway/internal/codexbridge/fakecli"
	"portway/internal/secure"
)

func testBridge(t *testing.T) (*Bridge, string) {
	t.Helper()
	codex, claude, codexHome, _ := fakeAgents(t)
	reg, err := OpenRegistry(filepath.Join(t.TempDir(), "bridge"))
	if err != nil {
		t.Fatal(err)
	}
	b := &Bridge{Workspace: codex.Workspace, Accounts: reg, Agents: []Agent{codex, claude}}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	b.init(ctx)
	return b, codexHome
}
func waitLogin(t *testing.T, b *Bridge, state string) Login {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if s := b.Snapshot(secure.ID()); s.Login != nil && s.Login.State == state {
			return *s.Login
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("login did not reach", state, b.Snapshot(secure.ID()).Login)
	return Login{}
}
func handle(b *Bridge, kind string, p CommandPayload) (string, string) {
	state, code := b.Handle(context.Background(), Command{ID: secure.ID(), Kind: kind, Payload: p})
	if code == nil {
		return state, ""
	}
	return state, *code
}

func TestBridgeSnapshotAndCommands(t *testing.T) {
	b, codexHome := testBridge(t)
	ctx := context.Background()
	b.RefreshAll(ctx, false)
	s := b.Snapshot(secure.ID())
	if err := s.Validate(); err != nil || len(s.Providers) != 2 {
		t.Fatal("snapshot invalid", err)
	}
	codex, claude := s.Providers[0], s.Providers[1]
	if !codex.Installed || codex.CLIVersion == nil || *codex.CLIVersion != "0.159.3" || len(codex.Accounts) != 1 || !codex.Accounts[0].LoggedIn || codex.Usage == nil || codex.Usage.Status != "ok" || len(codex.Models) != 2 {
		t.Fatal("codex provider", codex)
	}
	if !claude.Installed || len(claude.Models) != 4 || !claude.Accounts[0].LoggedIn || claude.Usage == nil || claude.Usage.Status != "unknown" {
		t.Fatal("claude provider", claude)
	}
	// An exhausted window is picked up by a forced refresh.
	if err := os.WriteFile(filepath.Join(codexHome, "limits.json"), []byte(`{"ordinaryUsageAllowed":false,"rateLimits":{"primary":{"usedPercent":100,"windowDurationMins":300,"resetsAt":1900000000}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	b.cache[ProviderCodex].usageAt = time.Now().Add(-time.Minute)
	if state, code := handle(b, "usage_refresh", CommandPayload{}); state != "done" || code != "" {
		t.Fatal("usage refresh", state, code)
	}
	if u := b.Snapshot(secure.ID()).Providers[0].Usage; u == nil || u.Status != "exhausted" || u.ResetAt == nil {
		t.Fatal("exhausted usage not refreshed", u)
	}
	if state, code := handle(b, "account_remove", CommandPayload{Provider: "codex", AccountID: "codex-system"}); state != "failed" || code != "AI_ACCOUNT_SYSTEM" {
		t.Fatal("system account removable")
	}
	if state, code := handle(b, "account_activate", CommandPayload{Provider: "codex", AccountID: "codex-missing"}); state != "failed" || code != "AI_ACCOUNT_NOT_FOUND" {
		t.Fatal("unknown account activated")
	}
}

func TestBridgeClaudePhoneLogin(t *testing.T) {
	b, _ := testBridge(t)
	ctx := context.Background()
	login := secure.ID()
	if state, code := handle(b, "login_code", CommandPayload{LoginID: login, Code: "x"}); state != "failed" || code != "AI_LOGIN_NOT_ACTIVE" {
		t.Fatal("code accepted without login")
	}
	if state, code := handle(b, "login_start", CommandPayload{LoginID: login, Provider: "claude", Method: "phone"}); state != "done" {
		t.Fatal("login start", code)
	}
	l := waitLogin(t, b, "waiting_user")
	if l.URL == nil || *l.URL != "https://claude.com/cai/oauth/authorize?code=true&client_id=fake&state=fake-state" || !l.NeedsCode || l.UserCode != nil {
		t.Fatal("phone flow url", l)
	}
	if state, _ := handle(b, "login_code", CommandPayload{LoginID: login, Code: "good-code#fake-state"}); state != "done" {
		t.Fatal("code not delivered")
	}
	l = waitLogin(t, b, "completed")
	if l.AccountID == nil || l.URL != nil || l.ErrorCode != nil {
		t.Fatal("completed login", l)
	}
	active := b.Accounts.Active(ProviderClaude)
	if active.ID != *l.AccountID || active.Label != "Claude 2" {
		t.Fatal("new account not active", active)
	}
	if _, err := os.Stat(filepath.Join(active.Dir, ".credentials.json")); err != nil {
		t.Fatal("credentials not in isolated profile")
	}
	b.RefreshAll(ctx, true)
	s := b.Snapshot(secure.ID())
	if len(s.Providers[1].Accounts) != 2 || !s.Providers[1].Accounts[1].Active || !s.Providers[1].Accounts[1].LoggedIn {
		t.Fatal("snapshot accounts", s.Providers[1].Accounts)
	}
	if state, _ := handle(b, "account_activate", CommandPayload{Provider: "claude", AccountID: "claude-system"}); state != "done" || b.Accounts.Active(ProviderClaude).ID != "claude-system" {
		t.Fatal("activate system")
	}
	if state, _ := handle(b, "account_remove", CommandPayload{Provider: "claude", AccountID: active.ID}); state != "done" {
		t.Fatal("remove")
	}
	if _, err := os.Stat(active.Dir); !os.IsNotExist(err) {
		t.Fatal("removed profile kept")
	}
	// A wrong code fails the flow and leaves no profile behind.
	login = secure.ID()
	handle(b, "login_start", CommandPayload{LoginID: login, Provider: "claude", Method: "phone"})
	waitLogin(t, b, "waiting_user")
	handle(b, "login_code", CommandPayload{LoginID: login, Code: "bad-code"})
	l = waitLogin(t, b, "failed")
	if l.ErrorCode == nil || *l.ErrorCode != "CODE_REJECTED" || len(b.Accounts.Profiles(ProviderClaude)) != 1 {
		t.Fatal("rejected code", l)
	}
}

func TestBridgeCodexDeviceLoginAndCancel(t *testing.T) {
	b, _ := testBridge(t)
	login := secure.ID()
	handle(b, "login_start", CommandPayload{LoginID: login, Provider: "codex", Method: "phone"})
	l := waitLogin(t, b, "waiting_user")
	if l.UserCode == nil || *l.UserCode != "ABCD-12345" || l.URL == nil || *l.URL != "https://auth.openai.com/codex/device" || l.NeedsCode {
		t.Fatal("device code", l)
	}
	entries, _ := os.ReadDir(filepath.Join(b.Accounts.Root, "accounts", "codex"))
	if len(entries) != 1 {
		t.Fatal("pending profile missing")
	}
	if err := fakecli.Approve(filepath.Join(b.Accounts.Root, "accounts", "codex", entries[0].Name())); err != nil {
		t.Fatal(err)
	}
	l = waitLogin(t, b, "completed")
	if b.Accounts.Active(ProviderCodex).ID != *l.AccountID {
		t.Fatal("codex account not active")
	}
	// Cancel: the flow becomes cancelled and its profile directory is deleted.
	login = secure.ID()
	handle(b, "login_start", CommandPayload{LoginID: login, Provider: "codex", Method: "phone"})
	waitLogin(t, b, "waiting_user")
	if state, _ := handle(b, "login_cancel", CommandPayload{LoginID: login}); state != "done" {
		t.Fatal("cancel")
	}
	waitLogin(t, b, "cancelled")
	entries, _ = os.ReadDir(filepath.Join(b.Accounts.Root, "accounts", "codex"))
	if len(entries) != 1 || len(b.Accounts.Profiles(ProviderCodex)) != 2 {
		t.Fatal("cancelled profile kept", len(entries))
	}
	// A missing CLI is reported, not crashed on.
	b.Agents[1] = &Claude{}
	if state, code := handle(b, "login_start", CommandPayload{LoginID: secure.ID(), Provider: "claude", Method: "phone"}); state != "failed" || code != "CLI_MISSING" {
		t.Fatal("missing CLI", state)
	}
}
