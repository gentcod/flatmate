package profile

import (
	"context"
	"strings"
	"testing"

	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/db"
)

func TestAddMonths(t *testing.T) {
	cases := []struct {
		start  string
		months int
		want   string
	}{
		{"2026-12-01", 6, "2027-06-01"},
		{"2026-12-01", 5, "2027-05-01"},
		{"2026-12-15", 2, "2027-02-15"},
		{"2026-12-01", 12, "2027-12-01"},
		{"2027-01-31", 1, "2027-02-28"}, // day clamps to short month
		{"2024-01-31", 1, "2024-02-29"}, // leap year
		{"2026-08-31", 6, "2027-02-28"},
	}
	for _, tc := range cases {
		got, err := AddMonths(tc.start, tc.months)
		if err != nil || got != tc.want {
			t.Errorf("AddMonths(%q, %d) = %q, %v; want %q", tc.start, tc.months, got, err, tc.want)
		}
	}
	for _, bad := range []string{"", "soon", "2026-13-01", "2026-12-32", "12/01/2026"} {
		if _, err := AddMonths(bad, 6); err == nil {
			t.Errorf("AddMonths(%q, 6): want error", bad)
		}
	}
	if _, err := AddMonths("2026-12-01", 0); err == nil {
		t.Error("AddMonths months=0: want error")
	}
}

func testProfile() *Profile {
	return &Profile{UserID: 1, FirstName: "Ada", Mode: ModeHaveRoom,
		AreaText: "Yaba", BudgetMin: 100, BudgetMax: 200, Currency: "NGN",
		MoveEarliest: "2026-12-01", DurationMonths: 6,
		Gender: GenderWoman, GenderPref: GenderPrefAnyone, Occupation: "Nurse"}
}

func TestValidateDurationGender(t *testing.T) {
	if err := Validate(testProfile()); err != nil {
		t.Fatalf("valid profile: %v", err)
	}
	cases := []struct {
		name string
		mut  func(*Profile)
	}{
		{"duration zero", func(p *Profile) { p.DurationMonths = 0 }},
		{"duration high", func(p *Profile) { p.DurationMonths = 37 }},
		{"bad earliest", func(p *Profile) { p.MoveEarliest = "soon" }},
		{"bad gender", func(p *Profile) { p.Gender = "female" }},
		{"empty gender", func(p *Profile) { p.Gender = "" }},
		{"bad pref", func(p *Profile) { p.GenderPref = "everyone" }},
		{"empty pref", func(p *Profile) { p.GenderPref = "" }},
		{"occupation long", func(p *Profile) { p.Occupation = strings.Repeat("x", 81) }},
	}
	for _, tc := range cases {
		p := testProfile()
		tc.mut(p)
		if err := Validate(p); err == nil {
			t.Errorf("%s: want error", tc.name)
		}
	}
}

func TestUpsertDerivesLatest(t *testing.T) {
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
		EmailHash: []byte("test-hash"), PassHash: []byte("pw"), CreatedAt: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	p := testProfile()
	p.UserID = id
	p.GenderPref = GenderPrefWomen
	if err := Upsert(ctx, d, p); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, found, err := Get(ctx, d, id)
	if err != nil || !found {
		t.Fatalf("get: found=%v err=%v", found, err)
	}
	if got.MoveLatest != "2027-06-01" {
		t.Errorf("derived latest = %q, want %q", got.MoveLatest, "2027-06-01")
	}
	if got.DurationMonths != 6 || got.Gender != GenderWoman || got.GenderPref != GenderPrefWomen || got.Occupation != "Nurse" {
		t.Errorf("round trip dropped fields: %+v", got)
	}
	if _, found, err := Get(ctx, d, id+999); err != nil || found {
		t.Errorf("missing profile: found=%v err=%v", found, err)
	}
}

func TestSearchingRoundTrip(t *testing.T) {
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
		EmailHash: []byte("search-hash"), PassHash: []byte("pw"), CreatedAt: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	p := testProfile()
	p.UserID = id
	p.Searching = true
	if err := Upsert(ctx, d, p); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, _, err := Get(ctx, d, id)
	if err != nil || got == nil || !got.Searching {
		t.Errorf("round trip true: got %+v err=%v", got, err)
	}
	p.Searching = false
	if err := Upsert(ctx, d, p); err != nil {
		t.Fatalf("upsert false: %v", err)
	}
	got, _, err = Get(ctx, d, id)
	if err != nil || got == nil || got.Searching {
		t.Errorf("round trip false: got %+v err=%v", got, err)
	}
}

func TestSearchingMigrationDefault(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := db.Migrate(d, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	res, err := database.New(d).CreateUser(context.Background(), database.CreateUserParams{
		EmailHash: []byte("legacy-hash"), PassHash: []byte("pw"), CreatedAt: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	// A row written before migration 004 carries no searching value;
	// it must read back as visible.
	_, err = d.Exec(`INSERT INTO profiles (user_id, first_name, mode, area_text, relocating, budget_min, budget_max, currency, move_earliest, move_latest, bio, contact_handle, duration_months, gender, gender_pref, occupation) VALUES (?, 'Ada', 'have-room', 'Yaba', 0, 100, 200, 'NGN', '2026-12-01', '2027-06-01', '', '', 6, 'woman', 'anyone', 'Nurse')`, id)
	if err != nil {
		t.Fatalf("legacy insert: %v", err)
	}
	got, found, err := Get(context.Background(), d, id)
	if err != nil || !found {
		t.Fatalf("legacy get: found=%v err=%v", found, err)
	}
	if !got.Searching {
		t.Error("legacy row: searching=false, want true")
	}
}
