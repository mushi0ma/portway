package execution

import (
	"encoding/json"
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"portway/internal/problem"
	"portway/internal/secure"
)

// ShellCommand is an immutable argv, not a shell-interpolated command string.
// A future Harness must remove credential values before proposing a command.
// These arguments are owner-only; events carry references and hashes only.
type ShellCommand struct {
	Argv           []string `json:"argv"`
	WorkingDir     string   `json:"working_dir"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type Approval struct {
	ID            string       `json:"id"`
	RunID         string       `json:"run_id"`
	ToolCallID    string       `json:"tool_call_id"`
	ToolName      string       `json:"tool_name"`
	Args          ShellCommand `json:"args"`
	ArgsHash      string       `json:"args_hash"`
	PolicyVersion string       `json:"policy_version"`
	State         string       `json:"state"`
	ToolState     string       `json:"tool_state"`
	Version       int64        `json:"version"`
	StopEpoch     int64        `json:"stop_epoch"`
	ChainEpoch    int64        `json:"chain_epoch"`
	CreatedAt     time.Time    `json:"created_at"`
	ExpiresAt     time.Time    `json:"expires_at"`
	DecidedAt     *time.Time   `json:"decided_at,omitempty"`
	DecisionActor string       `json:"decision_actor,omitempty"`
	Reason        string       `json:"reason,omitempty"`
}

type Decision struct {
	Decision        string `json:"decision"`
	ExpectedVersion int64  `json:"expected_version"`
	ArgsHash        string `json:"args_hash"`
}

func (d Decision) Validate() error {
	if (d.Decision != "approve" && d.Decision != "reject") || d.ExpectedVersion < 1 || !revision(d.ArgsHash) || len(d.ArgsHash) != 64 {
		return problem.Invalid("Expected approve/reject, positive version and SHA-256 args_hash")
	}
	return nil
}

func commandHash(command ShellCommand) (string, error) {
	if len(command.Argv) == 0 || len(command.Argv) > 64 || command.Argv[0] == "" || command.TimeoutSeconds < 1 || command.TimeoutSeconds > 60 || command.WorkingDir == "" || path.IsAbs(command.WorkingDir) || path.Clean(command.WorkingDir) != command.WorkingDir || command.WorkingDir == ".." || strings.HasPrefix(command.WorkingDir, "../") || strings.Contains(command.WorkingDir, "\\") {
		return "", problem.Invalid("Invalid workspace command or tool timeout")
	}
	for _, value := range append(append([]string{}, command.Argv...), command.WorkingDir) {
		if !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return "", problem.Invalid("Command contains control characters")
		}
	}
	b, _ := json.Marshal(command)
	if len(b) > 8192 {
		return "", problem.Invalid("Command is too large")
	}
	return secure.Hash(string(b)), nil
}

func (c *Chain) approval(id string) (*Run, *Approval, error) {
	for i := range c.Runs {
		for j := range c.Runs[i].Approvals {
			if c.Runs[i].Approvals[j].ID == id {
				return &c.Runs[i], &c.Runs[i].Approvals[j], nil
			}
		}
	}
	return nil, nil, problem.NotFound()
}

func (c *Chain) approvalEvent(r *Run, a *Approval, kind string, now time.Time) Event {
	r.Seq++
	payload, _ := json.Marshal(map[string]any{"approval_id": a.ID, "tool_call_id": a.ToolCallID, "tool_name": a.ToolName, "args_hash": a.ArgsHash, "expires_at": a.ExpiresAt, "state": a.State, "tool_state": a.ToolState, "version": a.Version, "reason": a.Reason})
	return Event{1, secure.ID(), r.ID, c.ID, r.Seq, kind, now, payload}
}

// RequestShellApproval is a trusted Harness boundary, never a public command
// execution endpoint. Owner decisions do not dispatch or mint Runtime grants.
func (c *Chain) RequestShellApproval(run string, command ShellCommand, now time.Time) (Approval, []Event, error) {
	r, err := c.Find(run)
	if err != nil {
		return Approval{}, nil, err
	}
	if err = c.live(r, now); err != nil {
		return Approval{}, nil, err
	}
	if r.State != "running" || len(r.Approvals) >= 20 {
		return Approval{}, nil, conflict("APPROVAL_NOT_ALLOWED")
	}
	allowed := false
	for _, t := range r.Snapshot.Config.Tools {
		allowed = allowed || t.Kind == "shell.exec"
	}
	if !allowed || r.Snapshot.Config.Policy["shell.exec"] != "approval" {
		return Approval{}, nil, problem.New(403, "POLICY_DENIED", "Pinned policy does not allow this tool")
	}
	hash, err := commandHash(command)
	if err != nil {
		return Approval{}, nil, err
	}
	for _, a := range r.Approvals {
		if a.ArgsHash == hash && (a.State == "rejected" || a.State == "expired") {
			return Approval{}, nil, problem.New(403, "ACTION_PREVIOUSLY_DENIED", "Use a different safe alternative")
		}
	}
	command.Argv = append([]string{}, command.Argv...)
	a := Approval{ID: secure.ID(), RunID: run, ToolCallID: secure.ID(), ToolName: "shell.exec", Args: command, ArgsHash: hash, PolicyVersion: r.Snapshot.ConfigHash, State: "pending", ToolState: "awaiting_approval", Version: 1, StopEpoch: r.StopEpoch, ChainEpoch: c.StopEpoch, CreatedAt: now, ExpiresAt: now.Add(180 * time.Second)}
	r.Approvals = append(r.Approvals, a)
	r.State = "awaiting_approval"
	r.Version++
	return a, []Event{c.approvalEvent(r, &a, "approval_requested", now), c.event(r, "run.state_changed", now)}, nil
}

func approvalError(status int, code string, a *Approval) error {
	p := problem.New(status, code, "Approval cannot accept this decision")
	p.Details = map[string]any{"approval": a}
	return p
}

// ResolveApproval records a business decision. A later dispatcher must repeat
// current device, policy, budget, epoch and deadline checks before any grant.
func (c *Chain) ResolveApproval(id, actor string, d Decision, now time.Time) (Approval, []Event, error) {
	if err := d.Validate(); err != nil {
		return Approval{}, nil, err
	}
	if !secure.UUID(actor) {
		return Approval{}, nil, problem.Invalid("Invalid decision actor")
	}
	r, a, err := c.approval(id)
	if err != nil {
		return Approval{}, nil, err
	}
	if d.ArgsHash != a.ArgsHash {
		return *a, nil, approvalError(409, "APPROVAL_ARGUMENTS_CHANGED", a)
	}
	if a.State != "pending" {
		if a.State == "expired" {
			return *a, nil, approvalError(422, "APPROVAL_EXPIRED", a)
		}
		if (a.State == "approved" && d.Decision == "approve" || a.State == "rejected" && d.Decision == "reject") && (d.ExpectedVersion == 1 || d.ExpectedVersion == a.Version) {
			return *a, nil, nil
		}
		return *a, nil, approvalError(409, "APPROVAL_ALREADY_RESOLVED", a)
	}
	if d.ExpectedVersion != a.Version {
		return *a, nil, approvalError(409, "VERSION_CONFLICT", a)
	}
	if !now.Before(a.ExpiresAt) {
		events := c.finishApproval(r, a, "expired", "denied", "approval_expired", "", now)
		return *a, events, approvalError(422, "APPROVAL_EXPIRED", a)
	}
	if a.StopEpoch != r.StopEpoch || a.ChainEpoch != c.StopEpoch || c.State != "active" || r.State != "awaiting_approval" || !now.Before(r.Deadline) || !now.Before(c.Deadline) || a.PolicyVersion != r.Snapshot.ConfigHash || r.Snapshot.Config.Policy["shell.exec"] != "approval" {
		events := c.finishApproval(r, a, "cancelled", "cancelled", "run_not_executable", "", now)
		return *a, events, approvalError(409, "APPROVAL_INVALIDATED", a)
	}
	if d.Decision == "approve" && (r.Budget.Spent+r.Budget.Reserved >= r.Budget.Limit || c.Budget.Spent+c.Budget.Reserved >= c.Budget.Limit) {
		events := c.finishApproval(r, a, "cancelled", "cancelled", "budget_exhausted", "", now)
		return *a, events, approvalError(422, "BUDGET_EXHAUSTED", a)
	}
	state, tool := "approved", "ready"
	if d.Decision == "reject" {
		state, tool = "rejected", "denied"
	}
	events := c.finishApproval(r, a, state, tool, "owner_decision", actor, now)
	return *a, events, nil
}

func (c *Chain) finishApproval(r *Run, a *Approval, state, tool, reason, actor string, now time.Time) []Event {
	a.State, a.ToolState, a.Reason, a.DecisionActor = state, tool, reason, actor
	a.DecidedAt = &now
	a.Version++
	events := []Event{c.approvalEvent(r, a, "approval.resolved", now)}
	// An approved tool still blocks the model loop until its outcome is recorded.
	if r.State == "awaiting_approval" && state != "approved" {
		r.State = "running"
		r.Version++
		events = append(events, c.event(r, "run.state_changed", now))
	}
	return events
}

func (c *Chain) ExpireApprovals(now time.Time) []Event {
	var events []Event
	for i := range c.Runs {
		r := &c.Runs[i]
		for j := range r.Approvals {
			a := &r.Approvals[j]
			if a.State == "pending" && !now.Before(a.ExpiresAt) {
				events = append(events, c.finishApproval(r, a, "expired", "denied", "approval_expired", "", now)...)
			}
		}
	}
	return events
}

func (c *Chain) cancelApprovals(r *Run, now time.Time) []Event {
	var events []Event
	for j := range r.Approvals {
		a := &r.Approvals[j]
		if a.State == "pending" {
			events = append(events, c.finishApproval(r, a, "cancelled", "cancelled", "run_stopped", "", now)...)
		} else if a.State == "approved" && a.ToolState == "ready" {
			// Approval is immutable; stopping revokes its unexecuted tool instead.
			a.ToolState, a.Reason = "cancelled", "run_stopped"
			a.Version++
			events = append(events, c.approvalEvent(r, a, "approval.resolved", now))
		}
	}
	return events
}
