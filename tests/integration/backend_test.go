package integration

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"portway/internal/config"
	"portway/internal/identity"
	"portway/internal/secure"
	"portway/internal/storage"
	"portway/internal/transport"
)

type fakeAuth struct {
	url, subject, session string
	key                   *ecdsa.PrivateKey
	calls                 atomic.Int32
	confirmed             atomic.Bool
	fail                  atomic.Bool
}

func (f *fakeAuth) user() identity.User {
	u := identity.User{ID: f.subject, Email: "owner@example.test"}
	if f.confirmed.Load() {
		now := time.Now().UTC()
		u.EmailConfirmedAt = &now
	}
	return u
}
func (f *fakeAuth) token(subject, session string) string {
	c := identity.Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: f.url, Subject: subject, Audience: jwt.ClaimStrings{"authenticated"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}, SessionID: session}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, c)
	token.Header["kid"] = "test-key"
	raw, _ := token.SignedString(f.key)
	return raw
}
func (f *fakeAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if f.fail.Load() {
		w.WriteHeader(503)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/.well-known/jwks.json":
		json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kid": "test-key", "alg": "ES256", "kty": "EC", "crv": "P-256", "x": base64.RawURLEncoding.EncodeToString(f.key.X.FillBytes(make([]byte, 32))), "y": base64.RawURLEncoding.EncodeToString(f.key.Y.FillBytes(make([]byte, 32)))}}})
	case "/signup":
		f.calls.Add(1)
		json.NewEncoder(w).Encode(f.user())
	case "/verify":
		var in map[string]string
		json.NewDecoder(r.Body).Decode(&in)
		if in["token_hash"] != "valid-confirmation-token-hash" || in["type"] != "email" {
			w.WriteHeader(400)
			return
		}
		f.confirmed.Store(true)
		json.NewEncoder(w).Encode(identity.Session{User: f.user()})
	case "/token":
		var in map[string]string
		json.NewDecoder(r.Body).Decode(&in)
		if !f.confirmed.Load() || in["password"] != "password-marker" && in["refresh_token"] != "refresh-marker" {
			w.WriteHeader(400)
			return
		}
		json.NewEncoder(w).Encode(identity.Session{AccessToken: f.token(f.subject, f.session), RefreshToken: "refresh-marker", TokenType: "bearer", ExpiresIn: 3600, User: f.user()})
	case "/user":
		json.NewEncoder(w).Encode(f.user())
	case "/logout":
		if r.URL.Query().Get("scope") != "local" {
			w.WriteHeader(400)
			return
		}
		w.WriteHeader(204)
	default:
		w.WriteHeader(404)
	}
}

type response struct {
	Data  json.RawMessage `json:"data"`
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func request(t *testing.T, client *http.Client, base, method, path, body, token, proof, key string, headers map[string]string) (int, response, []byte) {
	t.Helper()
	r, err := http.NewRequest(method, base+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if proof != "" {
		r.Header.Set("X-Device-Session", proof)
	}
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	v, err := client.Do(r)
	if err != nil {
		t.Error(err)
		return 0, response{}, nil
	}
	defer v.Body.Close()
	b, _ := io.ReadAll(v.Body)
	var out response
	_ = json.Unmarshal(b, &out)
	return v.StatusCode, out, b
}
func want(t *testing.T, got, want int, b []byte) {
	t.Helper()
	if got != want {
		t.Fatalf("status=%d want=%d body=%s", got, want, b)
	}
}
func TestBackend(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("PORTWAY_REQUIRE_INTEGRATION") == "1" {
			t.Fatal("TEST_DATABASE_URL required")
		}
		t.Skip("use make docker-test for isolated PostgreSQL integration")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Path != "/portway_test" {
		t.Fatal("integration requires disposable database named portway_test")
	}
	ctx := context.Background()
	installation := secure.ID()
	root, err := storage.Open(ctx, dsn, installation)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err = root.Migrate(ctx); err != nil {
		t.Fatal("fresh migration", err)
	}
	if err = root.Migrate(ctx); err != nil {
		t.Fatal("repeat migration", err)
	}
	if err = root.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	app, err := storage.Open(ctx, os.Getenv("TEST_APP_DATABASE_URL"), installation)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if err = app.CheckApplicationRole(ctx); err != nil {
		t.Fatal("app role rejected", err)
	}
	if root.CheckApplicationRole(ctx) == nil {
		t.Fatal("privileged role accepted for API")
	}
	setup := secure.Token()
	if err = app.EnsureInstallation(ctx, "Integration", secure.Hash(setup)); err != nil {
		t.Fatal(err)
	}
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	auth := &fakeAuth{subject: secure.ID(), session: secure.ID(), key: k}
	upstream := httptest.NewServer(auth)
	defer upstream.Close()
	auth.url = upstream.URL
	master := bytes.Repeat([]byte{0x17}, 32)
	c := config.Config{Environment: "test", InstallationID: installation, InstallationName: "Integration", AuthURL: auth.url, AuthKey: "public", MasterKey: master, AllowedEndpoints: []string{"http://model:11434/v1"}}
	var logs bytes.Buffer
	var logMu sync.Mutex
	logger := slog.New(slog.NewJSONHandler(&lockedWriter{&logMu, &logs}, nil))
	api := httptest.NewServer(transport.New(c, app, logger).Handler())
	defer api.Close()
	client := api.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	call := func(method, path, body, token, proof, key string, headers map[string]string) (int, response, []byte) {
		return request(t, client, api.URL, method, path, body, token, proof, key, headers)
	}
	code, _, b := call("GET", "/healthz", "", "", "", "", nil)
	want(t, code, 200, b)
	code, _, b = call("GET", "/readyz", "", "", "", "", nil)
	want(t, code, 503, b)
	code, _, b = call("GET", "/v1/agents", "", "", "", "", nil)
	want(t, code, 401, b)
	body := `{"name":"Owner","email":"owner@example.test","password":"password-marker"}`
	code, _, b = call("POST", "/v1/auth/register", body, "", "", "registration-key", nil)
	want(t, code, 403, b)
	code, first, b := call("POST", "/v1/auth/register", body, "", "", "registration-key", map[string]string{"X-Setup-Proof": setup})
	want(t, code, 202, b)
	code, replay, b := call("POST", "/v1/auth/register", body, "", "", "registration-key", map[string]string{"X-Setup-Proof": setup})
	want(t, code, 202, b)
	if !bytes.Equal(first.Data, replay.Data) || auth.calls.Load() != 1 {
		t.Fatal("signup replay created another external action")
	}
	code, _, b = call("POST", "/v1/auth/login", `{"email":"owner@example.test","password":"password-marker"}`, "", "", "", nil)
	want(t, code, 401, b)
	code, _, b = call("GET", "/auth/confirm?token_hash=valid-confirmation-token-hash&type=email&next=https://attacker.example", "", "", "", "", nil)
	want(t, code, 400, b)
	r, err := client.Get(api.URL + "/auth/confirm?token_hash=valid-confirmation-token-hash&type=email")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 303 || r.Header.Get("Location") != "portway://auth/confirmed" {
		t.Fatal("callback leaked credentials or wrong redirect")
	}
	code, _, b = call("POST", "/v1/auth/register", body, "", "", "other-register-key", map[string]string{"X-Setup-Proof": setup})
	want(t, code, 409, b)
	login := func() identity.LoginResult {
		code, v, b := call("POST", "/v1/auth/login", `{"email":"owner@example.test","password":"password-marker","device_label":"CI"}`, "", "", "", nil)
		want(t, code, 200, b)
		var result identity.LoginResult
		if err := json.Unmarshal(v.Data, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	one, two := login(), login()
	token, proof := one.Session.AccessToken, one.DeviceProof
	code, _, b = call("GET", "/v1/me", "", token, proof, "", nil)
	want(t, code, 200, b)
	code, _, b = call("GET", "/v1/me", "", auth.token(secure.ID(), auth.session), proof, "", nil)
	want(t, code, 403, b)
	code, _, b = call("GET", "/v1/me", "", auth.token(auth.subject, secure.ID()), proof, "", nil)
	want(t, code, 403, b)
	code, _, b = call("POST", "/v1/auth/refresh", `{"refresh_token":"refresh-marker"}`, "", proof, "", nil)
	want(t, code, 200, b)
	code, _, b = call("GET", "/v1/roles", "", token, proof, "", nil)
	want(t, code, 200, b)
	if bytes.Count(b, []byte(`"catalog_version":1`)) != 9 {
		t.Fatalf("role catalog wrong: %s", b)
	}
	secretBody := `{"alias":"model_key","value":"provider-secret-marker"}`
	code, secret, b := call("POST", "/v1/secret-refs", secretBody, token, proof, "secret-key-1", nil)
	want(t, code, 201, b)
	if bytes.Contains(b, []byte("provider-secret-marker")) {
		t.Fatal("secret write response leaked value")
	}
	var secretRef struct {
		ID string `json:"id"`
	}
	json.Unmarshal(secret.Data, &secretRef)
	var ciphertext, nonce []byte
	if err = app.Pool.QueryRow(ctx, `SELECT ciphertext,nonce FROM portway.secrets WHERE id=$1`, secretRef.ID).Scan(&ciphertext, &nonce); err != nil {
		t.Fatal(err)
	}
	vault, _ := secure.NewVault(master)
	plain, err := vault.Open(ciphertext, nonce, installation+":"+secretRef.ID+":1")
	if err != nil || string(plain) != "provider-secret-marker" {
		t.Fatal("vault DB roundtrip failed")
	}
	provider := fmt.Sprintf(`{"label":"Local","endpoint":"http://model:11434/v1","model_id":"test-model","secret_ref":%q}`, secretRef.ID)
	code, _, b = call("POST", "/v1/providers", strings.Replace(provider, "http://model:11434/v1", "http://169.254.169.254", 1), token, proof, "provider-bad", nil)
	want(t, code, 403, b)
	code, prov, b := call("POST", "/v1/providers", provider, token, proof, "provider-good", nil)
	want(t, code, 201, b)
	var pv struct {
		ID string `json:"id"`
	}
	json.Unmarshal(prov.Data, &pv)
	configJSON := fmt.Sprintf(`{"instructions":"Implement a focused change","role_code":"coder","role_catalog_version":1,"tools":[{"alias":"read","kind":"workspace.read"}],"provider_ref":%q}`, pv.ID)
	agentBody := `{"name":"Coder","config":` + configJSON + `}`
	code, _, b = call("POST", "/v1/agents", agentBody, token, proof, "", nil)
	want(t, code, 400, b)
	code, _, b = call("POST", "/v1/agents", `{"name":"bad","config":`+strings.Replace(configJSON, `"role_code":"coder"`, `"role_code":"reviewer","policy":{"workspace.write":"allow"}`, 1)+`}`, token, proof, "escalation-key", nil)
	want(t, code, 400, b)
	var wg sync.WaitGroup
	responses := make(chan response, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, v, raw := request(t, client, api.URL, "POST", "/v1/agents", agentBody, token, proof, "parallel-agent-key", nil)
			if status != 201 {
				t.Errorf("parallel create: %d %s", status, raw)
			}
			responses <- v
		}()
	}
	wg.Wait()
	close(responses)
	var created struct {
		Agent struct {
			ID               string `json:"id"`
			CurrentVersionID string `json:"current_version_id"`
		} `json:"agent"`
	}
	for v := range responses {
		var next struct {
			Agent struct {
				ID               string `json:"id"`
				CurrentVersionID string `json:"current_version_id"`
			} `json:"agent"`
		}
		json.Unmarshal(v.Data, &next)
		if created.Agent.ID != "" && next.Agent.ID != created.Agent.ID {
			t.Fatal("idempotency duplicate")
		}
		created = next
	}
	code, _, b = call("POST", "/v1/agents", strings.Replace(agentBody, "Coder", "Other", 1), token, proof, "parallel-agent-key", nil)
	want(t, code, 409, b)
	var count int
	app.Pool.QueryRow(ctx, `SELECT count(*) FROM portway.agents WHERE installation_id=$1`, installation).Scan(&count)
	if count != 1 {
		t.Fatalf("created %d agents", count)
	}
	versionPath := "/v1/agent-versions/" + created.Agent.CurrentVersionID
	code, old, b := call("GET", versionPath, "", token, proof, "", nil)
	want(t, code, 200, b)
	versionBody := fmt.Sprintf(`{"base_version_id":%q,"config":%s}`, created.Agent.CurrentVersionID, strings.Replace(configJSON, "focused change", "different change", 1))
	var successes, conflicts atomic.Int32
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			status, _, raw := request(t, client, api.URL, "POST", "/v1/agents/"+created.Agent.ID+"/versions", versionBody, token, proof, fmt.Sprintf("version-key-%d", i), map[string]string{"If-Match": "1"})
			switch status {
			case 201:
				successes.Add(1)
			case 409:
				conflicts.Add(1)
			default:
				t.Errorf("version race: %d %s", status, raw)
			}
		}(i)
	}
	wg.Wait()
	if successes.Load() != 1 || conflicts.Load() != 1 {
		t.Fatal("stale update was not rejected")
	}
	code, unchanged, b := call("GET", versionPath, "", token, proof, "", nil)
	want(t, code, 200, b)
	if !bytes.Equal(old.Data, unchanged.Data) {
		t.Fatal("historical version changed")
	}
	if _, err = app.Pool.Exec(ctx, `UPDATE portway.agent_versions SET config_hash=repeat('0',64) WHERE id=$1`, created.Agent.CurrentVersionID); err == nil {
		t.Fatal("app can mutate immutable version")
	}
	if _, err = root.Pool.Exec(ctx, `UPDATE portway.agent_versions SET config_hash=repeat('0',64) WHERE id=$1`, created.Agent.CurrentVersionID); err == nil {
		t.Fatal("DB trigger did not reject mutation")
	}
	for _, sql := range []string{`DELETE FROM portway.audit_entries`, `UPDATE portway.schema_migrations SET checksum='x'`, `CREATE TABLE portway.untrusted(id int)`} {
		if _, err = app.Pool.Exec(ctx, sql); err == nil {
			t.Fatalf("app role can execute %s", sql)
		}
	}
	anonymous, err := root.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = anonymous.Exec(ctx, `SET LOCAL ROLE portway_anonymous`); err != nil {
		t.Fatal(err)
	}
	if _, err = anonymous.Exec(ctx, `SELECT * FROM portway.agents`); err == nil {
		t.Fatal("anonymous schema exposure")
	}
	anonymous.Rollback(ctx)
	otherInstallation := secure.ID()
	otherStore, err := storage.Open(ctx, os.Getenv("TEST_APP_DATABASE_URL"), otherInstallation)
	if err != nil {
		t.Fatal(err)
	}
	defer otherStore.Close()
	if err = otherStore.EnsureInstallation(ctx, "Other", secure.Hash(secure.Token())); err != nil {
		t.Fatal(err)
	}
	otherConfig := c
	otherConfig.InstallationID = otherInstallation
	otherAPI := httptest.NewServer(transport.New(otherConfig, otherStore, logger).Handler())
	defer otherAPI.Close()
	code, _, b = request(t, client, otherAPI.URL, "GET", versionPath, "", token, proof, "", nil)
	want(t, code, 401, b)
	for _, path := range []string{"/v1/secret-refs", "/v1/providers", "/v1/audit", "/v1/devices", "/v1/status"} {
		code, _, b = call("GET", path, "", token, proof, "", nil)
		want(t, code, 200, b)
		for _, marker := range []string{"provider-secret-marker", "password-marker", "refresh-marker", proof, setup} {
			if bytes.Contains(b, []byte(marker)) {
				t.Fatalf("leak in %s", path)
			}
		}
	}
	code, _, b = call("DELETE", "/v1/devices/"+one.DeviceID, "", two.Session.AccessToken, two.DeviceProof, "revoke-device-one", nil)
	want(t, code, 200, b)
	code, _, b = call("GET", "/v1/me", "", token, proof, "", nil)
	want(t, code, 401, b)
	code, _, b = call("POST", "/v1/auth/refresh", `{"refresh_token":"refresh-marker"}`, "", proof, "", nil)
	want(t, code, 401, b)
	auth.fail.Store(true)
	code, _, b = call("POST", "/v1/auth/logout", "", two.Session.AccessToken, two.DeviceProof, "", nil)
	want(t, code, 200, b)
	if !bytes.Contains(b, []byte(`"device_revoked":true`)) {
		t.Fatal("offline upstream prevented local revoke")
	}
	code, _, b = call("GET", "/v1/me", "", two.Session.AccessToken, two.DeviceProof, "", nil)
	want(t, code, 401, b)
	logMu.Lock()
	defer logMu.Unlock()
	for _, marker := range []string{"provider-secret-marker", "password-marker", "refresh-marker", proof, setup, "valid-confirmation-token-hash"} {
		if strings.Contains(logs.String(), marker) {
			t.Fatal("secret marker leaked to logs")
		}
	}
}

type lockedWriter struct {
	mu     *sync.Mutex
	buffer *bytes.Buffer
}

func (w *lockedWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(b)
}

func TestMigrationChecksum(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("PORTWAY_REQUIRE_INTEGRATION") == "1" {
			t.Fatal("database required")
		}
		t.Skip("requires PostgreSQL")
	}
	ctx := context.Background()
	s, err := storage.Open(ctx, dsn, secure.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var checksum string
	if err = s.Pool.QueryRow(ctx, `SELECT checksum FROM portway.schema_migrations WHERE version=1`).Scan(&checksum); err != nil {
		t.Fatal(err)
	}
	defer s.Pool.Exec(ctx, `UPDATE portway.schema_migrations SET checksum=$1 WHERE version=1`, checksum)
	if _, err = s.Pool.Exec(ctx, `UPDATE portway.schema_migrations SET checksum='tampered' WHERE version=1`); err != nil {
		t.Fatal(err)
	}
	if s.Ready(ctx) == nil || s.Migrate(ctx) == nil {
		t.Fatal("changed migration was accepted")
	}
}
