package storage

import (
	"context"
	"testing"
)

func TestSmallPool(t *testing.T) {
	ctx := context.Background()
	dsn := "postgres://test:test@localhost:5432/test?sslmode=disable"
	s, err := Open(ctx, dsn, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Pool.Config().MaxConns != 2 || s.Pool.Config().MinConns != 0 {
		t.Fatal("unexpected default pool")
	}
	m, err := OpenWithLimit(ctx, dsn, "test", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if m.Pool.Config().MaxConns != 1 {
		t.Fatal("migration pool not limited")
	}
	if _, err = OpenWithLimit(ctx, dsn, "test", 0); err == nil {
		t.Fatal("invalid pool limit accepted")
	}
}
