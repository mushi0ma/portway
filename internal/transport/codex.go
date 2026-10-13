package transport

import (
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"portway/internal/codexbridge"
	"portway/internal/httpjson"
	"portway/internal/identity"
	"portway/internal/problem"
	"portway/internal/secure"
)

func (s *Server) codexStore() codexbridge.Store {
	return codexbridge.Store{DB: s.Store, Vault: s.Vault}
}
func (s *Server) codexEnabled() error {
	if s.Config.CodexBridgeKey == "" {
		return problem.New(503, "CODEX_DISABLED", "Native Codex profile is disabled")
	}
	return nil
}
func (s *Server) codexStatus(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	out, err := s.codexStore().Status(r.Context())
	return 200, out, err
}
func (s *Server) codexCreate(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	var in struct {
		Prompt    string  `json:"prompt"`
		Workspace string  `json:"workspace"`
		Provider  *string `json:"provider"`
		Model     *string `json:"model"`
		Effort    *string `json:"effort"`
	}
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if !utf8.ValidString(in.Prompt) || strings.TrimSpace(in.Prompt) == "" || len(in.Prompt) > 16384 || strings.ContainsRune(in.Prompt, 0) || in.Workspace != "portway" {
		return 0, nil, problem.Invalid("workspace must be portway; prompt must contain 1–16384 UTF-8 bytes")
	}
	create := codexbridge.CreateInput{Prompt: in.Prompt, Provider: codexbridge.ProviderCodex}
	if in.Provider != nil {
		create.Provider = *in.Provider
	}
	if in.Model != nil {
		create.Model = *in.Model
	}
	if in.Effort != nil {
		create.Effort = *in.Effort
	}
	if !codexbridge.ValidProvider(create.Provider) || (in.Model != nil && !codexbridge.ValidModel(create.Model)) || (in.Effort != nil && !codexbridge.ValidEffort(create.Effort)) {
		return 0, nil, problem.Invalid("provider must be codex or claude; model must match ^[A-Za-z0-9._:/\\[\\]-]{1,100}$; effort must be low, medium, high, xhigh or max")
	}
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		out, err := s.codexStore().Create(r.Context(), tx, p.OwnerID, p.DeviceID, create)
		return 202, out, err
	})
}
func (s *Server) codexList(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	q := r.URL.Query()
	f := codexbridge.ListFilter{Provider: q.Get("provider"), State: q.Get("state")}
	states := map[string]bool{"": true, "queued": true, "running": true, "completed": true, "failed": true, "stopped": true}
	if (f.Provider != "" && !codexbridge.ValidProvider(f.Provider)) || !states[f.State] {
		return 0, nil, problem.Invalid("provider must be codex or claude; state must be a run state")
	}
	out, err := s.codexStore().List(r.Context(), f)
	return 200, map[string]any{"items": out}, err
}
func (s *Server) codexGet(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	out, err := s.codexStore().Get(r.Context(), r.PathValue("id"))
	return 200, out, err
}
func (s *Server) codexStop(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	var in struct{}
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		out, err := s.codexStore().Stop(r.Context(), tx, p.OwnerID, r.PathValue("id"))
		return 200, out, err
	})
}
func (s *Server) codexEvents(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	after := 0
	var err error
	if raw := r.URL.Query().Get("after"); raw != "" {
		after, err = strconv.Atoi(raw)
		if err != nil || after < 0 || after > 256 {
			return 0, nil, problem.Invalid("after must be an integer from 0 to 256")
		}
	}
	out, err := s.codexStore().Events(r.Context(), r.PathValue("id"), after)
	next := after
	if len(out) > 0 {
		next = out[len(out)-1].Sequence
	}
	return 200, map[string]any{"items": out, "next_after": next}, err
}

// Account management: owner intents become one-time bridge commands (202 + command_id).
func (s *Server) codexLogin(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	var in struct {
		Provider string `json:"provider"`
		Method   string `json:"method"`
	}
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if !codexbridge.ValidProvider(in.Provider) || (in.Method != "phone" && in.Method != "computer") {
		return 0, nil, problem.Invalid("provider must be codex or claude; method must be phone or computer")
	}
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		out, err := s.codexStore().StartLogin(r.Context(), tx, p.OwnerID, p.DeviceID, in.Provider, in.Method)
		return 202, out, err
	})
}
func (s *Server) codexLoginCode(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	login := r.PathValue("login_id")
	var in struct {
		Code string `json:"code"`
	}
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	code := strings.TrimSpace(in.Code)
	if !secure.UUID(login) || code == "" || len(code) > 2048 || !utf8.ValidString(code) || strings.IndexFunc(code, func(c rune) bool { return unicode.IsSpace(c) || unicode.IsControl(c) }) >= 0 {
		return 0, nil, problem.Invalid("login code must contain 1–2048 characters without inner whitespace")
	}
	// The idempotency hash is a keyed MAC; the code itself is never stored outside the encrypted command.
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		out, err := s.codexStore().LoginCommand(r.Context(), tx, p.OwnerID, p.DeviceID, login, "login_code", code)
		return 202, out, err
	})
}
func (s *Server) codexLoginCancel(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	login := r.PathValue("login_id")
	var in struct{}
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if !secure.UUID(login) {
		return 0, nil, problem.Invalid("Invalid UUID")
	}
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		out, err := s.codexStore().LoginCommand(r.Context(), tx, p.OwnerID, p.DeviceID, login, "login_cancel", "")
		return 202, out, err
	})
}
func (s *Server) codexActivate(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	var in struct {
		Provider string `json:"provider"`
	}
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	account := r.PathValue("account_id")
	if !codexbridge.ValidProvider(in.Provider) || !codexbridge.ValidAccountID(in.Provider, account) {
		return 0, nil, problem.Invalid("provider must be codex or claude and own the account id")
	}
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		out, err := s.codexStore().AccountCommand(r.Context(), tx, p.OwnerID, p.DeviceID, "account_activate", in.Provider, account)
		return 202, out, err
	})
}
func (s *Server) codexRemoveAccount(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	provider, account := r.URL.Query().Get("provider"), r.PathValue("account_id")
	if !codexbridge.ValidProvider(provider) || !codexbridge.ValidAccountID(provider, account) {
		return 0, nil, problem.Invalid("provider query must be codex or claude and own the account id")
	}
	return s.mutate(r, p, map[string]string{"provider": provider, "account_id": account}, func(tx pgx.Tx) (int, any, error) {
		out, err := s.codexStore().AccountCommand(r.Context(), tx, p.OwnerID, p.DeviceID, "account_remove", provider, account)
		return 202, out, err
	})
}
func (s *Server) codexUsageRefresh(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexEnabled(); err != nil {
		return 0, nil, err
	}
	var in struct{}
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	return s.mutate(r, p, in, func(tx pgx.Tx) (int, any, error) {
		out, err := s.codexStore().RefreshUsage(r.Context(), tx, p.OwnerID, p.DeviceID)
		return 202, out, err
	})
}

func (s *Server) codexBridgeAuth(r *http.Request) error {
	if err := s.codexEnabled(); err != nil {
		return err
	}
	if !secure.Equal(r.Header.Get("X-Portway-Bridge-Key"), s.Config.CodexBridgeKey) {
		return problem.New(403, "CODEX_BRIDGE_DENIED", "Bridge authentication required")
	}
	return nil
}
func (s *Server) codexPoll(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexBridgeAuth(r); err != nil {
		return 0, nil, err
	}
	var in codexbridge.Poll
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if !secure.UUID(in.Instance) || (in.RunID != "" && (!secure.UUID(in.RunID) || len(in.Token) != 43)) || (in.RunID == "" && in.Token != "") {
		return 0, nil, problem.Invalid("Invalid bridge poll")
	}
	out, err := s.codexStore().Poll(r.Context(), in)
	return 200, out, err
}
func (s *Server) codexState(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexBridgeAuth(r); err != nil {
		return 0, nil, err
	}
	var in codexbridge.Snapshot
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if err := in.Validate(); err != nil {
		return 0, nil, err
	}
	err := s.codexStore().SaveState(r.Context(), in)
	return 200, map[string]any{"accepted": err == nil}, err
}
func (s *Server) codexAck(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexBridgeAuth(r); err != nil {
		return 0, nil, err
	}
	var in codexbridge.Ack
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if (in.State != "done" && in.State != "failed") || (in.ErrorCode != nil && !codexbridge.ValidErrorCode(*in.ErrorCode)) || (in.State == "done" && in.ErrorCode != nil) {
		return 0, nil, problem.Invalid("Invalid command outcome")
	}
	err := s.codexStore().AckCommand(r.Context(), r.PathValue("id"), in)
	return 200, map[string]any{"accepted": err == nil}, err
}
func (s *Server) codexReport(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.codexBridgeAuth(r); err != nil {
		return 0, nil, err
	}
	var in codexbridge.Report
	if err := httpjson.Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	e := in.Event
	if !secure.UUID(in.RunID) || len(in.Token) != 43 || e.Sequence < 1 || e.Sequence > 256 || !utf8.ValidString(e.Text) {
		return 0, nil, problem.Invalid("Invalid bridge event")
	}
	if (e.Meta != nil && e.Type != "completed") || (e.Code != "" && e.Type != "failed") || !e.Meta.Validate() {
		return 0, nil, problem.Invalid("Invalid bridge event metadata")
	}
	switch e.Type {
	case "started", "progress":
		if len(e.Text) > 8192 {
			return 0, nil, problem.Invalid("Event too long")
		}
	case "completed":
		if len(e.Text) > 65536 || strings.TrimSpace(e.Text) == "" {
			return 0, nil, problem.Invalid("Invalid answer")
		}
	case "failed":
		if e.Text != "" || (e.Code != "" && !codexbridge.RunFailureCode(e.Code)) {
			return 0, nil, problem.Invalid("Failure details must be omitted")
		}
	default:
		return 0, nil, problem.Invalid("Unknown bridge event")
	}
	err := s.codexStore().Report(r.Context(), in)
	return 200, map[string]any{"accepted": err == nil}, err
}
