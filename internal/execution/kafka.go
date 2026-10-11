package execution

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/twmb/franz-go/pkg/kgo"
	"portway/internal/secure"
)

const Topic = "portway.execution.events.v1"

type Envelope struct {
	SchemaVersion  int       `json:"schema_version"`
	EventID        string    `json:"event_id"`
	InstallationID string    `json:"installation_id"`
	Producer       string    `json:"producer"`
	Type           string    `json:"type"`
	AggregateID    string    `json:"aggregate_id"`
	CorrelationID  string    `json:"correlation_id"`
	OccurredAt     time.Time `json:"occurred_at"`
	Payload        Event     `json:"payload"`
}

func (e Envelope) Validate(now time.Time) error {
	if e.SchemaVersion != 1 || e.Producer != "execution" || !secure.UUID(e.EventID) || !secure.UUID(e.InstallationID) || !secure.UUID(e.AggregateID) || !secure.UUID(e.CorrelationID) || e.Payload.ID != e.EventID || e.Payload.RunID != e.AggregateID || e.Payload.CorrelationID != e.CorrelationID || e.Payload.Type != e.Type || e.Payload.SchemaVersion != 1 || e.Payload.Seq < 1 || !e.Payload.ServerTime.Equal(e.OccurredAt) || (e.Type != "run.created" && e.Type != "run.state_changed") || e.OccurredAt.Before(now.Add(-24*time.Hour)) || e.OccurredAt.After(now.Add(time.Minute)) {
		return errors.New("invalid execution event")
	}
	return nil
}
func (s Store) RelayOne(ctx context.Context, publish func(context.Context, Envelope) error) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.Background())
	var locked bool
	err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1,76242))`, s.Installation).Scan(&locked)
	if err != nil || !locked {
		return false, err
	}
	var id string
	var b []byte
	err = tx.QueryRow(ctx, `SELECT id,body FROM portway_execution.outbox WHERE installation_id=$1 AND published_at IS NULL ORDER BY sequence LIMIT 1 FOR UPDATE`, s.Installation).Scan(&id, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var e Envelope
	if err = json.Unmarshal(b, &e); err != nil {
		return false, err
	}
	if err = e.Validate(time.Now()); err != nil {
		return false, err
	}
	if e.InstallationID != s.Installation || e.EventID != id {
		return false, errors.New("outbox identity mismatch")
	}
	if err = publish(ctx, e); err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, `UPDATE portway_execution.outbox SET published_at=clock_timestamp() WHERE installation_id=$1 AND id=$2`, s.Installation, id)
	if err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}
func Publish(client *kgo.Client) func(context.Context, Envelope) error {
	return func(ctx context.Context, e Envelope) error {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		return client.ProduceSync(ctx, &kgo.Record{Topic: Topic, Key: []byte(e.InstallationID + "/" + e.CorrelationID), Value: b}).FirstErr()
	}
}
