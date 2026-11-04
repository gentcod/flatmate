package admin

import (
	"context"
	"testing"

	"github.com/gentcod/flatmate/internal/auth"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/db"
)

func testQueries(t *testing.T) *database.Queries {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if err := db.Migrate(d, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	return database.New(d)
}

func TestSetAdminByHash(t *testing.T) {
	q := testQueries(t)
	ctx := context.Background()
	hash := auth.HashEmail("Op@Example.com")
	if _, err := q.CreateUser(ctx, database.CreateUserParams{
		EmailHash: hash, PassHash: []byte("x"), CreatedAt: "2026-01-01T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	u, err := q.GetUserByEmailHash(ctx, hash)
	if err != nil {
		t.Fatal(err)
	}
	if IsAdmin(ctx, q, u.ID) {
		t.Fatal("fresh user is admin, want not")
	}
	if err := SetAdminByHash(ctx, q, auth.HashEmail("op@example.com"), true); err != nil {
		t.Fatal(err)
	}
	if !IsAdmin(ctx, q, u.ID) {
		t.Fatal("promote failed")
	}
	if err := SetAdminByHash(ctx, q, hash, false); err != nil {
		t.Fatal(err)
	}
	if IsAdmin(ctx, q, u.ID) {
		t.Fatal("demote failed")
	}
	if err := SetAdminByHash(ctx, q, auth.HashEmail("nobody@example.com"), true); err == nil {
		t.Fatal("expected error for unknown email")
	}
}
