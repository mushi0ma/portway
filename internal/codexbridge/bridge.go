package codexbridge

import (
	"context"
	"sync"
	"time"

	"portway/internal/secure"
)

// Inspection is what one CLI reports about one account profile without inference.
type Inspection struct {
	LoggedIn    bool
	Email, Plan *string
	Usage       *Usage
	Models      []Model
}

// Result of one native run; Code is set when the run failed.
type Result struct {
	Answer string
	Code   string
	Meta   RunMeta
}

// Agent is one official CLI on the laptop (Codex CLI or Claude Code).
type Agent interface {
	ID() string
	Name() string
	Installed() bool
	Version(context.Context) *string
	Inspect(ctx context.Context, p Profile, usage, models bool) (Inspection, error)
	Check(ctx context.Context, p Profile) bool
	Login(ctx context.Context, p Profile, f *LoginFlow) string
	Logout(ctx context.Context, p Profile) error
	Run(ctx context.Context, p Profile, job Job, emit func(Event) error) (Result, error)
}

// LoginFlow is the bridge-side state of one OAuth login driven by an official CLI.
type LoginFlow struct {
	mu        sync.Mutex
	state     Login
	code      chan string
	ready     chan struct{}
	readyOnce sync.Once
	cancelled bool
	notify    func()
}

func (f *LoginFlow) Set(fn func(*Login)) {
	f.mu.Lock()
	fn(&f.state)
	f.state.UpdatedAt = time.Now().UTC()
	ready := f.state.State != "starting"
	f.mu.Unlock()
	if ready {
		f.readyOnce.Do(func() { close(f.ready) })
	}
	if f.notify != nil {
		f.notify()
	}
}
func (f *LoginFlow) Snapshot() Login {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}

// Codes delivers codes pasted on the phone (Claude code-paste flow).
func (f *LoginFlow) Codes() <-chan string { return f.code }
func (f *LoginFlow) Method() string       { return f.Snapshot().Method }

type providerCache struct {
	version   *string
	checked   bool
	accounts  map[string]Inspection
	usage     *Usage
	usageAt   time.Time
	usageFor  string
	models    []Model
	modelsAt  time.Time
	modelsFor string
	at        time.Time
}

// Bridge owns the laptop-side state: agents, account profiles, caches and the login flow.
type Bridge struct {
	Workspace string
	Agents    []Agent
	Accounts  *Registry
	// ForceInterval bounds forced usage re-reads (default 10 s; tests shorten it).
	ForceInterval time.Duration
	mu            sync.Mutex
	ctx           context.Context
	cache         map[string]*providerCache
	flow          *LoginFlow
	stopFlow      context.CancelFunc
	flowDone      chan struct{}
	busy          map[string]string
	pending       map[string]bool
	changed       chan struct{}
}

func (b *Bridge) init(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ctx = ctx
	if b.cache == nil {
		b.cache, b.busy, b.pending, b.changed = map[string]*providerCache{}, map[string]string{}, map[string]bool{}, make(chan struct{}, 1)
	}
	for _, a := range b.Agents {
		if b.cache[a.ID()] == nil {
			b.cache[a.ID()] = &providerCache{accounts: map[string]Inspection{}}
		}
	}
}
func (b *Bridge) agent(id string) Agent {
	for _, a := range b.Agents {
		if a.ID() == id {
			return a
		}
	}
	return nil
}
func (b *Bridge) notify() {
	select {
	case b.changed <- struct{}{}:
	default:
	}
}

// RequestRefresh schedules a forced usage read for a provider and a state push.
func (b *Bridge) RequestRefresh(provider string) {
	b.mu.Lock()
	b.pending[provider] = true
	b.mu.Unlock()
	b.notify()
}
func (b *Bridge) takePending() map[string]bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.pending
	b.pending = map[string]bool{}
	return out
}

// Refresh re-reads account state; usage is read at most once a minute per provider
// (forced reads: at most every 10 seconds). No inference is used.
func (b *Bridge) Refresh(ctx context.Context, provider string, force bool) {
	a := b.agent(provider)
	if a == nil {
		return
	}
	b.mu.Lock()
	c := b.cache[provider]
	b.mu.Unlock()
	if !c.checked {
		version := a.Version(ctx)
		b.mu.Lock()
		c.version, c.checked = version, true
		b.mu.Unlock()
	}
	if !a.Installed() {
		return
	}
	active := b.Accounts.Active(provider)
	now := time.Now()
	minimum := b.ForceInterval
	if minimum <= 0 {
		minimum = 10 * time.Second
	}
	seen := map[string]bool{}
	for _, p := range b.Accounts.Profiles(provider) {
		seen[p.ID] = true
		isActive := p.ID == active.ID
		wantUsage := isActive && (c.usageFor != p.ID || now.Sub(c.usageAt) >= time.Minute || (force && now.Sub(c.usageAt) >= minimum))
		wantModels := isActive && (c.modelsFor != p.ID || now.Sub(c.modelsAt) >= 10*time.Minute)
		// Account state is re-read at most once a minute unless forced or usage/models are due.
		if !force && !wantUsage && !wantModels && now.Sub(c.at) < time.Minute {
			if _, ok := c.accounts[p.ID]; ok {
				continue
			}
		}
		info, err := a.Inspect(ctx, p, wantUsage, wantModels)
		if err != nil && ctx.Err() != nil {
			return
		}
		b.mu.Lock()
		c.accounts[p.ID] = Inspection{LoggedIn: info.LoggedIn, Email: info.Email, Plan: info.Plan}
		if wantUsage {
			c.usage, c.usageAt, c.usageFor = info.Usage, now, p.ID
		}
		if wantModels && info.Models != nil {
			c.models, c.modelsAt, c.modelsFor = info.Models, now, p.ID
		}
		b.mu.Unlock()
	}
	b.mu.Lock()
	for id := range c.accounts {
		if !seen[id] {
			delete(c.accounts, id)
		}
	}
	if now.Sub(c.at) >= time.Minute || force {
		c.at = now
	}
	b.mu.Unlock()
}
func (b *Bridge) RefreshAll(ctx context.Context, force bool) {
	for _, a := range b.Agents {
		b.Refresh(ctx, a.ID(), force)
	}
}

// Snapshot is the state pushed to the API (emails stay encrypted there).
func (b *Bridge) Snapshot(instance string) Snapshot {
	out := Snapshot{Instance: instance, Providers: []ProviderState{}}
	for _, a := range b.Agents {
		active := b.Accounts.Active(a.ID())
		profiles := b.Accounts.Profiles(a.ID())
		b.mu.Lock()
		c := b.cache[a.ID()]
		p := ProviderState{ID: a.ID(), Name: a.Name(), Installed: a.Installed(), CLIVersion: c.version, Accounts: []Account{}, Models: []Model{}}
		if c.modelsFor == active.ID && c.models != nil {
			p.Models = c.models
		}
		activeID := active.ID
		p.ActiveAccountID = &activeID
		for _, prof := range profiles {
			info := c.accounts[prof.ID]
			p.Accounts = append(p.Accounts, Account{ID: prof.ID, Label: prof.Label, Email: info.Email, Plan: info.Plan, LoggedIn: info.LoggedIn && p.Installed, Active: prof.ID == active.ID, System: prof.System})
			if prof.ID == active.ID && info.LoggedIn && c.usageFor == active.ID {
				p.Usage = c.usage
			}
		}
		b.mu.Unlock()
		out.Providers = append(out.Providers, p)
	}
	b.mu.Lock()
	if b.flow != nil {
		if l := b.flow.Snapshot(); time.Since(l.UpdatedAt) < loginWindow {
			out.Login = &l
		}
	}
	b.mu.Unlock()
	return out
}
func (b *Bridge) defaultModel(provider string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if c := b.cache[provider]; c != nil {
		for _, m := range c.models {
			if m.Default {
				return m.ID
			}
		}
	}
	return ""
}
func (b *Bridge) setBusy(provider, account string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if account == "" {
		delete(b.busy, provider)
	} else {
		b.busy[provider] = account
	}
}

// Handle executes one owner command and returns its outcome for the ack.
func (b *Bridge) Handle(ctx context.Context, c Command) (string, *string) {
	fail := func(code string) (string, *string) { return "failed", &code }
	p := c.Payload
	switch c.Kind {
	case "login_start":
		if !secure.UUID(p.LoginID) || !ValidProvider(p.Provider) || (p.Method != "phone" && p.Method != "computer") {
			return fail("INVALID_COMMAND")
		}
		if code := b.startLogin(p.LoginID, p.Provider, p.Method); code != "" {
			return fail(code)
		}
	case "login_code":
		b.mu.Lock()
		f := b.flow
		b.mu.Unlock()
		if f == nil {
			return fail("AI_LOGIN_NOT_ACTIVE")
		}
		l := f.Snapshot()
		if l.ID != p.LoginID || l.Terminal() || l.Provider != ProviderClaude || p.Code == "" {
			return fail("AI_LOGIN_NOT_ACTIVE")
		}
		select {
		case f.code <- p.Code:
		default:
			return fail("AI_LOGIN_BUSY")
		}
	case "login_cancel":
		b.mu.Lock()
		f, stop := b.flow, b.stopFlow
		var done chan struct{}
		if f != nil && f.Snapshot().ID == p.LoginID && !f.Snapshot().Terminal() && stop != nil {
			f.mu.Lock()
			f.cancelled = true
			f.mu.Unlock()
			stop()
			done = b.flowDone
		}
		b.mu.Unlock()
		if done != nil {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
			}
		}
	case "account_activate":
		if !ValidAccountID(p.Provider, p.AccountID) {
			return fail("INVALID_COMMAND")
		}
		if err := b.Accounts.Activate(p.Provider, p.AccountID); err != nil {
			return fail("AI_ACCOUNT_NOT_FOUND")
		}
		b.Refresh(ctx, p.Provider, true)
	case "account_remove":
		if !ValidAccountID(p.Provider, p.AccountID) {
			return fail("INVALID_COMMAND")
		}
		prof, ok := b.Accounts.Get(p.Provider, p.AccountID)
		if !ok {
			return fail("AI_ACCOUNT_NOT_FOUND")
		}
		if prof.System {
			return fail("AI_ACCOUNT_SYSTEM")
		}
		b.mu.Lock()
		inUse := b.busy[p.Provider] == p.AccountID
		b.mu.Unlock()
		if inUse {
			return fail("AI_ACCOUNT_BUSY")
		}
		if a := b.agent(p.Provider); a != nil && a.Installed() {
			logoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			_ = a.Logout(logoutCtx, prof)
			cancel()
		}
		if err := b.Accounts.Remove(p.Provider, p.AccountID); err != nil {
			return fail("AI_ACCOUNT_REMOVE_FAILED")
		}
		b.Refresh(ctx, p.Provider, true)
	case "usage_refresh":
		b.RefreshAll(ctx, true)
	default:
		return fail("INVALID_COMMAND")
	}
	return "done", nil
}

// startLogin runs the CLI login in a fresh profile and waits until the URL/code is
// known (or the flow fails). Completion is reported through later snapshots.
func (b *Bridge) startLogin(id, provider, method string) string {
	b.mu.Lock()
	if b.flow != nil && !b.flow.Snapshot().Terminal() && b.stopFlow != nil {
		b.flow.mu.Lock()
		b.flow.cancelled = true
		b.flow.mu.Unlock()
		b.stopFlow()
		done := b.flowDone
		b.mu.Unlock()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		b.mu.Lock()
	}
	now := time.Now().UTC()
	expires := now.Add(loginWindow)
	f := &LoginFlow{state: Login{ID: id, Provider: provider, Method: method, State: "starting", ExpiresAt: &expires, UpdatedAt: now}, code: make(chan string, 1), ready: make(chan struct{}), notify: b.notify}
	b.flow = f
	parent := b.ctx
	b.mu.Unlock()
	fail := func(code string) string {
		f.Set(func(l *Login) { l.State, l.ErrorCode = "failed", &code })
		return code
	}
	a := b.agent(provider)
	if a == nil || !a.Installed() {
		return fail("CLI_MISSING")
	}
	prof, err := b.Accounts.NewPending(provider)
	if err != nil {
		return fail("LOGIN_FAILED")
	}
	ctx, cancel := context.WithTimeout(parent, loginWindow)
	done := make(chan struct{})
	b.mu.Lock()
	b.stopFlow, b.flowDone = cancel, done
	b.mu.Unlock()
	go func() {
		defer close(done)
		defer cancel()
		code := a.Login(ctx, prof, f)
		if code == "" {
			if err := b.Accounts.Commit(prof); err != nil {
				code = "LOGIN_FAILED"
			}
		}
		f.mu.Lock()
		cancelled := f.cancelled
		f.mu.Unlock()
		if code != "" {
			b.Accounts.Discard(prof)
		}
		f.Set(func(l *Login) {
			l.URL, l.UserCode, l.NeedsCode = nil, nil, false
			switch {
			case code == "":
				account := prof.ID
				l.State, l.AccountID = "completed", &account
			case cancelled:
				l.State = "cancelled"
			case ctx.Err() == context.DeadlineExceeded:
				timeout := "LOGIN_TIMEOUT"
				l.State, l.ErrorCode = "expired", &timeout
			default:
				l.State, l.ErrorCode = "failed", &code
			}
		})
		b.RequestRefresh(provider)
	}()
	select {
	case <-f.ready:
	case <-time.After(45 * time.Second):
	}
	if l := f.Snapshot(); l.State == "failed" && l.ErrorCode != nil {
		return *l.ErrorCode
	}
	return ""
}
