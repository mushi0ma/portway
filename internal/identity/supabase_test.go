package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"portway/internal/problem"
)

func TestSignupUsesConfiguredCallback(t *testing.T) {
	callback := "https://tunnel.example/auth/confirm"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/signup" || r.URL.Query().Get("redirect_to") != callback {
			t.Error("wrong confirmation callback")
		}
		if r.Header.Get("apikey") != "publishable-fixture" {
			t.Error("missing publishable key")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"00000000-0000-4000-8000-000000000001"}`))
	}))
	defer server.Close()
	auth := NewSupabase(server.URL, "publishable-fixture")
	auth.ConfirmationURL = callback
	if _, err := auth.Signup(context.Background(), "owner@example.test", "fixture"); err != nil {
		t.Fatal(err)
	}
}

func TestAuthErrorDoesNotExposeProviderMessage(t *testing.T) {
	for _, tc := range []struct{ body, code string }{
		{`{"error_code":"email_not_confirmed","msg":"secret-address"}`, "EMAIL_NOT_CONFIRMED"},
		{`{"error_code":"invalid_credentials","msg":"secret-address"}`, "AUTH_REQUIRED"},
		{`{"error_code":"unknown","msg":"secret-address"}`, "AUTH_REQUIRED"},
		{`invalid upstream body`, "AUTH_REQUIRED"},
	} {
		t.Run(tc.code+tc.body[:1], func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(400)
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			_, err := NewSupabase(srv.URL, "public").Login(context.Background(), "owner@example.test", "test")
			if p := problem.Public(err); p.Code != tc.code || p.Message == "secret-address" {
				t.Fatalf("unsafe or incorrect error: %v", p)
			}
		})
	}
}
