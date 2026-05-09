package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the top-level configuration loaded from TOML + env + secret files.
type Config struct {
	Backend BackendConf
	Stripe  StripeConf
	Email   EmailConf
}

type BackendConf struct {
	// Bind address (e.g. 0.0.0.0, localhost) — not the externally-reachable host.
	Host string
	Port int
	// Externally-reachable URL behind any reverse proxy, used for Stripe redirects and email links.
	PublicURL string
	// 16/24/32 bytes (AES-128/192/256) for cookie HMAC signing.
	CookiePrivateKey string

	// Where to find the Hugo-built static site at runtime.
	StaticDir string
	// SQLite/libsql DB file path.
	DBPath string
	// Cookie name used by the session store. Distinct per deployment domain.
	SessionCookieName string
	// True for non-production deployments (skips production-only behaviour, enables seedTestUser).
	IsTest bool
	// MembersHost is the Host header for the member portal subdomain
	// (e.g. "members.development.makerspace.olaru.dk"). The router registers
	// dashboard + auth handlers on this host; the apex serves static Hugo
	// content only. Empty disables host-based routing (falls back to
	// any-host patterns — useful for tests / local stand-alone runs).
	MembersHost string
	// MarketingBaseURL is the full URL of the apex/marketing site
	// (e.g. "https://development.makerspace.olaru.dk"). Used for cross-host
	// redirects (logout returns the user to marketing). Empty falls back to
	// PublicURL — fine for tests / local stand-alone runs.
	MarketingBaseURL string
}

type StripeConf struct {
	Key            string // sk_xxx...xxx — secret key, server-only.
	PublishableKey string // pk_xxx...xxx — publishable key, safe to expose to the browser.
	EndpointSecret string // whsec_xxx...xxx
	PriceID        string // price_xxx — fallback price for re-checkout flows when the form doesn't carry a price_id (e.g. session-recovery). New signups should pick from /checkout/prices.
}

type EmailConf struct {
	User     string
	Host     string
	Port     int
	Password string
}

// Defaults applied when no config file is present and no env override is set.
var defaults = Config{
	Backend: BackendConf{
		Host:              "localhost",
		PublicURL:         "http://localhost:4242",
		StaticDir:         "./public",
		DBPath:            "./local.db",
		SessionCookieName: "otm-session",
	},
	Email: EmailConf{
		User: "info@theotowngarage.com",
		Host: "smtppro.zoho.com",
		Port: 465,
	},
}

// Load reads the configuration from (in priority order, highest wins per field):
//  1. Environment variables
//  2. Secret files at /run/secrets/<name>
//  3. TOML config file at BACKEND_CONFIG_PATH (or ./config.toml)
//  4. Built-in defaults
//
// Load is fatal on missing required secrets (cookie key) so misconfiguration
// surfaces at startup rather than at first request.
func Load() Config {
	c := defaults

	cfgPath := os.Getenv("BACKEND_CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "./config.toml"
	}
	if _, err := toml.DecodeFile(cfgPath, &c); err != nil {
		log.Printf("config: %s not loaded (%v); using defaults + env", cfgPath, err)
	}

	envOverride(&c.Backend.Host, "BACKEND_HOST")
	envOverride(&c.Backend.PublicURL, "BACKEND_PUBLIC_URL")
	envOverrideInt(&c.Backend.Port, "BACKEND_PORT")
	envOverride(&c.Backend.StaticDir, "BACKEND_STATIC_DIR")
	envOverride(&c.Backend.DBPath, "BACKEND_DB_PATH")
	envOverride(&c.Backend.SessionCookieName, "SESSION_COOKIE_NAME")
	envOverride(&c.Backend.MembersHost, "MEMBERS_HOST")
	envOverride(&c.Backend.MarketingBaseURL, "MARKETING_BASE_URL")

	secretOverride(&c.Backend.CookiePrivateKey, "COOKIE_STORE_KEY", "cookie_store_key")
	secretOverride(&c.Stripe.Key, "STRIPE_KEY", "stripe_key")
	secretOverride(&c.Stripe.PublishableKey, "STRIPE_PUBLISHABLE_KEY", "stripe_publishable_key")
	secretOverride(&c.Stripe.EndpointSecret, "STRIPE_WEBHOOK_SECRET", "stripe_webhook_secret")
	envOverride(&c.Stripe.PriceID, "STRIPE_PRICE_ID")
	secretOverride(&c.Email.Password, "EMAIL_PASSWORD", "email_password")

	// SITE_RELEASE presence (any value) marks production. Absence = test.
	_, isRelease := os.LookupEnv("SITE_RELEASE")
	c.Backend.IsTest = !isRelease

	if c.Backend.CookiePrivateKey == "" {
		log.Fatal("config: COOKIE_STORE_KEY must be set via environment variable, secret file, or config")
	}
	return c
}

// HostAddr returns the listen address (Host:Port) for http.ListenAndServe.
func (c BackendConf) HostAddr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// secretOverride reads from env var, then /run/secrets/<file>, in that priority.
func secretOverride(field *string, envKey, secretFile string) {
	if val, ok := os.LookupEnv(envKey); ok && val != "" {
		*field = val
		return
	}
	if data, err := os.ReadFile("/run/secrets/" + secretFile); err == nil {
		if val := strings.TrimSpace(string(data)); val != "" {
			*field = val
		}
	}
}

func envOverride(field *string, envKey string) {
	if val, ok := os.LookupEnv(envKey); ok && val != "" {
		*field = val
	}
}

func envOverrideInt(field *int, envKey string) {
	if val, ok := os.LookupEnv(envKey); ok && val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			*field = n
		}
	}
}
