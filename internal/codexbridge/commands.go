package codexbridge

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/problem"
	"portway/internal/secure"
)

const loginWindow = 15 * time.Minute

// Account and usage commands are owner intents for the laptop bridge. They are
// delivered once; the bridge reports the outcome by ack and by its next snapshot.
func (s Store) online(ctx context.Context, tx pgx.Tx) error {
	if err := s.sweep(ctx, tx); err != nil {
		return err
	}
	var connected bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM portway.codex_bridge WHERE installation_id=$1 AND last_seen_at>now()-interval '15 seconds')`, s.DB.Installation).Scan(&connected); err != nil {
		return err
	}
	if !connected {
		return problem.New(503, "CODEX_OFFLINE", "Laptop bridge is offline")
	}
	return nil
}
func (s Store) queue(ctx context.Context, tx pgx.Tx, owner, device, kind string, p CommandPayload) (string, error) {
	var pending int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM portway.codex_commands WHERE installation_id=$1 AND state IN ('queued','delivered')`, s.DB.Installation).Scan(&pending); err != nil {
		return "", err
	}
	if pending >= 10 {
		return "", problem.New(429, "AI_COMMANDS_BUSY", "Too many pending laptop commands")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM portway.codex_commands WHERE installation_id=$1 AND created_at<now()-interval '1 day' AND state IN ('done','failed')`, s.DB.Installation); err != nil {
		return "", err
	}
	id := secure.ID()
	body, _ := json.Marshal(p)
	ct, nonce := s.Vault.Seal(string(body), s.aad(id, "command"))
	_, err := tx.Exec(ctx, `INSERT INTO portway.codex_commands(id,installation_id,owner_id,device_id,kind,state,provider,method,login_id,payload_ciphertext,payload_nonce) VALUES($1,$2,$3,$4,$5,'queued',$6,$7,$8,$9,$10)`, id, s.DB.Installation, owner, device, kind, opt(p.Provider), opt(p.Method), opt(p.LoginID), ct, nonce)
	if err != nil {
		return "", err
	}
	return id, s.DB.Audit(ctx, tx, owner, "codex.command_queued", "codex_command", id)
}

// currentLogin derives the latest login flow from the owner's command and the bridge snapshot.
func (s Store) currentLogin(ctx context.Context, q querier, state *Snapshot, stateAt *time.Time) (*Login, error) {
	l := Login{}
	var cmdState string
	var code *string
	var created time.Time
	err := q.QueryRow(ctx, `SELECT login_id,provider,method,state,error_code,created_at,updated_at FROM portway.codex_commands WHERE installation_id=$1 AND kind='login_start' AND created_at>now()-interval '15 minutes' ORDER BY created_at DESC,id DESC LIMIT 1`, s.DB.Installation).Scan(&l.ID, &l.Provider, &l.Method, &cmdState, &code, &created, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if state != nil && state.Login != nil && state.Login.ID == l.ID {
		out := *state.Login
		if !out.Terminal() && out.ExpiresAt != nil && time.Now().After(out.ExpiresAt.Add(time.Minute)) {
			expired := "LOGIN_TIMEOUT"
			out.State, out.ErrorCode, out.URL, out.UserCode, out.NeedsCode = "expired", &expired, nil, nil, false
		}
		return &out, nil
	}
	expires := created.Add(loginWindow)
	l.State, l.ExpiresAt = "starting", &expires
	switch {
	case cmdState == "failed":
		l.State, l.ErrorCode = "failed", code
		if code == nil {
			failed := "LOGIN_FAILED"
			l.ErrorCode = &failed
		}
	case cmdState == "done" && stateAt != nil && stateAt.After(l.UpdatedAt.Add(5*time.Second)):
		// The bridge acknowledged the start but later reported no such flow (restart).
		failed := "LOGIN_FAILED"
		l.State, l.ErrorCode = "failed", &failed
	case time.Now().After(expires):
		expired := "LOGIN_TIMEOUT"
		l.State, l.ErrorCode = "expired", &expired
	}
	return &l, nil
}
func (s Store) loginView(ctx context.Context, tx pgx.Tx) (*Login, *Snapshot, error) {
	state, at, _, _, err := s.loadState(ctx, tx)
	if err != nil {
		return nil, nil, err
	}
	l, err := s.currentLogin(ctx, tx, state, at)
	return l, state, err
}
func (s Store) StartLogin(ctx context.Context, tx pgx.Tx, owner, device, provider, method string) (any, error) {
	if err := s.online(ctx, tx); err != nil {
		return nil, err
	}
	current, _, err := s.loginView(ctx, tx)
	if err != nil {
		return nil, err
	}
	if current != nil && !current.Terminal() {
		e := problem.New(409, "AI_LOGIN_ACTIVE", "Another account login is in progress")
		e.Details = map[string]any{"login_id": current.ID}
		return nil, e
	}
	login := secure.ID()
	id, err := s.queue(ctx, tx, owner, device, "login_start", CommandPayload{LoginID: login, Provider: provider, Method: method})
	return map[string]any{"command_id": id, "login_id": login}, err
}

// LoginCommand queues a pasted code or a cancellation for an existing login flow.
func (s Store) LoginCommand(ctx context.Context, tx pgx.Tx, owner, device, login, kind, code string) (any, error) {
	if err := s.online(ctx, tx); err != nil {
		return nil, err
	}
	var provider string
	err := tx.QueryRow(ctx, `SELECT provider FROM portway.codex_commands WHERE installation_id=$1 AND kind='login_start' AND login_id=$2`, s.DB.Installation, login).Scan(&provider)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, problem.NotFound()
	}
	if err != nil {
		return nil, err
	}
	if kind == "login_code" {
		current, _, err := s.loginView(ctx, tx)
		if err != nil {
			return nil, err
		}
		if current == nil || current.ID != login || current.Terminal() || provider != ProviderClaude {
			return nil, problem.New(409, "AI_LOGIN_NOT_ACTIVE", "This login does not accept a code")
		}
	}
	id, err := s.queue(ctx, tx, owner, device, kind, CommandPayload{LoginID: login, Code: code})
	return map[string]any{"command_id": id}, err
}
func (s Store) AccountCommand(ctx context.Context, tx pgx.Tx, owner, device, kind, provider, account string) (any, error) {
	if err := s.online(ctx, tx); err != nil {
		return nil, err
	}
	if kind == "account_remove" && account == provider+"-system" {
		return nil, problem.New(409, "AI_ACCOUNT_SYSTEM", "The laptop's default CLI login cannot be removed")
	}
	state, _, _, _, err := s.loadState(ctx, tx)
	if err != nil {
		return nil, err
	}
	if p := state.provider(provider); state != nil {
		found := false
		if p != nil {
			for _, a := range p.Accounts {
				found = found || a.ID == account
			}
		}
		if !found {
			return nil, problem.NotFound()
		}
	}
	id, err := s.queue(ctx, tx, owner, device, kind, CommandPayload{Provider: provider, AccountID: account})
	return map[string]any{"command_id": id}, err
}
func (s Store) RefreshUsage(ctx context.Context, tx pgx.Tx, owner, device string) (any, error) {
	if err := s.online(ctx, tx); err != nil {
		return nil, err
	}
	id, err := s.queue(ctx, tx, owner, device, "usage_refresh", CommandPayload{})
	return map[string]any{"command_id": id}, err
}

// deliver hands queued commands to the polling bridge once and wipes their payload.
func (s Store) deliver(ctx context.Context, tx pgx.Tx) ([]Command, error) {
	rows, err := tx.Query(ctx, `SELECT id,kind,payload_ciphertext,payload_nonce FROM portway.codex_commands WHERE installation_id=$1 AND state='queued' ORDER BY created_at,id LIMIT 10 FOR UPDATE`, s.DB.Installation)
	if err != nil {
		return nil, err
	}
	out := []Command{}
	for rows.Next() {
		var c Command
		var ct, nonce []byte
		if err = rows.Scan(&c.ID, &c.Kind, &ct, &nonce); err != nil {
			rows.Close()
			return nil, err
		}
		plain, err := s.Vault.Open(ct, nonce, s.aad(c.ID, "command"))
		if err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(plain, &c.Payload); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, c)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for _, c := range out {
		if _, err = tx.Exec(ctx, `UPDATE portway.codex_commands SET state='delivered',delivered_at=now(),updated_at=now(),payload_ciphertext=NULL,payload_nonce=NULL WHERE id=$1`, c.ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}
func (s Store) AckCommand(ctx context.Context, id string, a Ack) error {
	return s.locked(ctx, func(tx pgx.Tx) error {
		var state string
		var code *string
		err := tx.QueryRow(ctx, `SELECT state,error_code FROM portway.codex_commands WHERE installation_id=$1 AND id=$2 FOR UPDATE`, s.DB.Installation, id).Scan(&state, &code)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.NotFound()
		}
		if err != nil {
			return err
		}
		if state == a.State && ((code == nil && a.ErrorCode == nil) || (code != nil && a.ErrorCode != nil && *code == *a.ErrorCode)) {
			return nil
		}
		if state != "delivered" {
			return problem.New(409, "CODEX_COMMAND_CLOSED", "Command is not awaiting an outcome")
		}
		_, err = tx.Exec(ctx, `UPDATE portway.codex_commands SET state=$2,error_code=$3,updated_at=now() WHERE id=$1`, id, a.State, a.ErrorCode)
		return err
	})
}
