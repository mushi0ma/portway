package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
