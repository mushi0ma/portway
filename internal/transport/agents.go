package transport

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"portway/internal/agents"
	"portway/internal/identity"
	"portway/internal/problem"
	"portway/internal/secure"
)

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in struct {
		Name   string        `json:"name"`
		Config agents.Config `json:"config"`
	}
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 120 {
		return 0, nil, problem.Invalid("Name must contain 1–120 bytes")
	}
	c, err := agents.Validate(in.Config)
	if err != nil {
		return 0, nil, err
	}
	in.Config = c
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		v, err := s.Agents.Create(r.Context(), tx, p.OwnerID, in.Name, c)
		return 201, v, err
	})
}
func (s *Server) newVersion(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	var in struct {
		Config        agents.Config `json:"config"`
		BaseVersionID string        `json:"base_version_id"`
	}
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	ver, err := expected(r)
	if err != nil {
		return 0, nil, err
	}
	if !secure.UUID(in.BaseVersionID) {
		return 0, nil, problem.Invalid("Invalid base_version_id")
	}
	c, err := agents.Validate(in.Config)
	if err != nil {
		return 0, nil, err
	}
	in.Config = c
	return s.mutate(r, p, struct {
		Body    any `json:"body"`
		Version int `json:"version"`
	}{in, ver}, func(tx pgx.Tx) (int, any, error) {
		v, err := s.Agents.NewVersion(r.Context(), tx, p.OwnerID, r.PathValue("id"), in.BaseVersionID, ver, c)
		return 201, v, err
	})
}
