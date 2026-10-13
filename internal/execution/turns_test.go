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

func turnFixture(t *testing.T) (Chain, string, string, time.Time) {
	t.Helper()
	c, now := fixture(t)
	r := &c.Runs[0]
	r.Snapshot.Config.Tools = []agents.Tool{{Alias: "shell", Kind: "shell.exec"}, {Alias: "other_shell", Kind: "shell.exec"}}
	r.Snapshot.ConfigHash = agents.Hash(r.Snapshot.Config)
	if _, err := c.Start(r.ID, now); err != nil {
		t.Fatal(err)
	}
	step := secure.ID()
	if _, err := c.BeginTurn(r.ID, step, 100, 1000, now); err != nil {
		t.Fatal(err)
	}
	return c, r.ID, step, now
}
func shellReply(arg string) ModelReply {
	return ModelReply{Finish: "tool", Usage: 123, UsageSource: "actual", Tool: &ShellProposal{Alias: "shell", Command: ShellCommand{Argv: []string{"echo", arg}, WorkingDir: ".", TimeoutSeconds: 60}}}
}
func finalReply() ModelReply {
	return ModelReply{Finish: "final", Text: "private final answer marker", Usage: 200, UsageSource: "actual"}
}
func assertValid(t *testing.T, c Chain) {
	t.Helper()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestTurnFinalReplayAndStop(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		c, id, step, now := turnFixture(t)
		if stopped {
			_, _ = c.Stop("", now)
		}
		events, err := c.AcceptReply(id, step, finalReply(), now)
		if err != nil {
			t.Fatal(err)
		}
		r := &c.Runs[0]
		if r.Budget.Spent != 200 || r.Budget.Reserved != 0 {
			t.Fatal("settlement lost")
		}
		if stopped {
			if r.Result != nil || r.State != "stopping" {
				t.Fatal("late final revived run")
			}
		} else if r.Result == nil || r.Result.ContentHash != secure.Hash(finalReply().Text) || c.State != "completed" {
			t.Fatal("final result missing")
		}
		raw, _ := json.Marshal(events)
		if strings.Contains(string(raw), "private final") {
			t.Fatal("answer in event")
		}
		for _, e := range events {
			if err := (Envelope{1, e.ID, c.InstallationID, "execution", e.Type, id, c.ID, e.ServerTime, e}).Validate(now); err != nil {
				t.Fatal(err)
			}
		}
		before, _ := json.Marshal(c)
		again, err := c.AcceptReply(id, step, finalReply(), now.Add(time.Second))
		after, _ := json.Marshal(c)
		if err != nil || len(again) != 0 || string(before) != string(after) {
			t.Fatal("reply replay mutated state", err)
		}
		changed := finalReply()
		changed.Text = "other"
		if _, err = c.AcceptReply(id, step, changed, now); problem.Public(err).Code != "MODEL_REPLY_CONFLICT" {
			t.Fatal(err)
		}
		assertValid(t, c)
	}
}
func TestTurnBadUsageAndIncompleteReply(t *testing.T) {
	cases := []struct {
		reply  ModelReply
		reason string
		spent  int
	}{
		{ModelReply{Finish: "final", Text: "answer", UsageSource: "unknown"}, "budget_uncertain", 1100},
		{ModelReply{Finish: "final", Text: "answer", UsageSource: "actual", Usage: 1101}, "budget_uncertain", 1100},
		{ModelReply{Finish: "final", UsageSource: "actual", Usage: 10}, "model_response_invalid", 10},
		{ModelReply{Finish: "partial", Text: "incomplete", UsageSource: "actual", Usage: 10}, "model_response_invalid", 10},
		{ModelReply{Finish: "error", Text: "provider-secret-marker", UsageSource: "actual", Usage: 10}, "model_failed", 10},
	}
	for _, tc := range cases {
		c, id, step, now := turnFixture(t)
		if _, err := c.AcceptReply(id, step, tc.reply, now); err != nil {
			t.Fatal(err)
		}
		if c.Runs[0].Reason != tc.reason || c.Runs[0].Budget.Spent != tc.spent || c.Runs[0].Result != nil {
			t.Fatalf("wrong failure: %+v", c.Runs[0])
		}
		assertValid(t, c)
	}
	c, id, step, now := turnFixture(t)
	_, err := c.AcceptReply(id, step, finalReply(), now.Add(900*time.Second))
	if err != nil || c.Runs[0].Reason != "deadline_exceeded" || c.Runs[0].Result != nil {
		t.Fatal("late deadline result", err)
	}
}
func TestTurnRejectionAlternativeAndPolicyLimit(t *testing.T) {
	c, id, step, now := turnFixture(t)
	if _, err := c.AcceptReply(id, step, shellReply("private command"), now); err != nil {
		t.Fatal(err)
	}
	a := c.Runs[0].Approvals[0]
	_, _, err := c.ResolveApproval(a.ID, secure.ID(), Decision{"reject", 1, a.ArgsHash}, now)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err = c.ObserveDecisions(id, now); err != nil {
			t.Fatal(err)
		}
	}
	if c.Runs[0].Denials != 1 || c.Runs[0].Turns[0].Observation.Code != "APPROVAL_REJECTED" {
		t.Fatal("denial not fed back once")
	}
	step = secure.ID()
	if _, err = c.BeginTurn(id, step, 200, 1000, now); err != nil {
		t.Fatal(err)
	}
	// Same arguments via another alias must not create another approval.
	reply := shellReply("private command")
	reply.Tool.Alias = "other_shell"
	if _, err = c.AcceptReply(id, step, reply, now); err != nil {
		t.Fatal(err)
	}
	if c.State != "failed" || c.Runs[0].Reason != "policy_exhausted" || len(c.Runs[0].Approvals) != 1 {
		t.Fatal("repeated denied command bypassed limit")
	}
	assertValid(t, c)
}
func TestToolRoundtripAndFences(t *testing.T) {
	for _, mode := range []string{"success", "stop", "expired", "timeout", "restart", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			c, id, step, now := turnFixture(t)
			_, err := c.AcceptReply(id, step, shellReply("tool args"), now)
			if err != nil {
				t.Fatal(err)
			}
			a := c.Runs[0].Approvals[0]
			dispatch := secure.ID()
			if _, err = c.BeginToolDispatch(id, step, dispatch, now); err == nil {
				t.Fatal("pending dispatched")
			}
			_, _, err = c.ResolveApproval(a.ID, secure.ID(), Decision{"approve", 1, a.ArgsHash}, now)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = c.BeginTurn(id, secure.ID(), 100, 100, now); err == nil {
				t.Fatal("model proceeded before outcome")
			}
			if mode == "expired" {
				if _, err = c.BeginToolDispatch(id, step, dispatch, now.Add(180*time.Second)); err == nil {
					t.Fatal("expired grant admitted")
				}
				return
			}
			if _, err = c.BeginToolDispatch(id, step, dispatch, now); err != nil {
				t.Fatal(err)
			}
			if _, err = c.BeginToolDispatch(id, step, dispatch, now); err == nil {
				t.Fatal("duplicate dispatch allowed")
			}
			if mode == "restart" {
				c.Reconcile(now)
				if c.Runs[0].Approvals[0].ToolState != "unknown" {
					t.Fatal("restart concealed unknown dispatch")
				}
				if _, err = c.CompleteTool(id, step, dispatch, "completed", "late", now); err == nil {
					t.Fatal("late outcome revived restart")
				}
				return
			}
			if mode == "stop" {
				_, _ = c.Stop("", now)
				if _, err = c.ConfirmStopped(id, now); err == nil {
					t.Fatal("cancelled while tool outstanding")
				}
			}
			when := now.Add(time.Second)
			if mode == "timeout" {
				when = now.Add(60 * time.Second)
			}
			outcome := "completed"
			if mode == "unknown" {
				outcome = "unknown"
			}
			if _, err = c.CompleteTool(id, step, dispatch, outcome, "bounded private output", when); err != nil {
				t.Fatal(err)
			}
			seq := c.Runs[0].Seq
			if ev, err := c.CompleteTool(id, step, dispatch, outcome, "bounded private output", when); err != nil || len(ev) != 0 || c.Runs[0].Seq != seq {
				t.Fatal("outcome not idempotent")
			}
			if _, err = c.CompleteTool(id, step, dispatch, "failed", "changed", when); err == nil {
				t.Fatal("conflicting outcome accepted")
			}
			switch mode {
			case "stop":
				if c.Runs[0].State != "stopping" {
					t.Fatal("tool revived run")
				}
				if _, err = c.ConfirmStopped(id, when); err != nil {
					t.Fatal(err)
				}
			case "timeout", "unknown":
				if c.Runs[0].State != "failed" {
					t.Fatal("uncertain outcome continued")
				}
			default:
				next := secure.ID()
				if _, err = c.BeginTurn(id, next, 300, 1000, when); err != nil {
					t.Fatal(err)
				}
				if _, err = c.AcceptReply(id, next, finalReply(), when); err != nil {
					t.Fatal(err)
				}
				if c.State != "completed" || c.Runs[0].Result == nil {
					t.Fatal("tool -> final not completed")
				}
			}
			assertValid(t, c)
		})
	}
}

func TestApprovalExpiryAndSuccessfulAlternativeReset(t *testing.T) {
	c, id, step, now := turnFixture(t)
	if _, err := c.AcceptReply(id, step, shellReply("first"), now); err != nil {
		t.Fatal(err)
	}
	later := now.Add(180 * time.Second)
	if _, err := c.ObserveDecisions(id, later); err != nil {
		t.Fatal(err)
	}
	if c.Runs[0].Denials != 1 || c.Runs[0].Turns[0].Observation.Code != "APPROVAL_EXPIRED" {
		t.Fatal("expiry feedback missing")
	}
	next := secure.ID()
	if _, err := c.BeginTurn(id, next, 100, 1000, later); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AcceptReply(id, next, shellReply("different"), later); err != nil {
		t.Fatal(err)
	}
	a := c.Runs[0].Approvals[1]
	if _, _, err := c.ResolveApproval(a.ID, secure.ID(), Decision{"approve", 1, a.ArgsHash}, later); err != nil {
		t.Fatal(err)
	}
	dispatch := secure.ID()
	if _, err := c.BeginToolDispatch(id, next, dispatch, later); err != nil {
		t.Fatal(err)
	}
	// Reaching the run deadline alone must not make an outstanding dispatch look
	// terminal; its trusted outcome still needs to be recorded.
	if _, err := c.ObserveDecisions(id, now.Add(900*time.Second)); err != nil || terminal(c.Runs[0].State) {
		t.Fatal("deadline concealed outstanding tool", err)
	}
	if _, err := c.CompleteTool(id, next, dispatch, "completed", "safe alternative", later.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if c.Runs[0].Denials != 0 || c.Runs[0].State != "running" {
		t.Fatal("safe action did not reset denials")
	}
	assertValid(t, c)
}
