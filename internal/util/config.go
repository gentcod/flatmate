package util

import (
	"net"
	"os"
	"strings"

	"github.com/gentcod/environ"
)

// Config centralizes every environment variable the app reads.
// Values load from a dotenv file when present; the process
// environment always wins; compiled defaults fill the rest.
type Config struct {
	Addr          string `json:"ADDR"`
	DBPath        string `json:"DB_PATH"`
	WeightsPath   string `json:"WEIGHTS_PATH"`
	WebDir        string `json:"WEB_DIR"`
	MigrationsDir string `json:"MIGRATIONS_DIR"`
	SeedPath      string `json:"SEED_PATH"`
	SessionKey    string `json:"FLATMATE_SESSION_KEY"`
	CookieSecure  string `json:"FLATMATE_COOKIE_SECURE"`
	ModelEndpoint string `json:"FLATMATE_MODEL_ENDPOINT"`
	ModelName     string `json:"FLATMATE_MODEL_NAME"`
	ModelToken    string `json:"FLATMATE_MODEL_TOKEN"`
	UI            string `json:"FLATMATE_UI"`
	ClientDist    string `json:"CLIENT_DIST"`
}

func defaults() Config {
	return Config{
		Addr:          ":8080",
		DBPath:        "flatmate.db",
		WeightsPath:   "weights.yaml",
		WebDir:        "web",
		MigrationsDir: "migrations",
		SeedPath:      "seeds/global.json",
		UI:            "legacy",
		ClientDist:    "client/dist",
	}
}

// LoadConfig loads path (a dotenv file) when it exists, then overlays
// the process environment. A missing file is fine.
func LoadConfig(path string) (Config, error) {
	c := defaults()
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := environ.Init(path, &c); err != nil {
				return c, err
			}
		}
	}
	overlayEnv(&c)
	return c, nil
}

// HasModel reports whether model calls are configured.
func (c Config) HasModel() bool {
	return c.ModelEndpoint != "" && c.ModelName != ""
}

// SecureCookies reports whether cookies should carry the Secure flag.
// Explicit env wins (1/true/yes or 0/false/no); otherwise anything but
// loopback serving is secure, so the local HTTP demo keeps working.
func (c Config) SecureCookies() bool {
	switch strings.ToLower(strings.TrimSpace(c.CookieSecure)) {
	case "0", "false", "no":
		return false
	case "1", "true", "yes":
		return true
	}
	return !LoopbackAddr(c.Addr)
}

// LoopbackAddr reports whether addr serves local development traffic:
// an empty host (":8080"), localhost, or a loopback IP.
func LoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if h := strings.ToLower(strings.Trim(host, "[] \t")); h == "" || h == "localhost" {
		return true
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

func overlayEnv(c *Config) {
	if v := os.Getenv("ADDR"); v != "" {
		c.Addr = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		c.DBPath = v
	}
	if v := os.Getenv("WEIGHTS_PATH"); v != "" {
		c.WeightsPath = v
	}
	if v := os.Getenv("WEB_DIR"); v != "" {
		c.WebDir = v
	}
	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		c.MigrationsDir = v
	}
	if v := os.Getenv("SEED_PATH"); v != "" {
		c.SeedPath = v
	}
	if v := os.Getenv("FLATMATE_SESSION_KEY"); v != "" {
		c.SessionKey = v
	}
	if v := os.Getenv("FLATMATE_COOKIE_SECURE"); v != "" {
		c.CookieSecure = v
	}
	if v := os.Getenv("FLATMATE_MODEL_ENDPOINT"); v != "" {
		c.ModelEndpoint = v
	}
	if v := os.Getenv("FLATMATE_MODEL_NAME"); v != "" {
		c.ModelName = v
	}
	if v := os.Getenv("FLATMATE_MODEL_TOKEN"); v != "" {
		c.ModelToken = v
	}
	if v := os.Getenv("FLATMATE_UI"); v != "" {
		c.UI = v
	}
	if v := os.Getenv("CLIENT_DIST"); v != "" {
		c.ClientDist = v
	}
}
