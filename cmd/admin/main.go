// Command admin grants or revokes operator privileges by email.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gentcod/flatmate/internal/admin"
	"github.com/gentcod/flatmate/internal/auth"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/db"
)

func main() {
	dbPath := flag.String("db", "flatmate.db", "sqlite database path")
	migDir := flag.String("migrations", "migrations", "migrations directory")
	promote := flag.String("promote", "", "email to grant admin")
	demote := flag.String("demote", "", "email to revoke admin")
	flag.Parse()
	if (*promote == "") == (*demote == "") {
		fmt.Fprintln(os.Stderr, "usage: admin -db x.db (-promote email | -demote email)")
		os.Exit(2)
	}
	d, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer d.Close()
	if err := db.Migrate(d, *migDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	email, on := *promote, true
	if *demote != "" {
		email, on = *demote, false
	}
	email = strings.TrimSpace(email)
	if err := admin.SetAdminByHash(context.Background(), database.New(d), auth.HashEmail(email), on); err != nil {
		if err == sql.ErrNoRows {
			log.Fatalf("no user with that email")
		}
		log.Fatalf("set admin: %v", err)
	}
	log.Printf("admin=%v for %s", on, email)
}
