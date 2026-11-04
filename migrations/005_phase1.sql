-- 005_phase1.sql adds room listings, mutual-interest chat messages,
-- and one-per-pair flatmate reviews for Phase 1.
CREATE TABLE listings (
    id            INTEGER PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         TEXT NOT NULL DEFAULT '',
    description   TEXT NOT NULL DEFAULT '',
    area_text     TEXT NOT NULL DEFAULT '',
    monthly_rent  INTEGER NOT NULL DEFAULT 0,
    currency      TEXT NOT NULL DEFAULT 'USD',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
CREATE INDEX listings_user_idx ON listings(user_id);

CREATE TABLE messages (
    id         INTEGER PRIMARY KEY,
    sender     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX messages_pair_idx ON messages(sender, recipient);

CREATE TABLE reviews (
    id         INTEGER PRIMARY KEY,
    author     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating     INTEGER NOT NULL DEFAULT 5,
    body       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    UNIQUE(author, target)
);
CREATE INDEX reviews_target_idx ON reviews(target);
