package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"portway/internal/agents"
	"portway/internal/execution"
	"portway/internal/identity"
	"portway/internal/secure"
	"portway/internal/storage"
)

// Uses the actual control guard, private HTTP authority callback and execution
// role. Only Supabase Auth is a fixture; there is no production run admission.
func exerciseSSE(t *testing.T, ctx context.Context, store execution.Store, root *storage.Store, client *http.Client, base string, login identity.LoginResult, auth *fakeAuth) func() {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	handedOff := false
	defer func() {
		if !handedOff {
			cancel()
		}
	}()
	t.Cleanup(cancel)
	provider := secure.ID()
	cfg, err := agents.Validate(agents.Config{Instructions: "sse-instruction-marker", RoleCode: "coder", RoleCatalogVersion: 1, ProviderRef: provider})
	if err != nil {
		t.Fatal(err)
	}
	chain, err := store.Create(ctx, login.Owner.ID, "sse-fixture-create", execution.CreateInput{Task: "sse-task-marker", Snapshot: execution.Snapshot{AgentVersionID: secure.ID(), ProviderID: provider, Config: cfg, ConfigHash: agents.Hash(cfg), RepositoryRef: "fixture", Revision: strings.Repeat("a", 40)}})
	if err != nil {
		t.Fatal(err)
	}
	run := chain.Runs[0].ID
	path := "/v1/runs/" + run + "/events"
	call := func(method, path, body, token, proof, key string, headers map[string]string) (int, response, []byte) {
		return request(t, client, base, method, path, body, token, proof, key, headers)
	}
	token, proof := login.Session.AccessToken, login.DeviceProof
	for _, tc := range []struct {
		path, token, proof string
		headers            map[string]string
		code               int
		reason             string
	}{
		{path, "", "", nil, 401, "AUTH_REQUIRED"},
		{path + "?access_token=forbidden", token, proof, nil, 400, "INVALID_CURSOR"},
		{path, token, proof, map[string]string{"Last-Event-ID": secure.ID() + ":0"}, 400, "INVALID_CURSOR"},
		{path, token, proof, map[string]string{"Last-Event-ID": run + ":2"}, 409, "CURSOR_AHEAD"},
		{path + "?cursor=" + run + ":0", token, proof, map[string]string{"Last-Event-ID": run + ":0"}, 400, "INVALID_CURSOR"},
		{"/v1/runs/" + secure.ID() + "/events", token, proof, nil, 404, "NOT_FOUND"},
	} {
		code, _, b := call("GET", tc.path, "", tc.token, tc.proof, "", tc.headers)
		want(t, code, tc.code, b)
		if !strings.Contains(string(b), tc.reason) {
			t.Fatal("wrong SSE preflight error", string(b))
		}
	}
	code, snapshot, b := call("GET", "/v1/runs/"+run, "", token, proof, "", nil)
	want(t, code, 200, b)
	var view execution.RunSnapshot
	if json.Unmarshal(snapshot.Data, &view) != nil || view.SnapshotCursor != run+":1" {
		t.Fatal("missing consistent snapshot cursor")
	}
	type stream struct {
		response *http.Response
		frames   <-chan string
	}
	open := func(cursor, access string) stream {
		t.Helper()
		req, _ := http.NewRequestWithContext(ctx, "GET", base+path, nil)
		req.Header.Set("Authorization", "Bearer "+access)
		req.Header.Set("X-Device-Session", proof)
		if cursor != "" {
			req.Header.Set("Last-Event-ID", cursor)
		}
		// Client Close cancels the old request asynchronously. Wait for its
		// server-side slot to release before opening a replacement connection.
		var r *http.Response
		retryUntil := time.Now().Add(2 * time.Second)
		for {
			r, err = client.Do(req.Clone(ctx))
			if err != nil {
				t.Fatal(err)
			}
			if r.StatusCode != 429 || time.Now().After(retryUntil) {
				break
			}
			r.Body.Close()
			time.Sleep(20 * time.Millisecond)
		}
		t.Cleanup(func() { r.Body.Close() })
		if r.StatusCode != 200 || !strings.HasPrefix(r.Header.Get("Content-Type"), "text/event-stream") || r.Header.Get("X-Accel-Buffering") != "no" {
			t.Fatal("stream headers", r.StatusCode, r.Header)
		}
		frames := make(chan string, 32)
		go func() {
			defer close(frames)
			reader := bufio.NewScanner(r.Body)
			reader.Buffer(make([]byte, 4096), 2<<20)
			var frame strings.Builder
			for reader.Scan() {
				line := reader.Text()
				if line == "" {
					select {
					case frames <- frame.String():
					case <-ctx.Done():
						return
					}
					frame.Reset()
				} else {
					frame.WriteString(line + "\n")
				}
			}
		}()
		return stream{r, frames}
	}
	readEvent := func(s stream, seq int64) {
		t.Helper()
		deadline := time.NewTimer(4 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case frame, ok := <-s.frames:
				if !ok {
					t.Fatal("stream closed before event")
				}
				if strings.Contains(frame, "sse-task-marker") || strings.Contains(frame, "sse-instruction-marker") || strings.Contains(frame, proof) {
					t.Fatal("SSE leaked secret or instructions")
				}
				if strings.HasPrefix(frame, ":") {
					continue
				}
				if !strings.HasPrefix(frame, "id: "+execution.Cursor(run, seq)+"\n") {
					t.Fatal("duplicate, skipped or wrong event", frame)
				}
				return
			case <-deadline.C:
				t.Fatal("SSE event timed out")
			}
		}
	}
	s := open("", token)
	readEvent(s, 1)
	_, err = store.Change(ctx, login.Owner.ID, chain.ID, func(c *execution.Chain, now time.Time) ([]execution.Event, error) {
		e, err := c.Start(run, now)
		return []execution.Event{e}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	readEvent(s, 2)
	s.response.Body.Close()
	s = open(run+":1", token)
	readEvent(s, 2)
	// Four connections are allowed; the fifth returns 429 without opening SSE.
	extra := []stream{open(run+":2", token), open(run+":2", token), open(run+":2", token)}
	code, _, b = call("GET", path, "", token, proof, "", nil)
	want(t, code, 429, b)
	for _, e := range extra {
		e.response.Body.Close()
	}
	// Exercise the production 20-second request and 25-second write limits.
	time.Sleep(26 * time.Second)
	code, _, b = call("POST", "/v1/runs/"+run+"/stop", "{}", token, proof, "sse-stop-fixture", nil)
	want(t, code, 202, b)
	readEvent(s, 3)
	s.response.Body.Close()
	if err = store.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`UPDATE portway_execution.events SET created_at=clock_timestamp()-interval '8 days' WHERE installation_id=$1 AND run_id=$2`,
		`UPDATE portway_execution.outbox SET published_at=clock_timestamp()-interval '8 days' WHERE installation_id=$1 AND id IN (SELECT id FROM portway_execution.events WHERE installation_id=$1 AND run_id=$2)`,
	} {
		if _, err = root.Pool.Exec(ctx, sql, store.Installation, run); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = store.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	code, expired, b := call("GET", path, "", token, proof, "", nil)
	want(t, code, 410, b)
	if !strings.Contains(string(b), "HISTORY_EXPIRED") || expired.Error.Code != "HISTORY_EXPIRED" || !strings.Contains(string(b), run+":4") {
		t.Fatal("missing expired-history recovery cursor", string(b))
	}
	// A valid short-lived JWT must also close an already established stream.
	claims := identity.Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: auth.url, Subject: auth.subject, Audience: jwt.ClaimStrings{"authenticated"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(3 * time.Second))}, SessionID: auth.session}
	signed := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed.Header["kid"] = "test-key"
	short, err := signed.SignedString(auth.key)
	if err != nil {
		t.Fatal(err)
	}
	expiring := open(run+":4", short)
	closed := func(s stream) {
		t.Helper()
		deadline := time.NewTimer(5 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case _, ok := <-s.frames:
				if !ok {
					return
				}
			case <-deadline.C:
				t.Fatal("SSE did not close within five seconds")
			}
		}
	}
	closed(expiring)
	code, _, b = call("GET", path, "", short, proof, "", map[string]string{"Last-Event-ID": run + ":4"})
	want(t, code, 401, b)
	revoked := open(run+":4", token)
	handedOff = true
	return func() {
		t.Helper()
		defer cancel()
		closed(revoked)
		code, _, b := call("GET", path, "", token, proof, "", nil)
		want(t, code, 401, b)
		_, _ = io.Copy(io.Discard, revoked.response.Body)
	}
}
