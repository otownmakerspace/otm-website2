// Command server is the makerspace.olaru.dk backend HTTP entrypoint.
//
// Responsibilities (kept intentionally small):
//   - load Config from env + secret files + TOML
//   - open the libsql DB and run migrations
//   - in test mode, seed the test user (real Stripe customer + sub)
//   - construct dependencies and the route mux
//   - in release mode, backfill the Gmail mailing-list label from the DB
//   - bind and serve
//
// Domain logic lives under internal/. Adding a new endpoint or flow does not
// require editing this file — it only needs registering in internal/http.Mux.
package main

import (
	"context"
	"database/sql"
	"log"
	stdhttp "net/http"
	"time"

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
	switch {
	case !contacts.Enabled():
		log.Print("google contacts: mailing-list sync DISABLED (GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET/GOOGLE_REFRESH_TOKEN not all set)")
	case cfg.Backend.IsTest:
		log.Printf("google contacts: mailing-list sync ENABLED (label %q); startup backfill SKIPPED — test-mode data must never reach the real mailing list", cfg.Google.ContactGroup)
	default:
		log.Printf("google contacts: mailing-list sync ENABLED (label %q); backfilling from the membership database", cfg.Google.ContactGroup)
		go backfillMailingList(database, contacts)
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

// backfillMailingList makes the Google Contacts mailing-list label mirror the
// active members in THIS deployment's database — the authoritative member
// list. Mirror means both directions: missing members are added AND labelled
// contacts who aren't active members are unlabelled (so drift from the days
// before the sync existed — or from the retired cmd/googlebackfill tool that
// once seeded the label from a local dev DB copy — heals on the next start).
// Consequently the label is database-owned: contacts added to it by hand in
// Gmail get stripped again here. Runs in the background on every release-mode
// start; SyncMembers is duplicate-safe and idempotent, so restarts converge
// to a no-op and a half-finished run heals itself on the next one.
func backfillMailingList(database *sql.DB, contacts *googlecontacts.Client) {
	users, err := db.ListActive(database)
	if err != nil {
		log.Print("google contacts: backfill: list active members: ", err)
		return
	}
	// Refuse to mirror an empty member set: zero active members is far more
	// likely a misconfigured DB path (fresh file, wrong volume mount) than a
	// makerspace with no members — and mirroring it would strip the whole
	// label. Manual pruning is the right tool for a genuinely empty roster.
	if len(users) == 0 {
		log.Print("google contacts: backfill SKIPPED — no active members in the database (empty label mirror refused)")
		return
	}
	members := make([]googlecontacts.Member, len(users))
	for i, u := range users {
		members[i] = googlecontacts.Member{Email: u.Email, Name: u.Name}
	}
	// Generous ceiling: the run is a couple of requests when converged, but a
	// first fill creates contacts at a quota-friendly pace (~0.7s each).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	created, labelled, already, removed, err := contacts.SyncMembers(ctx, members)
	if err != nil {
		log.Print("google contacts: backfill INCOMPLETE (self-heals on next restart): ", err)
	}
	log.Printf("google contacts: backfill done — %d active member(s): %d contact(s) created, %d labelled, %d already in place, %d unlabelled",
		len(members), created, labelled, already, removed)
}
