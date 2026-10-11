package transport

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"portway/internal/identity"
	"portway/internal/problem"
	"portway/internal/secure"
)

func (s *Server) status(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	db := "ready"
	if s.Store.Ready(r.Context()) != nil {
		db = "unavailable"
	}
	pending, oldest, eventErr := s.Store.EventStatus(r.Context())
	eventState := "caught_up"
	if pending > 0 {
		eventState = "pending"
	}
	configurationReady := db == "ready" && eventErr == nil && pending < 10000 && (oldest == nil || time.Since(*oldest) < 24*time.Hour)
	if !configurationReady {
		eventState = "unavailable"
	}
	return 200, map[string]any{
		"stage": "foundation", "ready": false,
		"operations": map[string]any{"configuration": configurationReady, "run": false},
		"dependencies": map[string]any{
			"db":        map[string]any{"state": db, "checked_at": time.Now().UTC()},
			"events":    map[string]any{"state": eventState, "pending": pending, "oldest_at": oldest, "broker_health": "check worker readiness"},
			"auth":      map[string]any{"state": "unknown", "verification": "JWT and device guard passed; upstream health not probed"},
			"execution": map[string]any{"configured": s.Config.ExecutionURL != "", "run_admission": false, "error_code": "EXECUTION_NOT_READY"},
			"runtime":   map[string]any{"state": "unavailable", "error_code": "NOT_IMPLEMENTED"},
			"provider":  map[string]any{"state": "unknown", "error_code": "CAPABILITY_CHECK_NOT_IMPLEMENTED"},
			"push":      map[string]any{"state": "unknown", "error_code": "NOT_IMPLEMENTED"},
		},
	}, nil
}
func (s *Server) audit(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	q := r.URL.Query()
	for k := range q {
		if k != "cursor" {
			return 0, nil, problem.Invalid("Only cursor is supported before run context is implemented")
		}
	}
	var cursorTime *time.Time
	var cursorID *string
	if cur := q.Get("cursor"); cur != "" {
		b, err := base64.RawURLEncoding.DecodeString(cur)
		parts := strings.SplitN(string(b), "|", 2)
		if err != nil || len(parts) != 2 || !secure.UUID(parts[1]) {
			return 0, nil, problem.New(400, "INVALID_CURSOR", "Invalid audit cursor")
		}
		t, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return 0, nil, problem.New(400, "INVALID_CURSOR", "Invalid audit cursor")
		}
		cursorTime = &t
		cursorID = &parts[1]
	}
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT id,actor_type,actor_id,action,object_type,object_id,outcome,reason,server_time FROM portway.audit_entries WHERE installation_id=$1 AND ($2::timestamptz IS NULL OR (server_time,id)<($2,$3::uuid)) ORDER BY server_time DESC,id DESC LIMIT 101`, s.Store.Installation, cursorTime, cursorID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	next := ""
	var last string
	for rows.Next() {
		var id, actorType, action, objectType, outcome, reason string
		var actorID, objectID *string
		var timestamp time.Time
		if err = rows.Scan(&id, &actorType, &actorID, &action, &objectType, &objectID, &outcome, &reason, &timestamp); err != nil {
			return 0, nil, err
		}
		if len(items) == 100 {
			next = last
			break
		}
		last = base64.RawURLEncoding.EncodeToString([]byte(timestamp.UTC().Format(time.RFC3339Nano) + "|" + id))
		items = append(items, map[string]any{"id": id, "actor_type": actorType, "actor_id": actorID, "action": action, "object_type": objectType, "object_id": objectID, "outcome": outcome, "reason": reason, "server_time": timestamp, "safe_metadata": map[string]any{}})
	}
	return 200, map[string]any{"items": items, "next_cursor": next}, rows.Err()
}
