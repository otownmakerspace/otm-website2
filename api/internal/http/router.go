// Package http builds the application's net/http ServeMux from the
// Stripe service and session store.
package http

import (
	"database/sql"
	stdhttp "net/http"

	"github.com/iustin94/makerspace/api/internal/email"
	"github.com/iustin94/makerspace/api/internal/session"
	"github.com/iustin94/makerspace/api/internal/stripe"
)

// Mux composes the route table.
//
// The application has two runtimes split by Host:
//
//   - Apex (e.g. development.makerspace.olaru.dk) — the marketing site,
//     served as static Hugo HTML via the file-server fallback. Only the
//     Stripe webhook is dynamic on this host (it doesn't need the cookie).
//
//   - Members (e.g. members.development.makerspace.olaru.dk) — every
//     auth-touching surface: dashboard, login, checkout, profile, billing.
//     Same Go binary, different Host. ServeMux's host-prefixed patterns
//     route requests by their Host header. When MembersHost is empty,
//     routes register without a host prefix (any-host) so tests and local
//     stand-alone runs still work.
func Mux(
	staticDir string,
	publicURL string,
	membersHost string,
	db *sql.DB,
	sessions *session.Store,
	mailer *email.Mailer,
	svc *stripe.Service,
) *stdhttp.ServeMux {
	mux := stdhttp.NewServeMux()

	// onMembers prefixes a pattern with the members Host so it only matches
	// the member portal subdomain. If membersHost is empty the prefix is
	// dropped — pattern stays any-host.
	onMembers := func(pattern string) string {
		if membersHost == "" {
			return pattern
		}
		// Patterns can include a method prefix: "GET /foo". Insert host between
		// the method and the path.
		// Safe approach: split on first space.
		for i := 0; i < len(pattern); i++ {
			if pattern[i] == ' ' {
				return pattern[:i+1] + membersHost + pattern[i+1:]
			}
		}
		// No method prefix — bare path.
		return membersHost + pattern
	}

	staticFallback := stdhttp.FileServer(stdhttp.Dir(staticDir))

	// ───────────────────────── apex (any host with no more-specific match) ──
	// Stripe webhook — Stripe POSTs to whatever URL is configured in the
	// dashboard. We accept it on any host so configuration mistakes don't
	// drop events silently.
	mux.HandleFunc("/webhook", svc.HandleWebhook)

	// ───────────────────────── members runtime (host-prefixed) ──────────────
	// Dashboard — backend-rendered page with auth-aware chrome.
	mux.HandleFunc(onMembers("GET /dashboard"), svc.ServeDashboardPage())
	mux.HandleFunc(onMembers("GET /da/dashboard"), svc.ServeDashboardPage())

	// Dashboard fragments (htmx-loaded into the page)
	mux.HandleFunc(onMembers("GET /dashboard/hero"), svc.ServeHero())
	mux.HandleFunc(onMembers("/subscriptions"), svc.ServeSubscriptions())
	mux.HandleFunc(onMembers("/cancel-subscription"), svc.CancelSubscription())
	mux.HandleFunc(onMembers("GET /cancel-modal"), svc.ServeCancelModal())
	mux.HandleFunc(onMembers("GET /cancel-modal/dismiss"), svc.DismissModal())

	// Profile + password
	mux.HandleFunc(onMembers("GET /profile"), svc.ServeProfile())
	mux.HandleFunc(onMembers("POST /profile"), svc.UpdateProfile())
	mux.HandleFunc(onMembers("GET /profile/edit"), svc.ServeProfileEdit())
	mux.HandleFunc(onMembers("GET /profile/password"), svc.ServePassword())
	mux.HandleFunc(onMembers("POST /profile/password"), svc.UpdatePassword())

	// Billing
	mux.HandleFunc(onMembers("GET /invoices"), svc.ServeInvoices())
	mux.HandleFunc(onMembers("GET /billing-portal"), svc.BillingPortal())

	// Checkout — signup form (Hugo page) + form handler. Both on members host.
	mux.Handle(onMembers("GET /checkout/"), sessions.RedirectIfLoggedIn("/dashboard", staticFallback))
	mux.HandleFunc(onMembers("POST /checkout/"), svc.CreateCheckoutSession())
	mux.HandleFunc(onMembers("GET /checkout/prices"), svc.ServePrices())
	mux.HandleFunc(onMembers("/re-checkout"), svc.CreateCheckoutSession())
	mux.Handle(onMembers("GET /da/checkout/"), sessions.RedirectIfLoggedIn("/dashboard", staticFallback))

	// Auth flows (login, logout, password reset)
	mux.HandleFunc(onMembers("/logout"), sessions.Logout)
	mux.Handle(onMembers("GET /login/"), sessions.RedirectIfLoggedIn("/dashboard", staticFallback))
	mux.Handle(onMembers("GET /da/login/"), sessions.RedirectIfLoggedIn("/dashboard", staticFallback))
	mux.HandleFunc(onMembers("POST /login/"), sessions.Login(db))
	mux.HandleFunc(onMembers("POST /request-reset"), session.RequestReset(db, mailer, publicURL))
	mux.HandleFunc(onMembers("POST /reset-password/"), session.ResetPassword(db))
	mux.HandleFunc(onMembers("POST /reset-password"), session.ResetPassword(db))

	// ───────────────────────── apex static fallback (any host) ──────────────
	// Serves Hugo-built marketing site. Registered last so member routes win
	// for the members host. The apex host has no more-specific patterns, so
	// every request falls here.
	mux.Handle("/", staticFallback)
	return mux
}
