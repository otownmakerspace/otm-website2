package stripe

import (
	"log"
	"net/http"

	stripepkg "github.com/stripe/stripe-go/v82"
	stripesession "github.com/stripe/stripe-go/v82/checkout/session"
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
//   - Delegate rendering to the static Hugo success page, which contains the
//     "Log in to your account" CTA.
//
// The handler is permissive on error: if anything in the verification/fulfill
// path fails we still render the success page rather than surface an error
// to a user whose payment succeeded. The webhook is the authoritative path —
// Stripe will retry it, and the user can still log in once it completes.
func (s *Service) ServeCheckoutSuccess(static http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			static.ServeHTTP(w, r)
			return
		}
		sess, err := stripesession.Get(sessionID, nil)
		if err != nil {
			log.Printf("checkout success: stripesession.Get(%s): %v", sessionID, err)
			static.ServeHTTP(w, r)
			return
		}
		if sess.PaymentStatus != stripepkg.CheckoutSessionPaymentStatusPaid &&
			sess.PaymentStatus != stripepkg.CheckoutSessionPaymentStatusNoPaymentRequired {
			log.Printf("checkout success: session %s payment_status=%s", sessionID, sess.PaymentStatus)
			static.ServeHTTP(w, r)
			return
		}
		if err := s.fulfillCheckout(sessionID); err != nil {
			log.Printf("checkout success: fulfillCheckout: %v", err)
		}
		static.ServeHTTP(w, r)
	}
}
