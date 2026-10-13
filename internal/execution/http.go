package execution

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"portway/internal/httpjson"
	"portway/internal/problem"
	"portway/internal/secure"
)

type Authority struct {
	OwnerID        string `json:"owner_id"`
	DeviceID       string `json:"device_id"`
	InstallationID string `json:"installation_id"`
}
type Authorizer func(context.Context, string, string) (Authority, error)
type HTTPServer struct {
	Store      Store
	ServiceKey string
	Authorize  Authorizer
	Ready      func(context.Context) bool
}

func (h HTTPServer) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	m.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if h.Ready == nil || !h.Ready(r.Context()) {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	})
	m.HandleFunc("/internal/v1/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		write := func(code int, data any, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			if err != nil {
				p := problem.Public(err)
				w.WriteHeader(p.Status)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": p})
				return
			}
			w.WriteHeader(code)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
		}
		if len(h.ServiceKey) < 32 || !secure.Equal(h.ServiceKey, r.Header.Get("X-Portway-Service-Key")) {
			write(0, nil, problem.New(401, "SERVICE_AUTH_REQUIRED", "Service authentication required"))
			return
		}
		if h.Authorize == nil {
			write(0, nil, problem.Unavailable())
			return
		}
		a, err := h.Authorize(ctx, r.Header.Get("Authorization"), r.Header.Get("X-Device-Session"))
		if err != nil {
			write(0, nil, err)
			return
		}
		if a.InstallationID != h.Store.Installation || !secure.UUID(a.OwnerID) || !secure.UUID(a.DeviceID) {
			write(0, nil, problem.New(403, "OWNER_REQUIRED", "Installation owner required"))
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/internal/v1/")
		parts := strings.Split(path, "/")
		if parts[0] == "approvals" {
			if r.URL.RawQuery != "" {
				write(0, nil, problem.Invalid("Query parameters are not supported"))
				return
			}
			if len(parts) == 1 && r.Method == "GET" {
				items, err := h.Store.ListApprovals(ctx, a.OwnerID)
				write(200, map[string]any{"items": items, "server_time": time.Now().UTC()}, err)
				return
			}
			if len(parts) < 2 || !secure.UUID(parts[1]) {
				write(0, nil, problem.NotFound())
				return
			}
			if len(parts) == 2 && r.Method == "GET" {
				out, err := h.Store.GetApproval(ctx, a.OwnerID, parts[1])
				write(200, out, err)
				return
			}
			if len(parts) == 3 && parts[2] == "decision" && r.Method == "POST" {
				var d Decision
				if err := httpjson.Decode(w, r, &d); err != nil {
					write(0, nil, err)
					return
				}
				out, err := h.Store.DecideApproval(ctx, a.OwnerID, a.DeviceID, parts[1], r.Header.Get("Idempotency-Key"), d)
				write(200, out, err)
				return
			}
			write(0, nil, problem.NotFound())
			return
		}
		if path == "runs" {
			switch r.Method {
			case "GET":
				out, err := h.Store.List(ctx, a.OwnerID)
				write(200, map[string]any{"items": out}, err)
			case "POST":
				write(0, nil, problem.New(503, "EXECUTION_NOT_READY", "Runtime and capability admission are not implemented"))
			default:
				write(0, nil, problem.NotFound())
			}
			return
		}
		if len(parts) < 2 || len(parts) > 3 || (parts[0] != "runs" && parts[0] != "chains") || !secure.UUID(parts[1]) {
			write(0, nil, problem.NotFound())
			return
		}
		isRun := parts[0] == "runs"
		id := parts[1]
		if len(parts) == 3 && parts[2] == "result" && isRun && r.Method == "GET" {
			if r.URL.RawQuery != "" {
				write(0, nil, problem.Invalid("Query parameters are not supported"))
				return
			}
			c, err := h.Store.Lookup(ctx, a.OwnerID, id, true)
			if err != nil {
				write(0, nil, err)
				return
			}
			run, _ := c.Find(id)
			if run.Result == nil {
				write(0, nil, problem.New(404, "RESULT_NOT_FOUND", "Run has no completed result"))
				return
			}
			write(200, run.Result, nil)
			return
		}
		if len(parts) == 3 && parts[2] == "stop" && r.Method == "POST" {
			c, err := h.Store.Stop(ctx, a.OwnerID, id, r.Header.Get("Idempotency-Key"), isRun)
			if err != nil {
				write(0, nil, err)
				return
			}
			code := 202
			if isRun {
				run, _ := c.Find(id)
				if terminal(run.State) {
					code = 200
				}
				write(code, run.View(), nil)
			} else {
				if terminal(c.State) {
					code = 200
				}
				write(code, c, nil)
			}
			return
		}
		if len(parts) == 3 && parts[2] == "events" && isRun && r.Method == "GET" {
			query, err := url.ParseQuery(r.URL.RawQuery)
			if err != nil || len(query) != 1 || len(query["after"]) != 1 {
				write(0, nil, problem.New(400, "INVALID_CURSOR", "Only one after parameter is accepted"))
				return
			}
			after, err := strconv.ParseInt(query.Get("after"), 10, 64)
			if err != nil || strconv.FormatInt(after, 10) != query.Get("after") {
				write(0, nil, problem.New(400, "INVALID_CURSOR", "Invalid event sequence"))
				return
			}
			out, err := h.Store.HistoryPage(ctx, a.OwnerID, id, after)
			write(200, out, err)
			return
		}
		if len(parts) != 2 || r.Method != "GET" {
			write(0, nil, problem.NotFound())
			return
		}
		c, err := h.Store.Lookup(ctx, a.OwnerID, id, isRun)
		if err != nil {
			write(0, nil, err)
			return
		}
		if isRun {
			run, _ := c.Find(id)
			write(200, run.View(), nil)
		} else {
			write(200, c, nil)
		}
	})
	return m
}

// ControlAuthorizer checks live owner/device state on every operation. A service
// key alone is insufficient. No cached authorization or Kafka revocation lag.
func ControlAuthorizer(base, key string, client *http.Client) Authorizer {
	return func(ctx context.Context, bearer, proof string) (Authority, error) {
		if bearer == "" || proof == "" {
			return Authority{}, problem.New(401, "AUTH_REQUIRED", "Owner and device credentials required")
		}
		req, err := http.NewRequestWithContext(ctx, "GET", base+"/internal/v1/execution/authorize", nil)
		if err != nil {
			return Authority{}, problem.Unavailable()
		}
		req.Header.Set("Authorization", bearer)
		req.Header.Set("X-Device-Session", proof)
		req.Header.Set("X-Portway-Service-Key", key)
		response, err := client.Do(req)
		if err != nil {
			return Authority{}, problem.Unavailable()
		}
		defer response.Body.Close()
		if response.StatusCode == 401 || response.StatusCode == 403 {
			return Authority{}, problem.New(response.StatusCode, "AUTH_REQUIRED", "Owner or device credentials rejected")
		}
		if response.StatusCode != 200 {
			return Authority{}, problem.Unavailable()
		}
		var out struct {
			Data Authority `json:"data"`
		}
		if err = json.NewDecoder(io.LimitReader(response.Body, 8192)).Decode(&out); err != nil {
			return Authority{}, errors.New("invalid authority response")
		}
		return out.Data, nil
	}
}
