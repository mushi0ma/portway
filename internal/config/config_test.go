package config

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"portway/internal/secure"
)

func TestConfigSecretsAndTLS(t *testing.T) {
	for _, name := range []string{"PORTWAY_DATABASE_URL", "PORTWAY_MIGRATION_DATABASE_URL", "PORTWAY_MASTER_KEY", "PORTWAY_SETUP_TOKEN", "PORTWAY_DATABASE_MAX_CONNS"} {
		t.Setenv(name, "")
	}
	dir := t.TempDir()
	setFile := func(env, name, value string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv(env, path)
	}
	t.Setenv("PORTWAY_ENV", "production")
	t.Setenv("PORTWAY_INSTALLATION_ID", secure.ID())
	t.Setenv("PORTWAY_AUTH_URL", "https://project.supabase.co/auth/v1")
	t.Setenv("PORTWAY_PUBLIC_URL", "https://portway.example")
	t.Setenv("PORTWAY_AUTH_PUBLISHABLE_KEY", "public-test-key")
	t.Setenv("PORTWAY_MIGRATION_DATABASE_URL_FILE", "")
	setFile("PORTWAY_DATABASE_URL_FILE", "database", "postgres://user:secret-marker@db.example/app?sslmode=verify-full")
	setFile("PORTWAY_MASTER_KEY_FILE", "master", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	setFile("PORTWAY_SETUP_TOKEN_FILE", "setup", secure.Token())
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	setFile("PORTWAY_DATABASE_URL_FILE", "database", "postgres://user:secret-marker@db.example/app?sslmode=require")
	if _, err := Load(); err == nil || strings.Contains(err.Error(), "secret-marker") {
		t.Fatal("TLS must verify hostname without exposing DSN")
	}
	path := filepath.Join(dir, "world-readable")
	os.WriteFile(path, []byte("secret-marker"), 0644)
	if _, err := ReadSecret(path); err == nil {
		t.Fatal("insecure permissions accepted")
	}
}

func TestSecretEnvironmentOrFile(t *testing.T) {
	const name = "PORTWAY_TEST_SECRET"
	t.Setenv(name, "marker-never-log")
	t.Setenv(name+"_FILE", "")
	value, err := Secret(name, true)
	if err != nil || value != "marker-never-log" {
		t.Fatal("environment secret not loaded")
	}
	t.Setenv(name+"_FILE", "/unreadable-marker-path")
	if _, err = Secret(name, true); err == nil || strings.Contains(err.Error(), "marker") {
		t.Fatal("ambiguous secret or unsafe error")
	}
	t.Setenv(name, "")
	t.Setenv(name+"_FILE", "")
	if _, err = Secret(name, true); err == nil {
		t.Fatal("missing required secret accepted")
	}
	if value, err = Secret(name, false); err != nil || value != "" {
		t.Fatal("optional secret rejected")
	}
}

func TestEnvironmentConfigAndSmallPool(t *testing.T) {
	for _, name := range []string{"PORTWAY_DATABASE_URL_FILE", "PORTWAY_MIGRATION_DATABASE_URL_FILE", "PORTWAY_MASTER_KEY_FILE", "PORTWAY_SETUP_TOKEN_FILE", "PORTWAY_MIGRATION_DATABASE_URL", "PORTWAY_DATABASE_MAX_CONNS"} {
		t.Setenv(name, "")
	}
	t.Setenv("PORTWAY_ENV", "production")
	t.Setenv("PORTWAY_INSTALLATION_ID", secure.ID())
	t.Setenv("PORTWAY_AUTH_URL", "https://project.supabase.co/auth/v1")
	t.Setenv("PORTWAY_PUBLIC_URL", "https://localhost")
	t.Setenv("PORTWAY_AUTH_PUBLISHABLE_KEY", "sb_publishable_test")
	t.Setenv("PORTWAY_DATABASE_URL", "postgres://portway_app:marker@db.example/postgres?sslmode=verify-full")
	t.Setenv("PORTWAY_MASTER_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("PORTWAY_SETUP_TOKEN", secure.Token())
	c, err := Load()
	if err != nil || c.DatabaseMaxConns != 2 {
		t.Fatal("small pool default", err)
	}
	for _, value := range []string{"0", "9", "-1", "invalid"} {
		t.Setenv("PORTWAY_DATABASE_MAX_CONNS", value)
		if _, err = Load(); err == nil {
			t.Fatal("invalid pool size accepted")
		}
	}
	t.Setenv("PORTWAY_DATABASE_MAX_CONNS", "1")
	c, err = Load()
	if err != nil || c.DatabaseMaxConns != 1 {
		t.Fatal("pool override", err)
	}
	t.Setenv("PORTWAY_AUTH_PUBLISHABLE_KEY", "sb_secret_marker")
	if _, err = Load(); err == nil || strings.Contains(err.Error(), "marker") {
		t.Fatal("secret API key must be rejected without logging")
	}
}
