package agents

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/problem"
	"portway/internal/secure"
	"portway/internal/storage"
)

type Service struct{ Store *storage.Store }
type Agent struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	CurrentVersionID string    `json:"current_version_id"`
	Version          int       `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
}
type Version struct {
	ID              string    `json:"id"`
	AgentID         string    `json:"agent_id"`
	Ordinal         int       `json:"ordinal"`
	Config          Config    `json:"config"`
	ConfigHash      string    `json:"config_hash"`
	CreatedAt       time.Time `json:"created_at"`
	Unresolved      []string  `json:"unresolved"`
	CapabilityNeeds []string  `json:"capability_needs"`
}
type Created struct {
	Agent   Agent   `json:"agent"`
	Version Version `json:"agent_version"`
}

func (s *Service) Create(ctx context.Context, tx pgx.Tx, owner, name string, c Config) (Created, error) {
	a := Agent{ID: secure.ID(), Name: name, CurrentVersionID: secure.ID(), Version: 1}
	v := Version{ID: a.CurrentVersionID, AgentID: a.ID, Ordinal: 1, Config: c, ConfigHash: Hash(c)}
	err := tx.QueryRow(ctx, `INSERT INTO portway.agents(id,installation_id,name,current_version_id) VALUES($1,$2,$3,$4) RETURNING created_at`, a.ID, s.Store.Installation, name, v.ID).Scan(&a.CreatedAt)
	if err != nil {
		return Created{}, err
	}
	if err = s.insertVersion(ctx, tx, &v); err != nil {
		return Created{}, err
	}
	if err = s.Store.Audit(ctx, tx, owner, "agent.created", "agent", a.ID); err != nil {
		return Created{}, err
	}
	return Created{a, v}, nil
}
func (s *Service) NewVersion(ctx context.Context, tx pgx.Tx, owner, id, base string, expected int, c Config) (Created, error) {
	var a Agent
	err := tx.QueryRow(ctx, `SELECT id,name,current_version_id,version,created_at FROM portway.agents WHERE installation_id=$1 AND id=$2 FOR UPDATE`, s.Store.Installation, id).Scan(&a.ID, &a.Name, &a.CurrentVersionID, &a.Version, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Created{}, problem.NotFound()
	}
	if err != nil {
		return Created{}, err
	}
	if a.Version != expected || a.CurrentVersionID != base {
		return Created{}, problem.New(409, "VERSION_CONFLICT", "Current agent version changed")
	}
	v := Version{ID: secure.ID(), AgentID: id, Ordinal: a.Version + 1, Config: c, ConfigHash: Hash(c)}
	if err = s.insertVersion(ctx, tx, &v); err != nil {
		return Created{}, err
	}
	a.Version++
	a.CurrentVersionID = v.ID
	if _, err = tx.Exec(ctx, `UPDATE portway.agents SET current_version_id=$3,version=$4 WHERE installation_id=$1 AND id=$2`, s.Store.Installation, id, v.ID, a.Version); err != nil {
		return Created{}, err
	}
	if err = s.Store.Audit(ctx, tx, owner, "agent.version_created", "agent_version", v.ID); err != nil {
		return Created{}, err
	}
	return Created{a, v}, nil
}
func (s *Service) insertVersion(ctx context.Context, tx pgx.Tx, v *Version) error {
	if v.Config.ProviderRef != "" {
		var ok bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM portway.providers WHERE installation_id=$1 AND id=$2)`, s.Store.Installation, v.Config.ProviderRef).Scan(&ok)
		if err != nil {
			return err
		}
		if !ok {
			return problem.Invalid("Unknown provider_ref")
		}
	}
	for _, m := range v.Config.MCP {
		if m.SecretRef == "" {
			continue
		}
		var ok bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM portway.secrets WHERE installation_id=$1 AND id=$2 AND state='active')`, s.Store.Installation, m.SecretRef).Scan(&ok)
		if err != nil {
			return err
		}
		if !ok {
			return problem.Invalid("Unknown MCP secret_ref")
		}
	}
	b, _ := json.Marshal(v.Config)
	err := tx.QueryRow(ctx, `INSERT INTO portway.agent_versions(id,installation_id,agent_id,ordinal,config_json,config_hash) VALUES($1,$2,$3,$4,$5,$6) RETURNING created_at`, v.ID, s.Store.Installation, v.AgentID, v.Ordinal, json.RawMessage(b), v.ConfigHash).Scan(&v.CreatedAt)
	dependencies(v)
	return err
}
func dependencies(v *Version) {
	v.Unresolved = []string{"runtime:not_implemented"}
	if v.Config.ProviderRef == "" {
		v.Unresolved = append(v.Unresolved, "provider_ref:required")
	} else {
		v.Unresolved = append(v.Unresolved, "provider_capabilities:unchecked")
	}
	for _, m := range v.Config.MCP {
		v.Unresolved = append(v.Unresolved, "mcp:"+m.Alias+":unbound")
	}
	v.CapabilityNeeds = []string{"streaming", "tool_calling", "reported_or_conservative_usage"}
}
func (s *Service) List(ctx context.Context) ([]Agent, error) {
	rows, err := s.Store.Pool.Query(ctx, `SELECT id,name,current_version_id,version,created_at FROM portway.agents WHERE installation_id=$1 ORDER BY created_at,id`, s.Store.Installation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Agent{}
	for rows.Next() {
		var a Agent
		if err = rows.Scan(&a.ID, &a.Name, &a.CurrentVersionID, &a.Version, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *Service) Get(ctx context.Context, id string) (Agent, error) {
	var a Agent
	err := s.Store.Pool.QueryRow(ctx, `SELECT id,name,current_version_id,version,created_at FROM portway.agents WHERE installation_id=$1 AND id=$2`, s.Store.Installation, id).Scan(&a.ID, &a.Name, &a.CurrentVersionID, &a.Version, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = problem.NotFound()
	}
	return a, err
}
func (s *Service) GetVersion(ctx context.Context, id string) (Version, error) {
	var v Version
	err := s.Store.Pool.QueryRow(ctx, `SELECT id,agent_id,ordinal,config_json,config_hash,created_at FROM portway.agent_versions WHERE installation_id=$1 AND id=$2`, s.Store.Installation, id).Scan(&v.ID, &v.AgentID, &v.Ordinal, &v.Config, &v.ConfigHash, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = problem.NotFound()
	}
	dependencies(&v)
	return v, err
}
func (s *Service) Versions(ctx context.Context, id string) ([]map[string]any, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.Store.Pool.Query(ctx, `SELECT id,ordinal,config_hash,created_at FROM portway.agent_versions WHERE installation_id=$1 AND agent_id=$2 ORDER BY ordinal`, s.Store.Installation, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var vid, hash string
		var ordinal int
		var created time.Time
		if err = rows.Scan(&vid, &ordinal, &hash, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": vid, "ordinal": ordinal, "config_hash": hash, "created_at": created})
	}
	return out, rows.Err()
}
