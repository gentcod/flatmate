// Command flatmate runs the Flatmate Matching Platform server.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/auth"
	"github.com/gentcod/flatmate/internal/db"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/util"
	"github.com/gentcod/flatmate/internal/web"
)

func main() {
	cfg, err := util.LoadConfig(".env")
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	addr := flag.String("addr", cfg.Addr, "listen address (ADDR)")
	dbPath := flag.String("db", cfg.DBPath, "sqlite database path (DB_PATH)")
	weights := flag.String("weights", cfg.WeightsPath, "scoring weights config (WEIGHTS_PATH)")
	webDir := flag.String("web", cfg.WebDir, "web assets directory (WEB_DIR)")
	migDir := flag.String("migrations", cfg.MigrationsDir, "migrations directory (MIGRATIONS_DIR)")
	uiMode := flag.String("ui", cfg.UI, "frontend: legacy templates or spa build (FLATMATE_UI)")
	clientDir := flag.String("client", cfg.ClientDist, "built SPA directory for -ui=spa (CLIENT_DIST)")
	flag.Parse()

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()
	if err := db.Migrate(database, *migDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	cfg.Addr = *addr // effective address drives cookie + key policy below
	secure := cfg.SecureCookies()
	sessions, err := sessionsFromConfig(cfg.SessionKey, cfg.Addr, secure)
	if err != nil {
		log.Fatalf("sessions: %v", err)
	}

	var pages map[string]*template.Template
	if *uiMode == "spa" {
		if _, err := os.Stat(filepath.Join(*clientDir, "index.html")); err != nil {
			log.Fatalf("client dist missing at %s (run make client-build first)", *clientDir)
		}
	} else {
		var err error
		pages, err = web.LoadPages(filepath.Join(*webDir, "templates"))
		if err != nil {
			log.Fatalf("load templates: %v", err)
		}
	}
	weightsCfg, err := match.LoadWeights(*weights)
	if err != nil {
		log.Fatalf("load weights: %v", err)
	}
	model := ai.Disabled()
	if cfg.HasModel() {
		model, err = ai.NewHTTPModel(cfg.ModelEndpoint, cfg.ModelName, cfg.ModelToken)
		if err != nil {
			log.Fatalf("model: %v", err)
		}
		log.Print("AI explanations enabled")
	}

	// fmt.Printf("%v", model)

	srv := &web.Server{DB: database, Sessions: sessions, Pages: pages, Weights: weightsCfg, AI: model,
		CookieSecure: secure,
		LoginLimiter: web.NewLimiter(10, time.Minute),
		AILimiter:    web.NewLimiter(30, time.Minute),
		MsgLimiter:   web.NewLimiter(30, time.Minute),
		UI:           *uiMode, ClientDist: *clientDir}
	log.Printf("db=%s weights=%s", *dbPath, *weights)
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, web.LogRequests(srv.NewMux(filepath.Join(*webDir, "static")))))
}

func sessionsFromConfig(key, addr string, secure bool) (*auth.Sessions, error) {
	if key != "" {
		return auth.NewSessions(key, secure)
	}
	if util.LoopbackAddr(addr) {
		log.Printf("WARNING: FLATMATE_SESSION_KEY unset; using ephemeral key (sessions die on restart)")
		return auth.NewEphemeralSessions(secure)
	}
	return nil, fmt.Errorf("FLATMATE_SESSION_KEY is required when serving on %q", addr)
}
