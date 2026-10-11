package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"portway/internal/config"
	"portway/internal/events"
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
	if len(os.Args) != 2 {
		return errors.New("usage: events [relay|audit|probe]")
	}
	mode := os.Args[1]
	if mode == "probe" {
		c := http.Client{Timeout: 4 * time.Second}
		r, e := c.Get("http://127.0.0.1:8081/readyz")
		if e != nil {
			return errors.New("worker probe failed")
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			return errors.New("worker not ready")
		}
		return nil
	}
	if mode != "relay" && mode != "audit" {
		return errors.New("unknown worker mode")
	}
	id := os.Getenv("PORTWAY_INSTALLATION_ID")
	if !secure.UUID(id) {
		return errors.New("invalid installation ID")
	}
	dsn, err := config.Secret("PORTWAY_DATABASE_URL", true)
	if err != nil {
		return err
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Host == "" || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Query().Get("sslmode") != "verify-full" {
		return errors.New("worker database requires PostgreSQL URL and verified TLS")
	}
	brokers := strings.Split(os.Getenv("PORTWAY_KAFKA_BROKERS"), ",")
	for _, b := range brokers {
		if _, _, err = net.SplitHostPort(b); err != nil {
			return errors.New("invalid Kafka broker address")
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := storage.OpenWithLimit(ctx, dsn, id, 1)
	if err != nil {
		return err
	}
	defer db.Close()
	s := events.Store{Pool: db.Pool, Installation: id}
	start, cancel := context.WithTimeout(ctx, 15*time.Second)
	err = s.CheckRole(start, mode)
	cancel()
	if err != nil {
		return errors.New("worker database role/schema check failed")
	}
	var client *kgo.Client
	if mode == "relay" {
		client, err = events.Producer(brokers)
	} else {
		client, err = events.Consumer(brokers, id)
	}
	if err != nil {
		return errors.New("Kafka client configuration failed")
	}
	defer client.Close()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	var ready atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		c, done := context.WithTimeout(r.Context(), 3*time.Second)
		defer done()
		if !ready.Load() || client.Ping(c) != nil || db.Pool.Ping(c) != nil {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	})
	server := &http.Server{Addr: ":8081", Handler: mux, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		if e := server.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			log.Error("worker health server failed")
			stop()
		}
	}()
	defer func() {
		c, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()
		_ = server.Shutdown(c)
	}()
	log.Info("event worker started", "service", mode)
	nextPrune := time.Now()
	failed := false
	for ctx.Err() == nil {
		op, done := context.WithTimeout(ctx, 12*time.Second)
		progressed := false
		if mode == "relay" {
			progressed, err = s.RelayOne(op, events.PublishKafka(client))
		} else {
			err = events.ConsumeOne(op, client, s)
		}
		done()
		if ctx.Err() != nil {
			break
		}
		if mode == "audit" && errors.Is(err, events.ErrIdle) {
			err = nil
		}
		if err != nil {
			ready.Store(false)
			if !failed {
				log.Error("event worker blocked; pending records retained", "service", mode)
			}
			failed = true
			// Recreate the consumer from committed offsets on any processing failure.
			if mode == "audit" {
				return errors.New("audit processing failed; offset not acknowledged")
			}
		} else {
			ready.Store(true)
			if failed {
				log.Info("event worker recovered", "service", mode)
			}
			failed = false
		}
		if time.Now().After(nextPrune) {
			c, end := context.WithTimeout(ctx, 5*time.Second)
			e := s.Prune(c, mode)
			end()
			if e != nil {
				log.Warn("event retention cleanup failed", "service", mode)
			}
			nextPrune = time.Now().Add(time.Hour)
		}
		if mode == "relay" && !progressed {
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
