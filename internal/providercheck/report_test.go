package providercheck

import (
	"testing"
	"time"

	"portway/internal/modelgateway"
	"portway/internal/secure"
)

func TestReportBindingAndExpiry(t *testing.T) {
	now := time.Now().UTC()
	expires := now.Add(ReportTTL)
	id := secure.ID()
	base := Report{CheckID: secure.ID(), ProviderID: id, Endpoint: "http://fixture/v1", ModelID: "m1", ConfigVersion: 1, Profile: modelgateway.ProbeProfile, State: "completed", StartedAt: now.Add(-time.Second), Deadline: now.Add(time.Second), CheckedAt: &now, ExpiresAt: &expires, Protocol: modelgateway.ProbeResult{Streaming: true, ToolCalling: true, ToolResult: true, ReportedUsage: true, Requests: 2}}
	valid := func(r Report) bool { return r.ProtocolValid(id, base.Endpoint, "m1", 1, []string{base.Endpoint}, now) }
	if !valid(base) || base.Runnable {
		t.Fatal("protocol success should not authorize a run")
	}
	for _, tc := range []struct {
		name string
		edit func(*Report)
	}{
		{"id", func(r *Report) { r.CheckID = "" }},
		{"provider", func(r *Report) { r.ProviderID = secure.ID() }},
		{"endpoint", func(r *Report) { r.Endpoint += "/other" }},
		{"model", func(r *Report) { r.ModelID = "m2" }},
		{"version", func(r *Report) { r.ConfigVersion++ }},
		{"profile", func(r *Report) { r.Profile = "old" }},
		{"checking", func(r *Report) { r.State = "checking" }},
		{"no-time", func(r *Report) { r.CheckedAt = nil }},
		{"no-expiry", func(r *Report) { r.ExpiresAt = nil }},
		{"expired", func(r *Report) { r.ExpiresAt = &now }},
		{"excessive-ttl", func(r *Report) { future := expires.Add(time.Second); r.ExpiresAt = &future }},
		{"late-response", func(r *Report) { r.Deadline = now }},
		{"future", func(r *Report) { future := now.Add(time.Second); r.CheckedAt = &future }},
		{"clock-regression", func(r *Report) { r.StartedAt = now.Add(time.Second) }},
		{"failed", func(r *Report) { r.Protocol.ErrorCode = "MODEL_UNAVAILABLE" }},
		{"no-usage", func(r *Report) { r.Protocol.ReportedUsage = false }},
		{"no-roundtrip", func(r *Report) { r.Protocol.ToolResult = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			tc.edit(&r)
			if valid(r) {
				t.Fatal("accepted invalid capability report", r)
			}
		})
	}
	if base.ProtocolValid(id, base.Endpoint, "m1", 1, nil, now) {
		t.Fatal("accepted endpoint removed from allowlist")
	}
}
