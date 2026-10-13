package codexbridge

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"portway/internal/secure"
)

// Opt-in checks against the real CLIs on the owner's laptop. They read account
// state without inference, start logins only in temporary profiles and cancel them,
// and never log out the default ~/.codex or ~/.claude logins. Secrets/emails are not logged.
//
//	PORTWAY_REAL_CLI=1 go test -run TestRealCLI -v ./internal/codexbridge
//	PORTWAY_REAL_CLI_RUN=codex|claude additionally performs ONE tiny real run.
func realAgents(t *testing.T) (*Codex, *Claude, string) {
	if os.Getenv("PORTWAY_REAL_CLI") != "1" {
		t.Skip("set PORTWAY_REAL_CLI=1 to use the real CLIs")
	}
	workspace := t.TempDir()
	codex, _ := exec.LookPath("codex")
	claude, _ := exec.LookPath("claude")
	return &Codex{Binary: codex, Workspace: workspace}, &Claude{Binary: claude, Workspace: workspace}, workspace
}
func describe(t *testing.T, name string, info Inspection, err error) {
	t.Helper()
	usage, reset, windows := "null", "null", []string{}
	if info.Usage != nil {
		usage = info.Usage.Status
		if info.Usage.ResetAt != nil {
			reset = info.Usage.ResetAt.Format(time.RFC3339)
		}
		for _, w := range info.Usage.Windows {
			windows = append(windows, w.ID+"="+strconv.FormatFloat(w.UsedPercent, 'f', -1, 64))
		}
	}
	plan := "null"
	if info.Plan != nil {
		plan = *info.Plan
	}
	t.Logf("%s: err=%v logged_in=%t email_present=%t plan=%s models=%d usage=%s reset_at=%s windows=%v", name, err != nil, info.LoggedIn, info.Email != nil, plan, len(info.Models), usage, reset, windows)
}

func TestRealCLIInspect(t *testing.T) {
	codex, claude, _ := realAgents(t)
	ctx := context.Background()
	system := Profile{System: true}
	t.Logf("codex version=%v claude version=%v", deref(codex.Version(ctx)), deref(claude.Version(ctx)))
	info, err := codex.Inspect(ctx, system, true, true)
	describe(t, "codex-system", info, err)
	for _, m := range info.Models {
		t.Logf("codex model id=%s default=%t efforts=%v", m.ID, m.Default, m.Efforts)
	}
	t.Logf("codex check(chatgpt login)=%t", codex.Check(ctx, system))
	info, err = claude.Inspect(ctx, system, true, true)
	describe(t, "claude-system", info, err)
	t.Logf("claude check(subscription login)=%t", claude.Check(ctx, system))
	empty := Profile{Provider: ProviderClaude, ID: "claude-empty", Dir: t.TempDir()}
	info, err = claude.Inspect(ctx, empty, true, true)
	describe(t, "claude-empty-profile", info, err)
	empty = Profile{Provider: ProviderCodex, ID: "codex-empty", Dir: t.TempDir()}
	info, err = codex.Inspect(ctx, empty, true, true)
	describe(t, "codex-empty-profile", info, err)
}

// Starts phone logins in temporary profiles far enough to obtain the URL/code, then cancels.
func TestRealCLILoginStartAndCancel(t *testing.T) {
	codex, claude, workspace := realAgents(t)
	reg, err := OpenRegistry(filepath.Join(t.TempDir(), "bridge"))
	if err != nil {
		t.Fatal(err)
	}
	b := &Bridge{Workspace: workspace, Accounts: reg, Agents: []Agent{codex, claude}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.init(ctx)
	for _, provider := range []string{ProviderCodex, ProviderClaude} {
		login := secure.ID()
		started := time.Now()
		state, code := b.Handle(ctx, Command{ID: secure.ID(), Kind: "login_start", Payload: CommandPayload{LoginID: login, Provider: provider, Method: "phone"}})
		l := b.Snapshot(secure.ID()).Login
		host, path, userCode := "null", "", "null"
		if l != nil && l.URL != nil {
			u, _ := url.Parse(*l.URL)
			host, path = u.Host, u.Path
		}
		if l != nil && l.UserCode != nil {
			userCode = regexp.MustCompile(`[A-Za-z0-9]`).ReplaceAllString(*l.UserCode, "X")
		}
		t.Logf("%s phone login: ack=%s code=%v state=%s url_host=%s url_path=%s user_code_shape=%s needs_code=%t in=%s", provider, state, deref(code), l.State, host, path, userCode, l.NeedsCode, time.Since(started).Round(time.Millisecond))
		state, _ = b.Handle(ctx, Command{ID: secure.ID(), Kind: "login_cancel", Payload: CommandPayload{LoginID: login}})
		l = b.Snapshot(secure.ID()).Login
		entries, _ := os.ReadDir(filepath.Join(reg.Root, "accounts", provider))
		t.Logf("%s cancel: ack=%s final_state=%s leftover_profile_dirs=%d registered=%d", provider, state, l.State, len(entries), len(reg.Profiles(provider))-1)
		if l.State != "cancelled" || len(entries) != 0 {
			t.Fatal("cancel did not clean up", provider)
		}
	}
}

// Exactly one tiny real run (owner-approved): PORTWAY_REAL_CLI_RUN=codex or claude.
func TestRealCLIRun(t *testing.T) {
	codex, claude, _ := realAgents(t)
	var agent Agent
	job := Job{ID: secure.ID(), Prompt: "Ответь одним словом: ок"}
	switch os.Getenv("PORTWAY_REAL_CLI_RUN") {
	case "codex":
		agent = codex
	case "claude":
		agent, job.Model = claude, "haiku"
	default:
		t.Skip("set PORTWAY_REAL_CLI_RUN=codex|claude for one real run")
	}
	if tee := os.Getenv("PORTWAY_REAL_CLI_TEE"); tee != "" {
		// Wrapper keeps the raw JSONL for the evidence log (inspected manually, never published).
		wrapper := filepath.Join(t.TempDir(), "cli")
		script := "#!/bin/sh\n'" + map[bool]string{true: codex.Binary, false: claude.Binary}[agent == Agent(codex)] + "' \"$@\" | tee '" + tee + "'\n"
		if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
		if agent == Agent(codex) {
			codex.Binary = wrapper
		} else {
			claude.Binary = wrapper
		}
	}
	events := 0
	start := time.Now()
	res, err := agent.Run(context.Background(), Profile{System: true}, job, func(Event) error { events++; return nil })
	tokens := "null"
	if res.Meta.Tokens != nil {
		tokens = strconv.FormatInt(res.Meta.Tokens.Input, 10) + "/" + strconv.FormatInt(res.Meta.Tokens.Output, 10)
	}
	t.Logf("%s run: ok=%t code=%s answer=%q resolved_model=%s tokens(in/out)=%s events=%d duration=%s", agent.ID(), err == nil, res.Code, res.Answer, deref(res.Meta.ResolvedModel), tokens, events, time.Since(start).Round(time.Millisecond))
}
func deref(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}
