package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"portway/internal/problem"
)

type User struct {
	ID               string     `json:"id"`
	Email            string     `json:"email"`
	EmailConfirmedAt *time.Time `json:"email_confirmed_at"`
}
type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	User         User   `json:"user"`
}
type Supabase struct {
	URL, Key        string
	ConfirmationURL string
	Client          *http.Client
	verifier        *Verifier
}

func NewSupabase(url, key string) *Supabase {
	s := &Supabase{URL: url, Key: key, Client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	s.verifier = &Verifier{auth: s}
	return s
}
func (s *Supabase) call(ctx context.Context, method, path, token string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	r, err := http.NewRequestWithContext(ctx, method, s.URL+path, reader)
	if err != nil {
		return problem.Unavailable()
	}
	r.Header.Set("apikey", s.Key)
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.Client.Do(r)
	if err != nil {
		return problem.Unavailable()
	}
	defer resp.Body.Close()
	if resp.StatusCode == 429 {
		return problem.New(429, "RATE_LIMITED", "Authentication rate limit reached")
	}
	if resp.StatusCode >= 500 || resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return problem.Unavailable()
	}
	if resp.StatusCode >= 400 {
		var failure struct {
			Code string `json:"error_code"`
		}
		// Only known codes cross the boundary. Provider messages may contain
		// addresses or credentials and must never be returned to the client.
		b, readErr := io.ReadAll(io.LimitReader(resp.Body, 4097))
		if readErr == nil && len(b) <= 4096 && json.Unmarshal(b, &failure) == nil {
			if failure.Code == "email_not_confirmed" {
				return problem.New(403, "EMAIL_NOT_CONFIRMED", "Email confirmation required")
			}
		}
		return problem.New(401, "AUTH_REQUIRED", "Authentication failed")
	}
	if out != nil {
		b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
		if err != nil || len(b) > 1<<20 || json.Unmarshal(b, out) != nil {
			return problem.Unavailable()
		}
	}
	return nil
}
func (s *Supabase) Login(ctx context.Context, email, password string) (Session, error) {
	var v Session
	err := s.call(ctx, "POST", "/token?grant_type=password", "", map[string]string{"email": email, "password": password}, &v)
	return v, err
}
func (s *Supabase) Refresh(ctx context.Context, refresh string) (Session, error) {
	var v Session
	err := s.call(ctx, "POST", "/token?grant_type=refresh_token", "", map[string]string{"refresh_token": refresh}, &v)
	return v, err
}
func (s *Supabase) Signup(ctx context.Context, email, password string) (User, error) {
	var v User
	path := "/signup"
	if s.ConfirmationURL != "" {
		path += "?redirect_to=" + url.QueryEscape(s.ConfirmationURL)
	}
	err := s.call(ctx, "POST", path, "", map[string]string{"email": email, "password": password}, &v)
	return v, err
}
func (s *Supabase) Confirm(ctx context.Context, hash string) (User, error) {
	var v Session
	err := s.call(ctx, "POST", "/verify", "", map[string]string{"type": "email", "token_hash": hash}, &v)
	return v.User, err
}
func (s *Supabase) User(ctx context.Context, token string) (User, error) {
	var v User
	err := s.call(ctx, "GET", "/user", token, nil, &v)
	return v, err
}
func (s *Supabase) Logout(ctx context.Context, token string) error {
	return s.call(ctx, "POST", "/logout?scope=local", token, nil, nil)
}
func (s *Supabase) Verify(ctx context.Context, token string) (Claims, error) {
	return s.verifier.Verify(ctx, token)
}
