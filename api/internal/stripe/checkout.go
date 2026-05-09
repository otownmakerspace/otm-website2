// Package stripe wires the Stripe checkout, webhook, and subscription endpoints.
//
// The package depends on db, email, session, user, and the Stripe SDK.
// It exposes Service via NewService(deps) so handlers reach their dependencies
// without package-level globals.
package stripe

import (
	"database/sql"
	"log"
	"net/http"
	"net/url"

	stripepkg "github.com/stripe/stripe-go/v82"
	stripesession "github.com/stripe/stripe-go/v82/checkout/session"
	"golang.org/x/crypto/bcrypt"

	"github.com/iustin94/makerspace/api/internal/config"
	dbpkg "github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/email"
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
}

// NewService returns a Service. All dependencies must be non-nil.
func NewService(cfg config.Config, db *sql.DB, sessions *session.Store, mailer *email.Mailer) *Service {
	return &Service{Cfg: cfg, DB: db, Sessions: sessions, Mailer: mailer}
}

// CreateCheckoutSession is the POST /checkout/ handler. For new users, it
// validates the form + uniqueness, hashes the password, and starts a Stripe
// checkout. For already-logged-in users, it skips form handling and starts a
// new checkout under their existing customer ID (re-subscribe path).
func (s *Service) CreateCheckoutSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		log.Print("checkout: incoming")

		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			log.Print("checkout: new member signup")
		} else if active, _ := sess.Values["active"].(bool); active {
			emailAddr, _ := sess.Values["email"].(string)
			log.Printf("checkout: %s already has active membership", emailAddr)
			http.Redirect(w, r, "/dashboard?error=already_active", http.StatusSeeOther)
			return
		} else {
			emailAddr, _ := sess.Values["email"].(string)
			log.Printf("checkout: %s is re-subscribing", emailAddr)
			s.serveCheckout(w, r, user.FromSession(*sess))
			return
		}

		if err := r.ParseForm(); err != nil || !validateCheckoutInput(r.Form) {
			log.Print("checkout: malformed request")
			http.Redirect(w, r, "/checkout/", http.StatusSeeOther)
			return
		}
		pass := r.Form.Get("pass")
		if len(pass) < session.MinPasswordLen || len(pass) > session.MaxPasswordLen {
			http.Redirect(w, r, "/checkout/", http.StatusSeeOther)
			return
		}
		exists, err := dbpkg.EmailExists(s.DB, r.Form.Get("email"))
		if err != nil {
			log.Print("checkout: email lookup failed: ", err)
			return
		}
		if exists {
			http.Redirect(w, r, "/checkout/?reason=email_exists", http.StatusSeeOther)
			return
		}

		// Hash the password now so only the bcrypt hash (not the plaintext) flows
		// through Stripe metadata.
		hashed, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			log.Print("checkout: bcrypt failed: ", err)
			http.Redirect(w, r, "/checkout/?reason=failed_crypt", http.StatusSeeOther)
			return
		}
		u := user.FromForm(r)
		u.Password = hashed
		s.serveCheckout(w, r, u)
	}
}

// serveCheckout creates a Stripe checkout session and redirects the user to it.
// Metadata round-trips through Stripe so the post-payment webhook can recreate
// the user row.
func (s *Service) serveCheckout(w http.ResponseWriter, r *http.Request, u user.User) {
	params := &stripepkg.CheckoutSessionParams{
		SuccessURL: stripepkg.String(s.Cfg.Backend.PublicURL + "/checkout/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripepkg.String(s.Cfg.Backend.PublicURL + "/checkout/?reason=stripe_cancel"),
		Mode:       stripepkg.String(string(stripepkg.CheckoutSessionModeSubscription)),
		LineItems: []*stripepkg.CheckoutSessionLineItemParams{
			{
				Price:    stripepkg.String(s.Cfg.Stripe.PriceID),
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
	checkoutSess, err := stripesession.New(params)
	if err != nil {
		log.Printf("checkout: stripesession.New: %v", err)
		http.Redirect(w, r, "/checkout/?reason=failed_session", http.StatusSeeOther)
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
