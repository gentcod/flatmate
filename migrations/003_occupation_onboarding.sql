-- 003_occupation_onboarding.sql adds an optional occupation to profiles
-- and stores AI onboarding descriptions with their drafted answers
-- so users can revisit what they wrote.
ALTER TABLE profiles ADD COLUMN occupation TEXT NOT NULL DEFAULT '';

CREATE TABLE onboarding_drafts (
    user_id     INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    description TEXT NOT NULL DEFAULT '',
    drafts      TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
