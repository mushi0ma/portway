package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"portway/internal/openrouter"
	"portway/internal/secure"
	"unicode"
)

type Config struct {
	Environment, HTTPAddr, InstallationID, InstallationName string
	DatabaseURL, MigrationURL, AuthURL, AuthKey, PublicURL  string
	ExecutionURL, ExecutionKey                              string
	OpenRouterKey, OpenRouterModel                          string
	CodexBridgeKey                                          string
	FreeModelsOnly                                          bool
	SetupHash                                               string
	MasterKey                                               []byte
	AllowedEndpoints                                        []string
	DatabaseMaxConns                                        int32
}

// Secret accepts either an environment value or a file, never an ambiguous pair.
// Values and paths are deliberately omitted from errors.
func Secret(name string, required bool) (string, error) {
	value, path := os.Getenv(name), os.Getenv(name+"_FILE")
	if value != "" && path != "" {
		return "", fmt.Errorf("set only %s or %s_FILE", name, name)
	}
	if path != "" {
		v, err := ReadSecret(path)
		if err != nil {
			return "", fmt.Errorf("%s_FILE: %w", name, err)
		}
		value = v
	}
	if required && strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s or %s_FILE is required", name, name)
	}
	return strings.TrimSpace(value), nil
}

func ReadSecret(path string) (string, error) {
	if path == "" {
		return "", errors.New("secret file path is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", errors.New("secret file is unreadable")
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > 16384 {
		return "", errors.New("invalid secret file")
	}
	if st.Mode().Perm()&0007 != 0 {
		return "", errors.New("secret file must not be accessible to other users")
	}
	b, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil {
		return "", errors.New("secret file is unreadable")
	}
	if len(b) > 16384 {
		return "", errors.New("secret file is too large")
	}
	return strings.TrimSpace(string(b)), nil
}
func Load() (Config, error) {
	c := Config{Environment: os.Getenv("PORTWAY_ENV"), HTTPAddr: os.Getenv("PORTWAY_HTTP_ADDR"), InstallationID: os.Getenv("PORTWAY_INSTALLATION_ID"), InstallationName: os.Getenv("PORTWAY_INSTALLATION_NAME"), AuthURL: strings.TrimRight(os.Getenv("PORTWAY_AUTH_URL"), "/"), AuthKey: os.Getenv("PORTWAY_AUTH_PUBLISHABLE_KEY"), PublicURL: strings.TrimRight(os.Getenv("PORTWAY_PUBLIC_URL"), "/")}
	if c.Environment == "" {
		c.Environment = "production"
	}
	if c.Environment != "production" && c.Environment != "test" {
		return c, errors.New("PORTWAY_ENV must be production or test")
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = "127.0.0.1:8080"
	}
	if _, _, err := net.SplitHostPort(c.HTTPAddr); err != nil {
		return c, errors.New("invalid PORTWAY_HTTP_ADDR")
	}
	if !secure.UUID(c.InstallationID) {
		return c, errors.New("PORTWAY_INSTALLATION_ID must be a lowercase UUID")
	}
	if c.InstallationName == "" {
		c.InstallationName = "Portway"
	}
	for label, value := range map[string]string{"PORTWAY_AUTH_URL": c.AuthURL, "PORTWAY_PUBLIC_URL": c.PublicURL} {
		u, err := url.Parse(value)
		if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(c.Environment == "test" && u.Scheme == "http")) {
			return c, fmt.Errorf("%s must be an HTTPS URL without credentials, query or fragment", label)
		}
	}
	if c.AuthKey == "" {
		return c, errors.New("PORTWAY_AUTH_PUBLISHABLE_KEY is required")
	}
	if strings.HasPrefix(c.AuthKey, "sb_secret_") {
		return c, errors.New("PORTWAY_AUTH_PUBLISHABLE_KEY requires a publishable key, not a secret key")
	}
	c.DatabaseMaxConns = 2
	if value := os.Getenv("PORTWAY_DATABASE_MAX_CONNS"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 8 {
			return c, errors.New("PORTWAY_DATABASE_MAX_CONNS must be between 1 and 8")
		}
		c.DatabaseMaxConns = int32(n)
	}
	var err error
	c.DatabaseURL, err = Secret("PORTWAY_DATABASE_URL", true)
	if err != nil {
		return c, err
	}
	c.MigrationURL, err = Secret("PORTWAY_MIGRATION_DATABASE_URL", false)
	if err != nil {
		return c, err
	}
	for _, dsn := range []string{c.DatabaseURL, c.MigrationURL} {
		if dsn == "" {
			continue
		}
		u, e := url.Parse(dsn)
		if e != nil || u.Host == "" || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
			return c, errors.New("database file must contain a PostgreSQL URL")
		}
		if c.Environment == "production" && u.Query().Get("sslmode") != "verify-full" {
			return c, errors.New("production database requires sslmode=verify-full")
		}
	}
	c.CodexBridgeKey, err = Secret("PORTWAY_CODEX_BRIDGE_KEY", false)
	if err != nil {
		return c, err
	}
	if c.CodexBridgeKey != "" && (len(c.CodexBridgeKey) < 32 || len(c.CodexBridgeKey) > 128 || strings.IndexFunc(c.CodexBridgeKey, unicode.IsSpace) >= 0) {
		return c, errors.New("invalid PORTWAY_CODEX_BRIDGE_KEY")
	}
	c.ExecutionURL = os.Getenv("PORTWAY_EXECUTION_URL")
	c.ExecutionKey, err = Secret("PORTWAY_EXECUTION_SERVICE_KEY", false)
	if err != nil {
		return c, err
	}
	if c.ExecutionURL != "" && (c.ExecutionURL != "http://execution:8082" || len(c.ExecutionKey) < 32) {
		return c, errors.New("execution requires its private Compose URL and a separate service key")
	}
	key, err := Secret("PORTWAY_MASTER_KEY", true)
	if err != nil {
		return c, err
	}
	c.MasterKey, err = base64.StdEncoding.DecodeString(key)
	if err != nil || len(c.MasterKey) != 32 {
		return c, errors.New("master key must be base64 of 32 random bytes")
	}
	setup, err := Secret("PORTWAY_SETUP_TOKEN", true)
	if err != nil {
		return c, err
	}
	b, err := base64.RawURLEncoding.DecodeString(setup)
	if err != nil || len(b) < 32 {
		return c, errors.New("setup token must contain at least 32 random bytes in base64url")
	}
	c.SetupHash = secure.Hash(setup)
	for _, e := range strings.Split(os.Getenv("PORTWAY_LLM_ALLOWED_ENDPOINTS"), ",") {
		e = strings.TrimRight(strings.TrimSpace(e), "/")
		if e == "" {
			continue
		}
		u, err := url.Parse(e)
		if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
			return c, errors.New("invalid allowlisted model endpoint")
		}
		c.AllowedEndpoints = append(c.AllowedEndpoints, e)
	}
	policy := os.Getenv("PORTWAY_MODEL_POLICY")
	if policy != "" && policy != "free_only" {
		return c, errors.New("PORTWAY_MODEL_POLICY must be free_only")
	}
	c.FreeModelsOnly = true
	c.OpenRouterKey, err = Secret("OPENROUTER_API_KEY", false)
	if err != nil {
		return c, err
	}
	if len(c.OpenRouterKey) > 8192 || strings.IndexFunc(c.OpenRouterKey, unicode.IsControl) >= 0 {
		return c, errors.New("invalid OPENROUTER_API_KEY")
	}
	c.OpenRouterModel = strings.TrimSpace(os.Getenv("PORTWAY_OPENROUTER_MODEL"))
	if c.OpenRouterModel != "" && !openrouter.FreeID(c.OpenRouterModel) {
		return c, errors.New("PORTWAY_OPENROUTER_MODEL must be an explicit :free model")
	}
	if c.OpenRouterKey != "" {
		found := false
		for _, endpoint := range c.AllowedEndpoints {
			found = found || endpoint == openrouter.Endpoint
		}
		if !found {
			c.AllowedEndpoints = append(c.AllowedEndpoints, openrouter.Endpoint)
		}
	}
	return c, nil
}
