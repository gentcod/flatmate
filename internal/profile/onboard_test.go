package profile

import (
	"context"
	"testing"

	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/db"
)

func TestSaveGetOnboarding(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := db.Migrate(d, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	res, err := database.New(d).CreateUser(ctx, database.CreateUserParams{
		EmailHash: []byte("onboard-hash"), PassHash: []byte("pw"), CreatedAt: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := GetOnboarding(ctx, d, id); err != nil || found {
		t.Fatalf("fresh user: found=%v err=%v", found, err)
	}
	if err := SaveOnboarding(ctx, d, id, "early riser", `[{"dimension":"sleep"}]`); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, found, err := GetOnboarding(ctx, d, id)
	if err != nil || !found {
		t.Fatalf("get: found=%v err=%v", found, err)
	}
	if got.Description != "early riser" || got.Drafts != `[{"dimension":"sleep"}]` {
		t.Errorf("round trip: %+v", got)
	}
	if err := SaveOnboarding(ctx, d, id, "night owl", ""); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got, _, _ = GetOnboarding(ctx, d, id)
	if got.Description != "night owl" || got.Drafts != "" {
		t.Errorf("overwrite kept stale data: %+v", got)
	}
}
