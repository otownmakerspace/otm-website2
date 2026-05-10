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

	"github.com/iustin94/makerspace/api/internal/config"
	"github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/email"
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
	mailer := email.NewMailer(cfg.Email, cfg.Brand)
	svc := stripe.NewService(cfg, database, sessions, mailer)

	mux := apphttp.Mux(cfg.Backend.StaticDir, cfg.Backend.PublicURL, cfg.Backend.MembersHost, database, sessions, mailer, svc)

	addr := cfg.Backend.HostAddr()
	log.Printf("listening on %s (public URL: %s, members host: %s)", addr, cfg.Backend.PublicURL, cfg.Backend.MembersHost)
	log.Fatal(stdhttp.ListenAndServe(addr, mux))
}
