// Package fakecli is a test-only stand-in for the Codex CLI and Claude Code.
// Test binaries call Main from TestMain; Install writes a wrapper script that
// re-executes the test binary in fake mode. It is never imported by production code.
package fakecli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Active reports whether the current process was started as a fake CLI.
func Active() bool { return os.Getenv("PORTWAY_FAKE_CLI") != "" }

// Install writes an executable wrapper named codex/claude in dir. home is the
// profile directory used when CODEX_HOME / CLAUDE_CONFIG_DIR is not set.
func Install(dir, kind, home string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(home, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, kind)
	script := fmt.Sprintf("#!/bin/sh\nPORTWAY_FAKE_CLI=%s PORTWAY_FAKE_HOME='%s' exec '%s' \"$@\"\n", kind, home, self)
	return path, os.WriteFile(path, []byte(script), 0700)
}

// Approve simulates the owner finishing the OAuth page for a waiting login in home.
func Approve(home string) error { return os.WriteFile(filepath.Join(home, "approve"), nil, 0600) }

// Main runs the fake CLI and exits.
func Main() {
	for _, kv := range os.Environ() {
		name := strings.SplitN(kv, "=", 2)[0]
		if (strings.HasPrefix(name, "PORTWAY_") && !strings.HasPrefix(name, "PORTWAY_FAKE_")) || strings.HasSuffix(name, "_API_KEY") || strings.HasPrefix(name, "CLAUDECODE") {
			fmt.Fprintln(os.Stderr, "inherited secret environment", name)
			os.Exit(97)
		}
	}
	args := os.Args[1:]
	if os.Getenv("PORTWAY_FAKE_CLI") == "claude" {
		os.Exit(claude(args))
	}
	os.Exit(codex(args))
}
func home(variable string) string {
	if v := os.Getenv(variable); v != "" {
		return v
	}
	return os.Getenv("PORTWAY_FAKE_HOME")
}
func exists(path string) bool { _, err := os.Stat(path); return err == nil }
func has(args []string, s string) bool {
	return strings.Contains(" "+strings.Join(args, " ")+" ", " "+s+" ")
}
func value(args []string, flag string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}
func line(v any) { b, _ := json.Marshal(v); fmt.Println(string(b)) }

func codex(args []string) int {
	dir := home("CODEX_HOME")
	auth := filepath.Join(dir, "auth.json")
	switch {
	case has(args, "--version"):
		fmt.Println("codex-cli 0.159.3")
		return 0
	case len(args) >= 2 && args[0] == "login" && args[1] == "status":
		if exists(auth) {
			fmt.Fprintln(os.Stderr, "Logged in using ChatGPT")
			return 0
		}
		fmt.Fprintln(os.Stderr, "Not logged in")
		return 1
	case len(args) >= 1 && args[0] == "app-server":
		appServer(dir, auth)
		return 0
	case has(args, "exec"):
		if !has(args, "--sandbox read-only") || !has(args, "shell_environment_policy.inherit=none") {
			return 5
		}
		prompt, _ := io.ReadAll(os.Stdin)
		switch strings.TrimSpace(string(prompt)) {
		case "limit":
			line(map[string]any{"type": "error", "message": "You've hit your usage limit. Upgrade to Pro or try again at 2:06 PM."})
			line(map[string]any{"type": "turn.failed", "error": map[string]any{"message": "You've hit your usage limit."}})
			return 1
		case "auth":
			line(map[string]any{"type": "error", "message": "unexpected status 401 Unauthorized"})
			return 1
		case "sleep":
			time.Sleep(30 * time.Second)
			return 0
		}
		effort := ""
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "-c" && strings.HasPrefix(args[i+1], "model_reasoning_effort=") {
				effort = strings.Trim(strings.TrimPrefix(args[i+1], "model_reasoning_effort="), `"`)
			}
		}
		line(map[string]any{"type": "thread.started", "thread_id": "t1"})
		line(map[string]any{"type": "item.completed", "item": map[string]any{"type": "reasoning", "text": "private-reasoning"}})
		line(map[string]any{"type": "item.completed", "item": map[string]any{"type": "agent_message", "text": fmt.Sprintf("answer model=%s effort=%s", value(args, "-m"), effort)}})
		line(map[string]any{"type": "turn.completed", "usage": map[string]any{"input_tokens": 11, "cached_input_tokens": 4, "output_tokens": 7}})
		return 0
	}
	return 2
}

func appServer(dir, auth string) {
	var wmu sync.Mutex
	out := func(v any) {
		wmu.Lock()
		defer wmu.Unlock()
		line(v)
	}
	stop := make(chan struct{})
	var once sync.Once
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		var m struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(scanner.Bytes(), &m) != nil || m.ID == nil {
			continue
		}
		reply := func(result any) { out(map[string]any{"id": *m.ID, "result": result}) }
		fail := func(code int, msg string) {
			out(map[string]any{"id": *m.ID, "error": map[string]any{"code": code, "message": msg}})
		}
		switch m.Method {
		case "initialize":
			reply(map[string]any{"userAgent": "fake"})
		case "account/read":
			if exists(auth) {
				reply(map[string]any{"account": map[string]any{"type": "chatgpt", "email": "owner@example.com", "planType": "plus"}, "requiresOpenaiAuth": true})
			} else {
				reply(map[string]any{"account": nil, "requiresOpenaiAuth": true})
			}
		case "account/rateLimits/read":
			if !exists(auth) {
				fail(-32600, "codex account authentication required to read rate limits")
				continue
			}
			if raw, err := os.ReadFile(filepath.Join(dir, "limits.json")); err == nil {
				out(map[string]any{"id": *m.ID, "result": json.RawMessage(raw)})
				continue
			}
			reset := time.Now().Add(time.Hour).Unix()
			reply(map[string]any{"ordinaryUsageAllowed": true, "rateLimits": map[string]any{"primary": map[string]any{"usedPercent": 42, "windowDurationMins": 300, "resetsAt": reset}, "secondary": map[string]any{"usedPercent": 10, "windowDurationMins": 10080, "resetsAt": reset + 86400}, "rateLimitReachedType": nil}})
		case "model/list":
			efforts := []any{map[string]any{"reasoningEffort": "low", "description": ""}, map[string]any{"reasoningEffort": "high", "description": ""}}
			reply(map[string]any{"data": []any{
				map[string]any{"id": "gpt-test", "model": "gpt-test", "displayName": "GPT Test", "description": "Fake default model", "isDefault": true, "hidden": false, "supportedReasoningEfforts": efforts, "defaultReasoningEffort": "low"},
				map[string]any{"id": "gpt-mini", "model": "gpt-mini", "displayName": "GPT Mini", "description": "Fake small model", "isDefault": false, "hidden": false, "supportedReasoningEfforts": efforts, "defaultReasoningEffort": "low"},
				map[string]any{"id": "gpt-hidden", "model": "gpt-hidden", "displayName": "Hidden", "description": "", "isDefault": false, "hidden": true, "supportedReasoningEfforts": efforts, "defaultReasoningEffort": "low"},
			}})
		case "account/login/start":
			var p struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(m.Params, &p)
			if p.Type == "chatgptDeviceCode" {
				reply(map[string]any{"type": p.Type, "loginId": "fake-login", "userCode": "ABCD-12345", "verificationUrl": "https://auth.openai.com/codex/device"})
			} else {
				reply(map[string]any{"type": p.Type, "loginId": "fake-login", "authUrl": "https://auth.openai.com/oauth/authorize?client=fake"})
			}
			go func() {
				for {
					select {
					case <-stop:
						out(map[string]any{"method": "account/login/completed", "params": map[string]any{"loginId": "fake-login", "success": false, "error": "Login was not completed"}})
						return
					case <-time.After(50 * time.Millisecond):
					}
					if exists(filepath.Join(dir, "approve")) {
						_ = os.WriteFile(auth, []byte(`{"fake":true}`), 0600)
						out(map[string]any{"method": "account/login/completed", "params": map[string]any{"loginId": "fake-login", "success": true, "error": nil}})
						return
					}
				}
			}()
		case "account/login/cancel":
			once.Do(func() { close(stop) })
			reply(map[string]any{"status": "canceled"})
		case "account/logout":
			_ = os.Remove(auth)
			reply(map[string]any{})
		default:
			fail(-32601, "method not found")
		}
	}
}

func claude(args []string) int {
	dir := home("CLAUDE_CONFIG_DIR")
	creds := filepath.Join(dir, ".credentials.json")
	const link = "https://claude.com/cai/oauth/authorize?code=true&client_id=fake&state=fake-state"
	switch {
	case has(args, "--version"):
		fmt.Println("2.1.296 (Claude Code)")
		return 0
	case len(args) >= 2 && args[0] == "auth" && args[1] == "status":
		if exists(creds) {
			line(map[string]any{"loggedIn": true, "authMethod": "claude.ai", "apiProvider": "firstParty", "email": "owner@example.com", "subscriptionType": nil})
			return 0
		}
		line(map[string]any{"loggedIn": false, "authMethod": "none", "apiProvider": "firstParty"})
		return 1
	case len(args) >= 2 && args[0] == "auth" && args[1] == "logout":
		_ = os.Remove(creds)
		return 0
	case len(args) >= 2 && args[0] == "auth" && args[1] == "login":
		write := func() {
			raw, _ := json.Marshal(map[string]any{"claudeAiOauth": map[string]any{"accessToken": "fake-access-token", "expiresAt": time.Now().Add(time.Hour).UnixMilli(), "subscriptionType": "pro"}})
			_ = os.WriteFile(creds, raw, 0600)
		}
		fmt.Print("Opening browser to sign in…\nIf the browser didn't open, visit: \x1b]8;;" + link + "\x07" + link + "\x1b]8;;\x07\nPaste code here if prompted > ")
		if os.Getenv("BROWSER") != "/bin/true" {
			for !exists(filepath.Join(dir, "approve")) {
				time.Sleep(50 * time.Millisecond)
			}
			write()
			fmt.Println("Login successful.")
			return 0
		}
		code, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.TrimSpace(code) != "good-code#fake-state" {
			fmt.Println("OAuth error: Invalid code")
			return 1
		}
		write()
		fmt.Println("Login successful.")
		return 0
	case has(args, "-p"):
		for _, want := range []string{"--output-format stream-json", "--tools Read,Grep,Glob", "--strict-mcp-config", "--permission-mode dontAsk", "--no-session-persistence"} {
			if !has(args, want) {
				return 5
			}
		}
		if !exists(creds) {
			line(map[string]any{"type": "result", "subtype": "success", "is_error": true, "result": "Not logged in · Please run /login"})
			return 1
		}
		prompt, _ := io.ReadAll(os.Stdin)
		model := value(args, "--model")
		resolved := "claude-default-fake"
		if model != "" {
			resolved = "claude-" + model + "-fake"
		}
		line(map[string]any{"type": "system", "subtype": "init", "model": resolved, "tools": []string{"Read", "Grep", "Glob"}})
		switch strings.TrimSpace(string(prompt)) {
		case "limit":
			line(map[string]any{"type": "rate_limit_event", "rate_limit_info": map[string]any{"status": "rejected", "resetsAt": time.Now().Add(time.Hour).Unix()}})
			line(map[string]any{"type": "result", "subtype": "success", "is_error": true, "result": "Claude AI usage limit reached|1760000000"})
			return 1
		case "sleep":
			time.Sleep(30 * time.Second)
			return 0
		}
		line(map[string]any{"type": "assistant", "message": map[string]any{"content": []any{map[string]any{"type": "tool_use", "name": "Read", "input": map[string]any{"file_path": "/secret/path"}}}}})
		line(map[string]any{"type": "assistant", "message": map[string]any{"content": []any{map[string]any{"type": "text", "text": "ok"}}}})
		line(map[string]any{"type": "result", "subtype": "success", "is_error": false, "result": fmt.Sprintf("answer model=%s effort=%s", model, value(args, "--effort")), "usage": map[string]any{"input_tokens": 5, "cache_creation_input_tokens": 2, "cache_read_input_tokens": 3, "output_tokens": 4}})
		return 0
	}
	return 2
}
