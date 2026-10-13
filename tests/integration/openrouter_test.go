package integration

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"portway/internal/config"
	"portway/internal/openrouter"
	"portway/internal/secure"
	"portway/internal/storage"
	"portway/internal/transport"
)

func exerciseOpenRouterProvision(t *testing.T, cfg config.Config, app *storage.Store, token, proof string) {
	t.Helper()
	cfg.FreeModelsOnly = true
	cfg.OpenRouterKey = "openrouter-environment-secret-marker"
	cfg.AllowedEndpoints = append(append([]string{}, cfg.AllowedEndpoints...), openrouter.Endpoint)
	newServer := func(key string) *httptest.Server {
		local := cfg
		local.OpenRouterKey = key
		s := transport.New(local, app, slog.New(slog.NewTextHandler(io.Discard, nil)))
		s.FreeModels = func(context.Context) ([]openrouter.Model, error) {
			return []openrouter.Model{{ID: "vendor/model:free", Name: "fixture", ContextLength: 32768, SupportedParameters: []string{"tools", "tool_choice"}}}, nil
		}
		return httptest.NewServer(s.Handler())
	}
	api := newServer(cfg.OpenRouterKey)
	defer api.Close()
	call := func(method, path, body, key string) (int, response, []byte) {
		return request(t, api.Client(), api.URL, method, path, body, token, proof, key, nil)
	}
	t.Run("openrouter-env-provision", func(t *testing.T) {
		code, _, raw := call("GET", "/v1/models", "", "")
		want(t, code, 200, raw)
		for _, body := range []string{`{"model_id":"vendor/paid"}`, `{"model_id":"openrouter/free"}`, `{"model_id":"vendor/not-listed:free"}`} {
			code, _, raw = call("POST", "/v1/providers/openrouter", body, "router-denied")
			want(t, code, 403, raw)
		}
		code, first, raw := call("POST", "/v1/providers/openrouter", `{"model_id":"vendor/model:free"}`, "router-provision")
		want(t, code, 201, raw)
		var provider struct {
			ID string `json:"id"`
		}
		json.Unmarshal(first.Data, &provider)
		for i := 0; i < 2; i++ {
			code, replay, raw := call("POST", "/v1/providers/openrouter", `{"model_id":"vendor/model:free"}`, "router-provision")
			want(t, code, 201, raw)
			var out struct {
				ID string `json:"id"`
			}
			json.Unmarshal(replay.Data, &out)
			if out.ID != provider.ID {
				t.Fatal("provision replay created another provider")
			}
		}
		var ciphertext, nonce []byte
		var secret string
		if err := app.Pool.QueryRow(context.Background(), `SELECT s.id,s.ciphertext,s.nonce FROM portway.secrets s JOIN portway.providers p ON p.installation_id=s.installation_id AND p.secret_ref=s.id WHERE p.id=$1`, provider.ID).Scan(&secret, &ciphertext, &nonce); err != nil {
			t.Fatal(err)
		}
		vault, _ := secure.NewVault(cfg.MasterKey)
		plain, err := vault.Open(ciphertext, nonce, app.Installation+":"+secret+":1")
		if err != nil || string(plain) != cfg.OpenRouterKey {
			t.Fatal("environment key not encrypted with secret binding")
		}
		clear(plain)
		if strings.Contains(string(ciphertext), cfg.OpenRouterKey) || strings.Contains(string(raw), cfg.OpenRouterKey) {
			t.Fatal("key exposed")
		}
		code, _, raw = call("POST", "/v1/providers", `{"label":"paid","endpoint":"https://api.groq.com/openai/v1","model_id":"paid"}`, "router-paid-provider")
		want(t, code, 403, raw)
		missing := newServer("")
		defer missing.Close()
		code, _, raw = request(t, missing.Client(), missing.URL, "POST", "/v1/providers/openrouter", `{"model_id":"vendor/model:free"}`, token, proof, "router-missing", nil)
		want(t, code, 503, raw)
	})
}
