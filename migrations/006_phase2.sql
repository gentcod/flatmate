-- 006_phase2.sql adds shared expense records and saved searches
-- for Phase 2, plus a self-confirmed contact flag on profiles.
ALTER TABLE profiles ADD COLUMN contact_self_confirmed INTEGER NOT NULL DEFAULT 0;

CREATE TABLE expenses (
    id          INTEGER PRIMARY KEY,
    payer       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    peer        INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount      INTEGER NOT NULL DEFAULT 0,
    currency    TEXT NOT NULL DEFAULT 'USD',
    note        TEXT NOT NULL DEFAULT '',
    split       TEXT NOT NULL DEFAULT 'equal',
    payer_share INTEGER NOT NULL DEFAULT 0,
    peer_share  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL
);
CREATE INDEX expenses_pair_idx ON expenses(payer, peer);

CREATE TABLE saved_searches (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL DEFAULT '',
    area       TEXT NOT NULL DEFAULT '',
    max_rent   INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);
CREATE INDEX saved_searches_user_idx ON saved_searches(user_id);
