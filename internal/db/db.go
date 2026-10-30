// Package db opens SQLite and applies plain-SQL migrations in order.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Open opens path with sane pragmas. Use ":memory:" for tests.
func Open(path string) (*sql.DB, error) {
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	for _, p := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	} {
		if _, err := d.Exec(p); err != nil {
			d.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}
	return d, nil
}

// Migrate applies pending *.sql files from dir in lexical order.
func Migrate(d *sql.DB, dir string) error {
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		name := filepath.Base(f)
		var n int
		if err := d.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename=?`, name).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		for _, stmt := range splitStatements(string(sqlBytes)) {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("migrate %s: %w", name, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (filename, applied_at) VALUES (?, ?)`,
			name, time.Now().UTC().Format(time.RFC3339)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// splitStatements splits on semicolons. Our migrations contain no
// triggers or quoted semicolons, so this stays deterministic.
func splitStatements(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}
