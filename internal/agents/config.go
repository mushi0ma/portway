package agents

import (
	"encoding/json"
	"regexp"
	"strings"

	"portway/internal/problem"
	"portway/internal/secure"
)

type Limits struct {
	RunSeconds   int `json:"run_seconds"`
	Steps        int `json:"steps"`
	Tokens       int `json:"tokens"`
	OutputTokens int `json:"output_tokens"`
}
type Skill struct {
	Alias   string `json:"alias"`
	Content string `json:"content"`
}
type Tool struct {
	Alias string `json:"alias"`
	Kind  string `json:"kind"`
}
type MCP struct {
	Alias         string `json:"alias"`
	AdapterKind   string `json:"adapter_kind"`
	EndpointAlias string `json:"endpoint_alias"`
	SecretRef     string `json:"secret_ref,omitempty"`
}
type Config struct {
	Instructions       string            `json:"instructions"`
	RoleCode           string            `json:"role_code"`
	RoleCatalogVersion int               `json:"role_catalog_version"`
	Skills             []Skill           `json:"skills"`
	Tools              []Tool            `json:"tools"`
	MCP                []MCP             `json:"mcp"`
	ProviderRef        string            `json:"provider_ref,omitempty"`
	Limits             Limits            `json:"limits"`
	Policy             map[string]string `json:"policy"`
}
type Role struct {
	Code           string            `json:"code"`
	Group          string            `json:"group"`
	CatalogVersion int               `json:"catalog_version"`
	Instructions   string            `json:"instructions"`
	BaselinePolicy map[string]string `json:"baseline_policy"`
}

func Roles() []Role {
	var out []Role
	for _, r := range []struct {
		code, group, instructions string
		write, shell              bool
	}{
		{"coder", "main", "Implement the requested change and explain validation.", true, true},
		{"reviewer", "main", "Review the pinned result independently; return a verdict without editing it.", false, false},
		{"tester", "main", "Test the requested change within the approved workspace.", true, true},
		{"researcher", "main", "Research using approved documentation and workspace reads.", false, false},
		{"documenter", "main", "Write documentation for the requested change.", true, false},
		{"code_reader", "subagent", "Read and explain code without changing it.", false, false},
		{"test_runner", "subagent", "Run approved test commands and report their results.", false, true},
		{"docs_lookup", "subagent", "Read approved documentation and cite the relevant facts.", false, false},
	} {
		p := map[string]string{"workspace.read": "allow", "workspace.write": "deny", "shell.exec": "deny", "docs.lookup": "allow"}
		if r.write {
			p["workspace.write"] = "allow"
		}
		if r.shell {
			p["shell.exec"] = "approval"
		}
		out = append(out, Role{r.code, r.group, 1, r.instructions, p})
	}
	return out
}

var aliasRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func Alias(s string) bool { return aliasRE.MatchString(s) }
func Validate(c Config) (Config, error) {
	bad := func() (Config, error) {
		return c, problem.Invalid("Invalid agent configuration or policy exceeds role permissions")
	}
	if strings.TrimSpace(c.Instructions) == "" || len(c.Instructions) > 65536 || c.RoleCatalogVersion != 1 {
		return bad()
	}
	var role *Role
	for _, r := range Roles() {
		if r.Code == c.RoleCode {
			role = &r
			break
		}
	}
	if role == nil {
		return bad()
	}
	if c.ProviderRef != "" && !secure.UUID(c.ProviderRef) {
		return bad()
	}
	if c.Limits == (Limits{}) {
		c.Limits = Limits{900, 20, 20000, 2048}
	}
	l := c.Limits
	if l.RunSeconds < 1 || l.RunSeconds > 900 || l.Steps < 1 || l.Steps > 20 || l.Tokens < 1 || l.Tokens > 20000 || l.OutputTokens < 1 || l.OutputTokens > 2048 || l.OutputTokens > l.Tokens {
		return bad()
	}
	if len(c.Tools) > 16 || len(c.Skills) > 64 || len(c.MCP) > 16 {
		return bad()
	}
	seen := map[string]bool{}
	known := map[string]bool{"workspace.read": true, "workspace.write": true, "shell.exec": true, "docs.lookup": true}
	for _, t := range c.Tools {
		if !Alias(t.Alias) || seen[t.Alias] || !known[t.Kind] || role.BaselinePolicy[t.Kind] == "deny" {
			return bad()
		}
		seen[t.Alias] = true
	}
	seen = map[string]bool{}
	total := 0
	for _, s := range c.Skills {
		total += len(s.Content)
		if !Alias(s.Alias) || seen[s.Alias] || strings.TrimSpace(s.Content) == "" || len(s.Content) > 65536 || total > 262144 {
			return bad()
		}
		seen[s.Alias] = true
	}
	seen = map[string]bool{}
	for _, m := range c.MCP {
		if !Alias(m.Alias) || seen[m.Alias] || m.AdapterKind != "docs.lookup" || !Alias(m.EndpointAlias) || m.SecretRef != "" && !secure.UUID(m.SecretRef) {
			return bad()
		}
		seen[m.Alias] = true
	}
	if c.Policy == nil {
		c.Policy = map[string]string{}
	}
	rank := map[string]int{"allow": 0, "approval": 1, "deny": 2}
	for k, v := range c.Policy {
		r, ok := rank[v]
		base, bok := role.BaselinePolicy[k]
		if !ok || !bok || r < rank[base] {
			return bad()
		}
	}
	for k, v := range role.BaselinePolicy {
		if _, ok := c.Policy[k]; !ok {
			c.Policy[k] = v
		}
	}
	if c.Skills == nil {
		c.Skills = []Skill{}
	}
	if c.Tools == nil {
		c.Tools = []Tool{}
	}
	if c.MCP == nil {
		c.MCP = []MCP{}
	}
	return c, nil
}
func Hash(c Config) string { b, _ := json.Marshal(c); return secure.Hash(string(b)) }
