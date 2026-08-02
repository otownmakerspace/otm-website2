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
	Brand   BrandConf
	Captcha CaptchaConf
	Google  GoogleConf
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

// BrandConf is the public-facing identity rendered into emails (and anywhere
// the backend speaks for the organisation). Three text fields because the
// wordmark is two-tone — keeping the split explicit avoids fragile
// string-parsing in templates and lets a deploy choose its own casing/colours.
//
// LogoURL, when set, swaps the text wordmark for an <img> at the top of every
// email. Email clients that block images fall back to the alt text styled to
// look like the brand wordmark. Leave empty to keep the text wordmark only.
type BrandConf struct {
	Name            string // Full display name, e.g. "O'Town Makerspace". Used in body copy and the footer.
	WordmarkLeading string // Dark/primary part of the text wordmark, e.g. "O'TOWN".
	WordmarkAccent  string // Accent-coloured part of the text wordmark, e.g. "MAKERSPACE".
	LogoURL         string // Absolute URL to a horizontal wordmark image (PNG or SVG). Empty triggers per-deployment derivation in Load() from MarketingBaseURL → PublicURL → makerspace.olaru.dk, so staging emails reference staging assets automatically.
}

// GoogleConf drives the Google Contacts mailing-list sync: on subscription
// create/cancel the backend adds/removes the member in a Contacts label on
// the admin's Gmail account (typing the label into To/BCC in Gmail expands to
// all members). Leaving any credential empty disables the sync — deployments
// without Google secrets behave exactly as before.
//
// The refresh token is minted once, interactively, with cmd/googleauth; the
// client id/secret come from a Google Cloud OAuth client (Desktop type) with
// the People API enabled.
type GoogleConf struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	ContactGroup string // Contacts label display name holding active members.
}

// CaptchaConf holds the HMAC key used by the Altcha proof-of-work CAPTCHA on
// the signup + password-reset forms. Empty key disables verification — only
// safe in local dev / tests.
type CaptchaConf struct {
	HMACKey string // random 32+ byte secret, set per environment
}

// Defaults applied when no config file is present and no env override is set.
//
// Email config (User/Host/Port) intentionally has no defaults — providers
// and mailbox addresses are deployment-specific. Set via EMAIL_ADDRESS /
// EMAIL_HOST / EMAIL_PORT env vars (CI/CD passes these from per-environment
// GitHub secrets/variables) or via a TOML config file. The Go SMTP client
// surfaces a clear error at first send if these are blank.
var defaults = Config{
	Backend: BackendConf{
		Host:              "localhost",
		PublicURL:         "http://localhost:4242",
		StaticDir:         "./public",
		DBPath:            "./local.db",
		SessionCookieName: "otm-session",
	},
	Brand: BrandConf{
		Name:            "O'Town Makerspace",
		WordmarkLeading: "O'TOWN",
		WordmarkAccent:  "MAKERSPACE",
		// LogoURL is intentionally left empty here. Load() derives it from
		// MarketingBaseURL (or PublicURL) after all overrides so each
		// deployment serves its own host's copy of the wordmark instead of
		// every environment pointing at production.
	},
	Google: GoogleConf{
		ContactGroup: "Makerspace Members",
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
	envOverride(&c.Email.Host, "EMAIL_HOST")
	envOverrideInt(&c.Email.Port, "EMAIL_PORT")
	secretOverride(&c.Email.User, "EMAIL_ADDRESS", "email_address")
	secretOverride(&c.Email.Password, "EMAIL_PASSWORD", "email_password")

	envOverride(&c.Brand.Name, "BRAND_NAME")
	envOverride(&c.Brand.WordmarkLeading, "BRAND_WORDMARK_LEADING")
	envOverride(&c.Brand.WordmarkAccent, "BRAND_WORDMARK_ACCENT")
	envOverride(&c.Brand.LogoURL, "BRAND_LOGO_URL")

	secretOverride(&c.Captcha.HMACKey, "ALTCHA_HMAC_KEY", "altcha_hmac_key")

	// Google Contacts sync. The client id isn't strictly secret, but riding
	// the secret-file mechanism keeps all three Google credentials in one
	// place operationally.
	secretOverride(&c.Google.ClientID, "GOOGLE_CLIENT_ID", "google_client_id")
	secretOverride(&c.Google.ClientSecret, "GOOGLE_CLIENT_SECRET", "google_client_secret")
	secretOverride(&c.Google.RefreshToken, "GOOGLE_REFRESH_TOKEN", "google_refresh_token")
	envOverride(&c.Google.ContactGroup, "GOOGLE_CONTACT_GROUP")

	// SITE_RELEASE presence (any value) marks production. Absence = test.
	_, isRelease := os.LookupEnv("SITE_RELEASE")
	c.Backend.IsTest = !isRelease

	if c.Backend.CookiePrivateKey == "" {
		log.Fatal("config: COOKIE_STORE_KEY must be set via environment variable, secret file, or config")
	}

	// Derive LogoURL from the deployment's own marketing host so staging
	// emails reference staging assets, not production. Explicit BRAND_LOGO_URL
	// (env / secret / toml) still wins because it sets c.Brand.LogoURL above.
	// If neither MarketingBaseURL nor PublicURL is configured (bare local
	// dev / tests), LogoURL stays empty and the email layout's text-wordmark
	// branch renders instead — no hardcoded production string in the binary.
	if c.Brand.LogoURL == "" {
		base := c.Backend.MarketingBaseURL
		if base == "" {
			base = c.Backend.PublicURL
		}
		if base != "" {
			c.Brand.LogoURL = strings.TrimRight(base, "/") + "/images/branding/logos/wordmark-horizontal.svg"
		}
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
