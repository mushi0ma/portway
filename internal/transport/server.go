package transport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/agents"
	"portway/internal/config"
	"portway/internal/identity"
	"portway/internal/problem"
	"portway/internal/secure"
	"portway/internal/storage"
)

type Server struct {
	Config     config.Config
	Store      *storage.Store
	Identity   *identity.Service
	Agents     *agents.Service
	Log        *slog.Logger
	Vault      *secure.Vault
	authMu     sync.Mutex
	authWindow time.Time
	authCount  int
}
type requestKey struct{}
type handler func(http.ResponseWriter, *http.Request, identity.Principal) (int, any, error)

func New(c config.Config, store *storage.Store, log *slog.Logger) *Server {
	v, _ := secure.NewVault(c.MasterKey)
	auth := identity.NewSupabase(c.AuthURL, c.AuthKey)
	auth.ConfirmationURL = c.PublicURL + "/auth/confirm"
	return &Server{Config: c, Store: store, Identity: &identity.Service{Store: store, Auth: auth, RequestHashKey: c.MasterKey}, Agents: &agents.Service{Store: store}, Log: log, Vault: v}
}
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { s.respond(w, r, 200, map[string]any{"alive": true}, nil) })
	m.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) { s.respond(w, r, 0, nil, problem.Unavailable()) })
	s.route(m, "GET /v1/client-config", false, s.clientConfig)
	s.route(m, "POST /v1/auth/register", false, s.register)
	s.route(m, "GET /auth/confirm", false, s.confirm)
	s.route(m, "POST /v1/auth/login", false, s.login)
	s.route(m, "POST /v1/auth/refresh", false, s.refresh)
	s.route(m, "POST /v1/auth/logout", false, s.logout)
	s.route(m, "GET /v1/me", true, s.me)
	s.route(m, "GET /v1/devices", true, s.devices)
	s.route(m, "DELETE /v1/devices/{id}", true, s.revoke)
	s.route(m, "GET /v1/roles", true, func(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
		return 200, map[string]any{"catalog_version": 1, "items": agents.Roles()}, nil
	})
	s.route(m, "GET /v1/agents", true, func(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
		v, e := s.Agents.List(r.Context())
		return 200, map[string]any{"items": v}, e
	})
	s.route(m, "POST /v1/agents", true, s.createAgent)
	s.route(m, "GET /v1/agents/{id}", true, func(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
		v, e := s.Agents.Get(r.Context(), r.PathValue("id"))
		w.Header().Set("ETag", strconv.Quote(strconv.Itoa(v.Version)))
		return 200, v, e
	})
	s.route(m, "GET /v1/agents/{id}/versions", true, func(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
		v, e := s.Agents.Versions(r.Context(), r.PathValue("id"))
		return 200, map[string]any{"items": v}, e
	})
	s.route(m, "POST /v1/agents/{id}/versions", true, s.newVersion)
	s.route(m, "GET /v1/agent-versions/{id}", true, func(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
		v, e := s.Agents.GetVersion(r.Context(), r.PathValue("id"))
		return 200, v, e
	})
	s.route(m, "GET /v1/providers", true, s.providers)
	s.route(m, "POST /v1/providers", true, s.createProvider)
	s.route(m, "PUT /v1/providers/{id}", true, s.updateProvider)
	s.route(m, "GET /v1/secret-refs", true, s.secrets)
	s.route(m, "POST /v1/secret-refs", true, s.createSecret)
	s.route(m, "GET /internal/v1/execution/authorize", true, s.executionAuthorize)
	s.route(m, "POST /v1/runs", true, s.executionProxy)
	s.route(m, "GET /v1/runs", true, s.executionProxy)
	s.route(m, "GET /v1/runs/{id}", true, s.executionProxy)
	s.route(m, "POST /v1/runs/{id}/stop", true, s.executionProxy)
	s.route(m, "GET /v1/chains/{id}", true, s.executionProxy)
	s.route(m, "POST /v1/chains/{id}/stop", true, s.executionProxy)
	s.route(m, "GET /v1/status", true, s.status)
	s.route(m, "GET /v1/audit", true, s.audit)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { s.respond(w, r, 0, nil, problem.NotFound()) })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		r = r.WithContext(context.WithValue(ctx, requestKey{}, secure.ID()))
		w.Header().Set("X-Request-ID", r.Context().Value(requestKey{}).(string))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		defer func() {
			if recover() != nil {
				s.Log.Error("request panic", "request_id", r.Context().Value(requestKey{}))
				s.respond(w, r, 0, nil, problem.New(500, "INTERNAL_ERROR", "Request failed"))
			}
		}()
		m.ServeHTTP(w, r)
	})
}
func (s *Server) route(m *http.ServeMux, pattern string, protected bool, h handler) {
	m.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var p identity.Principal
		var err error
		if !protected && (strings.Contains(pattern, "/auth/") || strings.Contains(pattern, "/auth/confirm")) {
			s.authMu.Lock()
			if time.Since(s.authWindow) >= time.Minute {
				s.authWindow = time.Now()
				s.authCount = 0
			}
			s.authCount++
			limited := s.authCount > 30
			s.authMu.Unlock()
			if limited {
				w.Header().Set("Retry-After", "60")
				s.respond(w, r, 0, nil, problem.New(429, "RATE_LIMITED", "Too many authentication attempts"))
				return
			}
		}
		if protected {
			p, err = s.Identity.Authenticate(r.Context(), bearer(r), r.Header.Get("X-Device-Session"))
			if err != nil {
				s.respond(w, r, 0, nil, err)
				return
			}
		}
		if id := r.PathValue("id"); id != "" && !secure.UUID(id) {
			s.respond(w, r, 0, nil, problem.Invalid("Invalid UUID"))
			return
		}
		code, data, err := h(w, r, p)
		if code != 0 || err != nil {
			s.respond(w, r, code, data, err)
		}
	})
}
func bearer(r *http.Request) string {
	a := r.Header.Get("Authorization")
	if !strings.HasPrefix(a, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(a, "Bearer ")
}
func (s *Server) respond(w http.ResponseWriter, r *http.Request, code int, data any, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	meta := map[string]any{"request_id": r.Context().Value(requestKey{}), "server_time": time.Now().UTC()}
	out := map[string]any{"data": data, "meta": meta}
	if err != nil {
		e := problem.Public(err)
		code = e.Status
		out = map[string]any{"error": e, "meta": meta}
		s.Log.Warn("request rejected", "code", e.Code, "request_id", r.Context().Value(requestKey{}))
	}
	if code == 0 {
		code = 200
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(out)
}
func (s *Server) mutate(r *http.Request, p identity.Principal, body any, fn func(pgx.Tx) (int, any, error)) (int, any, error) {
	b, _ := json.Marshal(body)
	code, out, err := s.Store.Mutate(r.Context(), p.OwnerID, p.DeviceID, r.Method+" "+r.URL.Path, r.Header.Get("Idempotency-Key"), secure.MAC(s.Config.MasterKey, b), fn)
	return code, out, err
}
func expected(r *http.Request) (int, error) {
	v := r.Header.Get("If-Match")
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = v[1 : len(v)-1]
	}
	i, err := strconv.Atoi(v)
	if err != nil || i < 1 {
		return 0, problem.Invalid("If-Match must contain the current integer version")
	}
	return i, nil
}
