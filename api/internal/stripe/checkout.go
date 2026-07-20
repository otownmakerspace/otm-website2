// Package stripe wires the Stripe checkout, webhook, and subscription endpoints.
//
// The package depends on db, email, session, user, and the Stripe SDK.
// It exposes Service via NewService(deps) so handlers reach their dependencies
// without package-level globals.
package stripe

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	stripepkg "github.com/stripe/stripe-go/v82"
	stripesession "github.com/stripe/stripe-go/v82/checkout/session"
	"golang.org/x/crypto/bcrypt"

	"github.com/iustin94/makerspace/api/internal/captcha"
	"github.com/iustin94/makerspace/api/internal/config"
	dbpkg "github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/email"
	"github.com/iustin94/makerspace/api/internal/i18n"
	"github.com/iustin94/makerspace/api/internal/session"
	"github.com/iustin94/makerspace/api/internal/user"
)

// Service holds the dependencies the Stripe handlers need. Constructed once
// in main() and used to register routes.
type Service struct {
	Cfg       config.Config
	DB        *sql.DB
	Sessions  *session.Store
	Mailer    *email.Mailer
	Captcha   *captcha.Service
	HugoHead  template.HTML // <head> contents lifted from Hugo's docs/index.html for the dashboard's chrome
	LogoLight template.HTML // inline SVG of the light-mode horizontal lockup, mirrors marketing-site usage
	LogoDark  template.HTML // inline SVG of the dark-mode wordmark, mirrors marketing-site usage
}

// NewService returns a Service. All dependencies must be non-nil. HugoHead is
// resolved from the static-files directory and may be empty if the static
// site isn't bundled — the dashboard template falls back to a minimal head.
// LogoLight/LogoDark are read from the same static dir; if missing they stay
// empty and the header simply renders without a logo (no crash).
func NewService(cfg config.Config, db *sql.DB, sessions *session.Store, mailer *email.Mailer, captchaSvc *captcha.Service) *Service {
	return &Service{
		Cfg:       cfg,
		DB:        db,
		Sessions:  sessions,
		Mailer:    mailer,
		Captcha:   captchaSvc,
		HugoHead:  loadHugoHead(cfg.Backend.StaticDir),
		LogoLight: loadInlineSVG(cfg.Backend.StaticDir, "images/branding/logos/horizontal-lockup-notagline.svg"),
		LogoDark:  loadInlineSVG(cfg.Backend.StaticDir, "images/branding/logos/wordmark-white-transparent.svg"),
	}
}

// CreateCheckoutSession is the POST /checkout/ handler. For new users, it
// validates the form + uniqueness, hashes the password, and starts a Stripe
// checkout. For already-logged-in users, it skips form handling and starts a
// new checkout under their existing customer ID (re-subscribe path).
func (s *Service) CreateCheckoutSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		lang := i18n.FromRequest(r)
		log.Print("checkout: incoming")

		// Authenticated users don't belong on the anonymous-signup path: they
		// carry no signup form and can't solve the Altcha challenge enforced
		// below. They normally can't even reach here (the signup page is
		// login-gated), but a stale form or crafted POST could — hand them to the
		// dedicated re-subscribe flow, which owns its own dashboard-bound error
		// routing instead of the /checkout/ redirects the signup path uses.
		if auth, ok := sess.Values["authenticated"].(bool); ok && auth {
			s.serveReCheckout(w, r)
			return
		}

		// New (anonymous) signup from the public /checkout/ form. Anti-abuse:
		// honeypot first (cheapest), then Altcha proof-of-work. Honeypot triggers
		// a silent fake-success so bots don't learn what tripped them; Altcha
		// failure surfaces a real reason to the user.
		if captcha.HoneypotTriggered(r) {
			log.Print("checkout: honeypot triggered, silent drop")
			http.Redirect(w, r, i18n.Path(lang, "/checkout/success"), http.StatusSeeOther)
			return
		}
		if err := s.Captcha.Verify(r); err != nil {
			log.Printf("checkout: captcha verify: %v", err)
			http.Redirect(w, r, i18n.Path(lang, "/checkout/?reason=captcha_failed"), http.StatusSeeOther)
			return
		}
		log.Print("checkout: new member signup")

		if err := r.ParseForm(); err != nil || !validateCheckoutInput(r.Form) {
			log.Print("checkout: malformed request")
			http.Redirect(w, r, i18n.Path(lang, "/checkout/"), http.StatusSeeOther)
			return
		}
		// Consent enforcement. The form has `required` on each checkbox so most
		// users can't submit without ticking them — but a determined user can
		// DOM-edit and bypass that. Server-side check is the real enforcement.
		if !validateConsent(r.Form) {
			log.Print("checkout: consent checkboxes missing")
			http.Redirect(w, r, i18n.Path(lang, "/checkout/?reason=accept_required"), http.StatusSeeOther)
			return
		}
		pass := r.Form.Get("pass")
		if len(pass) < session.MinPasswordLen || len(pass) > session.MaxPasswordLen {
			http.Redirect(w, r, i18n.Path(lang, "/checkout/"), http.StatusSeeOther)
			return
		}

		// Price is server-side authoritative: always the configured
		// STRIPE_PRICE_ID. The form does not submit price_id (the prices.html
		// fragment renders a read-only display, not a selector). Fail fast if
		// deployment config is incomplete rather than handing Stripe an empty
		// price.
		priceID := s.Cfg.Stripe.PriceID
		if priceID == "" {
			log.Print("checkout: STRIPE_PRICE_ID is not configured")
			http.Redirect(w, r, i18n.Path(lang, "/checkout/?reason=invalid_price"), http.StatusSeeOther)
			return
		}

		exists, err := dbpkg.EmailExists(s.DB, r.Form.Get("email"))
		if err != nil {
			log.Print("checkout: email lookup failed: ", err)
			return
		}
		if exists {
			http.Redirect(w, r, i18n.Path(lang, "/checkout/?reason=email_exists"), http.StatusSeeOther)
			return
		}

		// Hash the password now so only the bcrypt hash (not the plaintext) flows
		// through Stripe metadata.
		hashed, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			log.Print("checkout: bcrypt failed: ", err)
			http.Redirect(w, r, i18n.Path(lang, "/checkout/?reason=failed_crypt"), http.StatusSeeOther)
			return
		}
		u := user.FromForm(r)
		u.Password = hashed
		s.serveCheckout(w, r, u, priceID, i18n.Path(lang, "/checkout/?reason=failed_session"))
	}
}

// ReCheckout is the POST /re-checkout handler behind the dashboard's "Become
// Member Again" button. It is a distinct entry point from the anonymous signup
// (POST /checkout/): no signup form, no captcha — the user already
// authenticated at login. Unauthenticated hits (expired session, stray
// bookmark) fall through to the public signup form; authenticated hits run the
// re-subscribe flow.
func (s *Service) ReCheckout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			lang := i18n.FromRequest(r)
			http.Redirect(w, r, i18n.Path(lang, "/checkout/"), http.StatusSeeOther)
			return
		}
		s.serveReCheckout(w, r)
	}
}

// serveReCheckout runs the authenticated re-subscribe flow. Callers must have
// already confirmed the session is authenticated.
//
// Every exit lands on /dashboard, never /checkout/ — that separation is the
// whole point of this being its own path. The signup URL is login-gated
// (RedirectIfLoggedIn), so routing a logged-in user's error through it would
// silently bounce them to the dashboard with the real cause erased. Here the
// dashboard is the deliberate destination and each redirect names its reason.
func (s *Service) serveReCheckout(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.Sessions.Get(r)
	lang := i18n.FromRequest(r)
	u := user.FromSession(*sess)

	// No form, so fall back to the configured default price. An empty price is a
	// deployment misconfiguration — surface it rather than handing Stripe an
	// empty price and failing downstream. (Checked before the Stripe lookup: no
	// point querying if we can't check out anyway.)
	if s.Cfg.Stripe.PriceID == "" {
		log.Print("re-checkout: STRIPE_PRICE_ID is not configured")
		http.Redirect(w, r, i18n.Path(lang, "/dashboard?error=invalid_price"), http.StatusSeeOther)
		return
	}

	// A single lookup drives the whole decision. We start a checkout ONLY from a
	// confirmed no-subscription state — subActive blocks (no double-subscribe),
	// subIndeterminate refuses (we couldn't confirm there's no existing sub, so
	// creating one could double-charge; by design we never guess), subCustomerGone
	// mints a fresh customer, subNone re-subscribes under the existing one.
	switch subscriptionStatus(u.CustomerID) {
	case subActive:
		log.Printf("re-checkout: %s already has active membership", u.Email)
		http.Redirect(w, r, i18n.Path(lang, "/dashboard?error=already_active"), http.StatusSeeOther)
		return
	case subIndeterminate:
		log.Printf("re-checkout: could not verify subscription status for %s; refusing to avoid a double-subscribe", u.Email)
		http.Redirect(w, r, i18n.Path(lang, "/dashboard?error=verify_failed"), http.StatusSeeOther)
		return
	case subCustomerGone:
		// Stored customer no longer exists — drop it so checkout mints a fresh
		// one; the completed-checkout webhook rebinds the new ID to this user by
		// email (fulfillCheckout -> ReactivateReturning).
		log.Printf("re-checkout: stored customer %s is gone; minting a fresh one for %s", u.CustomerID, u.Email)
		u.CustomerID = ""
	case subNone:
		// Existing customer (or none stored) with no active sub — normal path.
	}

	log.Printf("re-checkout: %s is re-subscribing", u.Email)
	s.serveCheckout(w, r, u, s.Cfg.Stripe.PriceID, i18n.Path(lang, "/dashboard?error=checkout_failed"))
}

// serveCheckout creates a Stripe checkout session and redirects the user to it.
// Metadata round-trips through Stripe so the post-payment webhook can recreate
// the user row.
//
// priceID must be non-empty — both callers guard `s.Cfg.Stripe.PriceID == ""`
// before calling — so we can trust it as a single tier line item. failURL is
// where the caller wants the
// user sent if Stripe rejects the session — signup keeps them on /checkout/,
// re-subscribe sends them to /dashboard — so a failure never lands a user on a
// URL that bounces for their auth state.
//
// The user's consent timestamps (terms / waiver / privacy) are stamped into
// the checkout session's metadata so they end up on the resulting Stripe
// customer + subscription — auditable from the Stripe Dashboard later.
func (s *Service) serveCheckout(w http.ResponseWriter, r *http.Request, u user.User, priceID, failURL string) {
	lang := i18n.FromRequest(r)
	params := &stripepkg.CheckoutSessionParams{
		SuccessURL: stripepkg.String(s.Cfg.Backend.PublicURL + i18n.Path(lang, "/checkout/success") + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripepkg.String(s.Cfg.Backend.PublicURL + i18n.Path(lang, "/checkout/?reason=stripe_cancel")),
		Mode:       stripepkg.String(string(stripepkg.CheckoutSessionModeSubscription)),
		LineItems: []*stripepkg.CheckoutSessionLineItemParams{
			{
				Price:    stripepkg.String(priceID),
				Quantity: stripepkg.Int64(1),
			},
		},
		SubscriptionData: &stripepkg.CheckoutSessionSubscriptionDataParams{
			BillingMode: &stripepkg.CheckoutSessionSubscriptionDataBillingModeParams{Type: stripepkg.String("flexible")},
		},
	}
	// Stripe rejects setting both Customer and CustomerEmail.
	if u.CustomerID != "" {
		params.Customer = stripepkg.String(u.CustomerID)
	} else {
		params.CustomerEmail = stripepkg.String(u.Email)
	}

	u.AddToStripeMetadata(params)
	// Audit trail: timestamp consent for each required doc. Same instant for
	// all three since they were submitted together; recorded separately so a
	// future legal review can see exactly what was accepted.
	if r.Form != nil && r.Form.Has("accept_terms") {
		now := time.Now().UTC().Format(time.RFC3339)
		params.AddMetadata("accepted_terms_at", now)
		params.AddMetadata("accepted_waiver_at", now)
		params.AddMetadata("accepted_privacy_at", now)
	}
	checkoutSess, err := stripesession.New(params)
	if err != nil {
		log.Printf("checkout: stripesession.New: %v", err)
		http.Redirect(w, r, failURL, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, checkoutSess.URL, http.StatusSeeOther)
}

// validateCheckoutInput rejects forms missing required signup fields.
func validateCheckoutInput(form url.Values) bool {
	for _, id := range []string{"email", "name", "pass"} {
		if !form.Has(id) {
			return false
		}
	}
	return true
}

// validateConsent confirms the user ticked all three required-doc checkboxes.
// Browsers send "on" for checked checkboxes by default; we just check presence.
func validateConsent(form url.Values) bool {
	for _, id := range []string{"accept_terms", "accept_waiver", "accept_privacy"} {
		if !form.Has(id) {
			return false
		}
	}
	return true
}
