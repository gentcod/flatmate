-- name: GetOnboardingDraft :one
SELECT * FROM onboarding_drafts WHERE user_id = sqlc.arg(user_id);

-- name: UpsertOnboardingDraft :exec
INSERT INTO onboarding_drafts (user_id, description, drafts, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(description), sqlc.arg(drafts), sqlc.arg(created_at), sqlc.arg(updated_at))
ON CONFLICT(user_id) DO UPDATE SET
  description = excluded.description,
  drafts = excluded.drafts,
  updated_at = excluded.updated_at;
