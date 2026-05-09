package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

var config = LoadConfig()

var defaultConfig = Config{
	Backend: BackendConf{
		Host:      "localhost",
		PublicURL: "http://localhost:4242",
	},
	Stripe: stripeConf{
		Key:            "",
		EndpointSecret: "",
	},
	Email: emailConf{
		User:     "info@theotowngarage.com",
		Host:     "smtppro.zoho.com",
		Port:     465,
		Password: "",
	},
}

type stripeConf struct {
	// sk_xxx...xxx
	Key string
	// whsec_xxx...xxx
	EndpointSecret string
}

type emailConf struct {
	User     string
	Host     string
	Port     int
	Password string
}

type BackendConf struct {
	// Bind address (e.g. 0.0.0.0, localhost) — not the externally-reachable host.
	Host string
	Port int
	// Externally-reachable URL behind any reverse proxy, used for Stripe redirects and email links.
	PublicURL string
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	CookiePrivateKey string
}

type Config struct {
	Backend BackendConf
	Stripe  stripeConf
	Email   emailConf
}

// secretOverride sets a config field from (in priority order):
// 1. Environment variable
// 2. Secret file at /run/secrets/<filename>
// 3. Existing config value (unchanged)
func secretOverride(field *string, envKey string, secretFile string) {
	if val, ok := os.LookupEnv(envKey); ok && len(val) > 0 {
		*field = val
		return
	}
	if data, err := os.ReadFile("/run/secrets/" + secretFile); err == nil {
		if val := strings.TrimSpace(string(data)); len(val) > 0 {
			*field = val
		}
	}
}

func envOverride(field *string, envKey string) {
	if val, ok := os.LookupEnv(envKey); ok && len(val) > 0 {
		*field = val
	}
}

func envOverrideInt(field *int, envKey string) {
	if val, ok := os.LookupEnv(envKey); ok && len(val) > 0 {
		if n, err := strconv.Atoi(val); err == nil {
			*field = n
		}
	}
}

func LoadConfig() Config {
	var parseconf Config
	_, err := toml.DecodeFile("../config/_default/config.toml", &parseconf)
	if err != nil {
		log.Print("Failed to load config - ", err)
		parseconf = defaultConfig
	}

	// Environment variables override file-based config
	envOverride(&parseconf.Backend.Host, "BACKEND_HOST")
	envOverride(&parseconf.Backend.PublicURL, "BACKEND_PUBLIC_URL")
	envOverrideInt(&parseconf.Backend.Port, "BACKEND_PORT")

	// Secrets: env var > /run/secrets/<file> > config.toml
	secretOverride(&parseconf.Backend.CookiePrivateKey, "COOKIE_STORE_KEY", "cookie_store_key")
	secretOverride(&parseconf.Stripe.Key, "STRIPE_KEY", "stripe_key")
	secretOverride(&parseconf.Stripe.EndpointSecret, "STRIPE_WEBSOCK_KEY", "stripe_webhook_secret")
	secretOverride(&parseconf.Email.Password, "EMAIL_PASSWORD", "email_password")

	if len(parseconf.Backend.CookiePrivateKey) == 0 {
		log.Fatal("COOKIE_STORE_KEY must be set via environment variable, secret file, or config")
	}

	return parseconf
}
