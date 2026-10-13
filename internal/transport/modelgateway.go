package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/execution"
	"portway/internal/identity"
	"portway/internal/modelgateway"
	"portway/internal/problem"
	"portway/internal/providercheck"
	"portway/internal/secure"
)

func (s *Server) modelService(r *http.Request) error {
	if len(s.Config.ExecutionKey) < 32 || !secure.Equal(s.Config.ExecutionKey, r.Header.Get("X-Portway-Service-Key")) {
		return problem.New(401, "SERVICE_AUTH_REQUIRED", "Service authentication required")
	}
	if r.URL.RawQuery != "" {
		return problem.Invalid("Query parameters are not supported")
	}
	return nil
}
func (s *Server) gatewayRun(ctx context.Context, r *http.Request, id string) (execution.Run, error) {
	var out struct {
		Data execution.RunSnapshot `json:"data"`
	}
	if !secure.UUID(id) || s.Config.ExecutionURL == "" {
		return out.Data.Run, problem.Invalid("Invalid gateway run")
	}
	bounded, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(bounded, "GET", s.Config.ExecutionURL+"/internal/v1/runs/"+id, nil)
	if err != nil {
		return out.Data.Run, problem.Unavailable()
	}
	req.Header.Set("Authorization", r.Header.Get("Authorization"))
	req.Header.Set("X-Device-Session", r.Header.Get("X-Device-Session"))
	req.Header.Set("X-Portway-Service-Key", s.Config.ExecutionKey)
	client := http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return out.Data.Run, problem.Unavailable()
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return out.Data.Run, problem.New(409, "MODEL_AUTHORITY_INVALIDATED", "Run authority is no longer valid")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, (2<<20)+1))
	if err != nil || len(raw) > 2<<20 || json.Unmarshal(raw, &out) != nil || out.Data.ID != id {
		return out.Data.Run, problem.Unavailable()
	}
	return out.Data.Run, nil
}
func (s *Server) prepareGateway(ctx context.Context, r *http.Request, runID, stepID string) (*modelgateway.Client, execution.Run, execution.Step, error) {
	run, err := s.gatewayRun(ctx, r, runID)
	step := execution.Step{}
	if err != nil {
		return nil, run, step, err
	}
	state := "running"
	if stepID != "" {
		state = "waiting_model"
	}
	if run.State != state || !time.Now().Before(run.Deadline) || run.Snapshot.ProviderVersion < 1 {
		return nil, run, step, problem.New(409, "MODEL_AUTHORITY_INVALIDATED", "Run is not ready for this model operation")
	}
	if stepID != "" {
		if len(run.Turns) == 0 || run.Turns[len(run.Turns)-1].StepID != stepID || run.Turns[len(run.Turns)-1].State != "waiting_model" {
			return nil, run, step, problem.Invalid("No current model turn")
		}
		for _, current := range run.Steps {
			if current.ID == stepID {
				step = current
			}
		}
		if step.ID == "" || step.Settled || step.Epoch != run.StopEpoch {
			return nil, run, step, problem.Invalid("No current model reservation")
		}
		// The reservation itself is not part of the counted prompt.
		run.Turns = run.Turns[:len(run.Turns)-1]
	}
	var endpoint, model, hash, currentVersion string
	var secret *string
	var version int
	var report providercheck.Report
	var now time.Time
	err = s.Store.Pool.QueryRow(ctx, `SELECT p.endpoint,p.model_id,p.secret_ref,p.config_version,p.capabilities,clock_timestamp(),v.config_hash,a.current_version_id
 FROM portway.providers p JOIN portway.agent_versions v ON v.installation_id=p.installation_id
 JOIN portway.agents a ON a.installation_id=v.installation_id AND a.id=v.agent_id
 WHERE p.installation_id=$1 AND p.id=$2 AND v.id=$3`, s.Store.Installation, run.Snapshot.ProviderID, run.Snapshot.AgentVersionID).Scan(&endpoint, &model, &secret, &version, &report, &now, &hash, &currentVersion)
	if err != nil {
		return nil, run, step, problem.New(409, "MODEL_BINDING_CHANGED", "Pinned model or agent configuration is unavailable")
	}
	if version != run.Snapshot.ProviderVersion || hash != run.Snapshot.ConfigHash || currentVersion != run.Snapshot.AgentVersionID || !report.ProtocolValid(run.Snapshot.ProviderID, endpoint, model, version, s.Config.AllowedEndpoints, now) {
		return nil, run, step, problem.New(409, "MODEL_BINDING_CHANGED", "Model report or pinned configuration is no longer current")
	}
	key := ""
	if secret != nil {
		var ciphertext, nonce []byte
		var keyVersion int
		err = s.Store.Pool.QueryRow(ctx, `SELECT ciphertext,nonce,key_version FROM portway.secrets WHERE installation_id=$1 AND id=$2 AND state='active'`, s.Store.Installation, *secret).Scan(&ciphertext, &nonce, &keyVersion)
		if err != nil || keyVersion != 1 || len(nonce) != 12 {
			return nil, run, step, problem.New(409, "PROVIDER_SECRET_UNAVAILABLE", "Provider secret is unavailable")
		}
		plain, err := s.Vault.Open(ciphertext, nonce, s.Store.Installation+":"+*secret+":1")
		if err != nil {
			return nil, run, step, problem.Unavailable()
		}
		key = string(plain)
		clear(plain)
	}
	client, err := modelgateway.New(modelgateway.Config{ProviderID: run.Snapshot.ProviderID, Endpoint: endpoint, AllowedEndpoints: s.Config.AllowedEndpoints, Model: model, APIKey: key, Count: s.TokenCounter, FreeOnly: s.Config.FreeModelsOnly})
	return client, run, step, err
}
func (s *Server) modelCount(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.modelService(r); err != nil {
		return 0, nil, err
	}
	var in modelgateway.CountRequest
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	client, run, _, err := s.prepareGateway(r.Context(), r, in.RunID, "")
	if err != nil {
		return 0, nil, err
	}
	defer client.Close()
	count, err := client.InputTokens(r.Context(), run)
	if err != nil {
		return 0, nil, err
	}
	hash, err := client.RequestHash(run)
	if err != nil {
		return 0, nil, err
	}
	return 200, modelgateway.CountReceipt{RunID: run.ID, Input: count, RequestHash: hash, ProviderVersion: run.Snapshot.ProviderVersion, Epoch: run.StopEpoch}, nil
}
func (s *Server) modelComplete(w http.ResponseWriter, r *http.Request, p identity.Principal) (int, any, error) {
	if err := s.modelService(r); err != nil {
		return 0, nil, err
	}
	var in modelgateway.CompleteRequest
	if err := Decode(w, r, &in); err != nil {
		return 0, nil, err
	}
	if !secure.UUID(in.StepID) {
		return 0, nil, problem.Invalid("Invalid model step")
	}
	client, run, step, err := s.prepareGateway(r.Context(), r, in.RunID, in.StepID)
	if err != nil {
		return 0, nil, err
	}
	defer client.Close()
	hash, err := client.RequestHash(run)
	if err != nil {
		return 0, nil, err
	}
	count, err := client.InputTokens(r.Context(), run)
	if err != nil {
		return 0, nil, err
	}
	if count != in.Input || count != step.Input || hash != in.RequestHash || in.ProviderVersion != run.Snapshot.ProviderVersion || in.Epoch != run.StopEpoch {
		return 0, nil, problem.New(409, "MODEL_COUNT_CHANGED", "Model request does not match its reservation")
	}
	cached, err := s.claimModel(r.Context(), p, in)
	if err != nil {
		return 0, nil, err
	}
	if cached != nil {
		return 200, cached, nil
	}
	callCtx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	// Recheck after durable claim as well as while a model is in flight. A stop,
	// revocation, config change, expired report or database failure cancels I/O.
	validate := func() error {
		fresh, _, _, err := s.prepareGateway(callCtx, r, in.RunID, in.StepID)
		if fresh != nil {
			fresh.Close()
		}
		return err
	}
	if err = validate(); err != nil {
		return 0, nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-callCtx.Done():
				return
			case <-ticker.C:
				if validate() != nil {
					cancel()
					return
				}
			}
		}
	}()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(125 * time.Second))
	reply, callErr := client.Complete(callCtx, run, step)
	if callErr == nil {
		callErr = validate()
	}
	cancel()
	<-done
	if callErr != nil {
		reply = execution.ModelReply{Finish: "error", UsageSource: "unknown"}
	}
	settleCtx, settleCancel := context.WithTimeout(context.WithoutCancel(r.Context()), 3*time.Second)
	defer settleCancel()
	if err = s.finishModel(settleCtx, in, reply); err != nil {
		return 0, nil, err
	}
	return 200, reply, nil
}
func dispatchHash(in modelgateway.CompleteRequest) string {
	raw, _ := json.Marshal(in)
	return secure.Hash(string(raw))
}
func (s *Server) claimModel(ctx context.Context, p identity.Principal, in modelgateway.CompleteRequest) (*execution.ModelReply, error) {
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT id FROM portway.installations WHERE id=$1 FOR UPDATE`, s.Store.Installation); err != nil {
		return nil, err
	}
	var id string
	if err = tx.QueryRow(ctx, `SELECT id FROM portway.device_sessions WHERE installation_id=$1 AND owner_id=$2 AND id=$3 AND revoked_at IS NULL FOR SHARE`, s.Store.Installation, p.OwnerID, p.DeviceID).Scan(&id); err != nil {
		return nil, problem.New(401, "DEVICE_REVOKED", "Device is no longer active")
	}
	var hash, state string
	var ciphertext, nonce []byte
	err = tx.QueryRow(ctx, `SELECT request_hash,state,ciphertext,nonce FROM portway.model_dispatches WHERE installation_id=$1 AND step_id=$2`, s.Store.Installation, in.StepID).Scan(&hash, &state, &ciphertext, &nonce)
	if err == nil {
		if hash != dispatchHash(in) {
			return nil, problem.New(409, "MODEL_DISPATCH_CONFLICT", "Step was claimed for a different request")
		}
		if state != "finished" {
			return nil, problem.New(409, "MODEL_DISPATCH_UNCERTAIN", "Step was already attempted; it will not be sent again")
		}
		if len(nonce) != 12 {
			return nil, problem.Unavailable()
		}
		plain, err := s.Vault.Open(ciphertext, nonce, s.Store.Installation+":"+in.StepID+":gateway:1")
		if err != nil {
			return nil, problem.Unavailable()
		}
		defer clear(plain)
		var reply execution.ModelReply
		if json.Unmarshal(plain, &reply) != nil {
			return nil, problem.Unavailable()
		}
		return &reply, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM portway.model_dispatches WHERE installation_id=$1 AND expires_at<=clock_timestamp()`, s.Store.Installation); err != nil {
		return nil, err
	}
	var total int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM portway.model_dispatches WHERE installation_id=$1`, s.Store.Installation).Scan(&total); err != nil {
		return nil, err
	}
	if total >= 10000 {
		return nil, problem.New(503, "GATEWAY_CAPACITY_LIMIT", "Gateway dispatch journal is full")
	}
	_, err = tx.Exec(ctx, `INSERT INTO portway.model_dispatches(installation_id,run_id,step_id,request_hash,state) VALUES($1,$2,$3,$4,'started')`, s.Store.Installation, in.RunID, in.StepID, dispatchHash(in))
	if err != nil {
		return nil, err
	}
	return nil, tx.Commit(ctx)
}
func (s *Server) finishModel(ctx context.Context, in modelgateway.CompleteRequest, reply execution.ModelReply) error {
	raw, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	ciphertext, nonce := s.Vault.Seal(string(raw), s.Store.Installation+":"+in.StepID+":gateway:1")
	tag, err := s.Store.Pool.Exec(ctx, `UPDATE portway.model_dispatches SET state='finished',ciphertext=$4,nonce=$5 WHERE installation_id=$1 AND step_id=$2 AND request_hash=$3 AND state='started'`, s.Store.Installation, in.StepID, dispatchHash(in), ciphertext, nonce)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return problem.New(409, "MODEL_DISPATCH_CONFLICT", "Dispatch settlement conflicted")
	}
	return nil
}
