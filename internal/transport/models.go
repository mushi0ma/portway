package transport

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/identity"
	"portway/internal/openrouter"
	"portway/internal/problem"
	"portway/internal/secure"
)

func (s *Server) freeModels(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if r.URL.RawQuery != "" {
		return 0, nil, problem.Invalid("Query parameters are not supported")
	}
	models, err := s.FreeModels(r.Context())
	return 200, map[string]any{"items": models, "endpoint": openrouter.Endpoint, "policy": "free_only", "checked_at": time.Now().UTC()}, err
}

// Environment provisioning is explicit and owner-authenticated. The API key is
// encrypted into a new secret reference; neither response nor replay stores it.
// A changed environment key does not silently rotate existing provider secrets.
func (s *Server) connectOpenRouter(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in struct {
		ModelID string `json:"model_id"`
	}
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if !openrouter.FreeID(in.ModelID) {
		return 0, nil, problem.New(403, "FREE_MODEL_REQUIRED", "Choose an explicit :free model")
	}
	if s.Config.OpenRouterKey == "" {
		return 0, nil, problem.New(503, "OPENROUTER_KEY_REQUIRED", "Set OPENROUTER_API_KEY in the control API environment")
	}
	// Metadata-only GET outside the transaction; never invokes generation.
	models, err := s.FreeModels(r.Context())
	if err != nil {
		return 0, nil, err
	}
	found := false
	for _, m := range models {
		found = found || m.ID == in.ModelID
	}
	if !found {
		return 0, nil, problem.New(403, "FREE_MODEL_REQUIRED", "Model is unavailable or not zero-priced")
	}
	input := providerInput{Label: "OpenRouter free", Endpoint: openrouter.Endpoint, ModelID: in.ModelID}
	if err := s.validateProvider(&input); err != nil {
		return 0, nil, err
	}
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		secretID, providerID := secure.ID(), secure.ID()
		ciphertext, nonce := s.Vault.Seal(s.Config.OpenRouterKey, s.Store.Installation+":"+secretID+":1")
		_, err := tx.Exec(r.Context(), `INSERT INTO portway.secrets(id,installation_id,alias,ciphertext,nonce) VALUES($1,$2,$3,$4,$5)`, secretID, s.Store.Installation, "openrouter_"+secretID, ciphertext, nonce)
		if err != nil {
			return 0, nil, err
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO portway.providers(id,installation_id,label,endpoint,model_id,secret_ref) VALUES($1,$2,$3,$4,$5,$6)`, providerID, s.Store.Installation, input.Label, input.Endpoint, input.ModelID, secretID)
		if err != nil {
			return 0, nil, err
		}
		for _, entry := range []struct{ action, kind, id string }{{"secret.created", "secret", secretID}, {"provider.created", "provider", providerID}} {
			if err = s.Store.Audit(r.Context(), tx, p.OwnerID, entry.action, entry.kind, entry.id); err != nil {
				return 0, nil, err
			}
		}
		return 201, map[string]any{"id": providerID, "config_version": 1, "capabilities": map[string]any{}, "checked_at": nil}, nil
	})
}
