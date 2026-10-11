// Package execution owns run/chain state. It never executes model or tool calls.
package execution

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"portway/internal/agents"
	"portway/internal/problem"
	"portway/internal/secure"
)

type Budget struct {
	Limit     int `json:"limit"`
	Spent     int `json:"spent"`
	Reserved  int `json:"reserved"`
	Steps     int `json:"steps"`
	StepLimit int `json:"step_limit"`
}
type Snapshot struct {
	AgentVersionID string        `json:"agent_version_id"`
	ProviderID     string        `json:"provider_id"`
	ConfigHash     string        `json:"config_hash"`
	Config         agents.Config `json:"config"`
	RepositoryRef  string        `json:"repository_ref"`
	Revision       string        `json:"revision"`
}
type Step struct {
	ID       string `json:"id"`
	Input    int    `json:"input"`
	Output   int    `json:"output"`
	Reserved int    `json:"reserved"`
	Charged  int    `json:"charged"`
	Source   string `json:"usage_source"`
	Settled  bool   `json:"settled"`
	Epoch    int64  `json:"epoch"`
}
type Run struct {
	ID        string        `json:"id"`
	ChainID   string        `json:"correlation_id"`
	State     string        `json:"state"`
	Reason    string        `json:"reason,omitempty"`
	Version   int64         `json:"version"`
	StopEpoch int64         `json:"stop_epoch"`
	Deadline  time.Time     `json:"deadline"`
	Task      string        `json:"task"`
	Snapshot  Snapshot      `json:"snapshot"`
	Limits    agents.Limits `json:"limits"`
	Budget    Budget        `json:"budgets"`
	Steps     []Step        `json:"steps"`
	Seq       int64         `json:"snapshot_seq"`
	Approvals []Approval    `json:"approvals,omitempty"`
}

// RunSummary bounds collection responses; full pinned instructions are fetched by ID.
type RunSummary struct {
	ID       string    `json:"id"`
	ChainID  string    `json:"correlation_id"`
	State    string    `json:"state"`
	Reason   string    `json:"reason,omitempty"`
	Version  int64     `json:"version"`
	Deadline time.Time `json:"deadline"`
	Budget   Budget    `json:"budgets"`
}
type Chain struct {
	ID             string    `json:"id"`
	InstallationID string    `json:"installation_id"`
	OwnerID        string    `json:"owner_id"`
	State          string    `json:"state"`
	StopEpoch      int64     `json:"stop_epoch"`
	CreatedAt      time.Time `json:"created_at"`
	Deadline       time.Time `json:"deadline"`
	Budget         Budget    `json:"budgets"`
	Runs           []Run     `json:"runs"`
}
type CreateInput struct {
	Snapshot Snapshot      `json:"snapshot"`
	Task     string        `json:"task"`
	Limits   agents.Limits `json:"limits"`
}
type Event struct {
	SchemaVersion int             `json:"schema_version"`
	ID            string          `json:"id"`
	RunID         string          `json:"run_id"`
	CorrelationID string          `json:"correlation_id"`
	Seq           int64           `json:"seq"`
	Type          string          `json:"type"`
	ServerTime    time.Time       `json:"server_time"`
	Payload       json.RawMessage `json:"payload"`
}

func terminal(s string) bool { return s == "completed" || s == "failed" || s == "cancelled" }
func conflict(code string) error {
	return problem.New(409, code, "Execution state does not permit this operation")
}

// NewChain is for trusted admission adapters. HTTP admission stays closed until
// the real Runtime, capability checks and revocation fencing are implemented.
func NewChain(installation, owner string, in CreateInput, now time.Time) (Chain, Event, error) {
	c := Chain{}
	cfg, err := agents.Validate(in.Snapshot.Config)
	if err != nil || agents.Hash(cfg) != in.Snapshot.ConfigHash || !secure.UUID(installation) || !secure.UUID(owner) || !secure.UUID(in.Snapshot.AgentVersionID) || !secure.UUID(in.Snapshot.ProviderID) || cfg.ProviderRef != in.Snapshot.ProviderID || strings.TrimSpace(in.Task) == "" || len(in.Task) > 65536 || !agents.Alias(in.Snapshot.RepositoryRef) || !revision(in.Snapshot.Revision) {
		return c, Event{}, problem.Invalid("Invalid pinned run snapshot")
	}
	l := in.Limits
	if l == (agents.Limits{}) {
		l = cfg.Limits
	}
	if l.RunSeconds < 1 || l.RunSeconds > cfg.Limits.RunSeconds || l.Steps < 1 || l.Steps > cfg.Limits.Steps || l.Tokens < 1 || l.Tokens > cfg.Limits.Tokens || l.OutputTokens < 1 || l.OutputTokens > cfg.Limits.OutputTokens || l.OutputTokens > l.Tokens {
		return c, Event{}, problem.Invalid("Run limits exceed pinned version")
	}
	// Detach maps and slices supplied by the admission adapter.
	in.Snapshot.Config = cfg
	b, _ := json.Marshal(in.Snapshot)
	var pinned Snapshot
	_ = json.Unmarshal(b, &pinned)
	c = Chain{ID: secure.ID(), InstallationID: installation, OwnerID: owner, State: "active", CreatedAt: now, Deadline: now.Add(1800 * time.Second), Budget: Budget{Limit: 40000, StepLimit: 40}}
	c.Runs = []Run{{ID: secure.ID(), ChainID: c.ID, State: "queued", Version: 1, Deadline: now.Add(time.Duration(l.RunSeconds) * time.Second), Task: in.Task, Snapshot: pinned, Limits: l, Budget: Budget{Limit: l.Tokens, StepLimit: l.Steps}, Steps: []Step{}}}
	e := c.event(&c.Runs[0], "run.created", now)
	return c, e, nil
}
func revision(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, ch := range s {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f') {
			return false
		}
	}
	return true
}
func (c *Chain) Find(id string) (*Run, error) {
	for i := range c.Runs {
		if c.Runs[i].ID == id {
			return &c.Runs[i], nil
		}
	}
	return nil, problem.NotFound()
}
func (c *Chain) event(r *Run, kind string, now time.Time) Event {
	r.Seq++
	payload, _ := json.Marshal(map[string]any{"state": r.State, "reason": r.Reason, "version": r.Version, "budgets": r.Budget})
	return Event{1, secure.ID(), r.ID, c.ID, r.Seq, kind, now, payload}
}
func (c *Chain) live(r *Run, now time.Time) error {
	if c.State != "active" || terminal(r.State) || r.State == "stopping" {
		return conflict("RUN_TERMINAL")
	}
	if !now.Before(c.Deadline) || !now.Before(r.Deadline) {
		return problem.New(422, "DEADLINE_EXCEEDED", "Execution deadline has passed")
	}
	return nil
}
func (c *Chain) Start(id string, now time.Time) (Event, error) {
	r, err := c.Find(id)
	if err != nil {
		return Event{}, err
	}
	if err = c.live(r, now); err != nil {
		return Event{}, err
	}
	if r.State != "queued" {
		return Event{}, conflict("INVALID_TRANSITION")
	}
	r.State = "running"
	r.Version++
	return c.event(r, "run.state_changed", now), nil
}
func (c *Chain) Reserve(id, stepID string, input, output int, now time.Time) (Step, Event, error) {
	r, err := c.Find(id)
	if err != nil {
		return Step{}, Event{}, err
	}
	if err = c.live(r, now); err != nil {
		return Step{}, Event{}, err
	}
	if !secure.UUID(stepID) || input < 0 || input > 40000 || output < 1 || output > r.Limits.OutputTokens {
		return Step{}, Event{}, problem.Invalid("Invalid token reservation")
	}
	for _, s := range r.Steps {
		if s.ID == stepID {
			return Step{}, Event{}, conflict("STEP_ALREADY_RESERVED")
		}
	}
	amount := input + output
	if r.State != "running" {
		return Step{}, Event{}, conflict("INVALID_TRANSITION")
	}
	if r.Budget.Steps >= r.Budget.StepLimit || c.Budget.Steps >= c.Budget.StepLimit || amount > r.Budget.Limit-r.Budget.Spent-r.Budget.Reserved || amount > c.Budget.Limit-c.Budget.Spent-c.Budget.Reserved {
		return Step{}, Event{}, problem.New(422, "BUDGET_EXHAUSTED", "No remaining execution budget")
	}
	s := Step{ID: stepID, Input: input, Output: output, Reserved: amount, Epoch: r.StopEpoch}
	r.Steps = append(r.Steps, s)
	r.Budget.Reserved += amount
	c.Budget.Reserved += amount
	r.Budget.Steps++
	c.Budget.Steps++
	r.State = "waiting_model"
	r.Version++
	return s, c.event(r, "run.state_changed", now), nil
}

// Settle releases unused tokens only for confirmed actual usage. Unknown usage
// consumes the entire reservation and fails the run, including after a stop.
func (c *Chain) Settle(id, stepID string, used int, source string, now time.Time) (*Event, error) {
	r, err := c.Find(id)
	if err != nil {
		return nil, err
	}
	var s *Step
	for i := range r.Steps {
		if r.Steps[i].ID == stepID {
			s = &r.Steps[i]
			break
		}
	}
	if s == nil {
		return nil, problem.NotFound()
	}
	if source != "actual" && source != "unknown" {
		return nil, problem.Invalid("Unsupported usage source")
	}
	if source == "unknown" {
		used = s.Reserved
	}
	if used < 0 || used > s.Reserved {
		return nil, problem.New(422, "BUDGET_UNCERTAIN", "Usage exceeds the confirmed reservation")
	}
	if s.Settled {
		if used == s.Charged && source == s.Source {
			return nil, nil
		}
		return nil, conflict("USAGE_CONFLICT")
	}
	s.Settled = true
	s.Charged = used
	s.Source = source
	r.Budget.Reserved -= s.Reserved
	c.Budget.Reserved -= s.Reserved
	r.Budget.Spent += used
	c.Budget.Spent += used
	if source == "unknown" {
		r.State = "failed"
		r.Reason = "budget_uncertain"
	} else if r.State == "waiting_model" {
		r.State = "running"
	}
	r.Version++
	c.finish()
	e := c.event(r, "run.state_changed", now)
	return &e, nil
}
func (c *Chain) Stop(id string, now time.Time) ([]Event, error) {
	if id != "" {
		if _, err := c.Find(id); err != nil {
			return nil, err
		}
	}
	out := []Event{}
	if terminal(c.State) {
		return out, nil
	}
	if id == "" && c.State != "stopping" {
		c.State = "stopping"
		c.StopEpoch++
	}
	for i := range c.Runs {
		r := &c.Runs[i]
		if (id == "" || r.ID == id) && !terminal(r.State) && r.State != "stopping" {
			r.State = "stopping"
			r.StopEpoch++
			r.Version++
			out = append(out, c.cancelApprovals(r, now)...)
			out = append(out, c.event(r, "run.state_changed", now))
		}
	}
	return out, nil
}

// ConfirmStopped is called only once controlled work is known to have ended.
func (c *Chain) ConfirmStopped(id string, now time.Time) (Event, error) {
	r, err := c.Find(id)
	if err != nil {
		return Event{}, err
	}
	if r.State != "stopping" || r.Budget.Reserved != 0 {
		return Event{}, conflict("STOP_NOT_CONFIRMED")
	}
	r.State = "cancelled"
	r.Reason = "owner_requested"
	r.Version++
	c.finish()
	return c.event(r, "run.state_changed", now), nil
}

// Reconcile never redispatches work after process restart. Outstanding reserves
// are conservatively charged, and nonterminal runs become failed/interrupted.
func (c *Chain) Reconcile(now time.Time) []Event {
	out := []Event{}
	if terminal(c.State) {
		return out
	}
	for i := range c.Runs {
		r := &c.Runs[i]
		if terminal(r.State) {
			continue
		}
		for j := range r.Steps {
			s := &r.Steps[j]
			if !s.Settled {
				s.Settled = true
				s.Source = "unknown"
				s.Charged = s.Reserved
				r.Budget.Spent += s.Reserved
				c.Budget.Spent += s.Reserved
				r.Budget.Reserved -= s.Reserved
				c.Budget.Reserved -= s.Reserved
			}
		}
		r.State = "failed"
		r.Reason = "interrupted"
		r.StopEpoch++
		r.Version++
		out = append(out, c.cancelApprovals(r, now)...)
		out = append(out, c.event(r, "run.state_changed", now))
	}
	c.StopEpoch++
	c.finish()
	return out
}
func (c *Chain) finish() {
	failed, cancelled := false, false
	for _, r := range c.Runs {
		if !terminal(r.State) {
			return
		}
		failed = failed || r.State == "failed"
		cancelled = cancelled || r.State == "cancelled"
	}
	if failed {
		c.State = "failed"
	} else if cancelled {
		c.State = "cancelled"
	} else {
		c.State = "completed"
	}
}
func (c Chain) Validate() error {
	spent, reserved, steps := 0, 0, 0
	for _, r := range c.Runs {
		if r.Budget.Spent < 0 || r.Budget.Reserved < 0 || r.Budget.Spent+r.Budget.Reserved > r.Budget.Limit || r.Budget.Steps > r.Budget.StepLimit {
			return errors.New("invalid run budget")
		}
		spent += r.Budget.Spent
		reserved += r.Budget.Reserved
		steps += r.Budget.Steps
	}
	if spent != c.Budget.Spent || reserved != c.Budget.Reserved || steps != c.Budget.Steps || spent+reserved > c.Budget.Limit || steps > c.Budget.StepLimit {
		return errors.New("invalid chain budget")
	}
	return nil
}
