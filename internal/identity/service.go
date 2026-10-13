package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/problem"
	"portway/internal/secure"
	"portway/internal/storage"
)

type Service struct {
	Store          *storage.Store
	Auth           *Supabase
	RequestHashKey []byte
}
type Principal struct{ OwnerID, DeviceID, Subject, SessionID string }
type Profile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
type LoginResult struct {
	Session     Session `json:"session"`
	DeviceProof string  `json:"device_proof"`
	DeviceID    string  `json:"device_id"`
	Owner       Profile `json:"owner"`
}
type Registration struct {
	ID    string `json:"registration_id"`
	State string `json:"state"`
}
type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func validEmail(email string) bool {
	a, err := mail.ParseAddress(email)
	return err == nil && a.Address == email && len(email) <= 254
}
func (s *Service) Register(ctx context.Context, in RegisterInput, proof, key string) (Registration, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if len(in.Name) < 1 || len(in.Name) > 120 || !validEmail(in.Email) || len(in.Password) < 8 || len(in.Password) > 1024 || len(key) < 8 || len(key) > 200 {
		return Registration{}, problem.Invalid("Invalid registration fields")
	}
	b, _ := json.Marshal(in)
	hash := secure.MAC(s.RequestHashKey, b)
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		return Registration{}, err
	}
	defer tx.Rollback(context.Background())
	var status string
	var setup *string
	err = tx.QueryRow(ctx, `SELECT status,setup_hash FROM portway.installations WHERE id=$1 FOR UPDATE`, s.Store.Installation).Scan(&status, &setup)
	if err != nil {
		return Registration{}, err
	}
	if status == "claimed" {
		return Registration{}, problem.New(409, "OWNER_ALREADY_BOUND", "Installation already has an owner")
	}
	if setup == nil || !secure.Equal(secure.Hash(proof), *setup) {
		return Registration{}, problem.New(403, "SETUP_DENIED", "Setup proof rejected")
	}
	var out Registration
	var oldKey, oldHash string
	err = tx.QueryRow(ctx, `SELECT id,state,key_hash,request_hash FROM portway.registration_intents WHERE installation_id=$1`, s.Store.Installation).Scan(&out.ID, &out.State, &oldKey, &oldHash)
	if err == nil {
		if oldKey != secure.Hash(key) || oldHash != hash {
			return out, problem.New(409, "IDEMPOTENCY_CONFLICT", "A registration is already reserved")
		}
		if out.State == "sent" {
			out.State = "pending_confirmation"
			return out, nil
		}
		return out, problem.New(503, "REGISTRATION_UNCERTAIN", "Registration requires operator reconciliation; do not create another signup")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	out = Registration{secure.ID(), "pending_confirmation"}
	_, err = tx.Exec(ctx, `INSERT INTO portway.registration_intents(id,installation_id,email_hash,display_name,state,key_hash,request_hash) VALUES($1,$2,$3,$4,'pending',$5,$6);`, out.ID, s.Store.Installation, secure.Hash(in.Email), in.Name, secure.Hash(key), hash)
	if err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `UPDATE portway.installations SET status='pending_confirmation' WHERE id=$1`, s.Store.Installation); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	// Never retry signup automatically after the durable intent: its external outcome may be unknown.
	u, callErr := s.Auth.Signup(ctx, in.Email, in.Password)
	if callErr != nil || !secure.UUID(u.ID) {
		_, _ = s.Store.Pool.Exec(ctx, `UPDATE portway.registration_intents SET state='uncertain' WHERE id=$1 AND state='pending'`, out.ID)
		return out, problem.New(503, "REGISTRATION_UNCERTAIN", "Signup outcome requires operator reconciliation")
	}
	_, err = s.Store.Pool.Exec(ctx, `UPDATE portway.registration_intents SET auth_user_id=$2,state='sent' WHERE id=$1 AND state='pending'`, out.ID, u.ID)
	return out, err
}
func (s *Service) Confirm(ctx context.Context, tokenHash string) error {
	if len(tokenHash) < 16 || len(tokenHash) > 512 {
		return problem.Invalid("Invalid confirmation")
	}
	u, err := s.Auth.Confirm(ctx, tokenHash)
	if err != nil {
		return err
	}
	return s.claimOwner(ctx, u)
}

// Both the custom callback and a password login after Supabase's standard
// confirmation email may finish setup. Only the reserved, verified subject can
// claim the installation; email or callback parameters alone are not authority.
func (s *Service) claimOwner(ctx context.Context, u User) error {
	if !secure.UUID(u.ID) || u.EmailConfirmedAt == nil {
		return problem.New(403, "SETUP_DENIED", "Email confirmation required")
	}
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var status string
	if err = tx.QueryRow(ctx, `SELECT status FROM portway.installations WHERE id=$1 FOR UPDATE`, s.Store.Installation).Scan(&status); err != nil {
		return err
	}
	var subject *string
	var emailHash, name, id, state string
	err = tx.QueryRow(ctx, `SELECT id,auth_user_id,email_hash,display_name,state FROM portway.registration_intents WHERE installation_id=$1`, s.Store.Installation).Scan(&id, &subject, &emailHash, &name, &state)
	if err != nil {
		return problem.New(403, "SETUP_DENIED", "No matching registration")
	}
	if subject == nil || *subject != u.ID || emailHash != secure.Hash(strings.ToLower(u.Email)) || (state != "sent" && state != "claimed") {
		return problem.New(403, "SETUP_DENIED", "Confirmation does not match registration")
	}
	if status == "claimed" {
		return nil
	}
	owner := secure.ID()
	_, err = tx.Exec(ctx, `INSERT INTO portway.owners(id,installation_id,auth_user_id,display_name,email) VALUES($1,$2,$3,$4,$5)`, owner, s.Store.Installation, u.ID, name, u.Email)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE portway.installations SET status='claimed',setup_hash=NULL WHERE id=$1`, s.Store.Installation); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE portway.registration_intents SET state='claimed' WHERE id=$1`, id); err != nil {
		return err
	}
	if err = s.Store.Audit(ctx, tx, owner, "owner.bound", "owner", owner); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Service) Login(ctx context.Context, email, password, label string) (LoginResult, error) {
	var out LoginResult
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) || password == "" || len(password) > 1024 || len(label) > 120 {
		return out, problem.Invalid("Invalid login fields")
	}
	session, err := s.Auth.Login(ctx, email, password)
	if err != nil {
		return out, err
	}
	c, err := s.Auth.Verify(ctx, session.AccessToken)
	if err != nil {
		return out, err
	}
	u, err := s.Auth.User(ctx, session.AccessToken)
	if err != nil {
		return out, err
	}
	if u.ID != c.Subject || u.EmailConfirmedAt == nil {
		return out, problem.New(403, "OWNER_REQUIRED", "Confirmed owner required")
	}
	var p Profile
	err = s.Store.Pool.QueryRow(ctx, `SELECT id,display_name,email FROM portway.owners WHERE installation_id=$1 AND auth_user_id=$2`, s.Store.Installation, c.Subject).Scan(&p.ID, &p.Name, &p.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = s.claimOwner(ctx, u); err != nil {
			return out, err
		}
		err = s.Store.Pool.QueryRow(ctx, `SELECT id,display_name,email FROM portway.owners WHERE installation_id=$1 AND auth_user_id=$2`, s.Store.Installation, c.Subject).Scan(&p.ID, &p.Name, &p.Email)
	}
	if err != nil {
		return out, err
	}
	proof, id := secure.Token(), secure.ID()
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `INSERT INTO portway.device_sessions(id,installation_id,owner_id,proof_hash,auth_session_id,label) VALUES($1,$2,$3,$4,$5,$6)`, id, s.Store.Installation, p.ID, secure.Hash(proof), c.SessionID, label); err != nil {
		return out, err
	}
	if err = s.Store.Audit(ctx, tx, p.ID, "device.login", "device", id); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return LoginResult{session, proof, id, p}, nil
}
func (s *Service) device(ctx context.Context, proof string, active bool) (Principal, error) {
	var p Principal
	var revoked *time.Time
	if len(proof) < 40 || len(proof) > 128 {
		return p, problem.New(401, "AUTH_REQUIRED", "Device proof required")
	}
	err := s.Store.Pool.QueryRow(ctx, `SELECT d.id,d.owner_id,o.auth_user_id,d.auth_session_id,d.revoked_at FROM portway.device_sessions d JOIN portway.owners o ON o.id=d.owner_id AND o.installation_id=d.installation_id WHERE d.installation_id=$1 AND d.proof_hash=$2`, s.Store.Installation, secure.Hash(proof)).Scan(&p.DeviceID, &p.OwnerID, &p.Subject, &p.SessionID, &revoked)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && active && revoked != nil {
		return p, problem.New(401, "DEVICE_REVOKED", "Device is unknown or revoked")
	}
	return p, err
}
func (s *Service) Authenticate(ctx context.Context, token, proof string) (Principal, error) {
	c, err := s.Auth.Verify(ctx, token)
	if err != nil {
		return Principal{}, err
	}
	p, err := s.device(ctx, proof, true)
	if err != nil {
		return p, err
	}
	if c.Subject != p.Subject || c.SessionID != p.SessionID {
		return p, problem.New(403, "OWNER_REQUIRED", "Token does not match device session")
	}
	return p, nil
}
func (s *Service) Refresh(ctx context.Context, token, proof string) (Session, error) {
	p, err := s.device(ctx, proof, true)
	if err != nil {
		return Session{}, err
	}
	v, err := s.Auth.Refresh(ctx, token)
	if err != nil {
		return v, err
	}
	c, err := s.Auth.Verify(ctx, v.AccessToken)
	if err != nil {
		return Session{}, err
	}
	if c.Subject != p.Subject || c.SessionID != p.SessionID {
		return Session{}, problem.New(403, "OWNER_REQUIRED", "Refreshed session does not match device")
	}
	u, err := s.Auth.User(ctx, v.AccessToken)
	if err != nil {
		return Session{}, err
	}
	if u.ID != p.Subject || u.EmailConfirmedAt == nil {
		return Session{}, problem.New(403, "OWNER_REQUIRED", "Confirmed owner required")
	}
	if _, err = s.device(ctx, proof, true); err != nil {
		return Session{}, err
	}
	return v, nil
}
func (s *Service) Logout(ctx context.Context, proof, token string) (string, error) {
	p, err := s.device(ctx, proof, false)
	if err != nil {
		return "", err
	}
	if err = s.Revoke(ctx, p, p.DeviceID); err != nil {
		return "", err
	}
	// An optional token may only revoke its matching upstream session, never an unrelated one.
	if token != "" {
		if c, err := s.Auth.Verify(ctx, token); err == nil && c.Subject == p.Subject && c.SessionID == p.SessionID {
			if s.Auth.Logout(ctx, token) == nil {
				return "revoked", nil
			}
			return "unavailable", nil
		}
	}
	return "not_attempted", nil
}
func (s *Service) Revoke(ctx context.Context, p Principal, id string) error {
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	ct, err := tx.Exec(ctx, `UPDATE portway.device_sessions SET revoked_at=coalesce(revoked_at,now()) WHERE installation_id=$1 AND owner_id=$2 AND id=$3`, s.Store.Installation, p.OwnerID, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return problem.NotFound()
	}
	if err = s.Store.Audit(ctx, tx, p.OwnerID, "device.revoked", "device", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
