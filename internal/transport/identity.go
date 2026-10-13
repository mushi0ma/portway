package transport

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/identity"
	"portway/internal/problem"
)

func (s *Server) clientConfig(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var status string
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT status FROM portway.installations WHERE id=$1`, s.Store.Installation).Scan(&status)
	return 200, map[string]any{"installation_id": s.Store.Installation, "name": s.Config.InstallationName, "api_version": "v1", "setup_required": status != "claimed"}, err
}
func (s *Server) register(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in identity.RegisterInput
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	v, err := s.Identity.Register(r.Context(), in, r.Header.Get("X-Setup-Proof"), r.Header.Get("Idempotency-Key"))
	return 202, v, err
}
func (s *Server) confirm(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	q := r.URL.Query()
	if r.URL.RawQuery == "" {
		// Standard Supabase email links have already verified the email. The
		// fragment is discarded locally, never used as a Portway credential.
		script := "history.replaceState(null, '', location.pathname);"
		hash := sha256.Sum256([]byte(script))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'sha256-"+base64.StdEncoding.EncodeToString(hash[:])+"'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		fmt.Fprintf(w, `<!doctype html><html lang="ru"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Portway — подтверждение почты</title><script>%s</script><h1>Вернитесь в Portway</h1><p>После подтверждения почты войдите в приложение с email и паролем. Сервер проверит подтверждение при входе.</p><p><a href="portway://auth/confirmed">Открыть Portway</a></p></html>`, script)
		return 0, nil, nil
	}
	if len(q) != 2 || len(q["type"]) != 1 || q.Get("type") != "email" || len(q["token_hash"]) != 1 {
		return 0, nil, problem.Invalid("Expected token_hash and type=email only")
	}
	if err := s.Identity.Confirm(r.Context(), q.Get("token_hash")); err != nil {
		return 0, nil, err
	}
	http.Redirect(w, r, "portway://auth/confirmed", http.StatusSeeOther)
	return 0, nil, nil
}
func (s *Server) login(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Label    string `json:"device_label"`
	}
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	v, err := s.Identity.Login(r.Context(), in.Email, in.Password, in.Label)
	return 200, v, err
}
func (s *Server) refresh(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if len(in.RefreshToken) < 1 || len(in.RefreshToken) > 16384 {
		return 0, nil, problem.Invalid("Invalid refresh token")
	}
	v, err := s.Identity.Refresh(r.Context(), in.RefreshToken, r.Header.Get("X-Device-Session"))
	return 200, v, err
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	v, err := s.Identity.Logout(r.Context(), r.Header.Get("X-Device-Session"), bearer(r))
	return 200, map[string]any{"device_revoked": true, "upstream_outcome": v}, err
}
func (s *Server) me(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var v identity.Profile
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT id,display_name,email FROM portway.owners WHERE installation_id=$1 AND id=$2`, s.Store.Installation, p.OwnerID).Scan(&v.ID, &v.Name, &v.Email)
	return 200, v, err
}
func (s *Server) devices(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT id,label,last_seen_at,revoked_at FROM portway.device_sessions WHERE installation_id=$1 AND owner_id=$2 ORDER BY created_at,id`, s.Store.Installation, p.OwnerID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, label string
		var seen time.Time
		var revoked *time.Time
		if err = rows.Scan(&id, &label, &seen, &revoked); err != nil {
			return 0, nil, err
		}
		out = append(out, map[string]any{"id": id, "label": label, "last_seen_at": seen, "revoked_at": revoked})
	}
	return 200, map[string]any{"items": out}, rows.Err()
}
func (s *Server) revoke(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	u, err := s.Identity.Auth.User(r.Context(), bearer(r))
	if err != nil {
		return 0, nil, err
	}
	if u.ID != p.Subject {
		return 0, nil, problem.New(403, "OWNER_REQUIRED", "Owner verification failed")
	}
	return s.mutate(r, p, map[string]string{"id": r.PathValue("id")}, func(tx pgx.Tx) (int, any, error) {
		var id string
		err := tx.QueryRow(r.Context(), `UPDATE portway.device_sessions SET revoked_at=coalesce(revoked_at,now()) WHERE installation_id=$1 AND owner_id=$2 AND id=$3 RETURNING id`, s.Store.Installation, p.OwnerID, r.PathValue("id")).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil, problem.NotFound()
		}
		if err != nil {
			return 0, nil, err
		}
		err = s.Store.Audit(r.Context(), tx, p.OwnerID, "device.revoked", "device", id)
		return 200, map[string]any{"id": id, "revoked": true}, err
	})
}
