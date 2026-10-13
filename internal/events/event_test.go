package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"portway/internal/secure"
)

func example() Event {
	actor, object := secure.ID(), secure.ID()
	return Event{SchemaVersion: 1, EventID: secure.ID(), InstallationID: secure.ID(), Producer: "control-api", Type: "audit.recorded", AggregateID: object, CorrelationID: secure.ID(), OccurredAt: time.Now().UTC(), Payload: Audit{ActorID: &actor, Action: "agent.created", ObjectType: "agent", ObjectID: &object, Outcome: "success"}}
}

func TestContract(t *testing.T) {
	e := example()
	b, _ := json.Marshal(e)
	if _, err := Decode(b, e.InstallationID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(b, secure.ID(), time.Now()); err == nil {
		t.Fatal("cross-installation event accepted")
	}
	poisoned := strings.Replace(string(b), `"payload":{`, `"payload":{"password":"secret-marker",`, 1)
	if _, err := Decode([]byte(poisoned), e.InstallationID, time.Now()); err == nil {
		t.Fatal("unrecognized payload field accepted")
	}
	cases := []func(*Event){
		func(e *Event) { e.SchemaVersion = 2 }, func(e *Event) { e.Producer = "other" },
		func(e *Event) { e.Payload.Action = "raw-secret-marker" }, func(e *Event) { e.Payload.ObjectType = "secret-marker" },
		func(e *Event) { e.OccurredAt = time.Now().Add(-25 * time.Hour) }, func(e *Event) { e.OccurredAt = time.Now().Add(time.Hour) },
		func(e *Event) { e.Payload.ActorID = nil }, func(e *Event) { e.AggregateID = secure.ID() },
	}
	for i, change := range cases {
		copy := e
		change(&copy)
		if copy.Validate(time.Now()) == nil {
			t.Fatalf("invalid case %d accepted", i)
		}
	}
}

func TestProviderCheckAuditContract(t *testing.T) {
	for _, action := range []string{"provider.check_started", "provider.check_completed", "codex.queued", "codex.stopped", "codex.completed", "codex.failed", "codex.command_queued"} {
		e := example()
		e.Payload.Action, e.Payload.ObjectType = action, "provider"
		if strings.HasPrefix(action, "codex.") {
			e.Payload.ObjectType = "codex_run"
		}
		if action == "codex.command_queued" {
			e.Payload.ObjectType = "codex_command"
		}
		b, _ := json.Marshal(e)
		if _, err := Decode(b, e.InstallationID, time.Now()); err != nil {
			t.Fatal("provider check would block the relay", action, err)
		}
		e.Payload.ObjectType = "secret"
		if e.Validate(time.Now()) == nil {
			t.Fatal("wrong audit object accepted")
		}
	}
}
