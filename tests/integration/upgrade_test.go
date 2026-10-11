package integration

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"portway/internal/secure"
	"portway/internal/storage"
)

func TestUpgradeFromIdentitySchema(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("PORTWAY_REQUIRE_INTEGRATION") == "1" {
			t.Fatal("database required")
		}
		t.Skip("requires disposable PostgreSQL")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Path != "/portway_test" {
		t.Fatal("disposable database required")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	name := "portway_upgrade_" + strings.ReplaceAll(secure.ID(), "-", "")
	quoted := pgx.Identifier{name}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err = admin.Exec(ctx, "DROP DATABASE "+quoted+" WITH (FORCE)"); err != nil {
			t.Error("cleanup upgrade DB", err)
		}
	}()
	u.Path = "/" + name
	installation := secure.ID()
	s, err := storage.Open(ctx, u.String(), installation)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, err := os.ReadFile("../../internal/storage/migrations/001_identity.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Pool.Exec(ctx, string(first)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Pool.Exec(ctx, `CREATE TABLE portway.schema_migrations(version integer PRIMARY KEY,checksum text NOT NULL,applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Pool.Exec(ctx, `INSERT INTO portway.schema_migrations(version,checksum) VALUES(1,$1)`, secure.Hash(string(first))); err != nil {
		t.Fatal(err)
	}
	if err = s.EnsureInstallation(ctx, "Preserved installation", secure.Hash(secure.Token())); err != nil {
		t.Fatal(err)
	}
	if s.Ready(ctx) == nil {
		t.Fatal("old schema marked compatible")
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal("upgrade", err)
	}
	if err = s.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	var preserved string
	if err = s.Pool.QueryRow(ctx, `SELECT name FROM portway.installations WHERE id=$1`, installation).Scan(&preserved); err != nil || preserved != "Preserved installation" {
		t.Fatal("upgrade lost data", err)
	}
}
