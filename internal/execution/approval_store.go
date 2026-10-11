package execution

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/problem"
	"portway/internal/secure"
)

func (s Store) approvalChain(ctx context.Context, owner, id string) (string, error) {
	var chain string
	err := s.Pool.QueryRow(ctx, `SELECT r.chain_id FROM portway_execution.approvals a
	 JOIN portway_execution.runs r ON r.installation_id=a.installation_id AND r.id=a.run_id
	 JOIN portway_execution.chains c ON c.installation_id=r.installation_id AND c.id=r.chain_id
	 WHERE a.installation_id=$1 AND c.owner_id=$2 AND a.id=$3`, s.Installation, owner, id).Scan(&chain)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", problem.NotFound()
	}
	return chain, err
}

func (s Store) GetApproval(ctx context.Context, owner, id string) (Approval, error) {
	chain, err := s.approvalChain(ctx, owner, id)
	if err != nil {
		return Approval{}, err
	}
	c, err := s.Change(ctx, owner, chain, func(c *Chain, now time.Time) ([]Event, error) { return c.ExpireApprovals(now), nil })
	if err != nil {
		return Approval{}, err
	}
	_, a, err := c.approval(id)
	if err != nil {
		return Approval{}, err
	}
	return *a, nil
}

func (s Store) ListApprovals(ctx context.Context, owner string) ([]Approval, error) {
	rows, err := s.Pool.Query(ctx, `SELECT a.id FROM portway_execution.approvals a
	 JOIN portway_execution.runs r ON r.installation_id=a.installation_id AND r.id=a.run_id
	 JOIN portway_execution.chains c ON c.installation_id=r.installation_id AND c.id=r.chain_id
	 WHERE a.installation_id=$1 AND c.owner_id=$2 AND a.state='pending' ORDER BY a.expires_at,a.id LIMIT 100`, s.Installation, owner)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []Approval{}
	for _, id := range ids {
		a, err := s.GetApproval(ctx, owner, id)
		if err != nil {
			return nil, err
		}
		if a.State == "pending" {
			out = append(out, a)
		}
	}
	return out, nil
}

func (s Store) DecideApproval(ctx context.Context, owner, actor, id, key string, d Decision) (Approval, error) {
	if err := d.Validate(); err != nil {
		return Approval{}, err
	}
	chain, err := s.approvalChain(ctx, owner, id)
	if err != nil {
		return Approval{}, err
	}
	tx, now, err := s.begin(ctx)
	if err != nil {
		return Approval{}, err
	}
	defer tx.Rollback(context.Background())
	b, _ := json.Marshal(d)
	op, hash := "approval-decision/"+id, secure.Hash(string(b))
	if _, err = s.replay(ctx, tx, owner, op, key, hash); err != nil {
		return Approval{}, err
	}
	c, err := s.lock(ctx, tx, owner, chain)
	if err != nil {
		return Approval{}, err
	}
	a, events, decisionErr := c.ResolveApproval(id, actor, d, now)
	if decisionErr != nil && len(events) == 0 {
		return a, decisionErr
	}
	if err = s.persist(ctx, tx, c, events); err != nil {
		return Approval{}, err
	}
	if decisionErr == nil {
		// Same key may be replayed after stop; return current tool outcome rather
		// than the cached ready state, and preserve the original decision actor.
		_, err = tx.Exec(ctx, `DELETE FROM portway_execution.request_keys WHERE installation_id=$1 AND owner_id=$2 AND operation=$3 AND key_hash=$4`, s.Installation, owner, op, secure.Hash(key))
		if err != nil {
			return Approval{}, err
		}
		if err = s.remember(ctx, tx, owner, op, key, hash, c); err != nil {
			return Approval{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Approval{}, err
	}
	return a, decisionErr
}

// ExpirePending bounds work and uses DB time inside each locked aggregate.
func (s Store) ExpirePending(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT c.owner_id,c.id FROM portway_execution.approvals a
	 JOIN portway_execution.runs r ON r.installation_id=a.installation_id AND r.id=a.run_id
	 JOIN portway_execution.chains c ON c.installation_id=r.installation_id AND c.id=r.chain_id
	 WHERE a.installation_id=$1 AND a.state='pending' AND a.expires_at<=clock_timestamp() LIMIT 100`, s.Installation)
	if err != nil {
		return err
	}
	type item struct{ owner, id string }
	var items []item
	for rows.Next() {
		var i item
		if err = rows.Scan(&i.owner, &i.id); err != nil {
			rows.Close()
			return err
		}
		items = append(items, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, i := range items {
		if _, err = s.Change(ctx, i.owner, i.id, func(c *Chain, now time.Time) ([]Event, error) { return c.ExpireApprovals(now), nil }); err != nil {
			return err
		}
	}
	return nil
}
