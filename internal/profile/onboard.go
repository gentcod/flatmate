package profile

import (
	"context"
	"database/sql"
	"time"

	"github.com/gentcod/flatmate/internal/database"
)

// Onboarding is a user's saved AI description flow state: the free text
// they wrote plus the JSON drafts the model extracted from it.
type Onboarding struct {
	UserID      int64
	Description string
	Drafts      string // JSON array of extracted answers, "" when none yet
}

// SaveOnboarding stores the description and drafts, replacing any prior row.
func SaveOnboarding(ctx context.Context, d *sql.DB, userID int64, description, drafts string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return database.New(d).UpsertOnboardingDraft(ctx, database.UpsertOnboardingDraftParams{
		UserID: userID, Description: description, Drafts: drafts,
		CreatedAt: now, UpdatedAt: now,
	})
}

// GetOnboarding loads the saved state. found=false when never saved.
func GetOnboarding(ctx context.Context, d *sql.DB, userID int64) (o *Onboarding, found bool, err error) {
	row, err := database.New(d).GetOnboardingDraft(ctx, userID)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &Onboarding{UserID: row.UserID, Description: row.Description, Drafts: row.Drafts}, true, nil
}
