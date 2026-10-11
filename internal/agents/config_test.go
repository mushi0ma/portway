package agents

import "testing"

func baseConfig() Config {
	return Config{Instructions: "Implement a small change", RoleCode: "coder", RoleCatalogVersion: 1, Tools: []Tool{{"read", "workspace.read"}, {"shell", "shell.exec"}}}
}
func TestRolePolicies(t *testing.T) {
	r := Roles()
	main, sub := 0, 0
	for _, v := range r {
		if v.Group == "main" {
			main++
		} else {
			sub++
		}
	}
	if main != 5 || sub != 3 {
		t.Fatal("role catalog must be 5+3")
	}
	c, err := Validate(baseConfig())
	if err != nil {
		t.Fatal(err)
	}
	if c.Limits != (Limits{900, 20, 20000, 2048}) || c.Policy["shell.exec"] != "approval" {
		t.Fatal("SRS defaults changed")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Config)
	}{
		{"shell auto allow", func(c *Config) { c.Policy = map[string]string{"shell.exec": "allow"} }},
		{"reviewer write", func(c *Config) { c.RoleCode = "reviewer"; c.Tools = []Tool{{"write", "workspace.write"}} }},
		{"unknown role", func(c *Config) { c.RoleCode = "admin" }},
		{"unknown tool", func(c *Config) { c.Tools = []Tool{{"docker", "docker.exec"}} }},
		{"duplicate alias", func(c *Config) { c.Tools = append(c.Tools, c.Tools[0]) }},
		{"budget escalation", func(c *Config) { c.Limits = Limits{901, 20, 20000, 2048} }},
		{"unknown permission", func(c *Config) { c.Policy = map[string]string{"host.root": "allow"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := baseConfig()
			tc.mutate(&c)
			if _, err := Validate(c); err == nil {
				t.Fatal("unsafe config accepted")
			}
		})
	}
	deny := baseConfig()
	deny.Policy = map[string]string{"shell.exec": "deny"}
	if _, err := Validate(deny); err != nil {
		t.Fatal("stricter policy must work", err)
	}
}
func TestCanonicalConfigHash(t *testing.T) {
	a, _ := Validate(baseConfig())
	b := baseConfig()
	b.Limits = Limits{900, 20, 20000, 2048}
	b.Policy = map[string]string{"shell.exec": "approval", "workspace.write": "allow", "workspace.read": "allow", "docs.lookup": "allow"}
	b, _ = Validate(b)
	if Hash(a) != Hash(b) {
		t.Fatal("equivalent configs must hash identically")
	}
	b.Instructions = "different"
	if Hash(a) == Hash(b) {
		t.Fatal("hash must cover instructions")
	}
}
