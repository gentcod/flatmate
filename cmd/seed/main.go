// Command seed imports seeds/global.json into the database.
// Existing emails are skipped, so re-running is safe.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/gentcod/flatmate/internal/auth"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/db"
	"github.com/gentcod/flatmate/internal/profile"
	"github.com/gentcod/flatmate/internal/util"
)

type seedAnswer struct {
	Value  string `json:"value"`
	Hard   bool   `json:"hard"`
	Weight int    `json:"weight"`
}

type seedProfile struct {
	FirstName      string   `json:"first_name"`
	Mode           string   `json:"mode"`
	AreaText       string   `json:"area_text"`
	Lat            *float64 `json:"lat"`
	Lng            *float64 `json:"lng"`
	Relocating     bool     `json:"relocating"`
	BudgetMin      int64    `json:"budget_min"`
	BudgetMax      int64    `json:"budget_max"`
	Currency       string   `json:"currency"`
	MoveEarliest   string   `json:"move_earliest"`
	DurationMonths int      `json:"duration_months"`
	Gender         string   `json:"gender"`
	GenderPref     string   `json:"gender_pref"`
	Occupation     string   `json:"occupation"`
	Bio            string   `json:"bio"`
	ContactHandle  string   `json:"contact_handle"`
}

type seedListing struct {
	Email       string `json:"email"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AreaText    string `json:"area_text"`
	MonthlyRent int64  `json:"monthly_rent"`
	Currency    string `json:"currency"`
	Photo       string `json:"photo"`
}

type seedUser struct {
	Email    string                `json:"email"`
	Password string                `json:"password"`
	Profile  seedProfile           `json:"profile"`
	Answers  map[string]seedAnswer `json:"answers"`
}

func main() {
	cfg, err := util.LoadConfig(".env")
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	dbPath := flag.String("db", cfg.DBPath, "sqlite database path (DB_PATH)")
	seedPath := flag.String("seeds", cfg.SeedPath, "seed file (SEED_PATH)")
	migDir := flag.String("migrations", cfg.MigrationsDir, "migrations directory (MIGRATIONS_DIR)")
	listingsPath := flag.String("listings", "seeds/listings.json", "listing seeds file")
	flag.Parse()

	raw, err := os.ReadFile(*seedPath)
	if err != nil {
		log.Fatalf("read seeds: %v", err)
	}
	var users []seedUser
	if err := json.Unmarshal(raw, &users); err != nil {
		log.Fatalf("parse seeds: %v", err)
	}

	dbc, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer dbc.Close()
	if err := db.Migrate(dbc, *migDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	added, skipped := 0, 0
	for _, u := range users {
		hash := auth.HashEmail(u.Email)
		exists, err := database.New(dbc).CountUsersByEmailHash(ctx, hash)
		if err != nil {
			log.Fatalf("lookup %s: %v", u.Email, err)
		}
		if exists > 0 {
			skipped++
			continue
		}
		pw, err := auth.HashPassword(u.Password)
		if err != nil {
			log.Fatalf("hash %s: %v", u.Email, err)
		}
		res, err := database.New(dbc).CreateUser(ctx, database.CreateUserParams{
			EmailHash: hash, PassHash: pw, CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			log.Fatalf("insert %s: %v", u.Email, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			log.Fatalf("id %s: %v", u.Email, err)
		}
		sp := u.Profile
		p := &profile.Profile{UserID: id, FirstName: sp.FirstName, Mode: sp.Mode,
			AreaText: sp.AreaText, Lat: sp.Lat, Lng: sp.Lng, Relocating: sp.Relocating,
			BudgetMin: sp.BudgetMin, BudgetMax: sp.BudgetMax, Currency: sp.Currency,
			MoveEarliest: sp.MoveEarliest, DurationMonths: sp.DurationMonths,
			Gender: sp.Gender, GenderPref: sp.GenderPref, Occupation: sp.Occupation,
			Bio: sp.Bio, ContactHandle: sp.ContactHandle, Searching: true}
		if err := profile.Upsert(ctx, dbc, p); err != nil {
			log.Fatalf("profile %s: %v", u.Email, err)
		}
		var answers []profile.Answer
		for dim, a := range u.Answers {
			answers = append(answers, profile.Answer{Dimension: dim, Value: a.Value, Hard: a.Hard, Weight: a.Weight})
		}
		if err := profile.SaveAnswers(ctx, dbc, id, answers); err != nil {
			log.Fatalf("answers %s: %v", u.Email, err)
		}
		added++
	}
	log.Printf("seeded %d users (%d skipped)", added, skipped)
	seedListings(ctx, dbc, *listingsPath)
}

func seedListings(ctx context.Context, dbc *sql.DB, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("no listing seeds at %s, skipping", path)
			return
		}
		log.Fatalf("read listing seeds: %v", err)
	}
	var listings []seedListing
	if err := json.Unmarshal(raw, &listings); err != nil {
		log.Fatalf("parse listing seeds: %v", err)
	}
	q := database.New(dbc)
	added, skipped := 0, 0
	for _, sl := range listings {
		u, err := q.GetUserByEmailHash(ctx, auth.HashEmail(sl.Email))
		if err != nil {
			log.Printf("listing %q: unknown user %s, skipping", sl.Title, sl.Email)
			skipped++
			continue
		}
		existing, err := q.ListListingsByUser(ctx, u.ID)
		if err != nil {
			log.Fatalf("list listings for %s: %v", sl.Email, err)
		}
		dup := false
		for _, e := range existing {
			if e.Title == sl.Title {
				dup = true
				break
			}
		}
		if dup {
			skipped++
			continue
		}
		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := q.CreateListing(ctx, database.CreateListingParams{
			UserID: u.ID, Title: sl.Title, Description: sl.Description,
			AreaText: sl.AreaText, MonthlyRent: sl.MonthlyRent, Currency: sl.Currency,
			PhotoUrl: sl.Photo, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			log.Fatalf("insert listing %q: %v", sl.Title, err)
		}
		added++
	}
	log.Printf("seeded %d listings (%d skipped)", added, skipped)
}
