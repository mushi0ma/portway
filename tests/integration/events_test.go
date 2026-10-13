package integration

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"portway/internal/events"
	"portway/internal/secure"
	"portway/internal/storage"
)

func TestKafkaAuditDelivery(t *testing.T) {
	dsn, brokers := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_KAFKA_BROKERS")
	if dsn == "" || brokers == "" {
		if os.Getenv("PORTWAY_REQUIRE_INTEGRATION") == "1" {
			t.Fatal("Kafka and disposable PostgreSQL required")
		}
		t.Skip("use make docker-test")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Path != "/portway_test" {
		t.Fatal("disposable database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	id := secure.ID()
	root, err := storage.Open(ctx, dsn, id)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err = root.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = root.EnsureInstallation(ctx, "Kafka fixture", secure.Hash(secure.Token())); err != nil {
		t.Fatal(err)
	}
	app, err := storage.Open(ctx, os.Getenv("TEST_APP_DATABASE_URL"), id)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	relayDB, err := storage.OpenWithLimit(ctx, os.Getenv("TEST_RELAY_DATABASE_URL"), id, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer relayDB.Close()
	auditDB, err := storage.OpenWithLimit(ctx, os.Getenv("TEST_AUDIT_DATABASE_URL"), id, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer auditDB.Close()
	relay := events.Store{Pool: relayDB.Pool, Installation: id}
	audit := events.Store{Pool: auditDB.Pool, Installation: id}
	if err = relay.CheckRole(ctx, "relay"); err != nil {
		t.Fatal(err)
	}
	if err = audit.CheckRole(ctx, "audit"); err != nil {
		t.Fatal(err)
	}
	for _, s := range []*storage.Store{relayDB, auditDB} {
		if _, err = s.Pool.Exec(ctx, `SELECT ciphertext FROM portway.secrets`); err == nil {
			t.Fatal("worker can read secrets")
		}
	}
	if _, err = app.Pool.Exec(ctx, `SELECT * FROM portway_audit.entries`); err == nil {
		t.Fatal("API can directly read foreign service schema")
	}
	if _, err = auditDB.Pool.Exec(ctx, `SELECT * FROM portway.outbox`); err == nil {
		t.Fatal("audit worker can read source schema")
	}
	if _, err = relayDB.Pool.Exec(ctx, `UPDATE portway.outbox SET action='tampered'`); err == nil {
		t.Fatal("relay can rewrite event")
	}
	actor, object := secure.ID(), secure.ID()
	tx, err := app.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = app.Audit(ctx, tx, actor, "agent.created", "agent", object); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback(ctx)
	var count int
	if err = root.Pool.QueryRow(ctx, `SELECT count(*) FROM portway.outbox WHERE installation_id=$1`, id).Scan(&count); err != nil || count != 0 {
		t.Fatal("rollback leaked event", err, count)
	}
	tx, err = app.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = app.Audit(ctx, tx, actor, "agent.created", "agent", object); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	// Broker failure: transaction rolls back, the fact is still pending.
	outage := errors.New("broker unavailable fixture")
	if _, err = relay.RelayOne(ctx, func(context.Context, events.Event) error { return outage }); !errors.Is(err, outage) {
		t.Fatal(err)
	}
	if err = root.Pool.QueryRow(ctx, `SELECT count(*) FROM portway.outbox WHERE installation_id=$1 AND published_at IS NULL`, id).Scan(&count); err != nil || count != 1 {
		t.Fatal("outage lost event", err, count)
	}
	producer, err := events.Producer(strings.Split(brokers, ","))
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	publish := events.PublishKafka(producer)
	var event events.Event
	// Crash after broker ACK, before DB commit: a real Kafka record exists, but
	// outbox is pending. Next relay attempt must publish the same event ID.
	_, err = relay.RelayOne(ctx, func(c context.Context, e events.Event) error {
		event = e
		if err := publish(c, e); err != nil {
			return err
		}
		return outage
	})
	if !errors.Is(err, outage) {
		t.Fatal(err)
	}
	if done, err := relay.RelayOne(ctx, publish); err != nil || !done {
		t.Fatal("relay retry", done, err)
	}
	consumer, err := events.Consumer(strings.Split(brokers, ","), id)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = events.ConsumeOne(ctx, consumer, audit); err != nil {
			consumer.Close()
			t.Fatal("consume", err)
		}
	}
	consumer.Close()
	if err = root.Pool.QueryRow(ctx, `SELECT count(*) FROM portway_audit.entries WHERE installation_id=$1`, id).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate effects", count, err)
	}
	if err = root.Pool.QueryRow(ctx, `SELECT count(*) FROM portway_audit.inbox WHERE installation_id=$1`, id).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate receipts", count, err)
	}
	if err = audit.Apply(ctx, event); err != nil {
		t.Fatal("DB commit before offset restart replay", err)
	}
	corrupted := event
	corrupted.Payload.Action = "agent.version_created"
	corrupted.Payload.ObjectType = "agent_version"
	if err = audit.Apply(ctx, corrupted); err == nil {
		t.Fatal("conflicting duplicate accepted")
	}
	// Capability checks use the same outbox contract. Verify their new action
	// codes traverse the actual relay/broker/consumer rather than blocking it.
	tx, err = app.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	provider := secure.ID()
	actions := []string{"provider.check_started", "provider.check_completed", "codex.queued", "codex.stopped", "codex.completed", "codex.failed", "codex.command_queued"}
	for _, action := range actions {
		kind := "provider"
		if strings.HasPrefix(action, "codex.") {
			kind = "codex_run"
		}
		if action == "codex.command_queued" {
			kind = "codex_command"
		}
		if err = app.Audit(ctx, tx, actor, action, kind, provider); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(actions); i++ {
		if done, err := relay.RelayOne(ctx, publish); err != nil || !done {
			t.Fatal("provider check audit relay", done, err)
		}
	}
	// Actual consumer restart: committed offsets mean no previously handled records.
	consumer, err = events.Consumer(strings.Split(brokers, ","), id)
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	for i := 0; i < len(actions); i++ {
		if err = events.ConsumeOne(ctx, consumer, audit); err != nil {
			t.Fatal("provider check audit consume", err)
		}
	}
	if err = root.Pool.QueryRow(ctx, `SELECT count(*) FROM portway_audit.entries WHERE installation_id=$1 AND action IN ('provider.check_started','provider.check_completed') AND object_type='provider' AND object_id=$2`, id, provider).Scan(&count); err != nil || count != 2 {
		t.Fatal("provider check audit not projected", count, err)
	}
	if err = root.Pool.QueryRow(ctx, `SELECT count(*) FROM portway_audit.entries WHERE installation_id=$1 AND action LIKE 'codex.%' AND object_type='codex_run' AND object_id=$2`, id, provider).Scan(&count); err != nil || count != 4 {
		t.Fatal("codex audit not projected", count, err)
	}
	if err = root.Pool.QueryRow(ctx, `SELECT count(*) FROM portway_audit.entries WHERE installation_id=$1 AND action='codex.command_queued' AND object_type='codex_command' AND object_id=$2`, id, provider).Scan(&count); err != nil || count != 1 {
		t.Fatal("codex command audit not projected", count, err)
	}
	idle, done := context.WithTimeout(ctx, 5*time.Second)
	// A group rebalance can yield an empty poll before the deadline. Inspect
	// raw records so an empty poll is not mistaken for a replay (and receipt
	// deduplication cannot hide a genuinely replayed committed record).
	for idle.Err() == nil {
		fetch := consumer.PollRecords(idle, 1)
		consumer.AllowRebalance()
		if fetch.NumRecords() != 0 {
			t.Fatal("consumer replayed committed offset")
		}
		if len(fetch.Errors()) != 0 && idle.Err() == nil {
			t.Fatal("Kafka idle poll failed")
		}
	}
	done()
	if err = relay.Prune(ctx, "relay"); err != nil {
		t.Fatal(err)
	}
	if err = audit.Prune(ctx, "audit"); err != nil {
		t.Fatal(err)
	}
	// Age limit closes normal admission but must not prevent device revocation.
	if _, err = root.Pool.Exec(ctx, `UPDATE portway.outbox SET published_at=NULL,occurred_at=now()-interval '25 hours' WHERE installation_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	tx, err = app.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = app.Audit(ctx, tx, actor, "agent.created", "agent", secure.ID())
	_ = tx.Rollback(ctx)
	if err == nil {
		t.Fatal("expired backlog admitted new configuration")
	}
	tx, err = app.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = app.Audit(ctx, tx, actor, "device.revoked", "device", secure.ID()); err != nil {
		t.Fatal("backlog prevented revocation", err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
