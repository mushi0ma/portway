package integration

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"portway/internal/agents"
	"portway/internal/execution"
	"portway/internal/problem"
	"portway/internal/secure"
	"portway/internal/storage"
)

func TestExecutionRetentionAndReplay(t *testing.T) {
	dsn, workerDSN := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_EXECUTION_DATABASE_URL")
	if dsn == "" || workerDSN == "" {
		if os.Getenv("PORTWAY_REQUIRE_INTEGRATION") == "1" {
			t.Fatal("execution test database required")
		}
		t.Skip("use make docker-test")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Path != "/portway_test" {
		t.Fatal("disposable database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	installation, owner := secure.ID(), secure.ID()
	root, err := storage.Open(ctx, dsn, installation)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err = root.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	worker, err := storage.OpenWithLimit(ctx, workerDSN, installation, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	s := execution.Store{Pool: worker.Pool, Installation: installation}
	provider := secure.ID()
	cfg, err := agents.Validate(agents.Config{Instructions: "retention secret marker", RoleCode: "coder", RoleCatalogVersion: 1, ProviderRef: provider})
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Create(ctx, owner, "retention-fixture", execution.CreateInput{Task: "fixture", Snapshot: execution.Snapshot{AgentVersionID: secure.ID(), ProviderID: provider, Config: cfg, ConfigHash: agents.Hash(cfg), RepositoryRef: "fixture", Revision: strings.Repeat("a", 40)}})
	if err != nil {
		t.Fatal(err)
	}
	run := c.Runs[0].ID
	// Synthetic persisted lifecycle pages exercise >100 replay and >200 cleanup.
	// No HTTP admission, model calls or runtime dispatch are involved.
	c, err = s.Change(ctx, owner, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
		events := []execution.Event{}
		for i := 0; i < 244; i++ {
			c.Runs[0].Seq++
			events = append(events, execution.Event{SchemaVersion: 1, ID: secure.ID(), RunID: run, CorrelationID: c.ID, Seq: c.Runs[0].Seq, Type: "run.state_changed", ServerTime: now, Payload: json.RawMessage(`{"state":"queued"}`)})
		}
		return events, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	execSQL := func(sql string, args ...any) {
		t.Helper()
		if _, err := root.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	execSQL(`UPDATE portway_execution.events SET created_at=clock_timestamp()-interval '8 days' WHERE installation_id=$1`, installation)
	execSQL(`UPDATE portway_execution.outbox SET published_at=clock_timestamp()-interval '8 days' WHERE installation_id=$1`, installation)
	if report, err := s.Cleanup(ctx); err != nil || report.EventsDeleted != 0 {
		t.Fatal("active chain pruned", report, err)
	}
	if err = s.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	page, err := s.HistoryPage(ctx, owner, run, 0)
	if err != nil || len(page.Items) != 100 || page.SnapshotSeq != 246 {
		t.Fatal("history page", page, err)
	}
	if _, err = s.HistoryPage(ctx, secure.ID(), run, 0); problem.Public(err).Status != 404 {
		t.Fatal("foreign owner history", err)
	}
	if _, err = s.HistoryPage(ctx, owner, run, 247); problem.Public(err).Code != "CURSOR_AHEAD" {
		t.Fatal("ahead cursor", err)
	}
	// A pending middle event protects the entire suffix, including published ones.
	execSQL(`UPDATE portway_execution.outbox SET published_at=NULL WHERE installation_id=$1 AND id IN (SELECT id FROM portway_execution.events WHERE installation_id=$1 AND seq=3)`, installation)
	report, err := s.Cleanup(ctx)
	if err != nil || report.EventsDeleted != 2 || report.PrunedThrough != 2 {
		t.Fatal("prefix pruning", report, err)
	}
	if _, err = s.HistoryPage(ctx, owner, run, 0); problem.Public(err).Code != "HISTORY_EXPIRED" || problem.Public(err).Details == nil {
		t.Fatal("expired cursor not explicit", err)
	}
	page, err = s.HistoryPage(ctx, owner, run, 2)
	if err != nil || page.Items[0].Seq != 3 {
		t.Fatal("watermark boundary", err)
	}
	if report, err = s.Cleanup(ctx); err != nil || report.EventsDeleted != 0 {
		t.Fatal("pending event pruned", report, err)
	}
	// Fresh ACKs retain seven days from publication, not just event creation.
	execSQL(`UPDATE portway_execution.outbox SET published_at=clock_timestamp() WHERE installation_id=$1 AND id IN (SELECT id FROM portway_execution.events WHERE installation_id=$1 AND seq=3)`, installation)
	if report, err = s.Cleanup(ctx); err != nil || report.EventsDeleted != 0 {
		t.Fatal("fresh ACK pruned", report, err)
	}
	execSQL(`UPDATE portway_execution.outbox SET published_at=clock_timestamp()-interval '8 days' WHERE installation_id=$1 AND id IN (SELECT id FROM portway_execution.events WHERE installation_id=$1 AND seq=3)`, installation)
	// An audit failure must roll back both deletion and watermark advancement.
	execSQL(`CREATE FUNCTION portway_execution.reject_cleanup_fixture() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture rollback'; END $$`)
	execSQL(`CREATE TRIGGER reject_cleanup_fixture BEFORE INSERT ON portway_execution.maintenance_log FOR EACH ROW EXECUTE FUNCTION portway_execution.reject_cleanup_fixture()`)
	_, rollbackErr := s.Cleanup(ctx)
	execSQL(`DROP TRIGGER reject_cleanup_fixture ON portway_execution.maintenance_log`)
	execSQL(`DROP FUNCTION portway_execution.reject_cleanup_fixture()`)
	if rollbackErr == nil {
		t.Fatal("expected cleanup rollback")
	}
	page, err = s.HistoryPage(ctx, owner, run, 2)
	if err != nil || page.PrunedThrough != 2 {
		t.Fatal("rollback lost history", err)
	}
	// Concurrent cleanup/replay may return the old consistent page or 410, never gaps.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			p, err := s.HistoryPage(ctx, owner, run, 2)
			if err != nil && problem.Public(err).Code != "HISTORY_EXPIRED" {
				t.Error(err)
				return
			}
			if err == nil && (len(p.Items) != 100 || p.Items[0].Seq != 3) {
				t.Error("inconsistent replay")
			}
		}
	}()
	report, err = s.Cleanup(ctx)
	wg.Wait()
	if err != nil || report.EventsDeleted != 200 || report.PrunedThrough != 202 {
		t.Fatal("batch limit", report, err)
	}
	report, err = s.Cleanup(ctx)
	if err != nil || report.EventsDeleted != 43 || report.PrunedThrough != 245 {
		t.Fatal("recent event protection", report, err)
	}
	page, err = s.HistoryPage(ctx, owner, run, 245)
	if err != nil || len(page.Items) != 1 || page.Items[0].Seq != 246 {
		t.Fatal("retained recent event", err)
	}
	// Expired idempotency keys are bounded; unexpired keys and other installations survive.
	execSQL(`INSERT INTO portway_execution.request_keys(installation_id,owner_id,operation,key_hash,request_hash,response,expires_at)
	 SELECT $1,$2,'fixture',n::text,'hash','{}',clock_timestamp()-interval '1 hour' FROM generate_series(1,250) n`, installation, owner)
	report, err = s.Cleanup(ctx)
	if err != nil || report.KeysDeleted != 200 {
		t.Fatal("key batch limit", report, err)
	}
	report, err = s.Cleanup(ctx)
	if err != nil || report.KeysDeleted != 50 {
		t.Fatal("key batch remainder", report, err)
	}
	var n int
	if err = root.Pool.QueryRow(ctx, `SELECT count(*) FROM portway_execution.request_keys WHERE installation_id=$1`, installation).Scan(&n); err != nil || n != 1 {
		t.Fatal("live key deleted", n, err)
	}
	foreign := execution.Store{Pool: worker.Pool, Installation: secure.ID()}
	if report, err = foreign.Cleanup(ctx); err != nil || report.Changed() {
		t.Fatal("cross-installation cleanup", report, err)
	}
	execSQL(`UPDATE portway_execution.maintenance_log SET created_at=clock_timestamp()-interval '31 days' WHERE installation_id=$1`, installation)
	report, err = s.Cleanup(ctx)
	if err != nil || report.LogsDeleted == 0 {
		t.Fatal("maintenance retention", report, err)
	}
	if err = root.Pool.QueryRow(ctx, `SELECT count(*) FROM portway_execution.maintenance_log WHERE installation_id=$1`, installation).Scan(&n); err != nil || n != 1 {
		t.Fatal("missing durable cleanup audit", n, err)
	}
}
