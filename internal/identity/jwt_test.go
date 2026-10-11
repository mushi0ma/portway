package identity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"portway/internal/secure"
)

func TestJWTValidationAndJWKSBound(t *testing.T) {
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	other, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	var calls atomic.Int32
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kid": "one", "kty": "EC", "alg": "ES256", "crv": "P-256", "x": base64.RawURLEncoding.EncodeToString(k.X.FillBytes(make([]byte, 32))), "y": base64.RawURLEncoding.EncodeToString(k.Y.FillBytes(make([]byte, 32)))}}})
	}))
	defer h.Close()
	s := NewSupabase(h.URL, "public")
	base := Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: h.URL, Audience: jwt.ClaimStrings{"authenticated"}, Subject: secure.ID(), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}, SessionID: secure.ID()}
	sign := func(c Claims, kid string, key *ecdsa.PrivateKey) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodES256, c)
		tok.Header["kid"] = kid
		raw, e := tok.SignedString(key)
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	if _, e := s.Verify(context.Background(), sign(base, "one", k)); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name string
		edit func(*Claims)
		kid  string
		key  *ecdsa.PrivateKey
	}{
		{"bad signature", func(*Claims) {}, "one", other},
		{"bad issuer", func(c *Claims) { c.Issuer = "https://attacker.example" }, "one", k},
		{"bad audience", func(c *Claims) { c.Audience = jwt.ClaimStrings{"other"} }, "one", k},
		{"expired", func(c *Claims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Second)) }, "one", k},
		{"no expiry", func(c *Claims) { c.ExpiresAt = nil }, "one", k},
		{"no session", func(c *Claims) { c.SessionID = "" }, "one", k},
		{"unknown kid", func(*Claims) {}, "missing", k},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			tc.edit(&c)
			if _, e := s.Verify(context.Background(), sign(c, tc.kid, tc.key)); e == nil {
				t.Fatal("invalid token accepted")
			}
		})
	}
	if calls.Load() != 1 {
		t.Fatalf("unbounded JWKS fetches: %d", calls.Load())
	}
	none := jwt.NewWithClaims(jwt.SigningMethodNone, base)
	raw, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, e := s.Verify(context.Background(), raw); e == nil {
		t.Fatal("alg none accepted")
	}
}
