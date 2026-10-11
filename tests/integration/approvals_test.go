package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"portway/internal/agents"
	"portway/internal/execution"
	"portway/internal/identity"
	"portway/internal/problem"
	"portway/internal/secure"
)

func exerciseApprovals(t *testing.T, ctx context.Context, store execution.Store, client *http.Client, base string, one, two identity.LoginResult) {
	t.Helper()
	provider := secure.ID()
	cfg, err := agents.Validate(agents.Config{Instructions: "approval instructions", RoleCode: "coder", RoleCatalogVersion: 1, ProviderRef: provider, Tools: []agents.Tool{{Alias: "shell", Kind: "shell.exec"}}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := store.Create(ctx, one.Owner.ID, "approval-fixture", execution.CreateInput{Task: "approval task", Snapshot: execution.Snapshot{AgentVersionID: secure.ID(), ProviderID: provider, Config: cfg, ConfigHash: agents.Hash(cfg), RepositoryRef: "fixture", Revision: strings.Repeat("a", 40)}})
	if err != nil {
		t.Fatal(err)
	}
	run := c.Runs[0].ID
	var approval execution.Approval
	c, err = store.Change(ctx, one.Owner.ID, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
		start, err := c.Start(run, now)
		if err != nil {
			return nil, err
		}
		a, ev, err := c.RequestShellApproval(run, execution.ShellCommand{Argv: []string{"go", "test", "./approval-private-args-marker"}, WorkingDir: ".", TimeoutSeconds: 60}, now)
		approval = a
		return append([]execution.Event{start}, ev...), err
	})
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body, key string, login identity.LoginResult) (int, response, []byte) {
		return request(t, client, base, method, path, body, login.Session.AccessToken, login.DeviceProof, key, nil)
	}
	code, list, b := call("GET", "/v1/approvals", "", "", one)
	want(t, code, 200, b)
	var items struct {
		Items []execution.Approval `json:"items"`
	}
	if json.Unmarshal(list.Data, &items) != nil || len(items.Items) != 1 || items.Items[0].ID != approval.ID {
		t.Fatal("missing pending approval", string(b))
	}
	path := "/v1/approvals/" + approval.ID
	code, _, b = call("GET", path, "", "", one)
	want(t, code, 200, b)
	if !strings.Contains(string(b), "approval-private-args-marker") {
		t.Fatal("owner cannot inspect command")
	}
	code, _, b = call("POST", path+"/decision", `{"decision":"approve","args_hash":"`+approval.ArgsHash+`","expected_version":1,"command":"injected"}`, "approval-invalid", one)
	want(t, code, 400, b)
	code, _, b = call("POST", path+"/decision", `{"decision":"approve","args_hash":"`+strings.Repeat("b", 64)+`","expected_version":1}`, "approval-wronghash", one)
	want(t, code, 409, b)
	// Competing owner devices must settle exactly one immutable decision.
	type result struct {
		code          int
		decision, key string
		body          []byte
		data          response
		login         identity.LoginResult
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i, decision := range []string{"approve", "reject"} {
		wg.Add(1)
		go func(i int, decision string) {
			defer wg.Done()
			login := one
			if i == 1 {
				login = two
			}
			body := fmt.Sprintf(`{"decision":%q,"args_hash":%q,"expected_version":1}`, decision, approval.ArgsHash)
			key := "competing-" + decision
			code, data, b := call("POST", path+"/decision", body, key, login)
			results <- result{code, decision, key, b, data, login}
		}(i, decision)
	}
	wg.Wait()
	close(results)
	wins := 0
	var winner result
	for result := range results {
		if result.code == 200 {
			wins++
			winner = result
		} else if result.code != 409 {
			t.Fatalf("unexpected decision response: %d %s", result.code, result.body)
		}
	}
	if wins != 1 {
		t.Fatal("decision race did not choose one winner")
	}
	var decided execution.Approval
	if json.Unmarshal(winner.data.Data, &decided) != nil || decided.DecisionActor != winner.login.DeviceID {
		t.Fatal("decision actor not recorded")
	}
	body := fmt.Sprintf(`{"decision":%q,"args_hash":%q,"expected_version":1}`, winner.decision, approval.ArgsHash)
	code, repeated, b := call("POST", path+"/decision", body, winner.key, winner.login)
	want(t, code, 200, b)
	if string(repeated.Data) != string(winner.data.Data) {
		t.Fatal("decision replay changed response")
	}
	code, _, b = call("POST", path+"/decision", fmt.Sprintf(`{"decision":%q,"args_hash":%q,"expected_version":2}`, winner.decision, approval.ArgsHash), winner.key, winner.login)
	want(t, code, 409, b)
	code, _, b = call("POST", "/v1/runs/"+run+"/stop", "{}", "approval-stop", one)
	want(t, code, 202, b)
	code, current, b := call("POST", path+"/decision", body, winner.key, winner.login)
	want(t, code, 200, b)
	if json.Unmarshal(current.Data, &decided) != nil || decided.State == "approved" && decided.ToolState != "cancelled" {
		t.Fatal("stopped approved action remained dispatchable")
	}
	if err = store.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	history, err := store.History(ctx, one.Owner.ID, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	requested, resolved := 0, 0
	for _, e := range history {
		if strings.Contains(string(e.Payload), "approval-private-args-marker") {
			t.Fatal("command leaked into events")
		}
		if e.Type == "approval_requested" {
			requested++
		}
		if e.Type == "approval.resolved" {
			resolved++
		}
		envelope := execution.Envelope{SchemaVersion: 1, EventID: e.ID, InstallationID: store.Installation, Producer: "execution", Type: e.Type, AggregateID: e.RunID, CorrelationID: e.CorrelationID, OccurredAt: e.ServerTime, Payload: e}
		if err = envelope.Validate(time.Now()); err != nil {
			t.Fatal("approval Kafka envelope rejected", err)
		}
	}
	if requested != 1 || resolved < 1 || resolved > 2 {
		t.Fatal("duplicate approval events", requested, resolved)
	}
	if _, err = store.GetApproval(ctx, secure.ID(), approval.ID); problem.Public(err).Status != 404 {
		t.Fatal("foreign owner approval access", err)
	}
	// Persist an already expired proposal using only trusted fixture time.
	c, err = store.Create(ctx, one.Owner.ID, "expiry-fixture", execution.CreateInput{Task: "expiry task", Snapshot: execution.Snapshot{AgentVersionID: secure.ID(), ProviderID: provider, Config: cfg, ConfigHash: agents.Hash(cfg), RepositoryRef: "fixture", Revision: strings.Repeat("a", 40)}})
	if err != nil {
		t.Fatal(err)
	}
	run = c.Runs[0].ID
	c, err = store.Change(ctx, one.Owner.ID, c.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
		past := now.Add(-181 * time.Second)
		start, err := c.Start(run, past)
		if err != nil {
			return nil, err
		}
		a, ev, err := c.RequestShellApproval(run, execution.ShellCommand{Argv: []string{"go", "version"}, WorkingDir: ".", TimeoutSeconds: 60}, past)
		approval = a
		return append([]execution.Event{start}, ev...), err
	})
	if err != nil {
		t.Fatal(err)
	}
	body = fmt.Sprintf(`{"decision":"approve","args_hash":%q,"expected_version":1}`, approval.ArgsHash)
	code, _, b = call("POST", "/v1/approvals/"+approval.ID+"/decision", body, "expired-decision", one)
	want(t, code, 422, b)
	got, err := store.GetApproval(ctx, one.Owner.ID, approval.ID)
	if err != nil || got.State != "expired" {
		t.Fatal("expired decision rollback lost state", err)
	}
	if err = store.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
}
