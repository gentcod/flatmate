// Package backup snapshots the SQLite database with VACUUM INTO,
// which reads the live file without requiring the server to stop.
package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Filename returns a timestamped backup name like flatmate-20261104-210000.db.
func Filename(prefix string, t time.Time) string {
	return fmt.Sprintf("%s-%s.db", prefix, t.UTC().Format("20060102-150405"))
}

// Backup copies src into dst. It creates dst's parent directory and
// replaces any existing dst file.
func Backup(ctx context.Context, src, dst string) error {
	if src == "" || dst == "" {
		return fmt.Errorf("backup: source and destination paths are required")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	d, err := sql.Open("sqlite", src)
	if err != nil {
		return err
	}
	defer d.Close()
	q := "'" + strings.ReplaceAll(dst, "'", "''") + "'"
	if _, err := d.ExecContext(ctx, "VACUUM INTO "+q); err != nil {
		return fmt.Errorf("backup: vacuum into: %w", err)
	}
	return nil
}
