package api

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestOpenAPIImplementedRoutes(t *testing.T) {
	b, err := os.ReadFile("openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		OpenAPI    string                                `json:"openapi"`
		Paths      map[string]map[string]json.RawMessage `json:"paths"`
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err = json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.OpenAPI != "3.1.0" {
		t.Fatal("unsupported contract version")
	}
	source, err := os.ReadFile("../internal/transport/server.go")
	if err != nil {
		t.Fatal(err)
	}
	implemented := map[string]bool{}
	for _, m := range regexp.MustCompile(`"(GET|POST|PUT|DELETE) (/[^" ]+)"`).FindAllStringSubmatch(string(source), -1) {
		implemented[m[1]+" "+m[2]] = true
	}
	for path, methods := range doc.Paths {
		for method, raw := range methods {
			route := strings.ToUpper(method) + " " + path
			if !implemented[route] {
				t.Errorf("OpenAPI promises unimplemented route: %s", route)
			}
			delete(implemented, route)
			var op struct {
				ID        string           `json:"operationId"`
				Security  []map[string]any `json:"security"`
				Responses map[string]any   `json:"responses"`
			}
			if err := json.Unmarshal(raw, &op); err != nil {
				t.Fatal(err)
			}
			if op.ID == "" || len(op.Responses) == 0 {
				t.Error("incomplete operation", route)
			}
			if strings.HasPrefix(path, "/v1/") && !strings.Contains(path, "/auth/") && path != "/v1/client-config" {
				if len(op.Security) != 1 || op.Security[0]["bearerAuth"] == nil || op.Security[0]["deviceProof"] == nil {
					t.Error("missing combined auth guard", route)
				}
			}
		}
	}
	for route := range implemented {
		t.Error("implemented route missing from OpenAPI", route)
	}
	for _, m := range regexp.MustCompile(`"\$ref":\s*"#/components/schemas/([^"]+)"`).FindAllStringSubmatch(string(b), -1) {
		if _, ok := doc.Components.Schemas[m[1]]; !ok {
			t.Error("unresolved schema", m[1])
		}
	}
}
