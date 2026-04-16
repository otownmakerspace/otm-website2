package main

import (
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

var config = LoadConfig()

var defaultConfig = Config{
	Backend: BackendConf{
		Host:     "localhost",
		Protocol: "http",
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
	// localhost, 10.11.12.13, theotowngarage.com
	Host string
	// http, https
	Protocol string
	Port     int
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	CookiePrivateKey string
}

type Config struct {
	Backend BackendConf
	Stripe  stripeConf
	Email   emailConf
}

// envOverride replaces a config value with an environment variable if set.
func envOverride(field *string, envKey string) {
	if val, ok := os.LookupEnv(envKey); ok && len(val) > 0 {
		*field = val
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
	envOverride(&parseconf.Backend.CookiePrivateKey, "COOKIE_STORE_KEY")
	envOverride(&parseconf.Stripe.Key, "STRIPE_KEY")
	envOverride(&parseconf.Stripe.EndpointSecret, "STRIPE_WEBSOCK_KEY")
	envOverride(&parseconf.Email.Password, "EMAIL_PASSWORD")

	if len(parseconf.Backend.CookiePrivateKey) == 0 {
		log.Fatal("COOKIE_STORE_KEY must be set via environment variable or config file")
	}

	return parseconf
}
