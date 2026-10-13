package execution

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"portway/internal/problem"
	"portway/internal/secure"
)

// ModelReply is a complete, normalized gateway response, never a partial stream.
// The trusted gateway validates provider framing and reports usage separately
// from model-generated text. A proposal cannot set policy, approval or outcome.
type ModelReply struct {
	Finish      string         `json:"finish"` // final, tool, error
	Text        string         `json:"text,omitempty"`
	Tool        *ShellProposal `json:"tool,omitempty"`
	Usage       int            `json:"usage"`
	UsageSource string         `json:"usage_source"` // actual or unknown
}
type ShellProposal struct {
	Alias   string       `json:"alias"`
	Command ShellCommand `json:"command"`
}
type Observation struct {
	Code         string   `json:"code"`
	ToolCallID   string   `json:"tool_call_id"`
	ArgsHash     string   `json:"denied_args_hash,omitempty"`
	Summary      string   `json:"summary,omitempty"`
	AllowedHints []string `json:"allowed_capability_hints,omitempty"`
}
type Turn struct {
	StepID           string         `json:"step_id"`
	State            string         `json:"state"`
	ReplyHash        string         `json:"reply_hash,omitempty"`
	Proposal         *ShellProposal `json:"proposal,omitempty"`
	ApprovalID       string         `json:"approval_id,omitempty"`
	DispatchID       string         `json:"dispatch_id,omitempty"`
	DispatchDeadline *time.Time     `json:"dispatch_deadline,omitempty"`
	OutcomeHash      string         `json:"outcome_hash,omitempty"`
	Observation      *Observation   `json:"observation,omitempty"`
}

// Result is a durable model answer. Repository artifacts/revisions are supplied
// by a future trusted workspace finalizer, never inferred from model prose.
type Result struct {
	ID             string    `json:"id"`
	RunID          string    `json:"run_id"`
	StepID         string    `json:"step_id"`
	Summary        string    `json:"summary"`
	SourceRevision string    `json:"source_revision"`
	ConfigHash     string    `json:"config_hash"`
	ContentHash    string    `json:"content_hash"`
	CreatedAt      time.Time `json:"created_at"`
}

func (c *Chain) BeginTurn(id, stepID string, input, output int, now time.Time) ([]Event, error) {
	r, err := c.Find(id)
	if err != nil {
		return nil, err
	}
	for _, t := range r.Turns {
		if t.State == "waiting_model" || t.State == "awaiting_approval" || t.State == "waiting_tool" {
			return nil, conflict("TURN_IN_PROGRESS")
		}
	}
	_, e, err := c.Reserve(id, stepID, input, output, now)
	if err != nil {
		return nil, err
	}
	r.Turns = append(r.Turns, Turn{StepID: stepID, State: "waiting_model"})
	return []Event{e}, nil
}
func turnFor(r *Run, id string) (*Turn, *Step, error) {
	var turn *Turn
	var step *Step
	for i := range r.Turns {
		if r.Turns[i].StepID == id {
			turn = &r.Turns[i]
			break
		}
	}
	for i := range r.Steps {
		if r.Steps[i].ID == id {
			step = &r.Steps[i]
			break
		}
	}
	if turn == nil || step == nil {
		return nil, nil, problem.NotFound()
	}
	return turn, step, nil
}
func (c *Chain) failTurn(r *Run, reason string, now time.Time) []Event {
	r.State, r.Reason = "failed", reason
	r.Version++
	events := c.cancelApprovals(r, now)
	c.finish()
	return append(events, c.event(r, "run.state_changed", now))
}
func (c *Chain) denyTurn(r *Run, t *Turn, code, call, hash string, now time.Time) []Event {
	t.State = "done"
	t.Observation = &Observation{Code: code, ToolCallID: call, ArgsHash: hash, AllowedHints: []string{"provide_final_explanation", "propose_different_action"}}
	r.Denials++
	r.Version++
	events := []Event{c.toolEvent(r, t, now)}
	if r.Denials >= 2 {
		events = append(events, c.failTurn(r, "policy_exhausted", now)...)
	}
	return events
}
func (c *Chain) toolEvent(r *Run, t *Turn, now time.Time) Event {
	r.Seq++
	payload, _ := json.Marshal(map[string]any{"tool_call_id": t.Observation.ToolCallID, "step_id": t.StepID, "outcome": t.Observation.Code})
	return Event{1, secure.ID(), r.ID, c.ID, r.Seq, "tool_result", now, payload}
}

// AcceptReply settles usage and accepts a complete reply in the SAME transaction.
// Late replies may settle a reservation but cannot revive stopped/recovered runs.
// Malformed responses produce durable failures, not rolled-back usage charges.
func (c *Chain) AcceptReply(id, stepID string, reply ModelReply, now time.Time) ([]Event, error) {
	r, err := c.Find(id)
	if err != nil {
		return nil, err
	}
	t, step, err := turnFor(r, stepID)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(reply)
	hash := secure.Hash(string(raw))
	if t.ReplyHash != "" {
		if t.ReplyHash == hash {
			return nil, nil
		}
		return nil, conflict("MODEL_REPLY_CONFLICT")
	}
	if t.State != "waiting_model" || step.Settled {
		return nil, conflict("STALE_MODEL_REPLY")
	}
	t.ReplyHash, t.State = hash, "done"
	source, used := reply.UsageSource, reply.Usage
	if source != "actual" || used < 0 || used > step.Reserved {
		source, used = "unknown", 0
	}
	e, err := c.Settle(id, stepID, used, source, now)
	if err != nil {
		return nil, err
	}
	events := []Event{}
	if e != nil {
		events = append(events, *e)
	}
	if source == "unknown" || terminal(r.State) || r.State == "stopping" || step.Epoch != r.StopEpoch {
		return events, nil
	}
	if !now.Before(r.Deadline) || !now.Before(c.Deadline) {
		return append(events, c.failTurn(r, "deadline_exceeded", now)...), nil
	}
	if len(raw) > 65536 || !utf8.ValidString(reply.Text) {
		return append(events, c.failTurn(r, "model_response_invalid", now)...), nil
	}
	switch reply.Finish {
	case "final":
		if reply.Tool != nil || strings.TrimSpace(reply.Text) == "" {
			break
		}
		r.Result = &Result{ID: secure.ID(), RunID: id, StepID: stepID, Summary: reply.Text, SourceRevision: r.Snapshot.Revision, ConfigHash: r.Snapshot.ConfigHash, ContentHash: secure.Hash(reply.Text), CreatedAt: now}
		r.State, r.Reason = "completed", ""
		r.Version++
		c.finish()
		r.Seq++
		payload, _ := json.Marshal(map[string]any{"result_id": r.Result.ID, "source_revision": r.Result.SourceRevision, "config_hash": r.Result.ConfigHash, "content_hash": r.Result.ContentHash})
		events = append(events, Event{1, secure.ID(), id, c.ID, r.Seq, "run.completed", now, payload})
		return append(events, c.event(r, "run.state_changed", now)), nil
	case "tool":
		if reply.Tool == nil || reply.Text != "" {
			break
		}
		// Canonical command validation precedes policy and any approval creation.
		argsHash, err := commandHash(reply.Tool.Command)
		if err != nil {
			break
		}
		proposal := *reply.Tool
		proposal.Command.Argv = append([]string{}, proposal.Command.Argv...)
		t.Proposal = &proposal
		declared := false
		for _, tool := range r.Snapshot.Config.Tools {
			declared = declared || tool.Alias == proposal.Alias && tool.Kind == "shell.exec"
		}
		if !declared {
			return append(events, c.denyTurn(r, t, "POLICY_DENIED", secure.ID(), argsHash, now)...), nil
		}
		a, ev, err := c.RequestShellApproval(id, proposal.Command, now)
		if err != nil {
			code := problem.Public(err).Code
			if code == "POLICY_DENIED" || code == "ACTION_PREVIOUSLY_DENIED" {
				return append(events, c.denyTurn(r, t, code, secure.ID(), argsHash, now)...), nil
			}
			return append(events, c.failTurn(r, "tool_proposal_invalid", now)...), nil
		}
		t.ApprovalID, t.State = a.ID, "awaiting_approval"
		return append(events, ev...), nil
	case "error":
		return append(events, c.failTurn(r, "model_failed", now)...), nil
	}
	return append(events, c.failTurn(r, "model_response_invalid", now)...), nil
}

// ObserveDecisions feeds durable denials back into the next model request.
// Approved actions remain blocked until a trusted Runtime adapter records an
// actual outcome. Repeated ticks cannot count the same rejection twice.
func (c *Chain) ObserveDecisions(id string, now time.Time) ([]Event, error) {
	r, err := c.Find(id)
	if err != nil {
		return nil, err
	}
	if terminal(r.State) || r.State == "stopping" {
		return nil, nil
	}
	if !now.Before(r.Deadline) || !now.Before(c.Deadline) {
		for _, t := range r.Turns {
			// Deadline cancels external work; it does not prove that it ended.
			if t.State == "waiting_tool" {
				return nil, nil
			}
		}
		if r.Budget.Reserved == 0 {
			return c.failTurn(r, "deadline_exceeded", now), nil
		}
		return nil, nil
	}
	events := c.ExpireApprovals(now)
	for i := range r.Turns {
		t := &r.Turns[i]
		if t.State != "awaiting_approval" {
			continue
		}
		_, a, err := c.approval(t.ApprovalID)
		if err != nil {
			return nil, err
		}
		if a.State == "rejected" || a.State == "expired" {
			code := "APPROVAL_REJECTED"
			if a.State == "expired" {
				code = "APPROVAL_EXPIRED"
			}
			events = append(events, c.denyTurn(r, t, code, a.ToolCallID, a.ArgsHash, now)...)
		} else if a.State == "cancelled" {
			return append(events, c.failTurn(r, "approval_invalidated", now)...), nil
		}
	}
	return events, nil
}

// BeginToolDispatch only records an immutable dispatch intent. It is NOT a
// Runtime grant. The future trusted adapter must perform current authority /
// restrictive-policy checks and Manager fencing before executing this intent.
// No production caller currently invokes this method.
func (c *Chain) BeginToolDispatch(id, stepID, dispatchID string, now time.Time) ([]Event, error) {
	r, err := c.Find(id)
	if err != nil {
		return nil, err
	}
	if err = c.live(r, now); err != nil {
		return nil, err
	}
	t, _, err := turnFor(r, stepID)
	if err != nil {
		return nil, err
	}
	if !secure.UUID(dispatchID) || t.DispatchID != "" || t.State != "awaiting_approval" {
		return nil, conflict("DISPATCH_NOT_ALLOWED")
	}
	_, a, err := c.approval(t.ApprovalID)
	if err != nil {
		return nil, err
	}
	if a.State != "approved" || a.ToolState != "ready" || a.StopEpoch != r.StopEpoch || a.ChainEpoch != c.StopEpoch || a.PolicyVersion != r.Snapshot.ConfigHash || r.Snapshot.Config.Policy["shell.exec"] != "approval" || !now.Before(a.ExpiresAt) || r.State != "awaiting_approval" {
		return nil, conflict("DISPATCH_NOT_ALLOWED")
	}
	if r.Budget.Spent+r.Budget.Reserved >= r.Budget.Limit || c.Budget.Spent+c.Budget.Reserved >= c.Budget.Limit {
		return nil, problem.New(422, "BUDGET_EXHAUSTED", "No remaining execution budget")
	}
	deadline := now.Add(time.Duration(a.Args.TimeoutSeconds) * time.Second)
	if r.Deadline.Before(deadline) {
		deadline = r.Deadline
	}
	if c.Deadline.Before(deadline) {
		deadline = c.Deadline
	}
	t.DispatchID, t.State, t.DispatchDeadline = dispatchID, "waiting_tool", &deadline
	a.ToolState = "dispatched"
	a.Version++
	r.Version++
	return []Event{c.approvalEvent(r, a, "approval.resolved", now)}, nil
}

// CompleteTool accepts a trusted adapter's bounded, credential-free observation,
// never a model claim of success. Timeout/unknown are terminal and never retried.
func (c *Chain) CompleteTool(id, stepID, dispatchID, outcome, summary string, now time.Time) ([]Event, error) {
	r, err := c.Find(id)
	if err != nil {
		return nil, err
	}
	t, step, err := turnFor(r, stepID)
	if err != nil {
		return nil, err
	}
	if !secure.UUID(dispatchID) || dispatchID != t.DispatchID || (outcome != "completed" && outcome != "failed" && outcome != "unknown" && outcome != "timeout" && outcome != "cancelled") || len(summary) > 16384 || !utf8.ValidString(summary) {
		return nil, problem.Invalid("Invalid tool outcome")
	}
	raw, _ := json.Marshal([]string{dispatchID, outcome, summary})
	hash := secure.Hash(string(raw))
	if t.OutcomeHash != "" {
		if t.OutcomeHash == hash {
			return nil, nil
		}
		return nil, conflict("TOOL_OUTCOME_CONFLICT")
	}
	if t.State != "waiting_tool" || terminal(r.State) {
		return nil, conflict("STALE_TOOL_OUTCOME")
	}
	_, a, err := c.approval(t.ApprovalID)
	if err != nil {
		return nil, err
	}
	t.State, t.OutcomeHash = "done", hash
	stopped := r.State == "stopping" || step.Epoch != r.StopEpoch
	if !stopped && (t.DispatchDeadline == nil || !now.Before(*t.DispatchDeadline)) {
		outcome = "timeout"
	}
	a.ToolState = outcome
	a.Version++
	r.Version++
	t.Observation = &Observation{Code: outcome, ToolCallID: a.ToolCallID, Summary: summary}
	events := []Event{c.toolEvent(r, t, now)}
	if stopped {
		return events, nil
	}
	if outcome == "unknown" || outcome == "timeout" || outcome == "cancelled" {
		return append(events, c.failTurn(r, "tool_"+outcome, now)...), nil
	}
	if outcome == "completed" {
		r.Denials = 0
	}
	r.State = "running"
	return append(events, c.event(r, "run.state_changed", now)), nil
}
