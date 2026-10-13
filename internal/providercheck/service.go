package providercheck

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/modelgateway"
	"portway/internal/problem"
	"portway/internal/secure"
	"portway/internal/storage"
)

type Service struct {
	Store            *storage.Store
	Vault            *secure.Vault
	AllowedEndpoints []string
	RequestHashKey   []byte
	FreeModelsOnly   bool
}

// Check durably reserves an attempt before any generation. All responses are
// replayable 202 acknowledgements; GET providers reads the current report.
// A crash/cancel never replays an uncertain model call. A new key can start a
// fresh bounded check once the old attempt's deadline has elapsed.
func (s Service) Check(ctx context.Context, owner, device, provider string, version int, key string) (int, json.RawMessage, error) {
	var report Report
	var client *modelgateway.Client
	defer func() {
		if client != nil {
			client.Close()
		}
	}()
	body, _ := json.Marshal(struct {
		Version int    `json:"version"`
		Profile string `json:"profile"`
	}{version, modelgateway.ProbeProfile})
	code, ack, err := s.Store.Mutate(ctx, owner, device, "POST /v1/providers/"+provider+"/capability-check", key, secure.MAC(s.RequestHashKey, body), func(tx pgx.Tx) (int, any, error) {
		var current int
		var endpoint, model string
		var secret *string
		var prior Report
		err := tx.QueryRow(ctx, `SELECT endpoint,model_id,secret_ref,config_version,capabilities FROM portway.providers WHERE installation_id=$1 AND id=$2 FOR UPDATE`, s.Store.Installation, provider).Scan(&endpoint, &model, &secret, &current, &prior)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil, problem.NotFound()
		}
		if err != nil {
			return 0, nil, err
		}
		if current != version {
			return 0, nil, problem.New(409, "VERSION_CONFLICT", "Provider configuration changed")
		}
		var now time.Time
		if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
			return 0, nil, err
		}
		if prior.State == "checking" && now.Before(prior.Deadline) {
			return 0, nil, problem.New(409, "CAPABILITY_CHECK_IN_PROGRESS", "A provider check is already in progress")
		}
		apiKey := ""
		if secret != nil {
			var ciphertext, nonce []byte
			var keyVersion int
			err = tx.QueryRow(ctx, `SELECT ciphertext,nonce,key_version FROM portway.secrets WHERE installation_id=$1 AND id=$2 AND state='active'`, s.Store.Installation, *secret).Scan(&ciphertext, &nonce, &keyVersion)
			if err != nil || keyVersion != 1 || s.Vault == nil || len(nonce) != 12 {
				return 0, nil, problem.New(409, "PROVIDER_SECRET_UNAVAILABLE", "Provider secret is unavailable")
			}
			plain, err := s.Vault.Open(ciphertext, nonce, s.Store.Installation+":"+*secret+":1")
			if err != nil {
				return 0, nil, problem.New(409, "PROVIDER_SECRET_UNAVAILABLE", "Provider secret is unavailable")
			}
			apiKey = string(plain)
			clear(plain)
		}
		client, err = modelgateway.New(modelgateway.Config{ProviderID: provider, Endpoint: endpoint, Model: model, APIKey: apiKey, AllowedEndpoints: s.AllowedEndpoints, FreeOnly: s.FreeModelsOnly})
		if err != nil {
			return 0, nil, err
		}
		report = Report{CheckID: secure.ID(), ProviderID: provider, Endpoint: endpoint, ModelID: model, ConfigVersion: version, Profile: modelgateway.ProbeProfile, State: "checking", StartedAt: now, Deadline: now.Add(AttemptTTL)}
		raw, err := json.Marshal(report)
		if err != nil {
			return 0, nil, err
		}
		_, err = tx.Exec(ctx, `UPDATE portway.providers SET capabilities=$3::jsonb,checked_at=NULL WHERE installation_id=$1 AND id=$2`, s.Store.Installation, provider, string(raw))
		if err == nil {
			err = s.Store.Audit(ctx, tx, owner, "provider.check_started", "provider", provider)
		}
		return 202, map[string]any{"check_id": report.CheckID, "provider_id": provider, "config_version": version, "profile": report.Profile, "status_url": "/v1/providers"}, err
	})
	if err != nil || client == nil {
		return code, ack, err
	}
	probeCtx, cancel := context.WithDeadline(ctx, report.Deadline)
	report.Protocol = client.Probe(probeCtx, func(ctx context.Context) error {
		var current bool
		err := s.Store.Pool.QueryRow(ctx, `SELECT EXISTS(
		 SELECT 1 FROM portway.providers p JOIN portway.device_sessions d ON d.installation_id=p.installation_id
		 WHERE p.installation_id=$1 AND p.id=$2 AND p.config_version=$3
		 AND p.capabilities->>'check_id'=$4 AND p.capabilities->>'state'='checking'
		 AND d.owner_id=$5 AND d.id=$6 AND d.revoked_at IS NULL
		 AND clock_timestamp()<$7)`, s.Store.Installation, provider, version, report.CheckID, owner, device, report.Deadline).Scan(&current)
		if err != nil || !current {
			return problem.New(409, "CAPABILITY_CHECK_INVALIDATED", "Provider check is no longer current")
		}
		return nil
	})
	cancel()
	// Settle after client disconnect too. Failure leaves an expiring checking
	// record, never an implicit success or a retry of an uncertain request.
	settleCtx, settleCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer settleCancel()
	if err = s.finish(settleCtx, owner, device, report); err != nil {
		return 0, nil, err
	}
	return code, ack, nil
}

func (s Service) finish(ctx context.Context, owner, device string, report Report) error {
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	// Same order as configuration/revocation commands. No locks span model I/O.
	if _, err = tx.Exec(ctx, `SELECT id FROM portway.installations WHERE id=$1 FOR UPDATE`, s.Store.Installation); err != nil {
		return err
	}
	var id string
	err = tx.QueryRow(ctx, `SELECT id FROM portway.device_sessions WHERE installation_id=$1 AND owner_id=$2 AND id=$3 AND revoked_at IS NULL FOR SHARE`, s.Store.Installation, owner, device).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		report.Protocol = modelgateway.ProbeResult{Requests: report.Protocol.Requests, ErrorCode: "DEVICE_REVOKED"}
	} else if err != nil {
		return err
	}
	var now time.Time
	if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return err
	}
	if !now.Before(report.Deadline) {
		report.Protocol = modelgateway.ProbeResult{Requests: report.Protocol.Requests, ErrorCode: "CAPABILITY_CHECK_EXPIRED"}
	}
	report.State, report.CheckedAt = "completed", &now
	expires := now.Add(ReportTTL)
	report.ExpiresAt = &expires
	// Config changes and newer attempts win over late replies from older calls.
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	changed, err := tx.Exec(ctx, `UPDATE portway.providers SET capabilities=$4::jsonb,checked_at=$5 WHERE installation_id=$1 AND id=$2 AND config_version=$3 AND capabilities->>'check_id'=$6 AND capabilities->>'state'='checking'`, s.Store.Installation, report.ProviderID, report.ConfigVersion, string(raw), now, report.CheckID)
	if err != nil {
		return err
	}
	if changed.RowsAffected() == 1 {
		if err = s.Store.Audit(ctx, tx, owner, "provider.check_completed", "provider", report.ProviderID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
