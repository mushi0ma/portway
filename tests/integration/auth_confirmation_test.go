package integration

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"portway/internal/config"
	"portway/internal/secure"
	"portway/internal/storage"
	"portway/internal/transport"
)

func exerciseStandardConfirmation(t *testing.T, c config.Config, base *storage.Store) {
	t.Helper()
	ctx := context.Background()
	store := &storage.Store{Pool: base.Pool, Installation: secure.ID()}
	c.InstallationID = store.Installation
	proof := secure.Token()
	if err := store.EnsureInstallation(ctx, "Email confirmation test", secure.Hash(proof)); err != nil {
		t.Fatal(err)
	}
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	auth := &fakeAuth{subject: secure.ID(), session: secure.ID(), key: k}
	upstream := httptest.NewServer(auth)
	defer upstream.Close()
	auth.url, c.AuthURL = upstream.URL, upstream.URL
	api := httptest.NewServer(transport.New(c, store, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler())
	defer api.Close()
	client := api.Client()
	login := func() int {
		code, _, _ := request(t, client, api.URL, "POST", "/v1/auth/login", `{"email":" OWNER@EXAMPLE.TEST ","password":"password-marker"}`, "", "", "", nil)
		return code
	}
	assertUnclaimed := func() {
		t.Helper()
		var count int
		if err := store.Pool.QueryRow(ctx, `SELECT count(*) FROM portway.owners WHERE installation_id=$1`, store.Installation).Scan(&count); err != nil || count != 0 {
			t.Fatalf("unexpected owner count=%d err=%v", count, err)
		}
	}
	// A valid, confirmed Supabase account cannot claim an unreserved installation.
	auth.confirmed.Store(true)
	if login() != 403 {
		t.Fatal("unreserved account accepted")
	}
	assertUnclaimed()
	auth.confirmed.Store(false)
	code, _, b := request(t, client, api.URL, "POST", "/v1/auth/register", `{"name":"Owner","email":"owner@example.test","password":"password-marker"}`, "", "", "registration-standard-email", map[string]string{"X-Setup-Proof": proof})
	want(t, code, 202, b)
	if login() != 401 {
		t.Fatal("unconfirmed account accepted")
	}
	assertUnclaimed()
	// Opening a callback without an OTP only displays instructions, not a claim.
	r, err := client.Get(api.URL + "/auth/confirm")
	if err != nil {
		t.Fatal(err)
	}
	b, _ = io.ReadAll(r.Body)
	r.Body.Close()
	if r.StatusCode != 200 || !strings.HasPrefix(r.Header.Get("Content-Type"), "text/html") || r.Header.Get("Cache-Control") != "no-store" || !strings.Contains(r.Header.Get("Content-Security-Policy"), "default-src 'none'") || !strings.Contains(string(b), "history.replaceState") {
		t.Fatal("unsafe confirmation landing page")
	}
	assertUnclaimed()
	// Supabase's standard email confirms upstream; Portway receives no OTP.
	auth.confirmed.Store(true)
	reserved := auth.subject
	auth.subject = secure.ID()
	if login() != 403 {
		t.Fatal("different confirmed subject accepted")
	}
	assertUnclaimed()
	auth.subject = reserved
	// The email must still match the intent, even when the subject is correct.
	_, err = store.Pool.Exec(ctx, `UPDATE portway.registration_intents SET email_hash=$2 WHERE installation_id=$1`, store.Installation, secure.Hash("other@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	if login() != 403 {
		t.Fatal("mismatched email accepted")
	}
	assertUnclaimed()
	_, err = store.Pool.Exec(ctx, `UPDATE portway.registration_intents SET email_hash=$2 WHERE installation_id=$1`, store.Installation, secure.Hash("owner@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if login() != 200 {
				t.Error("confirmed login failed")
			}
		}()
	}
	wg.Wait()
	var count int
	if err = store.Pool.QueryRow(ctx, `SELECT count(*) FROM portway.audit_entries WHERE installation_id=$1 AND action='owner.bound'`, store.Installation).Scan(&count); err != nil || count != 1 {
		t.Fatalf("owner bound more than once: %d %v", count, err)
	}
	var status string
	var setup *string
	if err = store.Pool.QueryRow(ctx, `SELECT status,setup_hash FROM portway.installations WHERE id=$1`, store.Installation).Scan(&status, &setup); err != nil || status != "claimed" || setup != nil {
		t.Fatal("setup proof not consumed")
	}
}
