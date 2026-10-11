package events

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

var ErrIdle = errors.New("no Kafka records before poll deadline")

func Producer(brokers []string) (*kgo.Client, error) {
	return kgo.NewClient(kgo.SeedBrokers(brokers...), kgo.ClientID("portway-outbox"),
		// Small audit records do not need compression. This also avoids loading
		// Snappy native libraries into the broker's noexec temporary filesystem.
		kgo.ProducerBatchCompression(kgo.NoCompression()),
		kgo.RequiredAcks(kgo.AllISRAcks()), kgo.RecordDeliveryTimeout(8*time.Second),
		kgo.ProducerBatchMaxBytes(8192), kgo.MaxBufferedRecords(32))
}

func Consumer(brokers []string, installation string) (*kgo.Client, error) {
	return kgo.NewClient(kgo.SeedBrokers(brokers...), kgo.ClientID("portway-audit"),
		kgo.ConsumerGroup("portway-audit-v1-"+installation), kgo.ConsumeTopics(Topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()), kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(), kgo.RebalanceTimeout(30*time.Second),
		kgo.FetchMaxBytes(1<<20), kgo.FetchMaxPartitionBytes(1<<20))
}

func PublishKafka(client *kgo.Client) Publish {
	return func(ctx context.Context, e Event) error {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		return client.ProduceSync(ctx, &kgo.Record{Topic: Topic, Key: []byte(e.Key()), Value: b}).FirstErr()
	}
}

// ConsumeOne commits only after the local receipt + projection transaction.
// Caller must stop/recreate the consumer on any error: polling again could skip
// a failed record because fetch position advances before offset commit.
func ConsumeOne(ctx context.Context, client *kgo.Client, s Store) error {
	fetch := client.PollRecords(ctx, 1)
	defer client.AllowRebalance()
	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) && fetch.NumRecords() == 0 {
			return ErrIdle
		}
		return ctx.Err()
	}
	if len(fetch.Errors()) != 0 {
		return errors.New("Kafka fetch failed")
	}
	for _, r := range fetch.Records() {
		e, err := Decode(r.Value, s.Installation, time.Now())
		if err != nil {
			return err
		}
		if string(r.Key) != e.Key() {
			return errors.New("event partition key mismatch")
		}
		if err = s.Apply(ctx, e); err != nil {
			return err
		}
		if err = client.CommitRecords(ctx, r); err != nil {
			return errors.New("Kafka offset commit failed")
		}
	}
	return nil
}
