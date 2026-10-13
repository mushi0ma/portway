package codexbridge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestClaudeRunReadOnlyStreamAndMeta(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "must-not-inherit")
	t.Setenv("PORTWAY_CODEX_BRIDGE_KEY", "must-not-inherit")
	_, claude, _, _ := fakeAgents(t)
	if !claude.Check(context.Background(), Profile{System: true}) {
		t.Fatal("subscription login not detected")
	}
	events := []Event{}
	res, err := claude.Run(context.Background(), Profile{System: true}, Job{Prompt: "hello", Model: "haiku", Effort: "low"}, func(e Event) error { events = append(events, e); return nil })
	if err != nil || res.Answer != "answer model=haiku effort=low" {
		t.Fatal(res, err)
	}
	if res.Meta.ResolvedModel == nil || *res.Meta.ResolvedModel != "claude-haiku-fake" || res.Meta.Tokens == nil || res.Meta.Tokens.Input != 10 || res.Meta.Tokens.Output != 4 {
		t.Fatal("meta not captured", res.Meta)
	}
	if len(events) != 3 || events[0].Type != "started" {
		t.Fatal("unexpected events", events)
	}
	for _, e := range events {
		if strings.Contains(e.Text, "/secret/path") {
			t.Fatal("tool input published")
		}
	}
	res, err = claude.Run(context.Background(), Profile{System: true}, Job{Prompt: "hello", Model: "default"}, func(Event) error { return nil })
	if err != nil || res.Answer != "answer model= effort=" || *res.Meta.ResolvedModel != "claude-default-fake" {
		t.Fatal("default model must omit --model", res, err)
	}
	res, err = claude.Run(context.Background(), Profile{System: true}, Job{Prompt: "limit"}, func(Event) error { return nil })
	if err == nil || res.Code != "AI_LIMIT_EXHAUSTED" {
		t.Fatal("limit not classified", res.Code, err)
	}
	empty := Profile{Provider: ProviderClaude, ID: "claude-empty", Dir: t.TempDir()}
	res, err = claude.Run(context.Background(), empty, Job{Prompt: "hello"}, func(Event) error { return nil })
	if err == nil || res.Code != "AI_AUTH_REQUIRED" || claude.Check(context.Background(), empty) {
		t.Fatal("logged-out profile not classified", res.Code, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err = claude.Run(ctx, Profile{System: true}, Job{Prompt: "sleep"}, func(Event) error { return nil }); err == nil || time.Since(start) > 4*time.Second {
		t.Fatal("cancellation failed")
	}
}

func TestClaudeUsageAndPlanFromOAuthAPI(t *testing.T) {
	_, claude, _, claudeHome := fakeAgents(t)
	var seen []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer good-token" || r.Header.Get("anthropic-beta") != "oauth-2025-04-20" {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/api/oauth/usage" {
			_, _ = w.Write([]byte(`{"five_hour":{"utilization":100.0,"resets_at":"2026-10-13T16:10:00.741387+00:00"},"seven_day":{"utilization":4.0,"resets_at":"2026-10-18T15:00:00+00:00"},"seven_day_opus":null,"limits":[{"kind":"session","percent":100}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"account":{"has_claude_pro":true},"organization":{"organization_type":"claude_max"}}`))
	}))
	defer api.Close()
	claude.APIBase = api.URL
	if err := os.WriteFile(filepath.Join(claudeHome, ".credentials.json"), []byte(`{"claudeAiOauth":{"accessToken":"good-token","expiresAt":`+strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10)+`,"subscriptionType":"pro"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := claude.Inspect(context.Background(), Profile{System: true}, true, true)
	if err != nil || !info.LoggedIn || info.Plan == nil || *info.Plan != "max" || len(info.Models) != 4 {
		t.Fatal("inspection failed", info, err)
	}
	u := info.Usage
	if u == nil || u.Status != "exhausted" || u.ResetAt == nil || u.ResetAt.Format(time.RFC3339) != "2026-10-13T16:10:00Z" || len(u.Windows) != 2 || u.Windows[1].ID != "weekly" {
		t.Fatal("usage not mapped", u)
	}
	// Expired tokens are never sent; usage becomes unknown instead of failing the snapshot.
	if err := os.WriteFile(filepath.Join(claudeHome, ".credentials.json"), []byte(`{"claudeAiOauth":{"accessToken":"good-token","expiresAt":1}}`), 0600); err != nil {
		t.Fatal(err)
	}
	before := len(seen)
	info, _ = claude.Inspect(context.Background(), Profile{System: true}, true, false)
	if info.Usage == nil || info.Usage.Status != "unknown" || len(seen) != before {
		t.Fatal("expired token used", info.Usage)
	}
}
