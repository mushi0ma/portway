package codexbridge

import (
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"portway/internal/problem"
	"portway/internal/secure"
)

// Shared Agent Bridge v2 vocabulary. The API validates every bridge snapshot
// against it, and the laptop bridge builds snapshots with the same types.
const (
	ProviderCodex  = "codex"
	ProviderClaude = "claude"
)

var (
	modelRE    = regexp.MustCompile(`^[A-Za-z0-9._:/\[\]-]{1,100}$`)
	accountRE  = regexp.MustCompile(`^(codex|claude)-[a-z0-9]{1,32}$`)
	errorRE    = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)
	windowRE   = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)
	userCodeRE = regexp.MustCompile(`^[A-Za-z0-9-]{1,32}$`)
	versionRE  = regexp.MustCompile(`^[0-9A-Za-z._+() -]{1,40}$`)
	efforts    = []string{"low", "medium", "high", "xhigh", "max"}
	loginHosts = map[string]bool{"auth.openai.com": true, "chatgpt.com": true, "claude.ai": true, "claude.com": true, "platform.claude.com": true, "console.anthropic.com": true}
	runCodes   = map[string]bool{"AI_LIMIT_EXHAUSTED": true, "AI_AUTH_REQUIRED": true, "AI_RUN_FAILED": true, "AI_TIMEOUT": true}
)

func ValidProvider(p string) bool { return p == ProviderCodex || p == ProviderClaude }
func ValidModel(m string) bool    { return modelRE.MatchString(m) }
func ValidAccountID(provider, id string) bool {
	return accountRE.MatchString(id) && strings.HasPrefix(id, provider+"-")
}
func ValidEffort(e string) bool {
	for _, v := range efforts {
		if v == e {
			return true
		}
	}
	return false
}
func ValidErrorCode(c string) bool { return errorRE.MatchString(c) }

// RunFailureCode lists the bridge-reported run failure codes.
func RunFailureCode(c string) bool { return runCodes[c] }

// SafeLoginURL returns the URL only for HTTPS pages of the expected identity providers.
func SafeLoginURL(raw string) *string {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 4096 || u.Scheme != "https" || u.User != nil || u.Port() != "" || !loginHosts[strings.ToLower(u.Hostname())] || !plain(raw, 4096) {
		return nil
	}
	return &raw
}

type Account struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Email    *string `json:"email"`
	Plan     *string `json:"plan"`
	LoggedIn bool    `json:"logged_in"`
	Active   bool    `json:"active"`
	System   bool    `json:"system"`
}
type Model struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Default       bool     `json:"default"`
	Efforts       []string `json:"efforts"`
	DefaultEffort *string  `json:"default_effort"`
}
type Window struct {
	ID            string     `json:"id"`
	UsedPercent   float64    `json:"used_percent"`
	WindowMinutes *int64     `json:"window_minutes"`
	ResetsAt      *time.Time `json:"resets_at"`
}
type Usage struct {
	Status    string     `json:"status"`
	CheckedAt time.Time  `json:"checked_at"`
	ResetAt   *time.Time `json:"reset_at"`
	Windows   []Window   `json:"windows"`
}
type ProviderState struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Installed       bool      `json:"installed"`
	CLIVersion      *string   `json:"cli_version"`
	ActiveAccountID *string   `json:"active_account_id"`
	Accounts        []Account `json:"accounts"`
	Models          []Model   `json:"models"`
	Usage           *Usage    `json:"usage"`
}
type Login struct {
	ID        string     `json:"id"`
	Provider  string     `json:"provider"`
	Method    string     `json:"method"`
	State     string     `json:"state"`
	URL       *string    `json:"url"`
	UserCode  *string    `json:"user_code"`
	NeedsCode bool       `json:"needs_code"`
	ErrorCode *string    `json:"error_code"`
	AccountID *string    `json:"account_id"`
	ExpiresAt *time.Time `json:"expires_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
type Snapshot struct {
	Instance  string          `json:"instance_id"`
	Providers []ProviderState `json:"providers"`
	Login     *Login          `json:"login"`
}

// Terminal reports whether a login flow can no longer change.
func (l Login) Terminal() bool {
	return l.State == "completed" || l.State == "failed" || l.State == "cancelled" || l.State == "expired"
}

// Commands are queued by the owner and delivered once to the active bridge.
type CommandPayload struct {
	LoginID   string `json:"login_id,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Method    string `json:"method,omitempty"`
	Code      string `json:"code,omitempty"`
	AccountID string `json:"account_id,omitempty"`
}
type Command struct {
	ID      string         `json:"id"`
	Kind    string         `json:"kind"`
	Payload CommandPayload `json:"payload"`
}
type CommandView struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	State     string    `json:"state"`
	ErrorCode *string   `json:"error_code"`
	UpdatedAt time.Time `json:"updated_at"`
}
type Ack struct {
	State     string  `json:"state"`
	ErrorCode *string `json:"error_code"`
}
type Tokens struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
}
type RunMeta struct {
	ResolvedModel *string `json:"resolved_model,omitempty"`
	AccountLabel  *string `json:"account_label,omitempty"`
	Tokens        *Tokens `json:"tokens,omitempty"`
}

// plain accepts bounded single-line UTF-8 text without control characters.
func plain(s string, max int) bool {
	return len(s) <= max && utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
}
func optional(s *string, max int) bool { return s == nil || (*s != "" && plain(*s, max)) }
func clampPercent(v float64) float64 {
	if v != v || v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// Validate checks a bridge snapshot before it is encrypted and served to the phone.
// Unsafe login URLs are dropped instead of rejecting the whole snapshot.
func (s *Snapshot) Validate() error {
	bad := problem.Invalid("Invalid bridge state")
	if !secure.UUID(s.Instance) || len(s.Providers) > 2 {
		return bad
	}
	seen := map[string]bool{}
	for i := range s.Providers {
		p := &s.Providers[i]
		if !ValidProvider(p.ID) || seen[p.ID] || p.Name == "" || !plain(p.Name, 40) || (p.CLIVersion != nil && !versionRE.MatchString(*p.CLIVersion)) || len(p.Accounts) > 10 || len(p.Models) > 64 {
			return bad
		}
		seen[p.ID] = true
		if p.Accounts == nil {
			p.Accounts = []Account{}
		}
		if p.Models == nil {
			p.Models = []Model{}
		}
		activeFound := p.ActiveAccountID == nil
		ids := map[string]bool{}
		for _, a := range p.Accounts {
			if !ValidAccountID(p.ID, a.ID) || ids[a.ID] || a.Label == "" || !plain(a.Label, 60) || !optional(a.Email, 254) || !optional(a.Plan, 40) {
				return bad
			}
			ids[a.ID] = true
			if p.ActiveAccountID != nil && *p.ActiveAccountID == a.ID {
				activeFound = true
			}
		}
		if !activeFound {
			return bad
		}
		models := map[string]bool{}
		for j := range p.Models {
			m := &p.Models[j]
			if !ValidModel(m.ID) || models[m.ID] || m.Name == "" || !plain(m.Name, 100) || !plain(m.Description, 1000) || len(m.Efforts) > len(efforts) || (m.DefaultEffort != nil && !ValidEffort(*m.DefaultEffort)) {
				return bad
			}
			models[m.ID] = true
			if m.Efforts == nil {
				m.Efforts = []string{}
			}
			for _, e := range m.Efforts {
				if !ValidEffort(e) {
					return bad
				}
			}
		}
		if u := p.Usage; u != nil {
			if (u.Status != "ok" && u.Status != "warning" && u.Status != "exhausted" && u.Status != "unknown") || u.CheckedAt.IsZero() || len(u.Windows) > 8 {
				return bad
			}
			if u.Windows == nil {
				u.Windows = []Window{}
			}
			for k := range u.Windows {
				w := &u.Windows[k]
				if !windowRE.MatchString(w.ID) || (w.WindowMinutes != nil && (*w.WindowMinutes < 0 || *w.WindowMinutes > 1000000)) {
					return bad
				}
				w.UsedPercent = clampPercent(w.UsedPercent)
			}
		}
	}
	if l := s.Login; l != nil {
		states := map[string]bool{"starting": true, "waiting_user": true, "verifying": true, "completed": true, "failed": true, "cancelled": true, "expired": true}
		if !secure.UUID(l.ID) || !ValidProvider(l.Provider) || (l.Method != "phone" && l.Method != "computer") || !states[l.State] || l.UpdatedAt.IsZero() ||
			(l.UserCode != nil && !userCodeRE.MatchString(*l.UserCode)) || (l.ErrorCode != nil && !ValidErrorCode(*l.ErrorCode)) || (l.AccountID != nil && !ValidAccountID(l.Provider, *l.AccountID)) {
			return bad
		}
		if l.URL != nil {
			l.URL = SafeLoginURL(*l.URL)
		}
	}
	return nil
}

// Validate checks the optional completion metadata reported by the bridge.
func (m *RunMeta) Validate() bool {
	return m == nil || ((m.ResolvedModel == nil || ValidModel(*m.ResolvedModel)) && optional(m.AccountLabel, 60) &&
		(m.Tokens == nil || (m.Tokens.Input >= 0 && m.Tokens.Output >= 0 && m.Tokens.Input < 1e12 && m.Tokens.Output < 1e12)))
}
