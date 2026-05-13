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
	Cfg      config.Config
	DB       *sql.DB
	Sessions *session.Store
	Mailer   *email.Mailer
	HugoHead template.HTML // <head> contents lifted from Hugo's docs/index.html for the dashboard's chrome
}

// NewService returns a Service. All dependencies must be non-nil. HugoHead is
// resolved from the static-files directory and may be empty if the static
// site isn't bundled — the dashboard template falls back to a minimal head.
func NewService(cfg config.Config, db *sql.DB, sessions *session.Store, mailer *email.Mailer) *Service {
	return &Service{
		Cfg:      cfg,
		DB:       db,
		Sessions: sessions,
		Mailer:   mailer,
		HugoHead: loadHugoHead(cfg.Backend.StaticDir),
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

		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			log.Print("checkout: new member signup")
		} else if active, _ := sess.Values["active"].(bool); active {
			emailAddr, _ := sess.Values["email"].(string)
			log.Printf("checkout: %s already has active membership", emailAddr)
			http.Redirect(w, r, i18n.Path(lang, "/dashboard?error=already_active"), http.StatusSeeOther)
			return
		} else {
			// Re-checkout: existing customer with a lapsed subscription. No form,
			// so we fall back to the configured default price (Cfg.Stripe.PriceID).
			emailAddr, _ := sess.Values["email"].(string)
			log.Printf("checkout: %s is re-subscribing", emailAddr)
			s.serveCheckout(w, r, user.FromSession(*sess), s.Cfg.Stripe.PriceID)
			return
		}

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
		s.serveCheckout(w, r, u, priceID)
	}
}

// serveCheckout creates a Stripe checkout session and redirects the user to it.
// Metadata round-trips through Stripe so the post-payment webhook can recreate
// the user row.
//
// priceID must already be validated by the caller (validatePriceID) so we can
// trust it as a single tier line item.
//
// The user's consent timestamps (terms / waiver / privacy) are stamped into
// the checkout session's metadata so they end up on the resulting Stripe
// customer + subscription — auditable from the Stripe Dashboard later.
func (s *Service) serveCheckout(w http.ResponseWriter, r *http.Request, u user.User, priceID string) {
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
		http.Redirect(w, r, i18n.Path(lang, "/checkout/?reason=failed_session"), http.StatusSeeOther)
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
