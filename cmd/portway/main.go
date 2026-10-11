package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"portway/internal/config"
	"portway/internal/storage"
	"portway/internal/transport"
)

var version = "dev"
var revision = "unknown"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	mode := "api"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	if mode == "version" {
		fmt.Printf("portway %s (%s)\n", version, revision)
		return nil
	}
	if mode == "probe" {
		c := http.Client{Timeout: 3 * time.Second}
		r, e := c.Get("http://127.0.0.1:8080/healthz")
		if e != nil {
			return errors.New("health probe failed")
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			return errors.New("health probe failed")
		}
		return nil
	}
	if mode != "api" && mode != "migrate" && mode != "check" {
		return errors.New("usage: portway [api|migrate|check|probe|version]")
	}
	c, err := config.Load()
	if err != nil {
		return err
	}
	if mode == "check" {
		fmt.Println("configuration valid (secret values omitted)")
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	dsn := c.DatabaseURL
	poolLimit := c.DatabaseMaxConns
	if mode == "migrate" {
		if c.MigrationURL == "" {
			return errors.New("migration requires PORTWAY_MIGRATION_DATABASE_URL or its _FILE variant")
		}
		dsn = c.MigrationURL
		poolLimit = 1
	}
	store, err := storage.OpenWithLimit(startup, dsn, c.InstallationID, poolLimit)
	if err != nil {
		return err
	}
	defer store.Close()
	if mode == "migrate" {
		if err = store.Migrate(startup); err != nil {
			return errors.New("migration failed; check schema version, checksums and migration role permissions")
		}
		fmt.Println("migrations applied")
		return nil
	}
	if err = store.Ready(startup); err != nil {
		return errors.New("database schema unavailable or incompatible; run migrations first")
	}
	if err = store.CheckApplicationRole(startup); err != nil {
		return errors.New("API requires a separate unprivileged database role")
	}
	if err = store.EnsureInstallation(startup, c.InstallationName, c.SetupHash); err != nil {
		return errors.New("installation initialization failed")
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	api := transport.New(c, store, log)
	httpServer := &http.Server{Addr: c.HTTPAddr, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 25 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	done := make(chan error, 1)
	go func() { done <- httpServer.ListenAndServe() }()
	log.Info("API started", "stage", "foundation", "version", version, "revision", revision)
	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.New("HTTP server failed")
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = httpServer.Shutdown(shutdown); err != nil {
			_ = httpServer.Close()
			return errors.New("HTTP shutdown timed out")
		}
	}
	return nil
}
