package identity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"portway/internal/problem"
	"portway/internal/secure"
)

type Claims struct {
	jwt.RegisteredClaims
	SessionID string `json:"session_id"`
}
type key struct {
	Value     any
	Algorithm string
}
type Verifier struct {
	auth    *Supabase
	mu      sync.Mutex
	keys    map[string]key
	checked time.Time
	lastErr error
}
type jwk struct {
	KID string `json:"kid"`
	KTY string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	X   string `json:"x"`
	Y   string `json:"y"`
	CRV string `json:"crv"`
}

func (v *Verifier) Verify(ctx context.Context, raw string) (Claims, error) {
	var c Claims
	if len(raw) > 16384 {
		return c, problem.New(401, "AUTH_REQUIRED", "Invalid access token")
	}
	opts := []jwt.ParserOption{jwt.WithValidMethods([]string{"RS256", "ES256", "HS256"}), jwt.WithIssuer(v.auth.URL), jwt.WithAudience("authenticated"), jwt.WithExpirationRequired(), jwt.WithIssuedAt()}
	unverified, _, err := jwt.NewParser().ParseUnverified(raw, &c)
	if err != nil {
		return c, problem.New(401, "AUTH_REQUIRED", "Invalid access token")
	}
	if unverified.Method.Alg() == "HS256" {
		// Legacy shared signing secrets never enter Portway; Auth validates the signature.
		u, err := v.auth.User(ctx, raw)
		if err != nil {
			return c, err
		}
		if err = jwt.NewValidator(opts...).Validate(&c); err != nil || u.ID != c.Subject || u.EmailConfirmedAt == nil {
			return c, problem.New(401, "AUTH_REQUIRED", "Invalid access token")
		}
	} else {
		_, err = jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
			kid, ok := t.Header["kid"].(string)
			if !ok || len(kid) > 200 {
				return nil, errors.New("missing key")
			}
			return v.publicKey(ctx, kid, t.Method.Alg())
		}, opts...)
		if err != nil {
			var dependency *problem.Error
			if errors.As(err, &dependency) && dependency.Status >= 500 {
				return c, dependency
			}
			if errors.Is(err, jwt.ErrTokenExpired) {
				return c, problem.New(401, "TOKEN_EXPIRED", "Access token expired")
			}
			return c, problem.New(401, "AUTH_REQUIRED", "Access token could not be verified")
		}
	}
	if !secure.UUID(c.Subject) || !secure.UUID(c.SessionID) {
		return c, problem.New(401, "AUTH_REQUIRED", "Invalid token identity")
	}
	return c, nil
}
func (v *Verifier) publicKey(ctx context.Context, kid, alg string) (any, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	k, ok := v.keys[kid]
	age := time.Since(v.checked)
	if ok && age < 5*time.Minute && k.Algorithm == alg {
		return k.Value, nil
	}
	if !ok && age < time.Minute {
		if v.lastErr != nil {
			return nil, v.lastErr
		}
		return nil, errors.New("unknown signing key")
	}
	v.checked = time.Now()
	v.keys = nil
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := v.auth.call(ctx, "GET", "/.well-known/jwks.json", "", nil, &set); err != nil {
		v.lastErr = err
		return nil, err
	}
	v.lastErr = nil
	keys := map[string]key{}
	for _, j := range set.Keys {
		if j.Use != "" && j.Use != "sig" {
			continue
		}
		decode := func(s string) *big.Int {
			b, err := base64.RawURLEncoding.DecodeString(s)
			if err != nil {
				return new(big.Int)
			}
			return new(big.Int).SetBytes(b)
		}
		switch {
		case j.KTY == "RSA" && (j.Alg == "RS256" || j.Alg == ""):
			n, e := decode(j.N), decode(j.E)
			if n.BitLen() < 2048 || !e.IsInt64() || e.Int64() < 3 || e.Int64() > 2147483647 {
				continue
			}
			keys[j.KID] = key{&rsa.PublicKey{N: n, E: int(e.Int64())}, "RS256"}
		case j.KTY == "EC" && j.CRV == "P-256" && (j.Alg == "ES256" || j.Alg == ""):
			x, y := decode(j.X), decode(j.Y)
			if !elliptic.P256().IsOnCurve(x, y) {
				continue
			}
			keys[j.KID] = key{&ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, "ES256"}
		}
	}
	v.keys = keys
	k, ok = keys[kid]
	if !ok || k.Algorithm != alg {
		return nil, errors.New("unknown signing key")
	}
	return k.Value, nil
}
