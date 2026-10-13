package transport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"portway/internal/agents"
	"portway/internal/identity"
	"portway/internal/modelgateway"
	"portway/internal/openrouter"
	"portway/internal/problem"
	"portway/internal/providercheck"
	"portway/internal/secure"
)

type providerInput struct {
	Label     string  `json:"label"`
	Endpoint  string  `json:"endpoint"`
	ModelID   string  `json:"model_id"`
	SecretRef *string `json:"secret_ref"`
}

func (s *Server) checkProvider(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in struct {
		Profile string `json:"profile"`
	}
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if in.Profile != modelgateway.ProbeProfile {
		return 0, nil, problem.Invalid("Unsupported capability check profile")
	}
	version, err := expected(r)
	if err != nil {
		return 0, nil, err
	}
	svc := providercheck.Service{Store: s.Store, Vault: s.Vault, AllowedEndpoints: s.Config.AllowedEndpoints, RequestHashKey: s.Config.MasterKey, FreeModelsOnly: s.Config.FreeModelsOnly}
	return svc.Check(r.Context(), p.OwnerID, p.DeviceID, r.PathValue("id"), version, r.Header.Get("Idempotency-Key"))
}

func (s *Server) validateProvider(in *providerInput) error {
	in.Label = strings.TrimSpace(in.Label)
	in.Endpoint = strings.TrimRight(in.Endpoint, "/")
	if s.Config.FreeModelsOnly && (in.Endpoint != openrouter.Endpoint || !openrouter.FreeID(in.ModelID)) {
		return problem.New(403, "FREE_MODEL_REQUIRED", "Only explicit OpenRouter :free models are enabled")
	}
	if in.Label == "" || len(in.Label) > 120 || strings.TrimSpace(in.ModelID) == "" || len(in.ModelID) > 256 || in.SecretRef != nil && !secure.UUID(*in.SecretRef) {
		return problem.Invalid("Invalid provider configuration")
	}
	for _, ep := range s.Config.AllowedEndpoints {
		if ep == in.Endpoint {
			return nil
		}
	}
	return problem.New(403, "POLICY_DENIED", "Model endpoint is not allowlisted")
}
func (s *Server) providers(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT jsonb_build_object('id',id,'label',label,'endpoint',endpoint,'model_id',model_id,'secret_ref',secret_ref,'config_version',config_version,'capabilities',capabilities,'checked_at',checked_at),clock_timestamp() FROM portway.providers WHERE installation_id=$1 ORDER BY id`, s.Store.Installation)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var v map[string]any
		var now time.Time
		if err = rows.Scan(&v, &now); err != nil {
			return 0, nil, err
		}
		var report providercheck.Report
		raw, _ := json.Marshal(v["capabilities"])
		_ = json.Unmarshal(raw, &report)
		v["protocol_valid"] = report.ProtocolValid(v["id"].(string), v["endpoint"].(string), v["model_id"].(string), int(v["config_version"].(float64)), s.Config.AllowedEndpoints, now)
		items = append(items, v)
	}
	return 200, map[string]any{"items": items}, rows.Err()
}
func (s *Server) createProvider(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in providerInput
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if err := s.validateProvider(&in); err != nil {
		return 0, nil, err
	}
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		id := secure.ID()
		_, err := tx.Exec(r.Context(), `INSERT INTO portway.providers(id,installation_id,label,endpoint,model_id,secret_ref) VALUES($1,$2,$3,$4,$5,$6)`, id, s.Store.Installation, in.Label, in.Endpoint, in.ModelID, in.SecretRef)
		if err != nil {
			return 0, nil, referenceError(err)
		}
		if err = s.Store.Audit(r.Context(), tx, p.OwnerID, "provider.created", "provider", id); err != nil {
			return 0, nil, err
		}
		return 201, map[string]any{"id": id, "config_version": 1, "capabilities": map[string]any{}, "checked_at": nil}, nil
	})
}
func (s *Server) updateProvider(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in providerInput
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if err := s.validateProvider(&in); err != nil {
		return 0, nil, err
	}
	ver, err := expected(r)
	if err != nil {
		return 0, nil, err
	}
	// PUT is naturally idempotent for its expected version; the request key also preserves a lost response.
	return s.mutate(r, p, struct {
		Body    any `json:"body"`
		Version int `json:"version"`
	}{in, ver}, func(tx pgx.Tx) (int, any, error) {
		var current int
		id := r.PathValue("id")
		err := tx.QueryRow(r.Context(), `SELECT config_version FROM portway.providers WHERE installation_id=$1 AND id=$2 FOR UPDATE`, s.Store.Installation, id).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil, problem.NotFound()
		}
		if err != nil {
			return 0, nil, err
		}
		if ver != current {
			return 0, nil, problem.New(409, "VERSION_CONFLICT", "Provider configuration changed")
		}
		_, err = tx.Exec(r.Context(), `UPDATE portway.providers SET label=$3,endpoint=$4,model_id=$5,secret_ref=$6,config_version=config_version+1,capabilities='{}',checked_at=NULL WHERE installation_id=$1 AND id=$2`, s.Store.Installation, id, in.Label, in.Endpoint, in.ModelID, in.SecretRef)
		if err != nil {
			return 0, nil, referenceError(err)
		}
		if err = s.Store.Audit(r.Context(), tx, p.OwnerID, "provider.updated", "provider", id); err != nil {
			return 0, nil, err
		}
		return 200, map[string]any{"id": id, "config_version": ver + 1, "capabilities": map[string]any{}, "checked_at": nil}, nil
	})
}
func referenceError(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		if p.Code == "23503" {
			return problem.Invalid("Unknown secret reference")
		}
		if p.Code == "23505" {
			return problem.New(409, "VERSION_CONFLICT", "Alias already exists")
		}
	}
	return err
}
func (s *Server) secrets(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT id,alias,state FROM portway.secrets WHERE installation_id=$1 ORDER BY alias,id`, s.Store.Installation)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	items := []map[string]string{}
	for rows.Next() {
		var id, alias, state string
		if err = rows.Scan(&id, &alias, &state); err != nil {
			return 0, nil, err
		}
		items = append(items, map[string]string{"id": id, "alias": alias, "state": state})
	}
	return 200, map[string]any{"items": items}, rows.Err()
}
func (s *Server) createSecret(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in struct {
		Alias string `json:"alias"`
		Value string `json:"value"`
	}
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if !agents.Alias(in.Alias) || len(in.Value) < 1 || len(in.Value) > 8192 {
		return 0, nil, problem.Invalid("Invalid secret alias or size")
	}
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		id := secure.ID()
		ct, nonce := s.Vault.Seal(in.Value, s.Store.Installation+":"+id+":1")
		_, err := tx.Exec(r.Context(), `INSERT INTO portway.secrets(id,installation_id,alias,ciphertext,nonce) VALUES($1,$2,$3,$4,$5)`, id, s.Store.Installation, in.Alias, ct, nonce)
		if err != nil {
			return 0, nil, referenceError(err)
		}
		if err = s.Store.Audit(r.Context(), tx, p.OwnerID, "secret.created", "secret", id); err != nil {
			return 0, nil, err
		}
		return 201, map[string]string{"id": id, "alias": in.Alias, "state": "active"}, nil
	})
}
