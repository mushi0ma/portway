package codexbridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodexUsageStates(t *testing.T) {
	now := time.Unix(1791880000, 0)
	exhausted := `{"ordinaryUsageAllowed":false,"rateLimits":{"primary":{"usedPercent":100,"windowDurationMins":300,"resetsAt":1791900408},"secondary":{"usedPercent":31,"windowDurationMins":10080,"resetsAt":1792400000},"credits":{"hasCredits":false,"unlimited":false},"planType":"plus","rateLimitReachedType":"rate_limit_reached"},"rateLimitUpsell":{"title":"You're out of Codex messages"}}`
	u := CodexUsage([]byte(exhausted), now)
	if u.Status != "exhausted" || u.ResetAt == nil || u.ResetAt.Unix() != 1791900408 || len(u.Windows) != 2 || u.Windows[0].ID != "five_hour" || u.Windows[1].ID != "weekly" || *u.Windows[1].WindowMinutes != 10080 {
		t.Fatal("exhausted usage wrong", u)
	}
	u = CodexUsage([]byte(`{"ordinaryUsageAllowed":true,"rateLimits":{"primary":{"usedPercent":85,"windowDurationMins":300},"secondary":null}}`), now)
	if u.Status != "warning" || u.ResetAt != nil || len(u.Windows) != 1 || u.Windows[0].ResetsAt != nil {
		t.Fatal("warning usage wrong", u)
	}
	u = CodexUsage([]byte(`{"rateLimits":{"primary":{"usedPercent":140,"windowDurationMins":60}}}`), now)
	if u.Status != "exhausted" || u.Windows[0].UsedPercent != 100 || u.Windows[0].ID != "primary" || u.ResetAt != nil {
		t.Fatal("clamp/unknown window wrong", u)
	}
	if CodexUsage([]byte(`not json`), now).Status != "unknown" {
		t.Fatal("invalid usage must be unknown")
	}
}
func TestClaudeUsageStates(t *testing.T) {
	now := time.Now()
	sample := `{"five_hour":{"utilization":24.0,"resets_at":"2026-10-13T16:10:00.741387+00:00","limit_dollars":null},"seven_day":{"utilization":4.0,"resets_at":"2026-10-18T15:00:00.741418+00:00"},"seven_day_opus":null,"seven_day_sonnet":{"utilization":81,"resets_at":null},"extra_usage":{"is_enabled":true},"limits":[{"kind":"session","percent":24,"severity":"normal"}]}`
	u := ClaudeUsage([]byte(sample), now)
	if u.Status != "warning" || len(u.Windows) != 3 || u.Windows[2].ID != "weekly_sonnet" || u.Windows[2].ResetsAt != nil || u.Windows[0].ResetsAt.Location() != time.UTC {
		t.Fatal("claude usage wrong", u)
	}
	u = ClaudeUsage([]byte(`{"five_hour":{"utilization":3,"resets_at":null},"limits":[{"kind":"weekly_all","percent":100}]}`), now)
	if u.Status != "exhausted" || u.ResetAt != nil {
		t.Fatal("limits percent ignored", u)
	}
	if ClaudeUsage([]byte(`{}`), now).Status != "unknown" || ClaudeUsage([]byte(`[`), now).Status != "unknown" {
		t.Fatal("empty usage must be unknown")
	}
	if ClaudePlan([]byte(`{"organization":{"organization_type":"claude_pro"}}`)) != "pro" || ClaudePlan([]byte(`{"account":{"has_claude_max":true}}`)) != "max" {
		t.Fatal("plan mapping")
	}
}
func TestLoginURLStripsOSC8(t *testing.T) {
	link := "https://claude.com/cai/oauth/authorize?code=true&client_id=x&redirect_uri=https%3A%2F%2Fplatform.claude.com%2Foauth%2Fcode%2Fcallback&state=abc"
	out := "Opening browser to sign in…\nIf the browser didn't open, visit: \x1b]8;;" + link + "\x07" + link + "\x1b]8;;\x07\nPaste code here if prompted > "
	if got := LoginURL(out); got != link {
		t.Fatal("url", got)
	}
	if LoginURL("visit: \x1b]8;;"+link+"\x1b\\"+link+"\x1b]8;;\x1b\\\n") != link {
		t.Fatal("ST terminated OSC-8")
	}
	if SafeLoginURL(link) == nil || SafeLoginURL("https://evil.example/login") != nil || SafeLoginURL("http://claude.ai/x") != nil || SafeLoginURL("https://claude.ai:8443/x") != nil || SafeLoginURL("https://user@claude.ai/x") != nil {
		t.Fatal("login URL allowlist")
	}
}
func TestClassifyAndVersion(t *testing.T) {
	cases := map[string]string{
		"You've hit your usage limit. Upgrade to Pro or try again at 2:06 PM.": "AI_LIMIT_EXHAUSTED",
		"Claude AI usage limit reached|1760000000":                             "AI_LIMIT_EXHAUSTED",
		"exceeded retry limit, last status: 429 Too Many Requests":             "AI_LIMIT_EXHAUSTED",
		"Invalid API key · Please run /login":                                  "AI_AUTH_REQUIRED",
		"unexpected status 401 Unauthorized":                                   "AI_AUTH_REQUIRED",
		"stream disconnected before completion":                                "",
	}
	for text, want := range cases {
		if got := Classify(text); got != want {
			t.Fatal(text, got)
		}
	}
	if v := cliVersion("codex-cli 0.159.3\n"); v == nil || *v != "0.159.3" {
		t.Fatal("codex version")
	}
	if v := cliVersion("2.1.296 (Claude Code)"); v == nil || *v != "2.1.296" {
		t.Fatal("claude version")
	}
}
func TestSnapshotValidation(t *testing.T) {
	email := "owner@example.com"
	active := "codex-system"
	evil, user := "https://evil.example/x", "ABCD-12345"
	valid := func() Snapshot {
		return Snapshot{Instance: "8a3d2f2e-9d7c-4b1a-9f6e-0c1d2e3f4a5b", Providers: []ProviderState{{ID: "codex", Name: "Codex", Installed: true, ActiveAccountID: &active,
			Accounts: []Account{{ID: active, Label: "ChatGPT", Email: &email, LoggedIn: true, Active: true, System: true}},
			Models:   []Model{{ID: "gpt-6.1-sol", Name: "GPT", Efforts: []string{"low", "xhigh"}}},
			Usage:    &Usage{Status: "exhausted", CheckedAt: time.Now(), Windows: []Window{{ID: "five_hour", UsedPercent: 250}}}}},
			Login: &Login{ID: "8a3d2f2e-9d7c-4b1a-9f6e-0c1d2e3f4a5c", Provider: "codex", Method: "phone", State: "waiting_user", URL: &evil, UserCode: &user, UpdatedAt: time.Now()}}
	}
	s := valid()
	if err := s.Validate(); err != nil || s.Login.URL != nil || s.Providers[0].Usage.Windows[0].UsedPercent != 100 {
		t.Fatal("sanitizing validation failed", err)
	}
	bad := []func(*Snapshot){
		func(s *Snapshot) { s.Instance = "x" },
		func(s *Snapshot) { s.Providers[0].ID = "gemini" },
		func(s *Snapshot) { s.Providers[0].Accounts[0].ID = "claude-system" },
		func(s *Snapshot) { other := "codex-other"; s.Providers[0].ActiveAccountID = &other },
		func(s *Snapshot) { s.Providers[0].Models[0].Efforts = []string{"turbo"} },
		func(s *Snapshot) { s.Providers[0].Usage.Status = "great" },
		func(s *Snapshot) { s.Providers[0].Usage.Windows[0].ID = "Five Hour" },
		func(s *Snapshot) { s.Login.State = "done" },
		func(s *Snapshot) { c := "lower"; s.Login.ErrorCode = &c },
		func(s *Snapshot) { s.Providers[0].Accounts[0].Label = "bad\nlabel" },
		func(s *Snapshot) { s.Providers = append(s.Providers, s.Providers[0]) },
	}
	for i, mutate := range bad {
		s := valid()
		mutate(&s)
		if s.Validate() == nil {
			t.Fatal("invalid snapshot accepted", i)
		}
	}
	raw, _ := json.Marshal(valid())
	if strings.Contains(string(raw), "token") {
		t.Fatal("snapshot shape leaks token fields")
	}
}
func TestPreview(t *testing.T) {
	if Preview("  a\n\tb  c ") != "a b c" {
		t.Fatal("whitespace not collapsed")
	}
	long := Preview(strings.Repeat("я", 300))
	if len([]rune(long)) != 120 || !strings.HasSuffix(long, "…") {
		t.Fatal("preview not bounded", len([]rune(long)))
	}
}
func TestRegistryProfilesAndPermissions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "bridge")
	reg, err := OpenRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if p := reg.Active(ProviderCodex); !p.System || p.ID != "codex-system" || p.Label != "ChatGPT" || p.Dir != "" {
		t.Fatal("system profile", p)
	}
	pending, err := reg.NewPending(ProviderClaude)
	if err != nil || !ValidAccountID(ProviderClaude, pending.ID) || pending.Label != "Claude 2" {
		t.Fatal("pending", pending, err)
	}
	st, _ := os.Stat(pending.Dir)
	if st.Mode().Perm() != 0700 {
		t.Fatal("profile dir mode", st.Mode())
	}
	if len(reg.Profiles(ProviderClaude)) != 1 {
		t.Fatal("pending profile listed before login")
	}
	if err = reg.Commit(pending); err != nil || reg.Active(ProviderClaude).ID != pending.ID {
		t.Fatal("commit", err)
	}
	st, _ = os.Stat(filepath.Join(root, "accounts.json"))
	if st.Mode().Perm() != 0600 {
		t.Fatal("registry mode", st.Mode())
	}
	orphan, _ := reg.NewPending(ProviderCodex)
	reopened, err := OpenRegistry(root)
	if err != nil || reopened.Active(ProviderClaude).ID != pending.ID || len(reopened.Profiles(ProviderClaude)) != 2 {
		t.Fatal("registry not persisted", err)
	}
	if _, err = os.Stat(orphan.Dir); !os.IsNotExist(err) {
		t.Fatal("interrupted login profile not cleaned")
	}
	next, _ := reopened.NewPending(ProviderClaude)
	if next.Label != "Claude 3" {
		t.Fatal("label numbering", next.Label)
	}
	reopened.Discard(next)
	if reopened.Remove(ProviderClaude, "claude-system") == nil || reopened.Activate(ProviderClaude, "claude-missing") == nil {
		t.Fatal("system remove / unknown activate accepted")
	}
	if err = reopened.Remove(ProviderClaude, pending.ID); err != nil || reopened.Active(ProviderClaude).ID != "claude-system" {
		t.Fatal("remove", err)
	}
	if _, err = os.Stat(pending.Dir); !os.IsNotExist(err) {
		t.Fatal("removed profile dir kept")
	}
}
