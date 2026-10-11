package execution

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"portway/internal/agents"
	"portway/internal/problem"
	"portway/internal/secure"
)

func fixture(t *testing.T) (Chain, time.Time) {
	t.Helper()
	now := time.Now().UTC()
	provider := secure.ID()
	cfg, err := agents.Validate(agents.Config{Instructions: "private task marker", RoleCode: "coder", RoleCatalogVersion: 1, ProviderRef: provider})
	if err != nil {
		t.Fatal(err)
	}
	c, e, err := NewChain(secure.ID(), secure.ID(), CreateInput{Task: "task-secret-marker", Snapshot: Snapshot{AgentVersionID: secure.ID(), ProviderID: provider, ConfigHash: agents.Hash(cfg), Config: cfg, RepositoryRef: "demo", Revision: strings.Repeat("a", 40)}}, now)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(e)
	if strings.Contains(string(b), "marker") {
		t.Fatal("event leaks private task")
	}
	return c, now
}
func TestReservationStopAndSettlement(t *testing.T) {
	c, now := fixture(t)
	id := c.Runs[0].ID
	if _, err := c.Start(id, now); err != nil {
		t.Fatal(err)
	}
	step, _, err := c.Reserve(id, secure.ID(), 100, 1000, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = c.Reserve(id, secure.ID(), 100, 1000, now); err == nil {
		t.Fatal("second concurrent model step admitted")
	}
	if _, err = c.Stop("", now); err != nil {
		t.Fatal(err)
	}
	epoch := c.StopEpoch
	if _, err = c.ConfirmStopped(id, now); err == nil {
		t.Fatal("cancelled with unsettled external usage")
	}
	if _, err = c.Stop("", now); err != nil || c.StopEpoch != epoch {
		t.Fatal("repeat stop changes fence", err)
	}
	if _, _, err = c.Reserve(id, secure.ID(), 100, 1000, now); err == nil {
		t.Fatal("reserve after stop")
	}
	if _, err = c.Settle(id, step.ID, 300, "actual", now); err != nil {
		t.Fatal(err)
	}
	if c.Runs[0].State != "stopping" {
		t.Fatal("late response revived run")
	}
	if e, err := c.Settle(id, step.ID, 300, "actual", now); err != nil || e != nil {
		t.Fatal("settlement not idempotent", err)
	}
	if _, err = c.Settle(id, step.ID, 301, "actual", now); err == nil {
		t.Fatal("conflicting usage accepted")
	}
	if _, err = c.ConfirmStopped(id, now); err != nil {
		t.Fatal(err)
	}
	if c.State != "cancelled" || c.Budget.Spent != 300 || c.Budget.Reserved != 0 {
		t.Fatalf("wrong outcome %+v", c.Budget)
	}
	if err = c.Validate(); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(c)
	if ev := c.Reconcile(now); len(ev) != 0 {
		t.Fatal("terminal reconciled")
	}
	after, _ := json.Marshal(c)
	if string(before) != string(after) {
		t.Fatal("terminal mutated")
	}
}
func TestUnknownAndRestartConserveBudget(t *testing.T) {
	for _, restart := range []bool{true, false} {
		c, now := fixture(t)
		id := c.Runs[0].ID
		_, _ = c.Start(id, now)
		step, _, err := c.Reserve(id, secure.ID(), 100, 1000, now)
		if err != nil {
			t.Fatal(err)
		}
		if restart {
			c.Reconcile(now)
		} else {
			_, err = c.Settle(id, step.ID, 0, "unknown", now)
			if err != nil {
				t.Fatal(err)
			}
		}
		if c.Budget.Spent != 1100 || c.Budget.Reserved != 0 || c.State != "failed" {
			t.Fatal("unknown usage refunded or run revived")
		}
		if err = c.Validate(); err != nil {
			t.Fatal(err)
		}
		if _, _, err = c.Reserve(id, secure.ID(), 0, 1, now); err == nil {
			t.Fatal("restarted failed run")
		}
	}
}
func TestLimitsDeadlineAndBudget(t *testing.T) {
	c, now := fixture(t)
	id := c.Runs[0].ID
	if _, err := c.Start(id, now.Add(900*time.Second)); err == nil {
		t.Fatal("deadline boundary admitted")
	}
	_, _ = c.Start(id, now)
	c.Budget.Limit = 500
	if _, _, err := c.Reserve(id, secure.ID(), 100, 401, now); err == nil {
		t.Fatal("chain cap exceeded")
	}
	if _, _, err := c.Reserve(id, secure.ID(), 100, 400, now); err != nil {
		t.Fatal(err)
	}
}
func TestServiceKeyDoesNotAuthorizeOwner(t *testing.T) {
	key := strings.Repeat("s", 32)
	calls := 0
	h := HTTPServer{ServiceKey: key, Store: Store{Installation: secure.ID()}, Authorize: func(context.Context, string, string) (Authority, error) {
		calls++
		return Authority{}, problem.New(401, "DEVICE_REVOKED", "Device revoked")
	}}.Handler()
	for _, provided := range []string{"", key} {
		r := httptest.NewRequest("POST", "/internal/v1/runs", strings.NewReader("{}"))
		r.Header.Set("X-Portway-Service-Key", provided)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	if calls != 1 {
		t.Fatal("wrong service key reached identity authority")
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestAuthorityChecksEveryRequestAndFailsClosed(t *testing.T) {
	calls := 0
	owner, device, installation := secure.ID(), secure.ID(), secure.ID()
	c := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("X-Portway-Service-Key") != "internal-key" || r.Header.Get("Authorization") != "Bearer owner" || r.Header.Get("X-Device-Session") != "proof" {
			t.Fatal("credentials not propagated")
		}
		status := 200
		body := `{"data":{"owner_id":"` + owner + `","device_id":"` + device + `","installation_id":"` + installation + `"}}`
		if calls == 2 {
			status = 401
			body = `{}`
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	auth := ControlAuthorizer("http://control.test", "internal-key", c)
	if a, err := auth(context.Background(), "Bearer owner", "proof"); err != nil || a.OwnerID != owner {
		t.Fatal(a, err)
	}
	if _, err := auth(context.Background(), "Bearer owner", "proof"); err == nil {
		t.Fatal("revoked device authorized from cache")
	}
}
func TestProductionAdmissionClosed(t *testing.T) {
	installation := secure.ID()
	key := strings.Repeat("s", 32)
	h := HTTPServer{ServiceKey: key, Store: Store{Installation: installation}, Authorize: func(context.Context, string, string) (Authority, error) {
		return Authority{OwnerID: secure.ID(), DeviceID: secure.ID(), InstallationID: installation}, nil
	}}.Handler()
	r := httptest.NewRequest("POST", "/internal/v1/runs", strings.NewReader("{}"))
	r.Header.Set("X-Portway-Service-Key", key)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "EXECUTION_NOT_READY") {
		t.Fatal(w.Code, w.Body.String())
	}
}
