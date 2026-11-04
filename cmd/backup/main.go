// Command backup snapshots the SQLite database to a timestamped copy.
package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"
	"time"

	"github.com/gentcod/flatmate/internal/backup"
)

func main() {
	dbPath := flag.String("db", "flatmate.db", "source sqlite database")
	out := flag.String("out", "", "destination file (default backups/<timestamp>.db)")
	flag.Parse()
	dst := *out
	if dst == "" {
		dst = filepath.Join("backups", backup.Filename("flatmate", time.Now()))
	}
	if err := backup.Backup(context.Background(), *dbPath, dst); err != nil {
		log.Fatalf("backup: %v", err)
	}
	log.Printf("backup wrote %s", dst)
}
