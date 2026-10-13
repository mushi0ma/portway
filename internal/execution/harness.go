package execution

import (
	"context"
	"time"

	"portway/internal/problem"
	"portway/internal/secure"
)

// Gateway is a trusted adapter, not an LLM-provided implementation. InputTokens
// must count/bound the complete serialized request (including tool schemas and
// conversation framing) without dispatching a model request. Complete performs
// at most one request, honors context cancellation, and never retries silently.
// Production wiring requires verified capabilities and live authority fencing;
// this package does not provide either and is not wired into cmd/execution yet.
type Gateway interface {
	InputTokens(context.Context, Run) (int, error)
	Complete(context.Context, Run, Step) (ModelReply, error)
}
type Harness struct {
	Store   Store
	Gateway Gateway
}

// Advance performs at most one model turn. Paused approvals / dispatched tools
// return their current state, allowing another device to decide through the API.
// External work is always outside Store.Change, after reservation commit.
func (h Harness) Advance(ctx context.Context, owner, runID string) (Chain, error) {
	c, err := h.Store.Lookup(ctx, owner, runID, true)
	if err != nil {
		return c, err
	}
	c, err = h.Store.Change(ctx, owner, c.ID, func(c *Chain, now time.Time) ([]Event, error) {
		r, err := c.Find(runID)
		if err != nil {
			return nil, err
		}
		ev, err := c.ObserveDecisions(runID, now)
		if err != nil {
			return nil, err
		}
		if r.State == "queued" {
			e, err := c.Start(runID, now)
			if err != nil {
				return nil, err
			}
			ev = append(ev, e)
		}
		return ev, nil
	})
	if err != nil {
		return c, err
	}
	r, _ := c.Find(runID)
	if r.State != "running" {
		return c, nil
	}
	if h.Gateway == nil {
		return c, problem.New(503, "GATEWAY_NOT_READY", "No verified model gateway is configured")
	}
	// Count and send the same prompt snapshot. The new reservation/turn metadata
	// is supplied separately and must not silently enlarge the counted request.
	request := *r
	input, err := h.Gateway.InputTokens(ctx, request)
	if err != nil {
		return c, problem.New(503, "TOKEN_COUNT_UNAVAILABLE", "Cannot bound model input usage")
	}
	version, stepID := r.Version, secure.ID()
	c, err = h.Store.Change(ctx, owner, c.ID, func(c *Chain, now time.Time) ([]Event, error) {
		current, err := c.Find(runID)
		if err != nil {
			return nil, err
		}
		if current.Version != version {
			return nil, conflict("RUN_CHANGED")
		}
		if err = c.live(current, now); err != nil {
			if problem.Public(err).Code == "DEADLINE_EXCEEDED" {
				return c.failTurn(current, "deadline_exceeded", now), nil
			}
			return nil, err
		}
		if input < 0 || input > 40000 {
			return c.failTurn(current, "budget_uncertain", now), nil
		}
		ev, err := c.BeginTurn(runID, stepID, input, current.Limits.OutputTokens, now)
		if problem.Public(err).Code == "BUDGET_EXHAUSTED" {
			return c.failTurn(current, "budget_exhausted", now), nil
		}
		return ev, err
	})
	if err != nil {
		return c, err
	}
	r, _ = c.Find(runID)
	if r.State != "waiting_model" {
		return c, nil
	}
	_, step, err := turnFor(r, stepID)
	if err != nil {
		return c, err
	}
	deadline := time.Now().Add(120 * time.Second)
	if r.Deadline.Before(deadline) {
		deadline = r.Deadline
	}
	if c.Deadline.Before(deadline) {
		deadline = c.Deadline
	}
	callCtx, cancel := context.WithDeadline(ctx, deadline)
	done := make(chan struct{})
	// Stop is durable first; the in-flight adapter then receives cancellation.
	// DB failure cancels too. No fresh dispatch is authorized by this polling loop.
	go func(chainID string, epoch int64) {
		defer close(done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-callCtx.Done():
				return
			case <-ticker.C:
				current, err := h.Store.Lookup(callCtx, owner, chainID, false)
				if err != nil {
					cancel()
					return
				}
				run, err := current.Find(runID)
				if err != nil || current.State != "active" || run.State != "waiting_model" || run.StopEpoch != epoch {
					cancel()
					return
				}
			}
		}
	}(c.ID, r.StopEpoch)
	reply, callErr := h.Gateway.Complete(callCtx, request, *step)
	cancel()
	<-done
	if callErr != nil {
		reply = ModelReply{Finish: "error", UsageSource: "unknown"}
	}
	// A cancelled request still owes a durable settlement. If DB is unavailable,
	// leave the reservation intact for conservative startup reconciliation.
	settleCtx, settleCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer settleCancel()
	return h.Store.Change(settleCtx, owner, c.ID, func(c *Chain, now time.Time) ([]Event, error) {
		return c.AcceptReply(runID, stepID, reply, now)
	})
}
