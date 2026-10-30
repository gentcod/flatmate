-- 002_duration_gender.sql: the required move-until date is replaced by a stay
-- duration (move_latest is now derived as move_earliest + duration_months);
-- profiles capture gender plus a gender preference for matching.
-- gender has no CHECK so legacy rows keep '' (unset) and Go validates new writes.

ALTER TABLE profiles ADD COLUMN duration_months INTEGER NOT NULL DEFAULT 6
  CHECK (duration_months BETWEEN 1 AND 36);

ALTER TABLE profiles ADD COLUMN gender TEXT NOT NULL DEFAULT '';

ALTER TABLE profiles ADD COLUMN gender_pref TEXT NOT NULL DEFAULT 'anyone'
  CHECK (gender_pref IN ('anyone', 'women', 'men', 'nonbinary'));
