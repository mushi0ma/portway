// Package codexbridge implements the opt-in, read-only native agent demo profile
// (Codex CLI and Claude Code on the owner's laptop).
package codexbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"portway/internal/problem"
	"portway/internal/secure"
	"portway/internal/storage"
)

type Store struct {
	DB    *storage.Store
	Vault *secure.Vault
}
type Run struct {
	ID            string     `json:"id"`
	State         string     `json:"state"`
	Workspace     string     `json:"workspace"`
	Sandbox       string     `json:"sandbox"`
	Provider      string     `json:"provider"`
	Model         *string    `json:"model"`
	Effort        *string    `json:"effort"`
	ResolvedModel *string    `json:"resolved_model"`
	AccountLabel  *string    `json:"account_label"`
	Tokens        *Tokens    `json:"tokens"`
	Prompt        string     `json:"prompt,omitempty"`
	Answer        string     `json:"answer,omitempty"`
	PromptPreview *string    `json:"prompt_preview"`
	ErrorCode     *string    `json:"error_code"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
}
type Status struct {
	Connected      bool            `json:"connected"`
	Workspace      string          `json:"workspace"`
	Sandbox        string          `json:"sandbox"`
	LastSeen       *time.Time      `json:"last_seen_at"`
	StateUpdatedAt *time.Time      `json:"state_updated_at"`
	Providers      []ProviderState `json:"providers"`
	Login          *Login          `json:"login"`
	Commands       []CommandView   `json:"commands"`
}
type Poll struct {
	Instance string `json:"instance_id"`
	RunID    string `json:"run_id,omitempty"`
	Token    string `json:"lease_token,omitempty"`
}
type Job struct {
	ID       string `json:"id"`
	Prompt   string `json:"prompt"`
	Token    string `json:"lease_token"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Effort   string `json:"effort,omitempty"`
}
type PollResult struct {
	Continue bool      `json:"continue"`
	Job      *Job      `json:"job"`
	Commands []Command `json:"commands"`
}
type Event struct {
	Sequence int      `json:"sequence"`
	Type     string   `json:"type"`
	Text     string   `json:"text,omitempty"`
	Code     string   `json:"code,omitempty"`
	Meta     *RunMeta `json:"meta,omitempty"`
}
type Report struct {
	RunID string `json:"run_id"`
	Token string `json:"lease_token"`
	Event Event  `json:"event"`
}

// CreateInput is the validated mobile request for one native agent run.
type CreateInput struct {
	Prompt   string
	Provider string
	Model    string
	Effort   string
}
type ListFilter struct{ Provider, State string }
type querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s Store) aad(id, kind string) string {
	return "codex/" + s.DB.Installation + "/" + id + "/" + kind
}
func opt(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// loadState decrypts the last bridge snapshot; nil means the bridge never reported one.
func (s Store) loadState(ctx context.Context, q querier) (*Snapshot, *time.Time, *time.Time, bool, error) {
	var ct, nonce []byte
	var at, seen *time.Time
	var connected bool
	err := q.QueryRow(ctx, `SELECT state_ciphertext,state_nonce,state_updated_at,last_seen_at,last_seen_at>now()-interval '15 seconds' FROM portway.codex_bridge WHERE installation_id=$1`, s.DB.Installation).Scan(&ct, &nonce, &at, &seen, &connected)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil, false, nil
	}
	if err != nil || ct == nil {
		return nil, nil, seen, connected, err
	}
	plain, err := s.Vault.Open(ct, nonce, s.aad("bridge", "state"))
	if err != nil {
		return nil, nil, seen, connected, err
	}
	var state Snapshot
	if err = json.Unmarshal(plain, &state); err != nil {
		return nil, nil, seen, connected, err
	}
	return &state, at, seen, connected, nil
}
func (st *Snapshot) provider(id string) *ProviderState {
	if st == nil {
		return nil
	}
	for i := range st.Providers {
		if st.Providers[i].ID == id {
			return &st.Providers[i]
		}
	}
	return nil
}
func (p *ProviderState) activeAccount() *Account {
	if p == nil || p.ActiveAccountID == nil {
		return nil
	}
	for i := range p.Accounts {
		if p.Accounts[i].ID == *p.ActiveAccountID {
			return &p.Accounts[i]
		}
	}
	return nil
}

func (s Store) Status(ctx context.Context) (Status, error) {
	out := Status{Workspace: "portway", Sandbox: "read-only", Providers: []ProviderState{}, Commands: []CommandView{}}
	err := s.locked(ctx, func(tx pgx.Tx) error {
		state, at, seen, connected, err := s.loadState(ctx, tx)
		if err != nil {
			return err
		}
		out.Connected, out.LastSeen, out.StateUpdatedAt = connected, seen, at
		if state != nil && state.Providers != nil {
			out.Providers = state.Providers
		}
		if out.Login, err = s.currentLogin(ctx, tx, state, at); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT id,kind,state,error_code,updated_at FROM portway.codex_commands WHERE installation_id=$1 AND created_at>now()-interval '10 minutes' ORDER BY created_at DESC,id DESC LIMIT 10`, s.DB.Installation)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c CommandView
			if err = rows.Scan(&c.ID, &c.Kind, &c.State, &c.ErrorCode, &c.UpdatedAt); err != nil {
				return err
			}
			out.Commands = append(out.Commands, c)
		}
		return rows.Err()
	})
	return out, err
}

// Sweep never puts an uncertain dispatch back in the queue.
func (s Store) sweep(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `UPDATE portway.codex_runs r SET state=CASE WHEN d.revoked_at IS NOT NULL THEN 'stopped' ELSE 'failed' END,error_code=CASE WHEN d.revoked_at IS NOT NULL THEN 'DEVICE_REVOKED' ELSE 'CODEX_BRIDGE_LOST' END,finished_at=now() FROM portway.device_sessions d WHERE r.installation_id=$1 AND d.id=r.device_id AND r.state IN ('queued','running') AND (d.revoked_at IS NOT NULL OR r.created_at<now()-interval '10 minutes' OR (r.state='queued' AND r.created_at<now()-interval '60 seconds') OR (r.state='running' AND r.lease_until<=now()))`, s.DB.Installation)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE portway.codex_commands SET state='failed',error_code=CASE WHEN state='queued' THEN 'COMMAND_EXPIRED' ELSE 'CODEX_BRIDGE_LOST' END,payload_ciphertext=NULL,payload_nonce=NULL,updated_at=now() WHERE installation_id=$1 AND ((state='queued' AND created_at<now()-interval '60 seconds') OR (state='delivered' AND delivered_at<now()-interval '2 minutes'))`, s.DB.Installation)
	return err
}

// admit applies the provider preflight from the last bridge snapshot.
func (s Store) admit(ctx context.Context, tx pgx.Tx, in *CreateInput) error {
	state, _, _, _, err := s.loadState(ctx, tx)
	if err != nil {
		return err
	}
	unavailable := problem.New(409, "AI_PROVIDER_UNAVAILABLE", "Selected AI CLI is not available on the laptop")
	if state == nil {
		// A bridge from the first demo slice reports no snapshot and runs Codex only.
		if in.Provider != ProviderCodex {
			return unavailable
		}
		return nil
	}
	p := state.provider(in.Provider)
	if p == nil || !p.Installed {
		return unavailable
	}
	if a := p.activeAccount(); a == nil || !a.LoggedIn {
		return problem.New(409, "AI_ACCOUNT_REQUIRED", "Connect an AI account for this provider on the laptop")
	}
	if u := p.Usage; u != nil && u.Status == "exhausted" && (u.ResetAt == nil || u.ResetAt.After(time.Now())) {
		e := problem.New(429, "AI_LIMIT_EXHAUSTED", "AI usage limit is exhausted for the active account")
		e.Details = map[string]any{"provider": in.Provider, "reset_at": u.ResetAt}
		return e
	}
	var model *Model
	for i := range p.Models {
		m := &p.Models[i]
		if (in.Model == "" && m.Default) || (in.Model != "" && m.ID == in.Model) {
			model = m
			break
		}
	}
	if in.Model != "" && model == nil && len(p.Models) > 0 {
		return problem.Invalid("Unknown model for the selected provider")
	}
	if in.Effort != "" && model != nil && len(model.Efforts) > 0 {
		supported := false
		for _, e := range model.Efforts {
			supported = supported || e == in.Effort
		}
		if !supported {
			if in.Provider == ProviderCodex {
				return problem.Invalid("Reasoning effort is not supported by the selected model")
			}
			in.Effort = ""
		}
	}
	if in.Effort != "" && in.Provider == ProviderClaude && model != nil && len(model.Efforts) == 0 {
		in.Effort = ""
	}
	return nil
}
func (s Store) Create(ctx context.Context, tx pgx.Tx, owner, device string, in CreateInput) (any, error) {
	if err := s.sweep(ctx, tx); err != nil {
		return nil, err
	}
	var connected bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM portway.codex_bridge WHERE installation_id=$1 AND last_seen_at>now()-interval '15 seconds')`, s.DB.Installation).Scan(&connected); err != nil {
		return nil, err
	}
	if !connected {
		return nil, problem.New(503, "CODEX_OFFLINE", "Laptop bridge is offline")
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM portway.codex_runs WHERE installation_id=$1 AND state IN ('queued','running')`, s.DB.Installation).Scan(&count); err != nil {
		return nil, err
	}
	if count != 0 {
		return nil, problem.New(409, "CODEX_BUSY", "One Codex request is already active")
	}
	if in.Provider == "" {
		in.Provider = ProviderCodex
	}
	if in.Model == "default" {
		in.Model = ""
	}
	if err := s.admit(ctx, tx, &in); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM portway.codex_runs WHERE installation_id=$1 AND expires_at<=now() AND state NOT IN ('queued','running')`, s.DB.Installation); err != nil {
		return nil, err
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM portway.codex_runs WHERE installation_id=$1`, s.DB.Installation).Scan(&count); err != nil {
		return nil, err
	}
	if count >= 1000 {
		return nil, problem.New(429, "CODEX_HISTORY_FULL", "History limit reached")
	}
	id := secure.ID()
	ct, nonce := s.Vault.Seal(in.Prompt, s.aad(id, "prompt"))
	run := Run{ID: id, State: "queued", Workspace: "portway", Sandbox: "read-only", Provider: in.Provider, Model: opt(in.Model), Effort: opt(in.Effort)}
	err := tx.QueryRow(ctx, `INSERT INTO portway.codex_runs(id,installation_id,owner_id,device_id,state,prompt_ciphertext,prompt_nonce,provider,model,effort) VALUES($1,$2,$3,$4,'queued',$5,$6,$7,$8,$9) RETURNING created_at`, id, s.DB.Installation, owner, device, ct, nonce, in.Provider, run.Model, run.Effort).Scan(&run.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err = s.DB.Audit(ctx, tx, owner, "codex.queued", "codex_run", id); err != nil {
		return nil, err
	}
	// The replay-safe response deliberately omits prompt text and preview.
	return run, nil
}

const runColumns = `id,state,error_code,created_at,started_at,finished_at,provider,model,effort,resolved_model,account_label,input_tokens,output_tokens,prompt_ciphertext,prompt_nonce`

// Preview is a whitespace-collapsed prefix used by the history list.
func Preview(prompt string) string {
	text := strings.Join(strings.FieldsFunc(prompt, unicode.IsSpace), " ")
	if utf8.RuneCountInString(text) <= 120 {
		return text
	}
	return string([]rune(text)[:119]) + "…"
}
func (s Store) scanRun(row pgx.Row, extra ...any) (Run, string, error) {
	r := Run{Workspace: "portway", Sandbox: "read-only"}
	var in, out *int64
	var ct, nonce []byte
	err := row.Scan(append([]any{&r.ID, &r.State, &r.ErrorCode, &r.CreatedAt, &r.StartedAt, &r.FinishedAt, &r.Provider, &r.Model, &r.Effort, &r.ResolvedModel, &r.AccountLabel, &in, &out, &ct, &nonce}, extra...)...)
	if err != nil {
		return r, "", err
	}
	if in != nil || out != nil {
		r.Tokens = &Tokens{}
		if in != nil {
			r.Tokens.Input = *in
		}
		if out != nil {
			r.Tokens.Output = *out
		}
	}
	plain, err := s.Vault.Open(ct, nonce, s.aad(r.ID, "prompt"))
	if err != nil {
		return r, "", err
	}
	preview := Preview(string(plain))
	r.PromptPreview = &preview
	return r, string(plain), nil
}
func (s Store) List(ctx context.Context, f ListFilter) ([]Run, error) {
	if err := s.locked(ctx, func(pgx.Tx) error { return nil }); err != nil {
		return nil, err
	}
	rows, err := s.DB.Pool.Query(ctx, `SELECT `+runColumns+` FROM portway.codex_runs WHERE installation_id=$1 AND expires_at>now() AND ($2='' OR provider=$2) AND ($3='' OR state=$3) ORDER BY created_at DESC,id DESC LIMIT 50`, s.DB.Installation, f.Provider, f.State)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		r, _, err := s.scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s Store) Get(ctx context.Context, id string) (Run, error) {
	r := Run{Workspace: "portway", Sandbox: "read-only"}
	if err := s.locked(ctx, func(pgx.Tx) error { return nil }); err != nil {
		return r, err
	}
	var answer, an []byte
	r, prompt, err := s.scanRun(s.DB.Pool.QueryRow(ctx, `SELECT `+runColumns+`,result_ciphertext,result_nonce FROM portway.codex_runs WHERE installation_id=$1 AND id=$2 AND expires_at>now()`, s.DB.Installation, id), &answer, &an)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, problem.NotFound()
	}
	if err != nil {
		return r, err
	}
	r.Prompt = prompt
	if answer != nil {
		plain, err := s.Vault.Open(answer, an, s.aad(id, "answer"))
		r.Answer = string(plain)
		return r, err
	}
	return r, nil
}
func (s Store) Events(ctx context.Context, id string, after int) ([]Event, error) {
	var exists bool
	if err := s.DB.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM portway.codex_runs WHERE installation_id=$1 AND id=$2 AND expires_at>now())`, s.DB.Installation, id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, problem.NotFound()
	}
	rows, err := s.DB.Pool.Query(ctx, `SELECT sequence,ciphertext,nonce FROM portway.codex_events WHERE run_id=$1 AND sequence>$2 ORDER BY sequence LIMIT 100`, id, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var seq int
		var ct, n []byte
		if err = rows.Scan(&seq, &ct, &n); err != nil {
			return nil, err
		}
		b, e := s.Vault.Open(ct, n, s.aad(id, fmt.Sprint(seq)))
		if e != nil {
			return nil, e
		}
		var event Event
		if e = json.Unmarshal(b, &event); e != nil {
			return nil, e
		}
		out = append(out, event)
	}
	return out, rows.Err()
}
func (s Store) Stop(ctx context.Context, tx pgx.Tx, owner, id string) (any, error) {
	var state string
	if err := tx.QueryRow(ctx, `SELECT state FROM portway.codex_runs WHERE installation_id=$1 AND id=$2 FOR UPDATE`, s.DB.Installation, id).Scan(&state); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, problem.NotFound()
		}
		return nil, err
	}
	if state == "queued" || state == "running" {
		state = "stopped"
		if _, err := tx.Exec(ctx, `UPDATE portway.codex_runs SET state='stopped',finished_at=now() WHERE id=$1`, id); err != nil {
			return nil, err
		}
		if err := s.DB.Audit(ctx, tx, owner, "codex.stopped", "codex_run", id); err != nil {
			return nil, err
		}
	}
	return map[string]any{"id": id, "state": state}, nil
}
func (s Store) locked(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var id string
	if err = tx.QueryRow(ctx, `SELECT id FROM portway.installations WHERE id=$1 FOR UPDATE`, s.DB.Installation).Scan(&id); err != nil {
		return err
	}
	if err = s.sweep(ctx, tx); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// claim registers the polling bridge instance; a second laptop cannot take over a live one.
func (s Store) claim(ctx context.Context, tx pgx.Tx, instance string) error {
	ct, err := tx.Exec(ctx, `INSERT INTO portway.codex_bridge(installation_id,instance_id,workspace) VALUES($1,$2,'portway') ON CONFLICT(installation_id) DO UPDATE SET instance_id=excluded.instance_id,last_seen_at=now() WHERE portway.codex_bridge.instance_id=excluded.instance_id OR portway.codex_bridge.last_seen_at<now()-interval '30 seconds'`, s.DB.Installation, instance)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return problem.New(409, "CODEX_BRIDGE_BUSY", "Another laptop bridge is active")
	}
	return nil
}
func (s Store) Poll(ctx context.Context, p Poll) (PollResult, error) {
	out := PollResult{Commands: []Command{}}
	err := s.locked(ctx, func(tx pgx.Tx) error {
		err := s.claim(ctx, tx, p.Instance)
		if err != nil {
			return err
		}
		if out.Commands, err = s.deliver(ctx, tx); err != nil {
			return err
		}
		if p.RunID != "" {
			var device string
			err = tx.QueryRow(ctx, `SELECT device_id FROM portway.codex_runs WHERE installation_id=$1 AND id=$2 AND instance_id=$3 AND lease_hash=$4 AND state='running'`, s.DB.Installation, p.RunID, p.Instance, secure.Hash(p.Token)).Scan(&device)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			var active bool
			if err = tx.QueryRow(ctx, `SELECT revoked_at IS NULL FROM portway.device_sessions WHERE id=$1 FOR SHARE`, device).Scan(&active); err != nil {
				return err
			}
			if !active {
				return nil
			}
			_, err = tx.Exec(ctx, `UPDATE portway.codex_runs SET lease_until=now()+interval '30 seconds' WHERE id=$1`, p.RunID)
			out.Continue = err == nil
			return err
		}
		var id, provider string
		var model, effort *string
		var ctPrompt, nonce []byte
		err = tx.QueryRow(ctx, `SELECT id,prompt_ciphertext,prompt_nonce,provider,model,effort FROM portway.codex_runs WHERE installation_id=$1 AND state='queued' FOR UPDATE`, s.DB.Installation).Scan(&id, &ctPrompt, &nonce, &provider, &model, &effort)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var device string
		if err = tx.QueryRow(ctx, `SELECT d.id FROM portway.device_sessions d JOIN portway.codex_runs r ON r.device_id=d.id WHERE r.id=$1 AND d.revoked_at IS NULL FOR SHARE OF d`, id).Scan(&device); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		plain, err := s.Vault.Open(ctPrompt, nonce, s.aad(id, "prompt"))
		if err != nil {
			return err
		}
		token := secure.Token()
		_, err = tx.Exec(ctx, `UPDATE portway.codex_runs SET state='running',started_at=now(),instance_id=$2,lease_hash=$3,lease_until=now()+interval '30 seconds' WHERE id=$1`, id, p.Instance, secure.Hash(token))
		if err != nil {
			return err
		}
		out.Job = &Job{ID: id, Prompt: string(plain), Token: token, Provider: provider}
		if model != nil {
			out.Job.Model = *model
		}
		if effort != nil {
			out.Job.Effort = *effort
		}
		return nil
	})
	return out, err
}
func (s Store) Report(ctx context.Context, r Report) error {
	body, _ := json.Marshal(r.Event)
	if len(body) > 69984 {
		return problem.Invalid("Encoded event exceeds size limit")
	}
	hash := secure.Hash(string(body))
	// Completion metadata goes to run columns; the replayable event keeps its public shape.
	meta := r.Event.Meta
	public := r.Event
	public.Meta = nil
	stored, _ := json.Marshal(public)
	return s.locked(ctx, func(tx pgx.Tx) error {
		var state, lease, device, owner string
		err := tx.QueryRow(ctx, `SELECT state,coalesce(lease_hash,''),device_id,owner_id FROM portway.codex_runs WHERE installation_id=$1 AND id=$2 FOR UPDATE`, s.DB.Installation, r.RunID).Scan(&state, &lease, &device, &owner)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.NotFound()
		}
		if err != nil {
			return err
		}
		if !secure.Equal(lease, secure.Hash(r.Token)) {
			return problem.New(403, "CODEX_LEASE_DENIED", "Lease rejected")
		}
		var old string
		err = tx.QueryRow(ctx, `SELECT request_hash FROM portway.codex_events WHERE run_id=$1 AND sequence=$2`, r.RunID, r.Event.Sequence).Scan(&old)
		if err == nil {
			if old != hash {
				return problem.New(409, "CODEX_EVENT_CONFLICT", "Event sequence was already used")
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var active bool
		if err = tx.QueryRow(ctx, `SELECT revoked_at IS NULL FROM portway.device_sessions WHERE id=$1 FOR SHARE`, device).Scan(&active); err != nil {
			return err
		}
		if state != "running" || !active {
			return problem.New(409, "CODEX_RUN_CLOSED", "Run no longer accepts output")
		}
		var last int
		if err = tx.QueryRow(ctx, `SELECT coalesce(max(sequence),0) FROM portway.codex_events WHERE run_id=$1`, r.RunID).Scan(&last); err != nil {
			return err
		}
		if r.Event.Sequence != last+1 {
			return problem.New(409, "CODEX_EVENT_GAP", "Expected the next event sequence")
		}
		ct, n := s.Vault.Seal(string(stored), s.aad(r.RunID, fmt.Sprint(r.Event.Sequence)))
		if _, err = tx.Exec(ctx, `INSERT INTO portway.codex_events(run_id,sequence,request_hash,ciphertext,nonce) VALUES($1,$2,$3,$4,$5)`, r.RunID, r.Event.Sequence, hash, ct, n); err != nil {
			return err
		}
		if r.Event.Type == "completed" || r.Event.Type == "failed" {
			next := r.Event.Type
			var answer, an []byte
			var code any
			if next == "completed" {
				answer, an = s.Vault.Seal(r.Event.Text, s.aad(r.RunID, "answer"))
			} else {
				code = "AI_RUN_FAILED"
				if r.Event.Code != "" {
					code = r.Event.Code
				}
			}
			var resolved, label any
			var input, output any
			if meta != nil {
				if meta.ResolvedModel != nil {
					resolved = *meta.ResolvedModel
				}
				if meta.AccountLabel != nil {
					label = *meta.AccountLabel
				}
				if meta.Tokens != nil {
					input, output = meta.Tokens.Input, meta.Tokens.Output
				}
			}
			if _, err = tx.Exec(ctx, `UPDATE portway.codex_runs SET state=$2,result_ciphertext=$3,result_nonce=$4,error_code=$5,finished_at=now(),resolved_model=coalesce($6,resolved_model),account_label=coalesce($7,account_label),input_tokens=coalesce($8,input_tokens),output_tokens=coalesce($9,output_tokens) WHERE id=$1`, r.RunID, next, answer, an, code, resolved, label, input, output); err != nil {
				return err
			}
			return s.DB.Audit(ctx, tx, owner, "codex."+next, "codex_run", r.RunID)
		}
		return nil
	})
}

// SaveState stores the latest validated laptop snapshot encrypted at rest.
func (s Store) SaveState(ctx context.Context, state Snapshot) error {
	body, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if len(body) > 65536 {
		return problem.New(413, "REQUEST_TOO_LARGE", "Bridge state exceeds 64 KiB")
	}
	return s.locked(ctx, func(tx pgx.Tx) error {
		if err := s.claim(ctx, tx, state.Instance); err != nil {
			return err
		}
		ct, nonce := s.Vault.Seal(string(body), s.aad("bridge", "state"))
		_, err := tx.Exec(ctx, `UPDATE portway.codex_bridge SET state_ciphertext=$2,state_nonce=$3,state_updated_at=now() WHERE installation_id=$1`, s.DB.Installation, ct, nonce)
		return err
	})
}
