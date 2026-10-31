-- 004_searching.sql adds a visibility flag to profiles. Searching users
-- appear in matching and users who found a roommate can hide themselves.
-- Existing rows default to visible.
ALTER TABLE profiles ADD COLUMN searching INTEGER NOT NULL DEFAULT 1;
