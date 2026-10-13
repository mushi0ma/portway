package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"portway/internal/agents"
	"portway/internal/config"
	"portway/internal/execution"
	"portway/internal/modelgateway"
	"portway/internal/problem"
	"portway/internal/secure"
	"portway/internal/storage"
	"portway/internal/transport"
)

type gatewayRoundTrip func(*http.Request) (*http.Response, error)

func (f gatewayRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func exercisePrivateGateway(t *testing.T, cfg config.Config, app, root *storage.Store, store execution.Store, token, proof string) {
	t.Helper()
	ctx := context.Background()
	var generations atomic.Int32
	var block atomic.Bool
	entered := make(chan struct{}, 1)
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		generations.Add(1)
		var body map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "text/event-stream")
		switch string(body["tool_choice"]) {
		case "":
			if block.Load() {
				select {
				case entered <- struct{}{}:
				default:
				}
				<-r.Context().Done()
				return
			}
			io.WriteString(w, "data: {\"id\":\"agent\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"agent final fixture\"},\"finish_reason\":\"stop\"}]}\n\ndata: {\"id\":\"agent\",\"choices\":[],\"usage\":{\"prompt_tokens\":50,\"completion_tokens\":10,\"total_tokens\":60}}\n\ndata: [DONE]\n\n")
		case `"none"`:
			io.WriteString(w, "data: {\"id\":\"probe\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"PORTWAY_PROBE_OK\"},\"finish_reason\":\"stop\"}]}\n\ndata: {\"id\":\"probe\",\"choices\":[],\"usage\":{\"prompt_tokens\":50,\"completion_tokens\":10,\"total_tokens\":60}}\n\ndata: [DONE]\n\n")
		default:
			args := `{"argv":["echo","PORTWAY_PROBE_OK"],"working_dir":".","timeout_seconds":1}`
			fmt.Fprintf(w, "data: {\"id\":\"probe\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call\",\"type\":\"function\",\"function\":{\"name\":\"portway_probe\",\"arguments\":%q}}]},\"finish_reason\":\"tool_calls\"}]}\n\ndata: {\"id\":\"probe\",\"choices\":[],\"usage\":{\"prompt_tokens\":50,\"completion_tokens\":30,\"total_tokens\":80}}\n\ndata: [DONE]\n\n", args)
		}
	}))
	defer model.Close()
	cfg.AllowedEndpoints = append(append([]string{}, cfg.AllowedEndpoints...), model.URL)
	server := transport.New(cfg, app, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.TokenCounter = func(context.Context, string, []byte) (int, error) { return 100, nil } // explicit test-only counter
	api := httptest.NewServer(server.Handler())
	defer api.Close()
	call := func(method, path, body, key string, headers map[string]string) (int, response, []byte) {
		return request(t, api.Client(), api.URL, method, path, body, token, proof, key, headers)
	}
	providerBody := fmt.Sprintf(`{"label":"Gateway fixture","endpoint":%q,"model_id":"fixture-model"}`, model.URL)
	status, p, raw := call("POST", "/v1/providers", providerBody, "gateway-provider", nil)
	want(t, status, 201, raw)
	var provider struct {
		ID string `json:"id"`
	}
	json.Unmarshal(p.Data, &provider)
	status, _, raw = call("POST", "/v1/providers/"+provider.ID+"/capability-check", `{"profile":"chat-completions-v1"}`, "gateway-probe", map[string]string{"If-Match": "1"})
	want(t, status, 202, raw)
	configJSON := fmt.Sprintf(`{"instructions":"Return a concise answer","role_code":"coder","role_catalog_version":1,"provider_ref":%q}`, provider.ID)
	status, a, raw := call("POST", "/v1/agents", `{"name":"Gateway agent","config":`+configJSON+`}`, "gateway-agent", nil)
	want(t, status, 201, raw)
	var created struct {
		Agent agents.Agent `json:"agent"`
	}
	json.Unmarshal(a.Data, &created)
	version, err := server.Agents.GetVersion(ctx, created.Agent.CurrentVersionID)
	if err != nil {
		t.Fatal(err)
	}
	var owner string
	if err = app.Pool.QueryRow(ctx, `SELECT id FROM portway.owners WHERE installation_id=$1`, app.Installation).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	createdChains := []string{}
	defer func() {
		for _, id := range createdChains {
			_, err := store.Change(ctx, owner, id, func(c *execution.Chain, now time.Time) ([]execution.Event, error) { return c.Reconcile(now), nil })
			if err != nil {
				t.Error(err)
			}
		}
	}()
	create := func(key string) execution.Chain {
		chain, err := store.Create(ctx, owner, key, execution.CreateInput{Task: "private gateway task", Snapshot: execution.Snapshot{AgentVersionID: version.ID, ProviderID: provider.ID, ProviderVersion: 1, ConfigHash: version.ConfigHash, Config: version.Config, RepositoryRef: "fixture", Revision: strings.Repeat("a", 40)}})
		if err != nil {
			t.Fatal(err)
		}
		createdChains = append(createdChains, chain.ID)
		return chain
	}
	rpcHTTP := &http.Client{Timeout: 10 * time.Second, Transport: gatewayRoundTrip(func(r *http.Request) (*http.Response, error) {
		copy := r.Clone(r.Context())
		copy.URL, _ = url.Parse(api.URL + r.URL.Path)
		return http.DefaultTransport.RoundTrip(copy)
	})}
	rpc, err := modelgateway.NewRPC("http://api:8080", cfg.ExecutionKey, "Bearer "+token, proof, rpcHTTP)
	if err != nil {
		t.Fatal(err)
	}
	defer rpc.Close()
	t.Run("private-gateway-harness-result", func(t *testing.T) {
		chain := create("private-gateway-complete")
		runID := chain.Runs[0].ID
		before := generations.Load()
		out, err := (execution.Harness{Store: store, Gateway: rpc}).Advance(ctx, owner, runID)
		if err != nil || out.Runs[0].State != "completed" || out.Runs[0].Result == nil || out.Runs[0].Result.Summary != "agent final fixture" || generations.Load() != before+1 {
			t.Fatal(out.Runs, err, generations.Load())
		}
		var ciphertext, nonce []byte
		var state string
		if err = root.Pool.QueryRow(ctx, `SELECT state,ciphertext,nonce FROM portway.model_dispatches WHERE installation_id=$1 AND run_id=$2`, app.Installation, runID).Scan(&state, &ciphertext, &nonce); err != nil {
			t.Fatal(err)
		}
		if state != "finished" || strings.Contains(string(ciphertext), "agent final fixture") || len(nonce) != 12 {
			t.Fatal("dispatch reply not encrypted")
		}
		if _, err = store.Pool.Exec(ctx, `SELECT * FROM portway.model_dispatches`); err == nil {
			t.Fatal("execution can read control dispatch journal")
		}
	})
	t.Run("private-gateway-replay-and-uncertainty", func(t *testing.T) {
		chain := create("private-gateway-replay")
		runID := chain.Runs[0].ID
		chain, err = store.Change(ctx, owner, chain.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
			e, err := c.Start(runID, now)
			return []execution.Event{e}, err
		})
		if err != nil {
			t.Fatal(err)
		}
		before := generations.Load()
		count, err := rpc.InputTokens(ctx, chain.Runs[0])
		if err != nil || count != 100 || generations.Load() != before {
			t.Fatal(count, err)
		}
		status, receiptResponse, raw := call("POST", "/internal/v1/model-gateway/count", fmt.Sprintf(`{"run_id":%q}`, runID), "", map[string]string{"X-Portway-Service-Key": cfg.ExecutionKey})
		want(t, status, 200, raw)
		var receipt modelgateway.CountReceipt
		json.Unmarshal(receiptResponse.Data, &receipt)
		stepID := secure.ID()
		chain, err = store.Change(ctx, owner, chain.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
			return c.BeginTurn(runID, stepID, count, c.Runs[0].Limits.OutputTokens, now)
		})
		if err != nil {
			t.Fatal(err)
		}
		step := chain.Runs[0].Steps[0]
		for i := 0; i < 2; i++ {
			reply, err := rpc.Complete(ctx, chain.Runs[0], step)
			if err != nil || reply.Finish != "final" || generations.Load() != before+1 {
				t.Fatal("dispatch replay", reply, err, generations.Load())
			}
		}
		in := modelgateway.CompleteRequest{CountReceipt: receipt, StepID: stepID}
		in.RequestHash = strings.Repeat("b", 64)
		body, _ := json.Marshal(in)
		status, _, raw = call("POST", "/internal/v1/model-gateway/complete", string(body), "", map[string]string{"X-Portway-Service-Key": cfg.ExecutionKey})
		want(t, status, 409, raw)
		if _, err = root.Pool.Exec(ctx, `UPDATE portway.model_dispatches SET state='started',ciphertext=NULL,nonce=NULL WHERE installation_id=$1 AND step_id=$2`, app.Installation, stepID); err != nil {
			t.Fatal(err)
		}
		if _, err = rpc.Complete(ctx, chain.Runs[0], step); problem.Public(err).Code != "MODEL_DISPATCH_UNCERTAIN" || generations.Load() != before+1 {
			t.Fatal("uncertain dispatch retried", err)
		}
		if _, err = store.Stop(ctx, owner, runID, "private-gateway-stop", true); err != nil {
			t.Fatal(err)
		}
		if _, err = rpc.Complete(ctx, chain.Runs[0], step); err == nil || generations.Load() != before+1 {
			t.Fatal("stopped run dispatched", err)
		}
		_, err = store.Change(ctx, owner, chain.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) { return c.Reconcile(now), nil })
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("private-gateway-stop-in-flight", func(t *testing.T) {
		chain := create("private-gateway-in-flight")
		runID := chain.Runs[0].ID
		block.Store(true)
		defer block.Store(false)
		done := make(chan execution.Chain, 1)
		failed := make(chan error, 1)
		go func() {
			out, err := (execution.Harness{Store: store, Gateway: rpc}).Advance(ctx, owner, runID)
			if err != nil {
				failed <- err
				return
			}
			done <- out
		}()
		select {
		case <-entered:
		case err := <-failed:
			t.Fatal(err)
		case <-time.After(5 * time.Second):
			t.Fatal("model request did not start")
		}
		if _, err = store.Stop(ctx, owner, runID, "private-gateway-live-stop", true); err != nil {
			t.Fatal(err)
		}
		select {
		case out := <-done:
			if out.Runs[0].Result != nil || out.Runs[0].Budget.Reserved != 0 || out.Runs[0].State == "completed" {
				t.Fatal("stopped model revived run", out.Runs[0])
			}
		case err := <-failed:
			t.Fatal(err)
		case <-time.After(5 * time.Second):
			t.Fatal("stop did not cancel private model request")
		}
		_, err = store.Change(ctx, owner, chain.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) { return c.Reconcile(now), nil })
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("private-gateway-guards", func(t *testing.T) {
		before := generations.Load()
		status, _, raw := call("POST", "/internal/v1/model-gateway/count", fmt.Sprintf(`{"run_id":%q}`, secure.ID()), "", nil)
		want(t, status, 401, raw)
		// The production default does not supply a guessed tokenizer.
		closed := transport.New(cfg, app, slog.New(slog.NewTextHandler(io.Discard, nil)))
		closedAPI := httptest.NewServer(closed.Handler())
		defer closedAPI.Close()
		chain := create("private-gateway-no-counter")
		runID := chain.Runs[0].ID
		chain, err = store.Change(ctx, owner, chain.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
			e, err := c.Start(runID, now)
			return []execution.Event{e}, err
		})
		if err != nil {
			t.Fatal(err)
		}
		status, _, raw = request(t, closedAPI.Client(), closedAPI.URL, "POST", "/internal/v1/model-gateway/count", fmt.Sprintf(`{"run_id":%q}`, runID), token, proof, "", map[string]string{"X-Portway-Service-Key": cfg.ExecutionKey})
		want(t, status, 502, raw)
		if generations.Load() != before {
			t.Fatal("missing counter generated")
		}
		status, _, raw = call("PUT", "/v1/providers/"+provider.ID, providerBody, "gateway-provider-changed", map[string]string{"If-Match": "1"})
		want(t, status, 200, raw)
		if _, err = rpc.InputTokens(ctx, chain.Runs[0]); problem.Public(err).Code != "MODEL_BINDING_CHANGED" || generations.Load() != before {
			t.Fatal("stale provider pin dispatched", err)
		}
		if _, err = store.Stop(ctx, owner, runID, "private-gateway-cleanup", true); err != nil {
			t.Fatal(err)
		}
	})
}
