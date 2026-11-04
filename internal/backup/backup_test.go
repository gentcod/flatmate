package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestBackupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.db")
	d, err := sql.Open("sqlite", src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`CREATE TABLE notes (body TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO notes (body) VALUES ('a'), ('b')`); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "nested", "copy.db")
	ctx := context.Background()
	if err := Backup(ctx, src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatal(err)
	}
	c, err := sql.Open("sqlite", dst)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var n int
	if err := c.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("rows=%d, want 2", n)
	}
	if err := Backup(ctx, src, dst); err != nil {
		t.Fatalf("second backup (overwrite): %v", err)
	}
}

func TestFilename(t *testing.T) {
	got := Filename("flatmate", time.Date(2026, 11, 4, 21, 0, 0, 0, time.UTC))
	if want := "flatmate-20261104-210000.db"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
