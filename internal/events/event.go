// Package events implements the first inter-service contract: safe audit facts.
package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"portway/internal/secure"
)

const Topic = "portway.control.audit.v1"

type Audit struct {
	ActorID    *string `json:"actor_id"`
	Action     string  `json:"action"`
	ObjectType string  `json:"object_type"`
	ObjectID   *string `json:"object_id"`
	Outcome    string  `json:"outcome"`
}

type Event struct {
	SchemaVersion  int       `json:"schema_version"`
	EventID        string    `json:"event_id"`
	InstallationID string    `json:"installation_id"`
	Producer       string    `json:"producer"`
	Type           string    `json:"type"`
	AggregateID    string    `json:"aggregate_id"`
	CorrelationID  string    `json:"correlation_id"`
	OccurredAt     time.Time `json:"occurred_at"`
	Payload        Audit     `json:"payload"`
}

func (e Event) Key() string { return e.InstallationID + "/" + e.AggregateID }

func (e Event) Validate(now time.Time) error {
	types := map[string]string{"owner.bound": "owner", "device.login": "device", "device.revoked": "device", "agent.created": "agent", "agent.version_created": "agent_version", "provider.created": "provider", "provider.updated": "provider", "provider.check_started": "provider", "provider.check_completed": "provider", "secret.created": "secret", "codex.queued": "codex_run", "codex.stopped": "codex_run", "codex.completed": "codex_run", "codex.failed": "codex_run", "codex.command_queued": "codex_command"}
	if e.SchemaVersion != 1 || e.Producer != "control-api" || e.Type != "audit.recorded" ||
		!secure.UUID(e.EventID) || !secure.UUID(e.InstallationID) || !secure.UUID(e.AggregateID) || !secure.UUID(e.CorrelationID) ||
		e.Payload.ActorID == nil || !secure.UUID(*e.Payload.ActorID) || e.Payload.ObjectID == nil || !secure.UUID(*e.Payload.ObjectID) ||
		e.AggregateID != *e.Payload.ObjectID || types[e.Payload.Action] == "" || types[e.Payload.Action] != e.Payload.ObjectType || e.Payload.Outcome != "success" {
		return errors.New("invalid audit event contract")
	}
	if e.OccurredAt.IsZero() || e.OccurredAt.After(now.Add(time.Minute)) || e.OccurredAt.Before(now.Add(-24*time.Hour)) {
		return errors.New("audit event outside delivery window")
	}
	return nil
}

func Decode(b []byte, installation string, now time.Time) (Event, error) {
	var e Event
	if len(b) > 4096 {
		return e, errors.New("event exceeds size limit")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&e); err != nil {
		return e, errors.New("invalid event JSON")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return e, errors.New("trailing event JSON")
	}
	if e.InstallationID != installation {
		return e, errors.New("event installation mismatch")
	}
	return e, e.Validate(now)
}
