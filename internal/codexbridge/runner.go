package codexbridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Codex drives the official Codex CLI (ChatGPT login) for one workspace.
type Codex struct {
	Binary, Workspace string
	mu                sync.Mutex
	slugs             map[string]string
}

func (c *Codex) ID() string      { return ProviderCodex }
func (c *Codex) Name() string    { return "Codex" }
func (c *Codex) Installed() bool { return c.Binary != "" }
func (c *Codex) env(p Profile, extra ...string) []string {
	if p.Dir != "" {
		return nativeEnv(append([]string{"CODEX_HOME=" + p.Dir}, extra...)...)
	}
	if v := os.Getenv("CODEX_HOME"); v != "" {
		extra = append([]string{"CODEX_HOME=" + v}, extra...)
	}
	return nativeEnv(extra...)
}
func (c *Codex) Version(ctx context.Context) *string {
	if !c.Installed() {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := native(ctx, c.Binary, c.Workspace, nativeEnv(), "--version").Output()
	if err != nil {
		return nil
	}
	return cliVersion(string(out))
}

// Check confirms a ChatGPT (subscription) login, never an API-key login.
func (c *Codex) Check(ctx context.Context, p Profile) bool {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := native(ctx, c.Binary, c.Workspace, c.env(p), "login", "status").CombinedOutput()
	return err == nil && strings.Contains(string(out), "Logged in using ChatGPT")
}

// Inspect reads account, rate limits and models through `codex app-server` (no inference).
func (c *Codex) Inspect(ctx context.Context, p Profile, usage, models bool) (Inspection, error) {
	out := Inspection{}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	client, err := startRPC(ctx, c.Binary, c.Workspace, c.env(p))
	if err != nil {
		if usage {
			out.Usage = &Usage{Status: "unknown", CheckedAt: time.Now().UTC(), Windows: []Window{}}
		}
		return out, err
	}
	defer client.Close()
	var account struct {
		Account *struct {
			Type     string  `json:"type"`
			Email    *string `json:"email"`
			PlanType *string `json:"planType"`
		} `json:"account"`
	}
	if err = client.call(ctx, "account/read", map[string]any{}, &account); err != nil {
		return out, err
	}
	if a := account.Account; a != nil && a.Type == "chatgpt" {
		out.LoggedIn = true
		if a.Email != nil && *a.Email != "" && plain(*a.Email, 254) {
			out.Email = a.Email
		}
		if a.PlanType != nil && *a.PlanType != "" && plain(*a.PlanType, 40) {
			out.Plan = a.PlanType
		}
	}
	if usage && out.LoggedIn {
		var raw json.RawMessage
		if client.call(ctx, "account/rateLimits/read", nil, &raw) == nil {
			out.Usage = CodexUsage(raw, time.Now())
		} else {
			out.Usage = &Usage{Status: "unknown", CheckedAt: time.Now().UTC(), Windows: []Window{}}
		}
	}
	if models {
		var list struct {
			Data []struct {
				ID            string `json:"id"`
				Model         string `json:"model"`
				DisplayName   string `json:"displayName"`
				Description   string `json:"description"`
				IsDefault     bool   `json:"isDefault"`
				Hidden        bool   `json:"hidden"`
				DefaultEffort string `json:"defaultReasoningEffort"`
				Efforts       []struct {
					Effort string `json:"reasoningEffort"`
				} `json:"supportedReasoningEfforts"`
			} `json:"data"`
		}
		if client.call(ctx, "model/list", map[string]any{}, &list) == nil {
			out.Models = []Model{}
			slugs := map[string]string{}
			for _, m := range list.Data {
				if m.Hidden || !ValidModel(m.ID) || len(out.Models) >= 64 {
					continue
				}
				model := Model{ID: m.ID, Name: trimText(m.DisplayName, 100), Description: trimText(m.Description, 500), Default: m.IsDefault, Efforts: []string{}}
				if model.Name == "" {
					model.Name = m.ID
				}
				for _, e := range m.Efforts {
					if ValidEffort(e.Effort) {
						model.Efforts = append(model.Efforts, e.Effort)
					}
				}
				if ValidEffort(m.DefaultEffort) {
					effort := m.DefaultEffort
					model.DefaultEffort = &effort
				}
				if ValidModel(m.Model) {
					slugs[m.ID] = m.Model
				}
				out.Models = append(out.Models, model)
			}
			c.mu.Lock()
			c.slugs = slugs
			c.mu.Unlock()
		}
	}
	return out, nil
}

// trimText keeps catalogue text single-line and bounded.
func trimText(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if !plain(s, 1<<20) {
		return ""
	}
	r := []rune(s)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

// Login uses ChatGPT device code (phone) or the browser flow on the laptop (computer).
func (c *Codex) Login(ctx context.Context, p Profile, f *LoginFlow) string {
	client, err := startRPC(ctx, c.Binary, c.Workspace, c.env(p))
	if err != nil {
		return "LOGIN_FAILED"
	}
	defer client.Close()
	method := f.Method()
	params := map[string]any{"type": "chatgptDeviceCode"}
	if method == "computer" {
		params = map[string]any{"type": "chatgpt"}
	}
	var start struct {
		LoginID         string `json:"loginId"`
		UserCode        string `json:"userCode"`
		VerificationURL string `json:"verificationUrl"`
		AuthURL         string `json:"authUrl"`
	}
	if err = client.call(ctx, "account/login/start", params, &start); err != nil {
		var rerr *rpcError
		if method == "phone" && errors.As(err, &rerr) && strings.Contains(strings.ToLower(rerr.Message), "device") {
			return "DEVICE_CODE_UNAVAILABLE"
		}
		return "LOGIN_FAILED"
	}
	link := start.VerificationURL
	if method == "computer" {
		link = start.AuthURL
		openBrowser(ctx, link)
	}
	f.Set(func(l *Login) {
		l.State, l.URL = "waiting_user", SafeLoginURL(link)
		if userCodeRE.MatchString(start.UserCode) {
			code := start.UserCode
			l.UserCode = &code
		}
	})
	for {
		select {
		case note := <-client.notes:
			if note.Method != "account/login/completed" {
				continue
			}
			var done struct {
				LoginID *string `json:"loginId"`
				Success bool    `json:"success"`
				Error   *string `json:"error"`
			}
			if json.Unmarshal(note.Params, &done) != nil || (done.LoginID != nil && *done.LoginID != start.LoginID) {
				continue
			}
			if !done.Success {
				if done.Error != nil && strings.Contains(strings.ToLower(*done.Error), "device") {
					return "DEVICE_CODE_UNAVAILABLE"
				}
				return "LOGIN_FAILED"
			}
			f.Set(func(l *Login) { l.State = "verifying" })
			var account struct {
				Account *struct {
					Type string `json:"type"`
				} `json:"account"`
			}
			if client.call(ctx, "account/read", map[string]any{}, &account) != nil || account.Account == nil || account.Account.Type != "chatgpt" {
				return "LOGIN_FAILED"
			}
			return ""
		case <-client.done:
			return "LOGIN_FAILED"
		case <-ctx.Done():
			cancelCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = client.call(cancelCtx, "account/login/cancel", map[string]string{"loginId": start.LoginID}, nil)
			cancel()
			return "LOGIN_CANCELLED"
		}
	}
}
func (c *Codex) Logout(ctx context.Context, p Profile) error {
	if p.System {
		return errors.New("system login is never logged out by the bridge")
	}
	client, err := startRPC(ctx, c.Binary, c.Workspace, c.env(p))
	if err != nil {
		return err
	}
	defer client.Close()
	return client.call(ctx, "account/logout", nil, nil)
}

// openBrowser opens the login page on the laptop desktop for the "computer" method.
func openBrowser(ctx context.Context, link string) {
	if SafeLoginURL(link) == nil {
		return
	}
	opener, err := exec.LookPath("xdg-open")
	if err != nil {
		return
	}
	cmd := native(context.WithoutCancel(ctx), opener, os.TempDir(), nativeEnv(displayEnv()...), link)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if cmd.Start() == nil {
		go func() { _ = cmd.Wait() }()
	}
}
func (c *Codex) slug(model string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s := c.slugs[model]; s != "" {
		return s
	}
	return model
}

// Run invokes exactly one native process. Prompts go through stdin, never a shell.
// Raw stderr, reasoning and command output are not published to the mobile API.
func (c *Codex) Run(ctx context.Context, p Profile, job Job, emit func(Event) error) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 9*time.Minute)
	defer cancel()
	args := []string{"-a", "never", "--no-daemon", "exec", "--json", "--ephemeral", "--ignore-user-config", "--ignore-rules", "--sandbox", "read-only", "--color", "never", "--skip-git-repo-check", "-C", c.Workspace, "-c", "shell_environment_policy.inherit=none", "-c", "web_search=disabled"}
	model := ""
	if job.Model != "" && ValidModel(job.Model) {
		model = c.slug(job.Model)
		args = append(args, "-m", model)
	}
	if ValidEffort(job.Effort) {
		args = append(args, "-c", `model_reasoning_effort="`+job.Effort+`"`)
	}
	args = append(args, "-")
	res := Result{Code: "AI_RUN_FAILED"}
	if model != "" {
		res.Meta.ResolvedModel = &model
	}
	cmd := native(ctx, c.Binary, c.Workspace, c.env(p), args...)
	cmd.Stdin = strings.NewReader(job.Prompt)
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return res, errors.New("Codex process unavailable")
	}
	if err = cmd.Start(); err != nil {
		return res, errors.New("Codex process unavailable")
	}
	seq := 0
	send := func(kind, text string) error { seq++; return emit(Event{Sequence: seq, Type: kind, Text: text}) }
	if err = send("started", ""); err != nil {
		cancel()
		_ = cmd.Wait()
		return res, err
	}
	scanner := bufio.NewScanner(io.LimitReader(stdout, 4<<20))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	answer, reason := "", ""
	complete, failed := false, false
	bytes := 0
	for scanner.Scan() {
		bytes += len(scanner.Bytes()) + 1
		var event struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Error   struct {
				Message string `json:"message"`
			} `json:"error"`
			Usage *struct {
				Input  int64 `json:"input_tokens"`
				Output int64 `json:"output_tokens"`
			} `json:"usage"`
			Item struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"item"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			failed = true
			break
		}
		switch event.Type {
		case "turn.completed":
			complete = true
			if event.Usage != nil && event.Usage.Input >= 0 && event.Usage.Output >= 0 {
				res.Meta.Tokens = &Tokens{Input: event.Usage.Input, Output: event.Usage.Output}
			}
		case "turn.failed", "error":
			failed = true
			if code := Classify(event.Message + " " + event.Error.Message); code != "" && reason == "" {
				reason = code
			}
		}
		if event.Type == "item.completed" && event.Item.Type == "agent_message" {
			if len(event.Item.Text) > 65536 {
				failed = true
				break
			}
			answer = event.Item.Text
			if seq < 250 && len(answer) <= 8192 {
				if err = send("progress", answer); err != nil {
					failed = true
					break
				}
			}
		}
		if event.Type == "item.started" && event.Item.Type == "command_execution" && seq < 250 {
			if err = send("progress", "Codex выполняет команду в режиме read-only"); err != nil {
				failed = true
				break
			}
		}
	}
	if failed || scanner.Err() != nil || bytes >= 4<<20 {
		cancel()
	}
	waitErr := cmd.Wait()
	if err != nil {
		return res, err
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		res.Code = "AI_TIMEOUT"
	} else if reason != "" {
		res.Code = reason
	}
	if failed || scanner.Err() != nil || waitErr != nil || !complete || strings.TrimSpace(answer) == "" || ctx.Err() != nil {
		return res, errors.New("Codex execution failed or was cancelled")
	}
	encoded, _ := json.Marshal(Event{Sequence: 256, Type: "completed", Text: answer, Meta: &res.Meta})
	if len(encoded) > 69984 {
		return res, errors.New("Codex answer exceeds encoded event limit")
	}
	res.Answer, res.Code = answer, ""
	return res, nil
}
