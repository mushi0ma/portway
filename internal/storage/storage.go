package storage

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"portway/internal/problem"
	"portway/internal/secure"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	Pool         *pgxpool.Pool
	Installation string
}

func Open(ctx context.Context, dsn, installation string) (*Store, error) {
	return OpenWithLimit(ctx, dsn, installation, 2)
}

func OpenWithLimit(ctx context.Context, dsn, installation string, maxConns int32) (*Store, error) {
	if maxConns < 1 || maxConns > 8 {
		return nil, errors.New("database pool size must be between 1 and 8")
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	c.MaxConns = maxConns
	c.MinConns = 0
	c.MaxConnIdleTime = 2 * time.Minute
	c.MaxConnLifetime = 30 * time.Minute
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	// QueryExecModeExec works with session and transaction poolers without named prepared statements.
	c.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	p, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, errors.New("database connection failed")
	}
	return &Store{p, installation}, nil
}
func (s *Store) Close() { s.Pool.Close() }

// EventStatus exposes only queue health, not event bodies, to the source API.
func (s *Store) EventStatus(ctx context.Context) (int64, *time.Time, error) {
	var pending int64
	var oldest *time.Time
	err := s.Pool.QueryRow(ctx, `SELECT pending,oldest_at FROM portway.event_status($1)`, s.Installation).Scan(&pending, &oldest)
	return pending, oldest, err
}
func (s *Store) CheckApplicationRole(ctx context.Context) error {
	var privileged bool
	err := s.Pool.QueryRow(ctx, `SELECT rolsuper OR rolbypassrls OR has_schema_privilege(current_user,'portway','CREATE') FROM pg_roles WHERE rolname=current_user`).Scan(&privileged)
	if err != nil {
		return err
	}
	if privileged {
		return errors.New("API database role must not be privileged or own the schema")
	}
	return nil
}
func MigrationCount() int { es, _ := migrations.ReadDir("migrations"); return len(es) }
func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(763485291)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS portway; REVOKE ALL ON SCHEMA portway FROM PUBLIC; CREATE TABLE IF NOT EXISTS portway.schema_migrations(version integer PRIMARY KEY, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	es, _ := migrations.ReadDir("migrations")
	sort.Slice(es, func(i, j int) bool { return es[i].Name() < es[j].Name() })
	var current int
	if err = tx.QueryRow(ctx, `SELECT coalesce(max(version),0) FROM portway.schema_migrations`).Scan(&current); err != nil {
		return err
	}
	if current > len(es) {
		return errors.New("database schema is newer than this binary")
	}
	for _, e := range es {
		version, _ := strconv.Atoi(strings.Split(e.Name(), "_")[0])
		body, _ := migrations.ReadFile("migrations/" + e.Name())
		h := sha256.Sum256(body)
		sum := hex.EncodeToString(h[:])
		var existing string
		err = tx.QueryRow(ctx, `SELECT checksum FROM portway.schema_migrations WHERE version=$1`, version).Scan(&existing)
		if err == nil {
			if existing != sum {
				return fmt.Errorf("migration %d checksum mismatch", version)
			}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err = tx.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("migration %d failed", version)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO portway.schema_migrations(version,checksum) VALUES($1,$2)`, version, sum); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (s *Store) Ready(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT version,checksum FROM portway.schema_migrations ORDER BY version`)
	if err != nil {
		return err
	}
	defer rows.Close()
	es, _ := migrations.ReadDir("migrations")
	i := 0
	for rows.Next() {
		var version int
		var sum string
		if err := rows.Scan(&version, &sum); err != nil {
			return err
		}
		if i >= len(es) || version != i+1 {
			return errors.New("incompatible schema")
		}
		b, _ := migrations.ReadFile("migrations/" + es[i].Name())
		if secure.Hash(string(b)) != sum {
			return errors.New("migration checksum mismatch")
		}
		i++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if i != len(es) {
		return errors.New("migrations required")
	}
	return nil
}
func (s *Store) EnsureInstallation(ctx context.Context, name, setupHash string) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO portway.installations(id,name,setup_hash) VALUES($1,$2,$3) ON CONFLICT(id) DO NOTHING`, s.Installation, name, setupHash)
	return err
}
func (s *Store) Audit(ctx context.Context, tx pgx.Tx, actor, action, kind, id string) error {
	_, err := tx.Exec(ctx, `INSERT INTO portway.audit_entries(id,installation_id,actor_type,actor_id,action,object_type,object_id,outcome) VALUES($1,$2,'owner',$3,$4,$5,$6,'success')`, secure.ID(), s.Installation, null(actor), action, kind, null(id))
	return err
}
func null(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Mutate commits the effect, safe replay response and audit in one transaction.
// The installation lock serializes configuration commands without granting UPDATE on owner identity.
func (s *Store) Mutate(ctx context.Context, owner, device, operation, key, hash string, fn func(pgx.Tx) (int, any, error)) (int, json.RawMessage, error) {
	if len(key) < 8 || len(key) > 200 {
		return 0, nil, problem.Invalid("Idempotency-Key must contain 8–200 characters")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback(context.Background())
	var id string
	if err = tx.QueryRow(ctx, `SELECT o.id FROM portway.owners o JOIN portway.installations i ON i.id=o.installation_id WHERE o.installation_id=$1 AND o.id=$2 FOR UPDATE OF i`, s.Installation, owner).Scan(&id); err != nil {
		return 0, nil, err
	}
	if err = tx.QueryRow(ctx, `SELECT id FROM portway.device_sessions WHERE installation_id=$1 AND owner_id=$2 AND id=$3 AND revoked_at IS NULL FOR SHARE`, s.Installation, owner, device).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil, problem.New(401, "DEVICE_REVOKED", "Device revoked")
		}
		return 0, nil, err
	}
	kh := secure.Hash(key)
	_, err = tx.Exec(ctx, `DELETE FROM portway.request_keys WHERE installation_id=$1 AND owner_id=$2 AND operation=$3 AND key_hash=$4 AND expires_at<=now()`, s.Installation, owner, operation, kh)
	if err != nil {
		return 0, nil, err
	}
	var oldHash string
	var code int
	var body json.RawMessage
	err = tx.QueryRow(ctx, `SELECT request_hash,response_status,response_body_safe FROM portway.request_keys WHERE installation_id=$1 AND owner_id=$2 AND operation=$3 AND key_hash=$4`, s.Installation, owner, operation, kh).Scan(&oldHash, &code, &body)
	if err == nil {
		if oldHash != hash {
			return 0, nil, problem.New(409, "IDEMPOTENCY_CONFLICT", "Key was used with a different request")
		}
		return code, body, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, err
	}
	code, data, err := fn(tx)
	if err != nil {
		return 0, nil, err
	}
	body, err = json.Marshal(data)
	if err != nil {
		return 0, nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO portway.request_keys(installation_id,owner_id,operation,key_hash,request_hash,response_status,response_body_safe) VALUES($1,$2,$3,$4,$5,$6,$7)`, s.Installation, owner, operation, kh, hash, code, body)
	if err != nil {
		return 0, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, nil, err
	}
	return code, body, nil
}
