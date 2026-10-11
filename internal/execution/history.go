package execution

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"portway/internal/problem"
	"portway/internal/secure"
)

const HistoryPageSize = 100

func Cursor(run string, seq int64) string { return run + ":" + strconv.FormatInt(seq, 10) }

// ParseCursor accepts only a canonical cursor for the requested run.
func ParseCursor(run, cursor string) (int64, error) {
	parts := strings.Split(cursor, ":")
	if len(parts) == 2 && secure.UUID(run) && parts[0] == run {
		n, err := strconv.ParseInt(parts[1], 10, 64)
		if err == nil && n >= 0 && Cursor(run, n) == cursor {
			return n, nil
		}
	}
	return 0, problem.New(400, "INVALID_CURSOR", "Cursor must identify this run and a nonnegative sequence")
}

type HistoryPage struct {
	Items         []Event `json:"items"`
	SnapshotSeq   int64   `json:"snapshot_seq"`
	PrunedThrough int64   `json:"events_pruned_through"`
}

type RunSnapshot struct {
	Run
	SnapshotCursor string `json:"snapshot_cursor"`
}

func (r Run) View() RunSnapshot { return RunSnapshot{Run: r, SnapshotCursor: Cursor(r.ID, r.Seq)} }

// HistoryPage uses one MVCC snapshot for authorization, high-water, watermark
// and events. Concurrent cleanup cannot silently remove a replay suffix.
func (s Store) HistoryPage(ctx context.Context, owner, run string, after int64) (HistoryPage, error) {
	p := HistoryPage{Items: []Event{}}
	if after < 0 {
		return p, problem.New(400, "INVALID_CURSOR", "Invalid event sequence")
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return p, err
	}
	defer tx.Rollback(context.Background())
	var b []byte
	err = tx.QueryRow(ctx, `SELECT c.body,r.events_pruned_through FROM portway_execution.runs r
	 JOIN portway_execution.chains c ON c.installation_id=r.installation_id AND c.id=r.chain_id
	 WHERE r.installation_id=$1 AND c.owner_id=$2 AND r.id=$3`, s.Installation, owner, run).Scan(&b, &p.PrunedThrough)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, problem.NotFound()
	}
	if err != nil {
		return p, err
	}
	var c Chain
	if err = json.Unmarshal(b, &c); err != nil {
		return p, err
	}
	r, err := c.Find(run)
	if err != nil {
		return p, err
	}
	p.SnapshotSeq = r.Seq
	if after > r.Seq {
		return p, conflict("CURSOR_AHEAD")
	}
	if after < p.PrunedThrough {
		err := problem.New(410, "HISTORY_EXPIRED", "Read the current run snapshot before reconnecting")
		err.Details = map[string]any{"snapshot_url": "/v1/runs/" + run, "snapshot_cursor": Cursor(run, r.Seq), "events_pruned_through": p.PrunedThrough}
		return p, err
	}
	rows, err := tx.Query(ctx, `SELECT body FROM portway_execution.events WHERE installation_id=$1 AND run_id=$2 AND seq>$3 AND seq<=$4 ORDER BY seq LIMIT 100`, s.Installation, run, after, r.Seq)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var e Event
		if err = rows.Scan(&b); err != nil {
			return p, err
		}
		if err = json.Unmarshal(b, &e); err != nil {
			return p, err
		}
		if e.RunID != run || e.Seq != after+int64(len(p.Items))+1 {
			return p, errors.New("execution history is not contiguous")
		}
		p.Items = append(p.Items, e)
	}
	if err = rows.Err(); err != nil {
		return p, err
	}
	if len(p.Items) < HistoryPageSize && after+int64(len(p.Items)) != r.Seq {
		return p, errors.New("execution history is incomplete")
	}
	return p, tx.Commit(ctx)
}
