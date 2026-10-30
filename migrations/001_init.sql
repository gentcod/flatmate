-- 001_init.sql: initial schema per CONTEXT.md §9.
-- Timestamps are TEXT in ISO8601 UTC. Booleans are INTEGER 0/1.

CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email_hash    BLOB NOT NULL UNIQUE,
    pass_hash     BLOB NOT NULL,
    created_at    TEXT NOT NULL,
    deactivated_at TEXT
);

CREATE TABLE profiles (
    user_id       INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_name    TEXT NOT NULL,
    mode          TEXT NOT NULL CHECK (mode IN ('have-room', 'need-room', 'find-together')),
    area_text     TEXT NOT NULL,
    lat           REAL,
    lng           REAL,
    relocating    INTEGER NOT NULL DEFAULT 0 CHECK (relocating IN (0, 1)),
    budget_min    INTEGER NOT NULL CHECK (budget_min >= 0),
    budget_max    INTEGER NOT NULL CHECK (budget_max >= budget_min),
    currency      TEXT NOT NULL DEFAULT 'USD',
    move_earliest TEXT NOT NULL,
    move_latest   TEXT NOT NULL CHECK (move_latest >= move_earliest),
    bio           TEXT NOT NULL DEFAULT '',
    contact_handle TEXT NOT NULL DEFAULT ''
);

CREATE TABLE answers (
    user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dimension TEXT NOT NULL,
    value     TEXT NOT NULL,
    hard      INTEGER NOT NULL DEFAULT 0 CHECK (hard IN (0, 1)),
    weight    INTEGER NOT NULL DEFAULT 1 CHECK (weight BETWEEN 1 AND 3),
    PRIMARY KEY (user_id, dimension)
);

CREATE TABLE actions (
    actor      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('save', 'dismiss', 'interest', 'reveal')),
    created_at TEXT NOT NULL,
    PRIMARY KEY (actor, target, kind),
    CHECK (actor != target)
);

CREATE TABLE blocks (
    blocker    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY (blocker, blocked),
    CHECK (blocker != blocked)
);

CREATE TABLE reports (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    reporter   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason     TEXT NOT NULL,
    created_at TEXT NOT NULL,
    CHECK (reporter != target)
);

CREATE INDEX idx_answers_user ON answers(user_id);
CREATE INDEX idx_actions_target ON actions(target, kind);
CREATE INDEX idx_profiles_area ON profiles(area_text);
