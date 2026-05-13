package stripe

import (
	"log"
	"net/http"

	stripepkg "github.com/stripe/stripe-go/v82"
	stripesession "github.com/stripe/stripe-go/v82/checkout/session"

	"github.com/iustin94/makerspace/api/internal/i18n"
)

// ServeCheckoutSuccess is the GET /checkout/success handler. Stripe redirects
// the user here after payment with ?session_id={CHECKOUT_SESSION_ID}.
//
// Responsibilities:
//   - Verify the session_id is a real, paid Stripe Checkout Session (defence
//     against a user pasting the URL with a forged id).
//   - Run fulfillCheckout idempotently so the user row exists even if the
//     webhook hasn't arrived yet (the redirect can beat the webhook on slow
//     networks).
//   - 303-redirect to the login page so the user can sign in with the
//     credentials they just set on the signup form.
//
// The handler is permissive on error: if anything in the verification/fulfill
// path fails we still redirect to login rather than surface an error to a
// user whose payment succeeded. The webhook is the authoritative fulfillment
// path — Stripe will retry it, and the user can still log in once it lands.
func (s *Service) ServeCheckoutSuccess() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.FromRequest(r)
		loginURL := i18n.Path(lang, "/login/")

		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Redirect(w, r, loginURL, http.StatusSeeOther)
			return
		}
		sess, err := stripesession.Get(sessionID, nil)
		if err != nil {
			log.Printf("checkout success: stripesession.Get(%s): %v", sessionID, err)
			http.Redirect(w, r, loginURL, http.StatusSeeOther)
			return
		}
		if sess.PaymentStatus != stripepkg.CheckoutSessionPaymentStatusPaid &&
			sess.PaymentStatus != stripepkg.CheckoutSessionPaymentStatusNoPaymentRequired {
			log.Printf("checkout success: session %s payment_status=%s", sessionID, sess.PaymentStatus)
			http.Redirect(w, r, loginURL, http.StatusSeeOther)
			return
		}
		if err := s.fulfillCheckout(sessionID); err != nil {
			log.Printf("checkout success: fulfillCheckout: %v", err)
		}
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
	}
}
