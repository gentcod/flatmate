ALTER TABLE listings ADD COLUMN status TEXT NOT NULL DEFAULT 'available';
CREATE TABLE stays (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_a INTEGER NOT NULL REFERENCES users(id),
  user_b INTEGER NOT NULL REFERENCES users(id),
  proposed_by INTEGER NOT NULL REFERENCES users(id),
  listing_id INTEGER NULL REFERENCES listings(id) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'proposed',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(user_a, user_b)
);
CREATE TABLE listing_reviews (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  listing_id INTEGER NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  author INTEGER NOT NULL REFERENCES users(id),
  rating INTEGER NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT 'visit',
  created_at TEXT NOT NULL,
  UNIQUE(listing_id, author)
);
