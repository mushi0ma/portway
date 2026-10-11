package execution

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"portway/internal/problem"
	"portway/internal/secure"
)

type Store struct {
	Pool         *pgxpool.Pool
	Installation string
}

func (s Store) Check(ctx context.Context) error {
	var ok bool
	err := s.Pool.QueryRow(ctx, `SELECT current_user='portway_execution' AND NOT rolsuper AND NOT rolbypassrls AND NOT has_schema_privilege(current_user,'portway_execution','CREATE') AND NOT has_schema_privilege(current_user,'portway','USAGE') AND NOT has_schema_privilege(current_user,'portway_audit','USAGE') FROM pg_roles WHERE rolname=current_user`).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("execution role must be isolated")
	}
	_, err = s.Pool.Exec(ctx, `SELECT id FROM portway_execution.chains LIMIT 0`)
	return err
}
func (s Store) Lookup(ctx context.Context, owner, id string, run bool) (Chain, error) {
	var body []byte
	sql := `SELECT body FROM portway_execution.chains WHERE installation_id=$1 AND owner_id=$2 AND id=$3`
	if run {
		sql = `SELECT c.body FROM portway_execution.chains c JOIN portway_execution.runs r ON r.installation_id=c.installation_id AND r.chain_id=c.id WHERE c.installation_id=$1 AND c.owner_id=$2 AND r.id=$3`
	}
	err := s.Pool.QueryRow(ctx, sql, s.Installation, owner, id).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return Chain{}, problem.NotFound()
	}
	if err != nil {
		return Chain{}, err
	}
	var c Chain
	err = json.Unmarshal(body, &c)
	return c, err
}
func (s Store) List(ctx context.Context, owner string) ([]RunSummary, error) {
	rows, err := s.Pool.Query(ctx, `SELECT body FROM portway_execution.chains WHERE installation_id=$1 AND owner_id=$2 ORDER BY created_at DESC,id DESC LIMIT 100`, s.Installation, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RunSummary{}
	for rows.Next() {
		var b []byte
		var c Chain
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &c); err != nil {
			return nil, err
		}
		for _, r := range c.Runs {
			out = append(out, RunSummary{ID: r.ID, ChainID: r.ChainID, State: r.State, Reason: r.Reason, Version: r.Version, Deadline: r.Deadline, Budget: r.Budget})
		}
	}
	return out, rows.Err()
}
func (s Store) begin(ctx context.Context) (pgx.Tx, time.Time, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	// One bounded installation lock also serializes admission capacity and keys.
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,76241))`, s.Installation)
	var now time.Time
	if err == nil {
		err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now)
	}
	if err != nil {
		_ = tx.Rollback(context.Background())
		return nil, now, err
	}
	return tx, now, nil
}
func (s Store) replay(ctx context.Context, tx pgx.Tx, owner, op, key, hash string) (*Chain, error) {
	if len(key) < 8 || len(key) > 200 {
		return nil, problem.Invalid("Idempotency-Key must contain 8–200 characters")
	}
	_, err := tx.Exec(ctx, `DELETE FROM portway_execution.request_keys WHERE installation_id=$1 AND owner_id=$2 AND operation=$3 AND key_hash=$4 AND expires_at<=clock_timestamp()`, s.Installation, owner, op, secure.Hash(key))
	if err != nil {
		return nil, err
	}
	var old string
	var body []byte
	err = tx.QueryRow(ctx, `SELECT request_hash,response FROM portway_execution.request_keys WHERE installation_id=$1 AND owner_id=$2 AND operation=$3 AND key_hash=$4`, s.Installation, owner, op, secure.Hash(key)).Scan(&old, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if old != hash {
		return nil, conflict("IDEMPOTENCY_CONFLICT")
	}
	var c Chain
	err = json.Unmarshal(body, &c)
	return &c, err
}
func (s Store) remember(ctx context.Context, tx pgx.Tx, owner, op, key, hash string, c Chain) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO portway_execution.request_keys(installation_id,owner_id,operation,key_hash,request_hash,response) VALUES($1,$2,$3,$4,$5,$6)`, s.Installation, owner, op, secure.Hash(key), hash, json.RawMessage(b))
	return err
}
func (s Store) persist(ctx context.Context, tx pgx.Tx, c Chain, ev []Event) error {
	if err := c.Validate(); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE portway_execution.chains SET state=$3,body=$4 WHERE installation_id=$1 AND id=$2`, s.Installation, c.ID, c.State, json.RawMessage(b))
	if err != nil {
		return err
	}
	for _, e := range ev {
		b, err = json.Marshal(e)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO portway_execution.events(installation_id,id,run_id,seq,body) VALUES($1,$2,$3,$4,$5)`, s.Installation, e.ID, e.RunID, e.Seq, json.RawMessage(b))
		if err != nil {
			return err
		}
		envelope := Envelope{1, e.ID, s.Installation, "execution", e.Type, e.RunID, e.CorrelationID, e.ServerTime, e}
		b, err = json.Marshal(envelope)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO portway_execution.outbox(installation_id,id,body) VALUES($1,$2,$3)`, s.Installation, e.ID, json.RawMessage(b))
		if err != nil {
			return err
		}
	}
	return nil
}
func (s Store) Create(ctx context.Context, owner, key string, in CreateInput) (Chain, error) {
	b, _ := json.Marshal(in)
	hash := secure.Hash(string(b))
	tx, now, err := s.begin(ctx)
	if err != nil {
		return Chain{}, err
	}
	defer tx.Rollback(context.Background())
	old, err := s.replay(ctx, tx, owner, "create", key, hash)
	if err != nil {
		return Chain{}, err
	}
	if old != nil {
		return *old, nil
	}
	var active, pending int
	var oldest *time.Time
	err = tx.QueryRow(ctx, `SELECT count(*) FROM portway_execution.chains WHERE installation_id=$1 AND state IN ('active','stopping')`, s.Installation).Scan(&active)
	if err != nil {
		return Chain{}, err
	}
	if active >= 2 {
		return Chain{}, problem.New(429, "CAPACITY_LIMIT", "Two active chains already exist")
	}
	err = tx.QueryRow(ctx, `SELECT count(*),min(created_at) FROM portway_execution.outbox WHERE installation_id=$1 AND published_at IS NULL`, s.Installation).Scan(&pending, &oldest)
	if err != nil {
		return Chain{}, err
	}
	if pending >= 10000 || oldest != nil && !now.Before(oldest.Add(24*time.Hour)) {
		return Chain{}, problem.Unavailable()
	}
	c, e, err := NewChain(s.Installation, owner, in, now)
	if err != nil {
		return c, err
	}
	b, _ = json.Marshal(c)
	_, err = tx.Exec(ctx, `INSERT INTO portway_execution.chains(installation_id,owner_id,id,state,body) VALUES($1,$2,$3,$4,$5)`, s.Installation, owner, c.ID, c.State, json.RawMessage(b))
	if err != nil {
		return c, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO portway_execution.runs(installation_id,id,chain_id) VALUES($1,$2,$3)`, s.Installation, c.Runs[0].ID, c.ID)
	if err != nil {
		return c, err
	}
	if err = s.persist(ctx, tx, c, []Event{e}); err != nil {
		return c, err
	}
	if err = s.remember(ctx, tx, owner, "create", key, hash, c); err != nil {
		return c, err
	}
	return c, tx.Commit(ctx)
}

// Change holds the chain row lock across budget, state, event and outbox writes.
// fn only mutates memory; it must never dispatch external work in this transaction.
func (s Store) Change(ctx context.Context, owner, id string, fn func(*Chain, time.Time) ([]Event, error)) (Chain, error) {
	tx, now, err := s.begin(ctx)
	if err != nil {
		return Chain{}, err
	}
	defer tx.Rollback(context.Background())
	c, err := s.lock(ctx, tx, owner, id)
	if err != nil {
		return c, err
	}
	ev, err := fn(&c, now)
	if err != nil {
		return c, err
	}
	if err = s.persist(ctx, tx, c, ev); err != nil {
		return c, err
	}
	return c, tx.Commit(ctx)
}
func (s Store) lock(ctx context.Context, tx pgx.Tx, owner, id string) (Chain, error) {
	var b []byte
	var c Chain
	err := tx.QueryRow(ctx, `SELECT body FROM portway_execution.chains WHERE installation_id=$1 AND owner_id=$2 AND id=$3 FOR UPDATE`, s.Installation, owner, id).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, problem.NotFound()
	}
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}
func (s Store) Stop(ctx context.Context, owner, id, key string, run bool) (Chain, error) {
	// Resolve only within this service's schema; no query of control-owned data.
	c, err := s.Lookup(ctx, owner, id, run)
	if err != nil {
		return c, err
	}
	tx, now, err := s.begin(ctx)
	if err != nil {
		return c, err
	}
	defer tx.Rollback(context.Background())
	op := "stop-chain/" + id
	target := ""
	if run {
		op = "stop-run/" + id
		target = id
	}
	if _, err = s.replay(ctx, tx, owner, op, key, secure.Hash(op)); err != nil {
		return c, err
	}
	// Repeated stop returns current state, never an obsolete cached 'stopping'.
	c, err = s.lock(ctx, tx, owner, c.ID)
	if err != nil {
		return c, err
	}
	ev, err := c.Stop(target, now)
	if err != nil {
		return c, err
	}
	if err = s.persist(ctx, tx, c, ev); err != nil {
		return c, err
	}
	_, err = tx.Exec(ctx, `DELETE FROM portway_execution.request_keys WHERE installation_id=$1 AND owner_id=$2 AND operation=$3 AND key_hash=$4`, s.Installation, owner, op, secure.Hash(key))
	if err != nil {
		return c, err
	}
	if err = s.remember(ctx, tx, owner, op, key, secure.Hash(op), c); err != nil {
		return c, err
	}
	return c, tx.Commit(ctx)
}
func (s Store) History(ctx context.Context, owner, run string, after int64) ([]Event, error) {
	c, err := s.Lookup(ctx, owner, run, true)
	if err != nil {
		return nil, err
	}
	r, _ := c.Find(run)
	if after < 0 {
		return nil, problem.New(400, "INVALID_CURSOR", "Invalid event sequence")
	}
	if after > r.Seq {
		return nil, conflict("CURSOR_AHEAD")
	}
	rows, err := s.Pool.Query(ctx, `SELECT body FROM portway_execution.events WHERE installation_id=$1 AND run_id=$2 AND seq>$3 AND seq<=$4 ORDER BY seq LIMIT 100`, s.Installation, run, after, r.Seq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var b []byte
		var e Event
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Reconcile is called at process startup under the dedicated singleton lock.
// It never changes already terminal runs or automatically retries dispatches.
func (s Store) Reconcile(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT owner_id,id FROM portway_execution.chains WHERE installation_id=$1 AND state IN ('active','stopping') ORDER BY id`, s.Installation)
	if err != nil {
		return err
	}
	type item struct{ owner, id string }
	items := []item{}
	for rows.Next() {
		var i item
		if err = rows.Scan(&i.owner, &i.id); err != nil {
			rows.Close()
			return err
		}
		items = append(items, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, i := range items {
		_, err = s.Change(ctx, i.owner, i.id, func(c *Chain, now time.Time) ([]Event, error) { return c.Reconcile(now), nil })
		if err != nil {
			return err
		}
	}
	return nil
}
