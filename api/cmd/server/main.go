// Command server is the makerspace.olaru.dk backend HTTP entrypoint.
//
// Responsibilities (kept intentionally small):
//   - load Config from env + secret files + TOML
//   - open the libsql DB and run migrations
//   - in test mode, seed the test user (real Stripe customer + sub)
//   - construct dependencies and the route mux
//   - bind and serve
//
// Domain logic lives under internal/. Adding a new endpoint or flow does not
// require editing this file — it only needs registering in internal/http.Mux.
package main

import (
	"log"
	stdhttp "net/http"

	stripepkg "github.com/stripe/stripe-go/v82"

	"github.com/iustin94/makerspace/api/internal/captcha"
	"github.com/iustin94/makerspace/api/internal/config"
	"github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/email"
	"github.com/iustin94/makerspace/api/internal/googlecontacts"
	apphttp "github.com/iustin94/makerspace/api/internal/http"
	"github.com/iustin94/makerspace/api/internal/session"
	"github.com/iustin94/makerspace/api/internal/stripe"
)

func main() {
	cfg := config.Load()
	if cfg.Backend.IsTest {
		log.Print("starting TEST backend")
	} else {
		log.Print("starting RELEASE backend")
	}

	stripepkg.Key = cfg.Stripe.Key

	database, err := db.Open(cfg.Backend)
	if err != nil {
		log.Fatal("db.Open: ", err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatal("db.Migrate: ", err)
	}
	if cfg.Backend.IsTest {
		db.SeedTest(database, cfg)
	}

	sessions := session.NewStore([]byte(cfg.Backend.CookiePrivateKey), cfg.Backend.SessionCookieName, cfg.Backend.MarketingBaseURL)

	// Email links derive from config, never hardcoded hosts. Marketing falls
	// back to the public URL; the member portal is the members host (apex's
	// public URL when host-based routing is off, e.g. local dev).
	marketingURL := cfg.Backend.MarketingBaseURL
	if marketingURL == "" {
		marketingURL = cfg.Backend.PublicURL
	}
	portalURL := cfg.Backend.PublicURL
	if cfg.Backend.MembersHost != "" {
		portalURL = "https://" + cfg.Backend.MembersHost
	}
	mailer := email.NewMailer(cfg.Email, cfg.Brand, marketingURL, portalURL)
	captchaSvc := captcha.NewService(cfg.Captcha.HMACKey)
	if !captchaSvc.Enabled() {
		log.Print("captcha: DISABLED (ALTCHA_HMAC_KEY not set) — protected forms accept any submission")
	}
	contacts := googlecontacts.New(cfg.Google)
	if contacts.Enabled() {
		log.Printf("google contacts: mailing-list sync ENABLED (label %q)", cfg.Google.ContactGroup)
	} else {
		log.Print("google contacts: mailing-list sync DISABLED (GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET/GOOGLE_REFRESH_TOKEN not all set)")
	}
	svc := stripe.NewService(cfg, database, sessions, mailer, captchaSvc, contacts)

	mux := apphttp.Mux(cfg.Backend.StaticDir, cfg.Backend.PublicURL, cfg.Backend.MembersHost, database, sessions, mailer, svc, captchaSvc)

	addr := cfg.Backend.HostAddr()
	log.Printf("listening on %s (public URL: %s, members host: %s)", addr, cfg.Backend.PublicURL, cfg.Backend.MembersHost)
	// Non-secret config dump — useful for spotting misconfigured env vars
	// (e.g. SMTP_HOST set to an email address, missing STRIPE_PRICE_ID,
	// wrong brand override) in the very first lines of docker logs.
	log.Printf("config: smtp=%s:%d stripe_price_id=%s", cfg.Email.Host, cfg.Email.Port, cfg.Stripe.PriceID)
	log.Printf("brand: name=%q wordmark=%q/%q logo=%s", cfg.Brand.Name, cfg.Brand.WordmarkLeading, cfg.Brand.WordmarkAccent, cfg.Brand.LogoURL)
	log.Fatal(stdhttp.ListenAndServe(addr, mux))
}
