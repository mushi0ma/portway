package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"portway/internal/execution"
	"portway/internal/identity"
	"portway/internal/problem"
	"portway/internal/secure"
)

func (s *Server) executionAuthorize(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if len(s.Config.ExecutionKey) < 32 || !secure.Equal(s.Config.ExecutionKey, r.Header.Get("X-Portway-Service-Key")) {
		return 0, nil, problem.New(401, "SERVICE_AUTH_REQUIRED", "Service authentication required")
	}
	return 200, execution.Authority{OwnerID: p.OwnerID, DeviceID: p.DeviceID, InstallationID: s.Config.InstallationID}, nil
}
func (s *Server) executionProxy(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if s.Config.ExecutionURL == "" || len(s.Config.ExecutionKey) < 32 {
		return 0, nil, problem.Unavailable()
	}
	var body []byte
	if r.Method == "POST" {
		if strings.HasSuffix(r.URL.Path, "/stop") {
			var empty struct{}
			if err := Decode(w, r, &empty); err != nil {
				return 0, nil, err
			}
			body = []byte("{}")
		} else if strings.HasSuffix(r.URL.Path, "/decision") {
			var decision execution.Decision
			if err := Decode(w, r, &decision); err != nil {
				return 0, nil, err
			}
			if err := decision.Validate(); err != nil {
				return 0, nil, err
			}
			body, _ = json.Marshal(decision)
		} else {
			// Validate the public DTO before returning the closed admission response.
			var in struct {
				AgentVersionID string          `json:"agent_version_id"`
				ProviderID     string          `json:"provider_id"`
				Task           string          `json:"task"`
				RepositoryRef  string          `json:"repository_ref"`
				Revision       string          `json:"revision"`
				Limits         json.RawMessage `json:"limits"`
			}
			if err := Decode(w, r, &in); err != nil {
				return 0, nil, err
			}
			if !secure.UUID(in.AgentVersionID) || !secure.UUID(in.ProviderID) || strings.TrimSpace(in.Task) == "" {
				return 0, nil, problem.Invalid("Invalid run request")
			}
			body, _ = json.Marshal(in)
		}
		key := r.Header.Get("Idempotency-Key")
		if len(key) < 8 || len(key) > 200 {
			return 0, nil, problem.Invalid("Idempotency-Key must contain 8–200 characters")
		}
	}
	if r.URL.RawQuery != "" {
		return 0, nil, problem.Invalid("Query parameters are not supported")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, r.Method, s.Config.ExecutionURL+"/internal"+r.URL.Path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, problem.Unavailable()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", r.Header.Get("Authorization"))
	req.Header.Set("X-Device-Session", r.Header.Get("X-Device-Session"))
	req.Header.Set("X-Portway-Service-Key", s.Config.ExecutionKey)
	req.Header.Set("Idempotency-Key", r.Header.Get("Idempotency-Key"))
	client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return 0, nil, problem.Unavailable()
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return 0, nil, problem.Unavailable()
	}
	var out struct {
		Data  json.RawMessage `json:"data"`
		Error *problem.Error  `json:"error"`
	}
	if json.Unmarshal(raw, &out) != nil {
		return 0, nil, problem.Unavailable()
	}
	if response.StatusCode >= 400 {
		if out.Error == nil {
			return 0, nil, problem.Unavailable()
		}
		out.Error.Status = response.StatusCode
		return 0, nil, out.Error
	}
	if response.StatusCode != 200 && response.StatusCode != 201 && response.StatusCode != 202 {
		return 0, nil, problem.Unavailable()
	}
	return response.StatusCode, out.Data, nil
}
