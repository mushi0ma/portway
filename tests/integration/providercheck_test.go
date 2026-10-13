package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"portway/internal/config"
	"portway/internal/providercheck"
	"portway/internal/secure"
	"portway/internal/storage"
	"portway/internal/transport"
)

func exerciseProviderChecks(t *testing.T, cfg config.Config, app, root *storage.Store, token, proof, secret string) {
	t.Helper()
	ctx := context.Background()
	var requests atomic.Int32
	var mode atomic.Value
	mode.Store("ok")
	started, release := make(chan struct{}, 1), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		raw, _ := io.ReadAll(r.Body)
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer provider-secret-marker" || bytes.Contains(raw, []byte("provider-secret-marker")) {
			t.Error("wrong probe connection or secret in body")
		}
		var in struct {
			Choice json.RawMessage `json:"tool_choice"`
			Max    int             `json:"max_tokens"`
		}
		if json.Unmarshal(raw, &in) != nil || in.Max != 128 {
			t.Error("unbounded probe")
		}
		if mode.Load() == "auth" {
			w.WriteHeader(401)
			io.WriteString(w, "provider-secret-marker")
			return
		}
		if mode.Load() == "block" {
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if string(in.Choice) == `"none"` {
			io.WriteString(w, "data: {\"id\":\"probe\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"PORTWAY_PROBE_OK\"},\"finish_reason\":\"stop\"}]}\n\n")
		} else {
			args := `{"argv":["echo","PORTWAY_PROBE_OK"],"working_dir":".","timeout_seconds":1}`
			fmt.Fprintf(w, "data: {\"id\":\"probe\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"proposal\",\"type\":\"function\",\"function\":{\"name\":\"portway_probe\",\"arguments\":%q}}]},\"finish_reason\":\"tool_calls\"}]}\n\n", args)
		}
		io.WriteString(w, "data: {\"id\":\"probe\",\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":30,\"total_tokens\":130}}\n\ndata: [DONE]\n\n")
	}))
	defer model.Close()
	cfg.AllowedEndpoints = append(append([]string{}, cfg.AllowedEndpoints...), model.URL+"/v1")
	var logs bytes.Buffer
	var mu sync.Mutex
	api := httptest.NewServer(transport.New(cfg, app, slog.New(slog.NewJSONHandler(&lockedWriter{&mu, &logs}, nil))).Handler())
	defer api.Close()
	call := func(method, path, body, key string, version int) (int, response, []byte) {
		return request(t, api.Client(), api.URL, method, path, body, token, proof, key, map[string]string{"If-Match": fmt.Sprint(version)})
	}
	body := fmt.Sprintf(`{"label":"Probe fixture","endpoint":%q,"model_id":"fixture-m1","secret_ref":%q}`, model.URL+"/v1", secret)
	code, created, raw := call("POST", "/v1/providers", body, "probe-provider-create", 1)
	want(t, code, 201, raw)
	var provider struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Data, &provider); err != nil {
		t.Fatal(err)
	}
	path := "/v1/providers/" + provider.ID + "/capability-check"
	input := `{"profile":"chat-completions-v1"}`
	get := func() (providercheck.Report, bool) {
		code, v, raw := call("GET", "/v1/providers", "", "", 1)
		want(t, code, 200, raw)
		if bytes.Contains(raw, []byte("provider-secret-marker")) {
			t.Fatal("secret leaked in report")
		}
		var out struct {
			Items []struct {
				ID     string               `json:"id"`
				Report providercheck.Report `json:"capabilities"`
				Valid  bool                 `json:"protocol_valid"`
			} `json:"items"`
		}
		if err := json.Unmarshal(v.Data, &out); err != nil {
			t.Fatal(err)
		}
		for _, item := range out.Items {
			if item.ID == provider.ID {
				return item.Report, item.Valid
			}
		}
		t.Fatal("missing provider")
		return providercheck.Report{}, false
	}

	t.Run("provider-check-validation", func(t *testing.T) {
		code, _, raw := request(t, api.Client(), api.URL, "POST", path, input, "", "", "probe-no-auth", nil)
		want(t, code, 401, raw)
		for _, tc := range []struct {
			body, key       string
			version, status int
		}{
			{`{"profile":"arbitrary"}`, "probe-invalid-profile", 1, 400},
			{`{"profile":"chat-completions-v1","max_tokens":999}`, "probe-invalid-body", 1, 400},
			{input, "", 1, 400}, {input, "probe-invalid-version", 2, 409},
		} {
			code, _, raw := call("POST", path, tc.body, tc.key, tc.version)
			want(t, code, tc.status, raw)
		}
		if requests.Load() != 0 {
			t.Fatal("invalid request reached model")
		}
	})
	t.Run("provider-check-success-replay-and-invalidation", func(t *testing.T) {
		code, first, raw := call("POST", path, input, "probe-success", 1)
		want(t, code, 202, raw)
		report, valid := get()
		if !valid || !report.Protocol.Compatible() || report.Runnable || report.ConfigVersion != 1 || report.CheckedAt == nil || requests.Load() != 2 {
			t.Fatal(report, valid, requests.Load())
		}
		for i := 0; i < 3; i++ {
			code, replay, raw := call("POST", path, input, "probe-success", 1)
			want(t, code, 202, raw)
			if requests.Load() != 2 {
				t.Fatal("replay made another external request")
			}
			// PostgreSQL JSONB may reorder keys in the stored acknowledgement.
			var expected, actual map[string]any
			if json.Unmarshal(first.Data, &expected) != nil || json.Unmarshal(replay.Data, &actual) != nil || !reflect.DeepEqual(expected, actual) {
				t.Fatal("replay changed acknowledgement", string(first.Data), string(replay.Data))
			}
		}
		// Expiry is checked at read/use time, not just when writing a success.
		if _, err := root.Pool.Exec(ctx, `UPDATE portway.providers SET capabilities=jsonb_set(capabilities,'{expires_at}',to_jsonb(clock_timestamp()-interval '1 second')) WHERE id=$1`, provider.ID); err != nil {
			t.Fatal(err)
		}
		if _, valid := get(); valid {
			t.Fatal("expired report accepted")
		}
		code, _, raw = call("PUT", "/v1/providers/"+provider.ID, body, "probe-update", 1)
		want(t, code, 200, raw)
		report, valid = get()
		if valid || report.CheckID != "" {
			t.Fatal("configuration update retained report")
		}
		code, _, raw = call("POST", path, input, "probe-success", 2)
		want(t, code, 409, raw)
		if requests.Load() != 2 {
			t.Fatal("conflicting replay dispatched")
		}
	})
	t.Run("provider-check-error-does-not-leak-or-retry", func(t *testing.T) {
		mode.Store("auth")
		code, _, raw := call("POST", path, input, "probe-auth-error", 2)
		want(t, code, 202, raw)
		report, valid := get()
		if valid || report.Protocol.ErrorCode != "MODEL_AUTH_FAILED" || requests.Load() != 3 {
			t.Fatal(report, valid)
		}
		code, _, raw = call("POST", path, input, "probe-auth-error", 2)
		want(t, code, 202, raw)
		if requests.Load() != 3 {
			t.Fatal("failure retried model call")
		}
	})
	t.Run("provider-check-race-with-update", func(t *testing.T) {
		mode.Store("block")
		done := make(chan int, 1)
		go func() { code, _, _ := call("POST", path, input, "probe-blocked", 2); done <- code }()
		defer unblock()
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("probe did not start")
		}
		code, _, raw := call("POST", path, input, "probe-concurrent-new-key", 2)
		want(t, code, 409, raw)
		code, _, raw = call("POST", path, input, "probe-blocked", 2)
		want(t, code, 202, raw)
		if requests.Load() != 4 {
			t.Fatal("concurrent request duplicated probe")
		}
		// This finishes while model I/O is blocked: no DB lock spans generation.
		code, _, raw = call("PUT", "/v1/providers/"+provider.ID, strings.Replace(body, "fixture-m1", "fixture-m2", 1), "probe-update-race", 2)
		want(t, code, 200, raw)
		unblock()
		select {
		case status := <-done:
			if status != 202 {
				t.Fatal(status)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("probe did not finish")
		}
		report, valid := get()
		if valid || report.CheckID != "" || requests.Load() != 4 {
			t.Fatal("late old-config report was published")
		}
	})
	t.Run("provider-check-abandoned-attempt-is-not-replayed", func(t *testing.T) {
		// Simulate a committed reservation followed by process death before I/O.
		var owner, device string
		if err := root.Pool.QueryRow(ctx, `SELECT owner_id,id FROM portway.device_sessions WHERE installation_id=$1 AND revoked_at IS NULL LIMIT 1`, app.Installation).Scan(&owner, &device); err != nil {
			t.Fatal(err)
		}
		checkID := secure.ID()
		hashBody := []byte(`{"version":3,"profile":"chat-completions-v1"}`)
		_, _, err := app.Mutate(ctx, owner, device, "POST "+path, "probe-abandoned", secure.MAC(cfg.MasterKey, hashBody), func(tx pgx.Tx) (int, any, error) {
			raw, _ := json.Marshal(providercheck.Report{CheckID: checkID, State: "checking", Deadline: time.Now().Add(-time.Second)})
			_, err := tx.Exec(ctx, `UPDATE portway.providers SET capabilities=$2::jsonb WHERE id=$1`, provider.ID, string(raw))
			return 202, map[string]any{"check_id": checkID}, err
		})
		if err != nil {
			t.Fatal(err)
		}
		before := requests.Load()
		code, _, raw := call("POST", path, input, "probe-abandoned", 3)
		want(t, code, 202, raw)
		if requests.Load() != before {
			t.Fatal("abandoned attempt was replayed")
		}
		mode.Store("ok")
		code, _, raw = call("POST", path, input, "probe-after-abandoned", 3)
		want(t, code, 202, raw)
		report, valid := get()
		if !valid || report.CheckID == checkID || report.ModelID != "fixture-m2" || requests.Load() != before+2 {
			t.Fatal(report, valid)
		}
	})
	t.Run("provider-check-revocation-during-probe", func(t *testing.T) {
		entered, resume := make(chan struct{}), make(chan struct{})
		var once sync.Once
		resumeProbe := func() { once.Do(func() { close(resume) }) }
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			_, _ = io.Copy(io.Discard, r.Body)
			close(entered)
			select {
			case <-resume:
			case <-r.Context().Done():
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			args := `{"argv":["echo","PORTWAY_PROBE_OK"],"working_dir":".","timeout_seconds":1}`
			fmt.Fprintf(w, "data: {\"id\":\"probe\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"proposal\",\"type\":\"function\",\"function\":{\"name\":\"portway_probe\",\"arguments\":%q}}]},\"finish_reason\":\"tool_calls\"}]}\n\ndata: {\"id\":\"probe\",\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":30,\"total_tokens\":130}}\n\ndata: [DONE]\n\n", args)
		}))
		defer server.Close()
		defer resumeProbe()
		var owner string
		if err := root.Pool.QueryRow(ctx, `SELECT id FROM portway.owners WHERE installation_id=$1`, app.Installation).Scan(&owner); err != nil {
			t.Fatal(err)
		}
		device := secure.ID()
		if _, err := app.Pool.Exec(ctx, `INSERT INTO portway.device_sessions(id,installation_id,owner_id,proof_hash,auth_session_id,label) VALUES($1,$2,$3,$4,$5,'probe-revocation-fixture')`, device, app.Installation, owner, secure.Hash(secure.Token()), secure.ID()); err != nil {
			t.Fatal(err)
		}
		id := secure.ID()
		if _, err := app.Pool.Exec(ctx, `INSERT INTO portway.providers(id,installation_id,label,endpoint,model_id,secret_ref) VALUES($1,$2,'revocation fixture',$3,'fixture',$4)`, id, app.Installation, server.URL, secret); err != nil {
			t.Fatal(err)
		}
		vault, _ := secure.NewVault(cfg.MasterKey)
		svc := providercheck.Service{Store: app, Vault: vault, AllowedEndpoints: []string{server.URL}, RequestHashKey: cfg.MasterKey}
		done := make(chan error, 1)
		go func() { _, _, err := svc.Check(ctx, owner, device, id, 1, "probe-revoked-device"); done <- err }()
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("probe did not start")
		}
		if _, err := app.Pool.Exec(ctx, `UPDATE portway.device_sessions SET revoked_at=clock_timestamp() WHERE id=$1`, device); err != nil {
			t.Fatal(err)
		}
		resumeProbe()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("probe did not settle")
		}
		var report providercheck.Report
		if err := app.Pool.QueryRow(ctx, `SELECT capabilities FROM portway.providers WHERE id=$1`, id).Scan(&report); err != nil {
			t.Fatal(err)
		}
		if calls.Load() != 1 || report.Protocol.Requests != 1 || report.Protocol.ErrorCode != "DEVICE_REVOKED" || report.Protocol.Compatible() {
			t.Fatal("revoked device continued probe", report, calls.Load())
		}
	})
	mu.Lock()
	defer mu.Unlock()
	if strings.Contains(logs.String(), "provider-secret-marker") {
		t.Fatal("secret leaked to logs")
	}
}
