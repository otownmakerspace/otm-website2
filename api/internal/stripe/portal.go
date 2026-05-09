package stripe

import (
	"log"
	"net/http"

	stripepkg "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/billingportal/session"

	"github.com/iustin94/makerspace/api/internal/i18n"
)

// BillingPortal returns the GET /billing-portal handler. Creates a Stripe-hosted
// Customer Portal session for the logged-in user and 303-redirects there.
//
// The Customer Portal lets the user update payment method, download invoices,
// and (depending on Stripe Dashboard configuration) cancel/upgrade their plan
// — all without us building UI for it. ReturnURL points back at /dashboard.
func (s *Service) BillingPortal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		lang := i18n.FromRequest(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			http.Redirect(w, r, i18n.Path(lang, "/login"), http.StatusSeeOther)
			return
		}
		customerID, _ := sess.Values["customer_id"].(string)
		if customerID == "" {
			log.Print("portal: missing customer_id in session")
			http.Redirect(w, r, i18n.Path(lang, "/dashboard?error=no_customer"), http.StatusSeeOther)
			return
		}

		ps, err := session.New(&stripepkg.BillingPortalSessionParams{
			Customer:  stripepkg.String(customerID),
			ReturnURL: stripepkg.String(s.Cfg.Backend.PublicURL + i18n.Path(lang, "/dashboard")),
		})
		if err != nil {
			log.Print("portal: stripe session.New: ", err)
			http.Redirect(w, r, i18n.Path(lang, "/dashboard?error=portal_failed"), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, ps.URL, http.StatusSeeOther)
	}
}
