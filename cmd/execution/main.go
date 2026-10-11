package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"portway/internal/config"
	"portway/internal/events"
	"portway/internal/execution"
	"portway/internal/secure"
	"portway/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) > 1 && os.Args[1] == "probe" {
		c := http.Client{Timeout: 4 * time.Second}
		r, e := c.Get("http://127.0.0.1:8082/readyz")
		if e != nil {
			return errors.New("execution probe failed")
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			return errors.New("execution not ready")
		}
		return nil
	}
	id := os.Getenv("PORTWAY_INSTALLATION_ID")
	if !secure.UUID(id) {
		return errors.New("invalid installation")
	}
	dsn, err := config.Secret("PORTWAY_DATABASE_URL", true)
	if err != nil {
		return err
	}
	u, e := url.Parse(dsn)
	if e != nil || u.Host == "" || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Query().Get("sslmode") != "verify-full" || u.Port() == "6543" {
		return errors.New("execution requires verified TLS and a direct/session-pooler connection")
	}
	key, err := config.Secret("PORTWAY_EXECUTION_SERVICE_KEY", true)
	if err != nil || len(key) < 32 {
		return errors.New("execution service key must contain at least 32 bytes")
	}
	control := os.Getenv("PORTWAY_CONTROL_URL")
	if control != "http://api:8080" {
		return errors.New("control URL must be the private Compose API service")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := storage.OpenWithLimit(ctx, dsn, id, 2)
	if err != nil {
		return err
	}
	defer db.Close()
	store := execution.Store{Pool: db.Pool, Installation: id}
	start, cancel := context.WithTimeout(ctx, 15*time.Second)
	err = store.Check(start)
	cancel()
	if err != nil {
		return errors.New("execution schema or role check failed")
	}
	// A held session lock prevents two processes from reconciling one installation.
	// The dedicated connection leaves one pool slot for bounded commands/relay.
	leaderCtx, leaderCancel := context.WithTimeout(ctx, 10*time.Second)
	leader, err := db.Pool.Acquire(leaderCtx)
	if err != nil {
		leaderCancel()
		return errors.New("execution leadership unavailable")
	}
	defer leader.Release()
	var locked bool
	err = leader.QueryRow(leaderCtx, `SELECT pg_try_advisory_lock(hashtextextended($1,76243))`, id).Scan(&locked)
	leaderCancel()
	if err != nil || !locked {
		return errors.New("another execution process owns this installation")
	}
	// Closing, not merely releasing, drops the session-scoped lock.
	defer func() {
		c, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		_ = leader.Conn().Close(c)
	}()
	reconcileCtx, reconcileCancel := context.WithTimeout(ctx, 15*time.Second)
	err = store.Reconcile(reconcileCtx)
	reconcileCancel()
	if err != nil {
		return errors.New("execution reconciliation failed")
	}
	broker, err := events.Producer(strings.Split(os.Getenv("PORTWAY_KAFKA_BROKERS"), ","))
	if err != nil {
		return errors.New("Kafka configuration failed")
	}
	defer broker.Close()
	var ready atomic.Bool
	handler := execution.HTTPServer{Store: store, ServiceKey: key, Authorize: execution.ControlAuthorizer(control, key, &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}), Ready: func(c context.Context) bool {
		bounded, end := context.WithTimeout(c, 3*time.Second)
		defer end()
		return ready.Load() && db.Pool.Ping(bounded) == nil && broker.Ping(bounded) == nil
	}}
	srv := &http.Server{Addr: ":8082", Handler: handler.Handler(), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 12 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 32 << 10}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			stop()
		}
	}()
	defer func() {
		c, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()
		_ = srv.Shutdown(c)
	}()
	slog.Info("execution service started", "run_admission", false)
	failed := false
	for ctx.Err() == nil {
		op, end := context.WithTimeout(ctx, 10*time.Second)
		if err := leader.Conn().Ping(op); err != nil {
			end()
			return errors.New("execution leadership connection lost")
		}
		progressed, err := store.RelayOne(op, execution.Publish(broker))
		end()
		ready.Store(err == nil)
		if err != nil && !failed {
			slog.Error("execution event relay blocked; records retained")
		}
		failed = err != nil
		if !progressed {
			timer := time.NewTimer(2 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
			case <-timer.C:
			}
		}
	}
	return nil
}
