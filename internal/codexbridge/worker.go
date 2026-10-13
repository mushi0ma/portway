package codexbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"portway/internal/secure"
)

type Worker struct {
	BaseURL, Key string
	Bridge       *Bridge
	Client       *http.Client
	commands     chan Command
}

func NewWorker(base, key string, bridge *Bridge) (*Worker, error) {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost"))) {
		return nil, errors.New("bridge URL requires HTTPS or HTTP loopback origin")
	}
	if len(key) < 32 {
		return nil, errors.New("bridge key requires at least 32 characters")
	}
	if bridge == nil || bridge.Accounts == nil {
		return nil, errors.New("bridge requires an account registry")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &Worker{BaseURL: base, Key: key, Bridge: bridge, Client: &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, commands: make(chan Command, 32)}, nil
}
func (w *Worker) call(ctx context.Context, path string, in, out any) error {
	raw, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", w.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Portway-Bridge-Key", w.Key)
	res, err := w.Client.Do(req)
	if err != nil {
		return errors.New("bridge API unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return errors.New("bridge API rejected request")
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 131073))
	if err != nil || len(body) > 131072 {
		return errors.New("invalid bridge API response")
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return errors.New("invalid bridge envelope")
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(envelope.Data, out)
}
func pause(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// Serve never requires a logged-in CLI: missing CLIs/logins are reported in the snapshot.
func (w *Worker) Serve(ctx context.Context) error {
	w.Bridge.init(ctx)
	instance := secure.ID()
	controlDone := make(chan struct{})
	go func() {
		defer close(controlDone)
		w.control(ctx, instance)
	}()
	for ctx.Err() == nil {
		var result PollResult
		if err := w.call(ctx, "/internal/v1/codex/poll", Poll{Instance: instance}, &result); err != nil {
			if !pause(ctx, 3*time.Second) {
				break
			}
			continue
		}
		w.enqueue(result.Commands)
		if result.Job != nil {
			w.execute(ctx, instance, *result.Job)
		}
		if !pause(ctx, 2*time.Second) {
			break
		}
	}
	<-controlDone
	return nil
}
func (w *Worker) enqueue(commands []Command) {
	for _, c := range commands {
		select {
		case w.commands <- c:
		default:
			code := "AI_COMMANDS_BUSY"
			_ = w.call(context.Background(), "/internal/v1/codex/commands/"+c.ID+"/ack", Ack{State: "failed", ErrorCode: &code}, nil)
		}
	}
}

// control serializes commands, usage refresh and state pushes.
func (w *Worker) control(ctx context.Context, instance string) {
	push := func() {
		if ctx.Err() == nil {
			_ = w.call(ctx, "/internal/v1/codex/state", w.Bridge.Snapshot(instance), nil)
		}
	}
	w.Bridge.RefreshAll(ctx, false)
	push()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.Bridge.mu.Lock()
			stop := w.Bridge.stopFlow
			w.Bridge.mu.Unlock()
			if stop != nil {
				stop()
			}
			return
		case c := <-w.commands:
			if !secure.UUID(c.ID) {
				continue
			}
			state, code := w.Bridge.Handle(ctx, c)
			if ctx.Err() == nil {
				_ = w.call(ctx, "/internal/v1/codex/commands/"+c.ID+"/ack", Ack{State: state, ErrorCode: code}, nil)
			}
			push()
		case <-w.Bridge.changed:
			for provider := range w.Bridge.takePending() {
				w.Bridge.Refresh(ctx, provider, true)
			}
			push()
		case <-ticker.C:
			w.Bridge.RefreshAll(ctx, false)
			push()
		}
	}
}
func (w *Worker) execute(ctx context.Context, instance string, job Job) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		for pause(runCtx, 2*time.Second) {
			var result PollResult
			if w.call(runCtx, "/internal/v1/codex/poll", Poll{Instance: instance, RunID: job.ID, Token: job.Token}, &result) != nil || !result.Continue {
				w.enqueue(result.Commands)
				cancel()
				return
			}
			w.enqueue(result.Commands)
		}
	}()
	seq := 0
	report := func(e Event) error {
		seq = e.Sequence
		// Repeating an event is safe by sequence+hash; a generation is never retried.
		for attempt := 0; attempt < 2; attempt++ {
			if err := w.call(runCtx, "/internal/v1/codex/report", Report{RunID: job.ID, Token: job.Token, Event: e}, nil); err == nil {
				return nil
			}
			if !pause(runCtx, 200*time.Millisecond) {
				break
			}
		}
		return errors.New("Codex event delivery failed")
	}
	if job.Provider == "" {
		job.Provider = ProviderCodex
	}
	res := Result{Code: "AI_RUN_FAILED"}
	var err = errors.New("AI provider unavailable")
	agent := w.Bridge.agent(job.Provider)
	profile := w.Bridge.Accounts.Active(job.Provider)
	if agent != nil && agent.Installed() {
		if job.Provider == ProviderCodex && job.Model == "" {
			job.Model = w.Bridge.defaultModel(job.Provider)
		}
		w.Bridge.setBusy(job.Provider, profile.ID)
		if agent.Check(runCtx, profile) {
			res, err = agent.Run(runCtx, profile, job, report)
		} else {
			res.Code = "AI_AUTH_REQUIRED"
		}
		w.Bridge.setBusy(job.Provider, "")
	}
	if runCtx.Err() == nil {
		e := Event{Sequence: seq + 1, Type: "completed", Text: res.Answer}
		if err != nil {
			e = Event{Sequence: seq + 1, Type: "failed", Code: res.Code}
			if !RunFailureCode(e.Code) {
				e.Code = "AI_RUN_FAILED"
			}
		} else {
			label := profile.Label
			res.Meta.AccountLabel = &label
			e.Meta = &res.Meta
		}
		_ = report(e)
	}
	cancel()
	<-heartbeatDone
	// Usage after a run (and immediately after a limit error) is re-read without inference.
	w.Bridge.RequestRefresh(job.Provider)
}
