package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"portway/internal/agents"
	"portway/internal/events"
	"portway/internal/execution"
	"portway/internal/secure"
	"portway/internal/storage"
)

func TestExecutionTransactionsAndKafka(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
	worker, err := storage.Open(ctx, workerDSN, installation)
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	store := execution.Store{Pool: worker.Pool, Installation: installation}
	if err = store.Check(ctx); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{`SELECT * FROM portway.secrets`, `SELECT * FROM portway.device_sessions`, `SELECT * FROM portway_audit.entries`} {
		if _, err = worker.Pool.Exec(ctx, sql); err == nil {
			t.Fatal("execution read another service schema")
		}
	}
	app, err := storage.Open(ctx, os.Getenv("TEST_APP_DATABASE_URL"), installation)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if _, err = app.Pool.Exec(ctx, `SELECT * FROM portway_execution.chains`); err == nil {
		t.Fatal("control read execution state directly")
	}
	provider := secure.ID()
	cfg, err := agents.Validate(agents.Config{Instructions: "sensitive-instruction-marker", RoleCode: "coder", RoleCatalogVersion: 1, ProviderRef: provider})
	if err != nil {
		t.Fatal(err)
	}
	input := execution.CreateInput{Task: "sensitive-task-marker", Snapshot: execution.Snapshot{AgentVersionID: secure.ID(), ProviderID: provider, Config: cfg, ConfigHash: agents.Hash(cfg), RepositoryRef: "fixture", Revision: strings.Repeat("f", 40)}}
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := store.Create(ctx, owner, "concurrent-create", input)
			if err != nil {
				t.Error(err)
				return
			}
			ids <- c.ID
		}()
	}
	wg.Wait()
	close(ids)
	chainID := ""
	for id := range ids {
		if chainID != "" && id != chainID {
			t.Fatal("duplicate create")
		}
		chainID = id
	}
	list, err := store.List(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	listBody, _ := json.Marshal(list)
	if len(list) != 1 || strings.Contains(string(listBody), "sensitive-") {
		t.Fatal("collection leaked full pinned task/config")
	}
	if chainID == "" {
		t.Fatal("no chain created")
	}
	c, err := store.Lookup(ctx, owner, chainID, false)
	if err != nil {
		t.Fatal(err)
	}
	runID := c.Runs[0].ID
	if _, err = store.Lookup(ctx, secure.ID(), chainID, false); err == nil {
		t.Fatal("foreign owner read")
	}
	changed := input
	changed.Task = "different task"
	if _, err = store.Create(ctx, owner, "concurrent-create", changed); err == nil {
		t.Fatal("key conflict accepted")
	}
	// Callback error rolls the mutation and its event back together.
	_, err = store.Change(ctx, owner, chainID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
		_, _ = c.Start(runID, now)
		return nil, errors.New("rollback fixture")
	})
	if err == nil {
		t.Fatal("expected rollback")
	}
	c, err = store.Lookup(ctx, owner, chainID, false)
	if err != nil || c.Runs[0].State != "queued" || c.Runs[0].Seq != 1 {
		t.Fatal("rollback leaked", err)
	}
	_, err = store.Change(ctx, owner, chainID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
		e, err := c.Start(runID, now)
		return []execution.Event{e}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	var admitted atomic.Int32
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Change(ctx, owner, chainID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
				_, e, err := c.Reserve(runID, secure.ID(), 100, 1000, now)
				return []execution.Event{e}, err
			})
			if err == nil {
				admitted.Add(1)
			}
		}()
	}
	wg.Wait()
	if admitted.Load() != 1 {
		t.Fatal("concurrent reservation admitted twice")
	}
	c, err = store.Stop(ctx, owner, runID, "stop-key-001", true)
	if err != nil || c.Runs[0].State != "stopping" {
		t.Fatal("stop", err)
	}
	epoch := c.Runs[0].StopEpoch
	c, err = store.Stop(ctx, owner, runID, "stop-key-001", true)
	if err != nil || c.Runs[0].StopEpoch != epoch {
		t.Fatal("repeat stop fence", err)
	}
	if err = store.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	c, err = store.Lookup(ctx, owner, chainID, false)
	if err != nil || c.State != "failed" || c.Budget.Spent != 1100 || c.Budget.Reserved != 0 {
		t.Fatal("restart lost reservations", err)
	}
	if err = c.Validate(); err != nil {
		t.Fatal(err)
	}
	c, err = store.Stop(ctx, owner, runID, "stop-key-001", true)
	if err != nil || c.Runs[0].State != "failed" {
		t.Fatal("stop replay returns stale state", err)
	}
	history, err := store.History(ctx, owner, runID, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i, e := range history {
		if e.Seq != int64(i+1) {
			t.Fatal("event sequence gap")
		}
		b, _ := json.Marshal(e)
		if strings.Contains(string(b), "sensitive-") {
			t.Fatal("private data leaked into event")
		}
	}
	if int64(len(history)) != c.Runs[0].Seq {
		t.Fatal("snapshot high-water mismatch")
	}
	if _, err = store.History(ctx, owner, runID, c.Runs[0].Seq+1); err == nil {
		t.Fatal("future cursor accepted")
	}
	second, err := store.Create(ctx, owner, "second-create", input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Create(ctx, owner, "third-create", input); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Create(ctx, owner, "fourth-create", input); err == nil {
		t.Fatal("active chain capacity exceeded")
	}
	if second.ID == chainID {
		t.Fatal("new chain reused terminal identity")
	}
	brokers := strings.Split(os.Getenv("TEST_KAFKA_BROKERS"), ",")
	producer, err := events.Producer(brokers)
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	publish := execution.Publish(producer)
	var eventID string
	_, err = store.RelayOne(ctx, func(ctx context.Context, e execution.Envelope) error {
		eventID = e.EventID
		if err := publish(ctx, e); err != nil {
			return err
		}
		return errors.New("crash after Kafka ACK")
	})
	if err == nil {
		t.Fatal("expected crash fixture")
	}
	if ok, err := store.RelayOne(ctx, func(ctx context.Context, e execution.Envelope) error {
		if e.EventID != eventID {
			t.Fatal("retry changed event identity")
		}
		return publish(ctx, e)
	}); err != nil || !ok {
		t.Fatal("relay retry", err)
	}
	consumer, err := kgo.NewClient(kgo.SeedBrokers(brokers...), kgo.ConsumeTopics(execution.Topic), kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	seen := 0
	for seen < 2 && ctx.Err() == nil {
		fetch := consumer.PollRecords(ctx, 100)
		if len(fetch.Errors()) != 0 {
			t.Fatal("execution Kafka fetch failed")
		}
		for _, record := range fetch.Records() {
			var e execution.Envelope
			if err = json.Unmarshal(record.Value, &e); err != nil {
				t.Fatal(err)
			}
			if e.InstallationID != installation {
				continue
			}
			if err = e.Validate(time.Now()); err != nil {
				t.Fatal(err)
			}
			if string(record.Key) != installation+"/"+e.CorrelationID {
				t.Fatal("chain partition key mismatch")
			}
			if e.EventID == eventID {
				seen++
			}
		}
	}
	if seen != 2 {
		t.Fatal("expected duplicate publication after ACK crash")
	}
}
