package codexbridge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"portway/internal/codexbridge/fakecli"
)

func TestMain(m *testing.M) {
	if fakecli.Active() {
		fakecli.Main()
	}
	os.Exit(m.Run())
}

// fakeRunner is a shell fixture for protocol edge cases.
func fakeRunner(t *testing.T, script string) *Codex {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	return &Codex{Binary: bin, Workspace: dir}
}

// fakeAgents installs Go fakes of both CLIs with logged-in system profiles.
func fakeAgents(t *testing.T) (*Codex, *Claude, string, string) {
	t.Helper()
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
	if err = os.WriteFile(filepath.Join(codexHome, "auth.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(claudeHome, ".credentials.json"), []byte(`{"claudeAiOauth":{"accessToken":"system-token","expiresAt":0}}`), 0600); err != nil {
		t.Fatal(err)
	}
	return &Codex{Binary: codexBin, Workspace: dir}, &Claude{Binary: claudeBin, Workspace: dir, APIBase: "http://127.0.0.1:1", SystemDir: claudeHome}, codexHome, claudeHome
}

func TestNativeRunnerProtocolAndEnvironment(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "must-not-inherit")
	t.Setenv("PORTWAY_CODEX_BRIDGE_KEY", "must-not-inherit")
	t.Setenv("PORTWAY_DATABASE_URL", "must-not-inherit")
	runner := fakeRunner(t, `if [ "$1" = login ]; then echo 'Logged in using ChatGPT' >&2; exit; fi
[ -z "$OPENAI_API_KEY$PORTWAY_CODEX_BRIDGE_KEY$PORTWAY_DATABASE_URL" ] || exit 4
case "$*" in *'--sandbox read-only'*'shell_environment_policy.inherit=none'*'-m gpt-x -c model_reasoning_effort="high" -') ;; *) exit 5;; esac
prompt=$(cat)
[ "$prompt" = '$(touch injection-marker)' ] || exit 6
echo '{"type":"item.completed","item":{"type":"reasoning","text":"private-reasoning"}}'
echo '{"type":"item.started","item":{"type":"command_execution","command":"secret command"}}'
echo '{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"private-output"}}'
echo '{"type":"item.completed","item":{"type":"agent_message","text":"answer"}}'
echo '{"type":"turn.completed","usage":{"input_tokens":12,"cached_input_tokens":2,"output_tokens":3}}'
`)
	if !runner.Check(context.Background(), Profile{System: true}) {
		t.Fatal("login check failed")
	}
	events := []Event{}
	res, err := runner.Run(context.Background(), Profile{System: true}, Job{Prompt: "$(touch injection-marker)", Model: "gpt-x", Effort: "high"}, func(e Event) error { events = append(events, e); return nil })
	if err != nil || res.Answer != "answer" || len(events) != 3 {
		t.Fatal(res, events, err)
	}
	if res.Meta.ResolvedModel == nil || *res.Meta.ResolvedModel != "gpt-x" || res.Meta.Tokens == nil || res.Meta.Tokens.Input != 12 || res.Meta.Tokens.Output != 3 {
		t.Fatal("run metadata missing", res.Meta)
	}
	for _, e := range events {
		if strings.Contains(e.Text, "private") || strings.Contains(e.Text, "secret") {
			t.Fatal("private process output published")
		}
	}
	if _, err = os.Stat(filepath.Join(runner.Workspace, "injection-marker")); !os.IsNotExist(err) {
		t.Fatal("prompt executed as shell")
	}
}
func TestNativeRunnerRejectsIncompleteAndCancels(t *testing.T) {
	for _, script := range []string{"echo '{\"type\":\"turn.completed\"}'", "echo '{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"answer\"}}'", "echo invalid-json", "echo '{\"type\":\"error\"}'"} {
		runner := fakeRunner(t, script)
		if res, err := runner.Run(context.Background(), Profile{}, Job{Prompt: "prompt"}, func(Event) error { return nil }); err == nil || res.Code != "AI_RUN_FAILED" {
			t.Fatal("incomplete result accepted", res.Code)
		}
	}
	runner := fakeRunner(t, "sleep 30 &\nwait")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := runner.Run(ctx, Profile{}, Job{Prompt: "prompt"}, func(Event) error { return nil }); err == nil {
		t.Fatal("cancelled process succeeded")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("process group did not cancel promptly")
	}
}
func TestNativeRunnerClassifiesFailures(t *testing.T) {
	for script, want := range map[string]string{
		`echo '{"type":"error","message":"You'"'"'ve hit your usage limit. Upgrade to Pro or try again at 2:06 PM."}'; echo '{"type":"turn.failed","error":{"message":"usage limit"}}'`: "AI_LIMIT_EXHAUSTED",
		`echo '{"type":"error","message":"unexpected status 401 Unauthorized: token expired"}'`:                                                                                         "AI_AUTH_REQUIRED",
		`echo '{"type":"turn.failed","error":{"message":"stream disconnected"}}'`:                                                                                                       "AI_RUN_FAILED",
	} {
		res, err := fakeRunner(t, script).Run(context.Background(), Profile{}, Job{Prompt: "p"}, func(Event) error { return nil })
		if err == nil || res.Code != want {
			t.Fatal("wrong failure code", res.Code, want)
		}
	}
}
func TestCodexProfileEnvironment(t *testing.T) {
	t.Setenv("CODEX_HOME", "")
	runner := fakeRunner(t, `[ "$CODEX_HOME" = "$EXPECTED" ] || [ "$CODEX_HOME" = /profile/codex-abc ] || exit 9
echo '{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}'
echo '{"type":"turn.completed"}'`)
	if _, err := runner.Run(context.Background(), Profile{ID: "codex-abc", Dir: "/profile/codex-abc"}, Job{Prompt: "p"}, func(Event) error { return nil }); err != nil {
		t.Fatal("profile CODEX_HOME not passed", err)
	}
}
func TestBridgeOriginValidation(t *testing.T) {
	reg, err := OpenRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"http://public.example", "https://user:password@example.test", "https://example.test?key=x", "https://example.test/path"} {
		if _, err := NewWorker(origin, strings.Repeat("k", 32), &Bridge{Accounts: reg}); err == nil {
			t.Fatal("unsafe origin accepted", origin)
		}
	}
}

func TestNativeRunnerBoundsEscapedAnswer(t *testing.T) {
	encoded, _ := json.Marshal(map[string]any{"type": "item.completed", "item": map[string]string{"type": "agent_message", "text": strings.Repeat(`"`, 40000)}})
	runner := fakeRunner(t, "cat <<'EVENTS'\n"+string(encoded)+"\n{\"type\":\"turn.completed\"}\nEVENTS\n")
	if _, err := runner.Run(context.Background(), Profile{}, Job{Prompt: "prompt"}, func(Event) error { return nil }); err == nil {
		t.Fatal("encoded answer exceeds database/wire cap")
	}
}

func TestCodexAppServerInspectAndModels(t *testing.T) {
	codex, _, codexHome, _ := fakeAgents(t)
	info, err := codex.Inspect(context.Background(), Profile{Provider: ProviderCodex, ID: "codex-system", System: true}, true, true)
	if err != nil || !info.LoggedIn || info.Email == nil || *info.Email != "owner@example.com" || info.Plan == nil || *info.Plan != "plus" {
		t.Fatal("account not read", info, err)
	}
	if info.Usage == nil || info.Usage.Status != "ok" || len(info.Usage.Windows) != 2 || info.Usage.Windows[0].ID != "five_hour" || info.Usage.Windows[1].ID != "weekly" {
		t.Fatal("usage not mapped", info.Usage)
	}
	if len(info.Models) != 2 || !info.Models[0].Default || len(info.Models[0].Efforts) != 2 || info.Models[0].DefaultEffort == nil {
		t.Fatal("models not mapped (hidden must be omitted)", info.Models)
	}
	if err = os.Remove(filepath.Join(codexHome, "auth.json")); err != nil {
		t.Fatal(err)
	}
	info, err = codex.Inspect(context.Background(), Profile{System: true}, true, false)
	if err != nil || info.LoggedIn || info.Usage != nil || codex.Check(context.Background(), Profile{System: true}) {
		t.Fatal("logged-out account reported as usable", info, err)
	}
}
