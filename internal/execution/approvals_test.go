package execution

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"portway/internal/agents"
	"portway/internal/problem"
	"portway/internal/secure"
)

func approvalFixture(t *testing.T) (Chain, Approval, time.Time) {
	t.Helper()
	c, now := fixture(t)
	r := &c.Runs[0]
	r.Snapshot.Config.Tools = []agents.Tool{{Alias: "shell", Kind: "shell.exec"}}
	r.Snapshot.ConfigHash = agents.Hash(r.Snapshot.Config)
	if _, err := c.Start(r.ID, now); err != nil {
		t.Fatal(err)
	}
	command := ShellCommand{Argv: []string{"go", "test", "./private-args-marker"}, WorkingDir: ".", TimeoutSeconds: 60}
	a, events, err := c.RequestShellApproval(r.ID, command, now)
	if err != nil {
		t.Fatal(err)
	}
	command.Argv[0] = "tampered"
	if a.Args.Argv[0] != "go" || r.Approvals[0].Args.Argv[0] != "go" {
		t.Fatal("proposal not detached")
	}
	if !a.ExpiresAt.Equal(now.Add(180*time.Second)) || a.State != "pending" || r.State != "awaiting_approval" {
		t.Fatal("invalid pending approval")
	}
	b, _ := json.Marshal(events)
	if strings.Contains(string(b), "private-args-marker") {
		t.Fatal("event leaked command")
	}
	return c, a, now
}

func TestApprovalDecisionAndStop(t *testing.T) {
	c, a, now := approvalFixture(t)
	d := Decision{Decision: "approve", ExpectedVersion: 1, ArgsHash: a.ArgsHash}
	actor := secure.ID()
	changed := d
	changed.ArgsHash = strings.Repeat("b", 64)
	if _, _, err := c.ResolveApproval(a.ID, actor, changed, now); problem.Public(err).Code != "APPROVAL_ARGUMENTS_CHANGED" {
		t.Fatal(err)
	}
	changed = d
	changed.ExpectedVersion = 2
	if _, _, err := c.ResolveApproval(a.ID, actor, changed, now); problem.Public(err).Code != "VERSION_CONFLICT" {
		t.Fatal(err)
	}
	approved, events, err := c.ResolveApproval(a.ID, actor, d, now)
	if err != nil || approved.State != "approved" || approved.ToolState != "ready" || approved.DecisionActor != actor || len(events) != 1 {
		t.Fatal(approved, err)
	}
	if _, _, err = c.Reserve(a.RunID, secure.ID(), 10, 20, now); err == nil {
		t.Fatal("model resumed before tool outcome")
	}
	seq := c.Runs[0].Seq
	if _, events, err = c.ResolveApproval(a.ID, secure.ID(), d, now); err != nil || len(events) != 0 || c.Runs[0].Seq != seq {
		t.Fatal("duplicate decision mutated state", err)
	}
	changed = d
	changed.Decision = "reject"
	if _, _, err = c.ResolveApproval(a.ID, actor, changed, now); problem.Public(err).Status != 409 {
		t.Fatal("opposite decision accepted", err)
	}
	if _, err = c.Stop("", now); err != nil {
		t.Fatal(err)
	}
	approved, _, err = c.ResolveApproval(a.ID, actor, d, now)
	if err != nil || approved.State != "approved" || approved.ToolState != "cancelled" || approved.DecisionActor != actor {
		t.Fatal("stop rewrote approval or replayed ready tool", approved, err)
	}
}

func TestApprovalExpiryRejectAndReconciliation(t *testing.T) {
	for _, kind := range []string{"expiry", "reject", "stop", "restart", "deadline", "budget"} {
		t.Run(kind, func(t *testing.T) {
			c, a, now := approvalFixture(t)
			d := Decision{Decision: "approve", ExpectedVersion: 1, ArgsHash: a.ArgsHash}
			switch kind {
			case "expiry":
				now = a.ExpiresAt
			case "reject":
				d.Decision = "reject"
			case "stop":
				_, _ = c.Stop(a.RunID, now)
			case "restart":
				c.Reconcile(now)
			case "deadline":
				c.Runs[0].Deadline = now
			case "budget":
				c.Runs[0].Budget.Spent = c.Runs[0].Budget.Limit
				c.Budget.Spent = c.Runs[0].Budget.Spent
			}
			out, _, err := c.ResolveApproval(a.ID, secure.ID(), d, now)
			if kind == "reject" {
				if err != nil || out.State != "rejected" || out.ToolState != "denied" || c.Runs[0].State != "running" {
					t.Fatal(out, err)
				}
			} else if err == nil || out.ToolState == "ready" {
				t.Fatal("invalid decision permitted", out, err)
			}
			if kind == "expiry" && (out.State != "expired" || problem.Public(err).Status != 422) {
				t.Fatal("expiry boundary", out, err)
			}
			if kind == "reject" {
				if _, _, err = c.RequestShellApproval(a.RunID, a.Args, now); problem.Public(err).Code != "ACTION_PREVIOUSLY_DENIED" {
					t.Fatal("same denied action reused", err)
				}
				alternative := a.Args
				alternative.Argv = []string{"go", "version"}
				if _, _, err = c.RequestShellApproval(a.RunID, alternative, now); err != nil {
					t.Fatal("different alternative rejected", err)
				}
			}
		})
	}
	c, a, _ := approvalFixture(t)
	if events := c.ExpireApprovals(a.ExpiresAt); len(events) != 2 || c.Runs[0].Approvals[0].State != "expired" {
		t.Fatal("expiry sweep failed")
	}
	if events := c.ExpireApprovals(a.ExpiresAt.Add(time.Minute)); len(events) != 0 {
		t.Fatal("expiry not idempotent")
	}
}

func TestApprovalPolicyAndCommandBounds(t *testing.T) {
	c, now := fixture(t)
	run := c.Runs[0].ID
	_, _ = c.Start(run, now)
	good := ShellCommand{Argv: []string{"go", "test"}, WorkingDir: ".", TimeoutSeconds: 60}
	if _, _, err := c.RequestShellApproval(run, good, now); problem.Public(err).Code != "POLICY_DENIED" {
		t.Fatal("undeclared tool permitted", err)
	}
	for _, cmd := range []ShellCommand{
		{Argv: []string{"go"}, WorkingDir: "../host", TimeoutSeconds: 60},
		{Argv: []string{"go"}, WorkingDir: "/tmp", TimeoutSeconds: 60},
		{Argv: []string{"go\nsecret"}, WorkingDir: ".", TimeoutSeconds: 60},
		{Argv: []string{"go"}, WorkingDir: ".", TimeoutSeconds: 61},
		{Argv: []string{"go", strings.Repeat("x", 8192)}, WorkingDir: ".", TimeoutSeconds: 60},
	} {
		if _, err := commandHash(cmd); err == nil {
			t.Fatal("invalid command accepted", cmd.WorkingDir)
		}
	}
}
