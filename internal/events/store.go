package events

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"portway/internal/secure"
)

type Store struct {
	Pool         *pgxpool.Pool
	Installation string
}

type Publish func(context.Context, Event) error

// RelayOne holds one source DB transaction until the broker acknowledges the
// record. A commit failure can duplicate delivery, never acknowledge an unsent fact.
func (s Store) RelayOne(ctx context.Context, publish Publish) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.Background())
	var locked bool
	err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtext('portway.relay'),hashtext($1))`, s.Installation).Scan(&locked)
	if err != nil || !locked {
		return false, err
	}
	var sequence int64
	e := Event{SchemaVersion: 1, Producer: "control-api", Type: "audit.recorded", InstallationID: s.Installation}
	err = tx.QueryRow(ctx, `SELECT sequence,event_id,actor_id,action,object_type,object_id,outcome,occurred_at FROM portway.outbox WHERE installation_id=$1 AND published_at IS NULL ORDER BY sequence LIMIT 1 FOR UPDATE`, s.Installation).Scan(&sequence, &e.EventID, &e.Payload.ActorID, &e.Payload.Action, &e.Payload.ObjectType, &e.Payload.ObjectID, &e.Payload.Outcome, &e.OccurredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	e.CorrelationID = e.EventID
	if e.Payload.ObjectID != nil {
		e.AggregateID = *e.Payload.ObjectID
	}
	if err = e.Validate(time.Now()); err != nil {
		return false, err
	}
	if err = publish(ctx, e); err != nil {
		return false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE portway.outbox SET published_at=now() WHERE sequence=$1`, sequence); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// Apply records receipt and projection atomically. Same ID with different
// contents fails closed instead of accepting producer corruption as a duplicate.
func (s Store) Apply(ctx context.Context, e Event) error {
	if e.InstallationID != s.Installation {
		return errors.New("event installation mismatch")
	}
	if err := e.Validate(time.Now()); err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	hash := secure.Hash(string(b))
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	tag, err := tx.Exec(ctx, `INSERT INTO portway_audit.inbox(installation_id,event_id,content_hash) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, e.InstallationID, e.EventID, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var old string
		if err = tx.QueryRow(ctx, `SELECT content_hash FROM portway_audit.inbox WHERE installation_id=$1 AND event_id=$2`, e.InstallationID, e.EventID).Scan(&old); err != nil {
			return err
		}
		if old != hash {
			return errors.New("conflicting duplicate event")
		}
		return tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `INSERT INTO portway_audit.entries(installation_id,event_id,actor_id,action,object_type,object_id,outcome,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, e.InstallationID, e.EventID, e.Payload.ActorID, e.Payload.Action, e.Payload.ObjectType, e.Payload.ObjectID, e.Payload.Outcome, e.OccurredAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s Store) CheckRole(ctx context.Context, mode string) error {
	role, schema, table := "portway_relay", "portway", "outbox"
	if mode == "audit" {
		role, schema, table = "portway_audit", "portway_audit", "inbox"
	}
	var current string
	var privileged bool
	err := s.Pool.QueryRow(ctx, `SELECT current_user,rolsuper OR rolbypassrls OR has_schema_privilege(current_user,$1,'CREATE') FROM pg_roles WHERE rolname=current_user`, schema).Scan(&current, &privileged)
	if err != nil {
		return errors.New("worker database unavailable")
	}
	if privileged || current != role {
		return errors.New("worker requires its dedicated database role")
	}
	_, err = s.Pool.Exec(ctx, "SELECT 1 FROM "+pgx.Identifier{schema, table}.Sanitize()+" LIMIT 0")
	return err
}

func (s Store) Prune(ctx context.Context, mode string) error {
	sql := `SELECT portway.prune_outbox($1)`
	if mode == "audit" {
		sql = `SELECT portway_audit.prune($1)`
	}
	_, err := s.Pool.Exec(ctx, sql, s.Installation)
	return err
}
