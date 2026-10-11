package execution

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const CleanupBatchSize = 200

type CleanupReport struct {
	RunID         string
	PrunedThrough int64
	EventsDeleted int64
	KeysDeleted   int64
	LogsDeleted   int64
}

func (r CleanupReport) Changed() bool { return r.EventsDeleted+r.KeysDeleted+r.LogsDeleted > 0 }

// Cleanup removes at most one contiguous prefix (200 events), 200 expired keys
// and 200 old maintenance records. It never removes a pending outbox record or
// skips a protected prefix. Relay ACKs and history deletion are independent.
func (s Store) Cleanup(ctx context.Context) (CleanupReport, error) {
	report := CleanupReport{}
	tx, now, err := s.begin(ctx)
	if err != nil {
		return report, err
	}
	defer tx.Rollback(context.Background())
	cutoff := now.Add(-7 * 24 * time.Hour)
	var run string
	var through int64
	err = tx.QueryRow(ctx, `SELECT r.id,r.events_pruned_through FROM portway_execution.runs r
	 JOIN portway_execution.chains c ON c.installation_id=r.installation_id AND c.id=r.chain_id
	 JOIN portway_execution.events e ON e.installation_id=r.installation_id AND e.run_id=r.id AND e.seq=r.events_pruned_through+1
	 JOIN portway_execution.outbox o ON o.installation_id=e.installation_id AND o.id=e.id
	 WHERE r.installation_id=$1 AND c.state IN ('completed','failed','cancelled')
	 AND e.created_at<=$2 AND o.published_at<=$2 ORDER BY e.created_at,r.id LIMIT 1 FOR UPDATE OF r`, s.Installation, cutoff).Scan(&run, &through)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return report, err
	}
	if err == nil {
		rows, err := tx.Query(ctx, `SELECT e.seq,e.created_at,o.published_at FROM portway_execution.events e
		 LEFT JOIN portway_execution.outbox o ON o.installation_id=e.installation_id AND o.id=e.id
		 WHERE e.installation_id=$1 AND e.run_id=$2 AND e.seq>$3 ORDER BY e.seq LIMIT 200`, s.Installation, run, through)
		if err != nil {
			return report, err
		}
		for rows.Next() {
			var seq int64
			var created time.Time
			var published *time.Time
			if err = rows.Scan(&seq, &created, &published); err != nil {
				rows.Close()
				return report, err
			}
			if seq != through+1 || created.After(cutoff) || published == nil || published.After(cutoff) {
				break
			}
			through = seq
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return report, err
		}
		_, err = tx.Exec(ctx, `DELETE FROM portway_execution.outbox WHERE installation_id=$1 AND id IN
		 (SELECT id FROM portway_execution.events WHERE installation_id=$1 AND run_id=$2 AND seq<=$3)`, s.Installation, run, through)
		if err != nil {
			return report, err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM portway_execution.events WHERE installation_id=$1 AND run_id=$2 AND seq<=$3`, s.Installation, run, through)
		if err != nil {
			return report, err
		}
		report.RunID, report.PrunedThrough, report.EventsDeleted = run, through, tag.RowsAffected()
		if _, err = tx.Exec(ctx, `UPDATE portway_execution.runs SET events_pruned_through=$3 WHERE installation_id=$1 AND id=$2`, s.Installation, run, through); err != nil {
			return report, err
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM portway_execution.request_keys WHERE ctid IN
	 (SELECT ctid FROM portway_execution.request_keys WHERE installation_id=$1 AND expires_at<=$2 ORDER BY expires_at LIMIT 200)`, s.Installation, now)
	if err != nil {
		return report, err
	}
	report.KeysDeleted = tag.RowsAffected()
	tag, err = tx.Exec(ctx, `DELETE FROM portway_execution.maintenance_log WHERE id IN
	 (SELECT id FROM portway_execution.maintenance_log WHERE installation_id=$1 AND created_at<=$2 ORDER BY created_at LIMIT 200)`, s.Installation, now.Add(-30*24*time.Hour))
	if err != nil {
		return report, err
	}
	report.LogsDeleted = tag.RowsAffected()
	if report.Changed() {
		_, err = tx.Exec(ctx, `INSERT INTO portway_execution.maintenance_log(installation_id,run_id,pruned_through,events_deleted,keys_deleted,logs_deleted)
		 VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6)`, s.Installation, report.RunID, report.PrunedThrough, report.EventsDeleted, report.KeysDeleted, report.LogsDeleted)
		if err != nil {
			return report, err
		}
	}
	return report, tx.Commit(ctx)
}
