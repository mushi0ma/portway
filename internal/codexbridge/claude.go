package codexbridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Claude drives the official Claude Code CLI (Claude subscription login).
type Claude struct {
	Binary, Workspace string
	// APIBase is the Anthropic OAuth API origin used for usage/profile reads (tests override it).
	APIBase string
	Client  *http.Client
	// SystemDir overrides the default ~/.claude location of the system login (tests).
	SystemDir string
}

// Fixed aliases accepted by `claude --model`; "default" keeps the account's default model.
var claudeModels = []Model{
	{ID: "default", Name: "По умолчанию", Description: "Модель Claude Code по умолчанию для этого аккаунта", Default: true, Efforts: []string{}},
	{ID: "opus", Name: "Claude Opus", Description: "Самая сильная модель Claude для сложных задач", Efforts: []string{"low", "medium", "high"}},
	{ID: "sonnet", Name: "Claude Sonnet", Description: "Сбалансированная модель Claude для повседневных задач", Efforts: []string{"low", "medium", "high"}},
	{ID: "haiku", Name: "Claude Haiku", Description: "Быстрая и экономная модель Claude", Efforts: []string{}},
}

// Read-only built-in tools; no Bash, edits, web or MCP servers.
const claudeTools = "Read,Grep,Glob"

func (c *Claude) ID() string      { return ProviderClaude }
func (c *Claude) Name() string    { return "Claude Code" }
func (c *Claude) Installed() bool { return c.Binary != "" }
func (c *Claude) configDir(p Profile) string {
	if p.Dir != "" {
		return p.Dir
	}
	if c.SystemDir != "" {
		return c.SystemDir
	}
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); filepath.IsAbs(v) {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}
func (c *Claude) env(p Profile, extra ...string) []string {
	if p.Dir != "" {
		return nativeEnv(append([]string{"CLAUDE_CONFIG_DIR=" + p.Dir}, extra...)...)
	}
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		extra = append([]string{"CLAUDE_CONFIG_DIR=" + v}, extra...)
	}
	return nativeEnv(extra...)
}
func (c *Claude) Version(ctx context.Context) *string {
	if !c.Installed() {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := native(ctx, c.Binary, c.Workspace, nativeEnv(), "--version").Output()
	if err != nil {
		return nil
	}
	return cliVersion(string(out))
}

type claudeStatus struct {
	LoggedIn     bool    `json:"loggedIn"`
	AuthMethod   string  `json:"authMethod"`
	Email        *string `json:"email"`
	Subscription *string `json:"subscriptionType"`
}

// status runs `claude auth status` (JSON; exit code 1 when logged out).
func (c *Claude) status(ctx context.Context, p Profile) (claudeStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var s claudeStatus
	cmd := native(ctx, c.Binary, c.Workspace, c.env(p), "auth", "status", "--json")
	out, _ := cmd.Output()
	if err := json.Unmarshal(out, &s); err != nil {
		return s, errors.New("claude auth status unavailable")
	}
	// Only a Claude subscription login is used; API-key billing is never selected.
	s.LoggedIn = s.LoggedIn && s.AuthMethod == "claude.ai"
	return s, nil
}
func (c *Claude) Check(ctx context.Context, p Profile) bool {
	s, err := c.status(ctx, p)
	return err == nil && s.LoggedIn
}
func (c *Claude) Inspect(ctx context.Context, p Profile, usage, models bool) (Inspection, error) {
	out := Inspection{}
	if models {
		out.Models = append([]Model(nil), claudeModels...)
	}
	s, err := c.status(ctx, p)
	if err != nil {
		if usage {
			out.Usage = &Usage{Status: "unknown", CheckedAt: time.Now().UTC(), Windows: []Window{}}
		}
		return out, err
	}
	out.LoggedIn = s.LoggedIn
	if s.Email != nil && *s.Email != "" && plain(*s.Email, 254) {
		out.Email = s.Email
	}
	if s.Subscription != nil && *s.Subscription != "" && plain(*s.Subscription, 40) {
		out.Plan = s.Subscription
	}
	if !out.LoggedIn {
		return out, nil
	}
	token, subscription := c.credentials(p)
	if out.Plan == nil && subscription != "" {
		out.Plan = &subscription
	}
	if !usage {
		return out, nil
	}
	out.Usage = &Usage{Status: "unknown", CheckedAt: time.Now().UTC(), Windows: []Window{}}
	if token == "" {
		return out, nil
	}
	if raw, err := c.oauthGet(ctx, token, "usage"); err == nil {
		out.Usage = ClaudeUsage(raw, time.Now())
	}
	if raw, err := c.oauthGet(ctx, token, "profile"); err == nil {
		if plan := ClaudePlan(raw); plan != "" {
			out.Plan = &plan
		}
	}
	return out, nil
}

// credentials reads the OAuth access token of a profile; the token never leaves this process.
func (c *Claude) credentials(p Profile) (string, string) {
	raw, err := os.ReadFile(filepath.Join(c.configDir(p), ".credentials.json"))
	if err != nil || len(raw) > 1<<20 {
		return "", ""
	}
	var creds struct {
		OAuth struct {
			AccessToken  string `json:"accessToken"`
			ExpiresAt    int64  `json:"expiresAt"`
			Subscription string `json:"subscriptionType"`
		} `json:"claudeAiOauth"`
	}
	if json.Unmarshal(raw, &creds) != nil {
		return "", ""
	}
	sub := ""
	if plain(creds.OAuth.Subscription, 40) {
		sub = creds.OAuth.Subscription
	}
	if creds.OAuth.ExpiresAt > 0 && time.UnixMilli(creds.OAuth.ExpiresAt).Before(time.Now()) {
		return "", sub
	}
	return creds.OAuth.AccessToken, sub
}
func (c *Claude) oauthGet(ctx context.Context, token, path string) ([]byte, error) {
	base := c.APIBase
	if base == "" {
		base = "https://api.anthropic.com"
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", base+"/api/oauth/"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "portway-bridge/2")
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Claude OAuth API unavailable")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil || res.StatusCode != 200 {
		return nil, errors.New("Claude OAuth API rejected request")
	}
	return body, nil
}

// lockedBuffer collects bounded CLI output from concurrent stdout/stderr writers.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buf.Len() < 64<<10 {
		b.buf.Write(p[:min(len(p), 64<<10-b.buf.Len())])
	}
	return len(p), nil
}
func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Login runs `claude auth login --claudeai`. Phone: no laptop browser, the phone pastes the
// code shown by platform.claude.com. Computer: the laptop browser completes the callback.
func (c *Claude) Login(ctx context.Context, p Profile, f *LoginFlow) string {
	method := f.Method()
	extra := []string{"BROWSER=/bin/true"}
	if method == "computer" {
		extra = displayEnv()
	}
	cmd := native(ctx, c.Binary, c.Workspace, c.env(p, extra...), "auth", "login", "--claudeai")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "LOGIN_FAILED"
	}
	output := &lockedBuffer{}
	cmd.Stdout, cmd.Stderr = output, output
	if err = cmd.Start(); err != nil {
		return "LOGIN_FAILED"
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	defer stdin.Close()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	shown, sent := false, false
	var deadline <-chan time.Time
	for {
		select {
		case <-tick.C:
			if text := output.String(); !shown {
				// Wait for the line end so a partially written URL is never reported.
				if link := LoginURL(text); link != "" && strings.Contains(text[strings.Index(text, link)+len(link):], "\n") {
					shown = true
					f.Set(func(l *Login) { l.State, l.URL, l.NeedsCode = "waiting_user", SafeLoginURL(link), method == "phone" })
				}
			}
		case code := <-f.Codes():
			if sent || !shown {
				continue
			}
			sent = true
			if _, err = io.WriteString(stdin, code+"\n"); err != nil {
				return "CODE_REJECTED"
			}
			f.Set(func(l *Login) { l.State, l.NeedsCode = "verifying", false })
			deadline = time.After(90 * time.Second)
		case err := <-exited:
			if err != nil {
				if sent {
					return "CODE_REJECTED"
				}
				if ctx.Err() != nil {
					return "LOGIN_CANCELLED"
				}
				return "LOGIN_FAILED"
			}
			f.Set(func(l *Login) { l.State = "verifying" })
			if !c.Check(context.WithoutCancel(ctx), p) {
				return "LOGIN_FAILED"
			}
			return ""
		case <-deadline:
			return "CODE_REJECTED"
		case <-ctx.Done():
			<-exited
			return "LOGIN_CANCELLED"
		}
	}
}
func (c *Claude) Logout(ctx context.Context, p Profile) error {
	if p.System {
		return errors.New("system login is never logged out by the bridge")
	}
	return native(ctx, c.Binary, c.Workspace, c.env(p), "auth", "logout").Run()
}

// Run executes one non-interactive Claude Code request with read-only tools only.
func (c *Claude) Run(ctx context.Context, p Profile, job Job, emit func(Event) error) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 9*time.Minute)
	defer cancel()
	args := []string{"-p", "--output-format", "stream-json", "--verbose", "--tools", claudeTools, "--allowedTools", claudeTools, "--permission-mode", "dontAsk", "--strict-mcp-config", "--no-session-persistence", "--safe-mode", "--restricted"}
	if job.Model != "" && job.Model != "default" && ValidModel(job.Model) {
		args = append(args, "--model", job.Model)
	}
	if ValidEffort(job.Effort) {
		args = append(args, "--effort", job.Effort)
	}
	res := Result{Code: "AI_RUN_FAILED"}
	cmd := native(ctx, c.Binary, c.Workspace, c.env(p), args...)
	cmd.Stdin = strings.NewReader(job.Prompt)
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return res, errors.New("Claude process unavailable")
	}
	if err = cmd.Start(); err != nil {
		return res, errors.New("Claude process unavailable")
	}
	seq := 0
	send := func(kind, text string) error { seq++; return emit(Event{Sequence: seq, Type: kind, Text: text}) }
	if err = send("started", ""); err != nil {
		cancel()
		_ = cmd.Wait()
		return res, err
	}
	scanner := bufio.NewScanner(io.LimitReader(stdout, 8<<20))
	scanner.Buffer(make([]byte, 4096), 2<<20)
	answer, reason := "", ""
	complete, failed, limited := false, false, false
	bytes := 0
	for scanner.Scan() {
		bytes += len(scanner.Bytes()) + 1
		var event struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
			Model   string `json:"model"`
			IsError bool   `json:"is_error"`
			Result  string `json:"result"`
			Message struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
			RateLimit struct {
				Status string `json:"status"`
			} `json:"rate_limit_info"`
			Usage *struct {
				Input       int64 `json:"input_tokens"`
				CacheCreate int64 `json:"cache_creation_input_tokens"`
				CacheRead   int64 `json:"cache_read_input_tokens"`
				Output      int64 `json:"output_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		switch event.Type {
		case "system":
			if event.Subtype == "init" && ValidModel(event.Model) {
				model := event.Model
				res.Meta.ResolvedModel = &model
			}
		case "rate_limit_event":
			limited = limited || event.RateLimit.Status == "rejected"
		case "assistant":
			for _, part := range event.Message.Content {
				text := ""
				if part.Type == "text" && strings.TrimSpace(part.Text) != "" && len(part.Text) <= 8192 {
					text = part.Text
				} else if part.Type == "tool_use" {
					text = "Claude читает файлы проекта в режиме read-only"
				}
				if text != "" && seq < 250 {
					if err = send("progress", text); err != nil {
						failed = true
					}
				}
			}
		case "result":
			complete = true
			if event.IsError || event.Subtype != "success" {
				failed = true
				reason = Classify(event.Result + " " + event.Subtype)
			} else {
				answer = event.Result
			}
			if u := event.Usage; u != nil && u.Input >= 0 && u.Output >= 0 && u.CacheCreate >= 0 && u.CacheRead >= 0 {
				res.Meta.Tokens = &Tokens{Input: u.Input + u.CacheCreate + u.CacheRead, Output: u.Output}
			}
		}
		if failed && err != nil {
			break
		}
	}
	if failed || scanner.Err() != nil || bytes >= 8<<20 {
		cancel()
	}
	waitErr := cmd.Wait()
	if err != nil {
		return res, err
	}
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		res.Code = "AI_TIMEOUT"
	case reason != "":
		res.Code = reason
	case limited:
		res.Code = "AI_LIMIT_EXHAUSTED"
	}
	if failed || scanner.Err() != nil || waitErr != nil || !complete || strings.TrimSpace(answer) == "" || len(answer) > 65536 || ctx.Err() != nil {
		return res, errors.New("Claude execution failed or was cancelled")
	}
	encoded, _ := json.Marshal(Event{Sequence: 256, Type: "completed", Text: answer, Meta: &res.Meta})
	if len(encoded) > 69984 {
		return res, errors.New("Claude answer exceeds encoded event limit")
	}
	res.Answer, res.Code = answer, ""
	return res, nil
}
