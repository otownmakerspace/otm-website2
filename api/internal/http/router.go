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

// Mux composes the route table. Hugo static files are served from staticDir;
// dynamic routes are registered before the catch-all so they take precedence.
func Mux(
	staticDir string,
	publicURL string,
	db *sql.DB,
	sessions *session.Store,
	mailer *email.Mailer,
	svc *stripe.Service,
) *stdhttp.ServeMux {
	mux := stdhttp.NewServeMux()

	// Stripe webhook (public, signature-verified inside handler)
	mux.HandleFunc("/webhook", svc.HandleWebhook)

	// Checkout
	staticFallback := stdhttp.FileServer(stdhttp.Dir(staticDir))
	mux.Handle("GET /checkout/", sessions.RedirectIfLoggedIn("/dashboard", staticFallback))
	mux.HandleFunc("POST /checkout/", svc.CreateCheckoutSession())
	mux.HandleFunc("/re-checkout", svc.CreateCheckoutSession())

	// Dashboard / subscriptions
	mux.HandleFunc("/subscriptions", svc.ServeSubscriptions())
	mux.HandleFunc("/cancel-subscription", svc.CancelSubscription())

	// Auth
	mux.HandleFunc("/logout", sessions.Logout)
	mux.Handle("GET /login/", sessions.RedirectIfLoggedIn("/dashboard", staticFallback))
	mux.HandleFunc("POST /login/", sessions.Login(db))

	// Password reset
	mux.HandleFunc("POST /request-reset", session.RequestReset(db, mailer, publicURL))
	mux.HandleFunc("POST /reset-password/", session.ResetPassword(db))
	mux.HandleFunc("POST /reset-password", session.ResetPassword(db))

	// Static fallback (Hugo). Registered last so dynamic routes win.
	mux.Handle("/", staticFallback)
	return mux
}
