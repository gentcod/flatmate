// Package profile stores user profiles and questionnaire answers.
package profile

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/match"
)

// Looking-for modes. All cross-pairs are eligible; mode never gates.
const (
	ModeHaveRoom     = "have-room"
	ModeNeedRoom     = "need-room"
	ModeFindTogether = "find-together"
)

// Genders users can select for themselves.
const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonbinary = "nonbinary"
)

// Gender preferences. Anyone (the default) matches everybody.
const (
	GenderPrefAnyone    = "anyone"
	GenderPrefWomen     = "women"
	GenderPrefMen       = "men"
	GenderPrefNonbinary = "nonbinary"
)

// Profile mirrors the profiles table. Lat/Lng are nil when unset.
// MoveLatest is derived as MoveEarliest + DurationMonths on every save.
type Profile struct {
	UserID         int64
	FirstName      string
	Mode           string
	AreaText       string
	Lat            *float64
	Lng            *float64
	Relocating     bool
	BudgetMin      int64
	BudgetMax      int64
	Currency       string
	MoveEarliest   string // YYYY-MM-DD
	MoveLatest     string // YYYY-MM-DD, derived
	DurationMonths int    // 1–36
	Gender         string // woman|man|nonbinary
	GenderPref     string // anyone|women|men|nonbinary
	Occupation     string // optional, shown to matches
	Bio            string
	ContactHandle  string
	Searching      bool // visible in matching; new profiles start true
}

// Validate checks user-supplied profile fields.
func Validate(p *Profile) error {
	switch {
	case len(p.FirstName) == 0 || len(p.FirstName) > 50:
		return fmt.Errorf("first name must be 1–50 characters")
	case p.Mode != ModeHaveRoom && p.Mode != ModeNeedRoom && p.Mode != ModeFindTogether:
		return fmt.Errorf("unknown looking-for mode")
	case len(p.AreaText) == 0 || len(p.AreaText) > 120:
		return fmt.Errorf("area must be 1–120 characters")
	case p.BudgetMin < 0 || p.BudgetMax < p.BudgetMin:
		return fmt.Errorf("budget range is invalid")
	case len(p.Currency) == 0 || len(p.Currency) > 8:
		return fmt.Errorf("currency is invalid")
	case !validDate(p.MoveEarliest):
		return fmt.Errorf("move-in date is invalid (use YYYY-MM-DD)")
	case p.DurationMonths < 1 || p.DurationMonths > 36:
		return fmt.Errorf("stay duration must be 1–36 months")
	case p.Gender != GenderWoman && p.Gender != GenderMan && p.Gender != GenderNonbinary:
		return fmt.Errorf("gender is required")
	case p.GenderPref != GenderPrefAnyone && p.GenderPref != GenderPrefWomen &&
		p.GenderPref != GenderPrefMen && p.GenderPref != GenderPrefNonbinary:
		return fmt.Errorf("unknown gender preference")
	case len(p.Occupation) > 80:
		return fmt.Errorf("occupation must be at most 80 characters")
	case len(p.Bio) > 500:
		return fmt.Errorf("bio must be at most 500 characters")
	case len(p.ContactHandle) > 120:
		return fmt.Errorf("contact must be at most 120 characters")
	}
	return nil
}

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// AddMonths shifts start forward by whole calendar months, clamping the
// day to the target month length (Jan 31 + 1 month = Feb 28).
func AddMonths(start string, months int) (string, error) {
	t, err := time.Parse("2006-01-02", start)
	if err != nil {
		return "", fmt.Errorf("bad date %q: want YYYY-MM-DD", start)
	}
	if months < 1 {
		return "", fmt.Errorf("months must be at least 1")
	}
	y, m, d := t.Date()
	total := int(m) - 1 + months
	ny, nm := y+total/12, time.Month(total%12+1)
	last := time.Date(ny, nm+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if d > last {
		d = last
	}
	return time.Date(ny, nm, d, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), nil
}

// GenderLabel renders a stored gender for display.
func GenderLabel(g string) string {
	switch g {
	case GenderWoman:
		return "Woman"
	case GenderMan:
		return "Man"
	case GenderNonbinary:
		return "Non-binary"
	}
	return "Not specified"
}

// PrefLabel renders a stored gender preference for display.
func PrefLabel(p string) string {
	switch p {
	case GenderPrefWomen:
		return "Women"
	case GenderPrefMen:
		return "Men"
	case GenderPrefNonbinary:
		return "Non-binary"
	default:
		return "Anyone"
	}
}

// Get loads a profile. found=false when the user has none yet.
func Get(ctx context.Context, d *sql.DB, userID int64) (p *Profile, found bool, err error) {
	row, err := database.New(d).GetProfile(ctx, userID)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	p = &Profile{
		UserID: row.UserID, FirstName: row.FirstName, Mode: row.Mode, AreaText: row.AreaText,
		Lat: nullFloatPtr(row.Lat), Lng: nullFloatPtr(row.Lng),
		Relocating: row.Relocating == 1,
		BudgetMin:  row.BudgetMin, BudgetMax: row.BudgetMax, Currency: row.Currency,
		MoveEarliest: row.MoveEarliest, MoveLatest: row.MoveLatest,
		DurationMonths: int(row.DurationMonths), Gender: row.Gender, GenderPref: row.GenderPref,
		Occupation: row.Occupation,
		Bio:        row.Bio, ContactHandle: row.ContactHandle,
		Searching: row.Searching == 1,
	}
	return p, true, nil
}

// Upsert creates or replaces a profile, deriving the move window end
// from the start date plus stay duration.
func Upsert(ctx context.Context, d *sql.DB, p *Profile) error {
	if err := Validate(p); err != nil {
		return err
	}
	latest, err := AddMonths(p.MoveEarliest, p.DurationMonths)
	if err != nil {
		return err
	}
	return database.New(d).UpsertProfile(ctx, database.UpsertProfileParams{
		UserID: p.UserID, FirstName: p.FirstName, Mode: p.Mode, AreaText: p.AreaText,
		Lat: floatNull(p.Lat), Lng: floatNull(p.Lng), Relocating: boolInt64(p.Relocating),
		BudgetMin: p.BudgetMin, BudgetMax: p.BudgetMax, Currency: p.Currency,
		MoveEarliest: p.MoveEarliest, MoveLatest: latest,
		DurationMonths: int64(p.DurationMonths), Gender: p.Gender, GenderPref: p.GenderPref,
		Occupation: p.Occupation,
		Bio:        p.Bio, ContactHandle: p.ContactHandle,
		Searching: boolInt64(p.Searching),
	})
}

// Answer is one questionnaire response.
type Answer struct {
	Dimension string
	Value     string
	Hard      bool
	Weight    int // 1–3
}

// GetAnswers loads all answers for a user keyed by dimension.
func GetAnswers(ctx context.Context, d *sql.DB, userID int64) (map[string]Answer, error) {
	rows, err := database.New(d).GetAnswers(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := map[string]Answer{}
	for _, r := range rows {
		out[r.Dimension] = Answer{Dimension: r.Dimension, Value: r.Value, Hard: r.Hard == 1, Weight: int(r.Weight)}
	}
	return out, nil
}

// SaveAnswers replaces all answers for a user. All 12 dimensions required.
func SaveAnswers(ctx context.Context, d *sql.DB, userID int64, answers []Answer) error {
	if len(answers) != len(match.Dimensions) {
		return fmt.Errorf("need all %d dimensions, got %d", len(match.Dimensions), len(answers))
	}
	seen := map[string]bool{}
	for _, a := range answers {
		if seen[a.Dimension] {
			return fmt.Errorf("duplicate dimension %q", a.Dimension)
		}
		seen[a.Dimension] = true
		if !match.ValidValue(a.Dimension, a.Value) {
			return fmt.Errorf("invalid value %q for %q", a.Value, a.Dimension)
		}
		if a.Weight < 1 || a.Weight > 3 {
			return fmt.Errorf("weight for %q must be 1–3", a.Dimension)
		}
	}
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	q := database.New(d).WithTx(tx)
	if err := q.DeleteAnswers(ctx, userID); err != nil {
		tx.Rollback()
		return err
	}
	for _, a := range answers {
		err := q.InsertAnswer(ctx, database.InsertAnswerParams{
			UserID: userID, Dimension: a.Dimension, Value: a.Value,
			Hard: boolInt64(a.Hard), Weight: int64(a.Weight),
		})
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func boolInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func nullFloatPtr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	return &n.Float64
}

func floatNull(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}
