// Package providercheck owns the control-side capability report lifecycle.
// Execution receives no database access or credentials through this package.
package providercheck

import (
	"time"

	"portway/internal/modelgateway"
	"portway/internal/secure"
)

const ReportTTL = 30 * time.Minute
const AttemptTTL = 15 * time.Second

type Report struct {
	CheckID       string                   `json:"check_id"`
	ProviderID    string                   `json:"provider_id"`
	Endpoint      string                   `json:"endpoint"`
	ModelID       string                   `json:"model_id"`
	ConfigVersion int                      `json:"config_version"`
	Profile       string                   `json:"profile"`
	State         string                   `json:"state"`
	StartedAt     time.Time                `json:"started_at"`
	Deadline      time.Time                `json:"deadline"`
	CheckedAt     *time.Time               `json:"checked_at"`
	ExpiresAt     *time.Time               `json:"expires_at"`
	Protocol      modelgateway.ProbeResult `json:"protocol"`
	// Deliberately never true: protocol probes do not establish a prompt bound,
	// private dispatch authorization, or Runtime readiness.
	Runnable bool `json:"runnable"`
}

// ProtocolValid must be evaluated at use time, with the CURRENT provider row
// and allowlist. Cached success does not survive a configuration or profile
// change. This is only one prerequisite; never use it alone for run admission.
func (r Report) ProtocolValid(provider, endpoint, model string, version int, allowed []string, now time.Time) bool {
	if !secure.UUID(r.CheckID) || r.ProviderID != provider || r.Endpoint != endpoint || r.ModelID != model || r.ConfigVersion != version || r.Profile != modelgateway.ProbeProfile || r.State != "completed" || r.CheckedAt == nil || r.ExpiresAt == nil || r.CheckedAt.Before(r.StartedAt) || !r.CheckedAt.Before(r.Deadline) || r.CheckedAt.After(now) || !now.Before(*r.ExpiresAt) || r.ExpiresAt.After(r.CheckedAt.Add(ReportTTL)) || !r.Protocol.Compatible() {
		return false
	}
	for _, entry := range allowed {
		if entry == endpoint {
			return true
		}
	}
	return false
}
