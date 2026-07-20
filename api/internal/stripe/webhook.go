package stripe

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	stripepkg "github.com/stripe/stripe-go/v82"
	stripesession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"

	dbpkg "github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/email"
	"github.com/iustin94/makerspace/api/internal/user"
)

// HandleWebhook is the POST /webhook handler. It verifies the Stripe signature
// against the configured endpoint secret, then dispatches by event type.
func (s *Service) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("webhook: read body: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event, err := webhook.ConstructEventWithOptions(
		body,
		r.Header.Get("Stripe-Signature"),
		s.Cfg.Stripe.EndpointSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		log.Printf("webhook: signature verification failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case stripepkg.EventTypeCheckoutSessionCompleted,
		stripepkg.EventTypeCheckoutSessionAsyncPaymentSucceeded:
		var checkoutSess stripepkg.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &checkoutSess); err != nil {
			log.Printf("webhook: parse JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := s.fulfillCheckout(checkoutSess.ID); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

	case stripepkg.EventTypeCustomerSubscriptionDeleted:
		var sub stripepkg.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("webhook: parse JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := s.handleSubscriptionEnded(sub); err != nil {
			log.Printf("webhook: handleSubscriptionEnded: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return

	case stripepkg.EventTypeCustomerSubscriptionUpdated:
		var sub stripepkg.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("webhook: parse JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// On a cycle advance, reset the reactivation counter. Stripe puts the
		// old value of any changed field in previous_attributes; we only care
		// that current_period_end was among the changed keys.
		//
		// Best-effort: a failed metadata write shouldn't bubble up as an error
		// to Stripe (that would trigger webhook retries for a cosmetic reset).
		if _, ok := event.Data.PreviousAttributes["current_period_end"]; ok {
			if _, err := subscription.Update(sub.ID, &stripepkg.SubscriptionParams{
				Params: stripepkg.Params{
					Metadata: map[string]string{
						reactivationsMetadataKey: "0",
					},
				},
			}); err != nil {
				log.Printf("webhook: failed to reset reactivations counter on sub %s: %v", sub.ID, err)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

// fulfillCheckout reads the metadata from the completed Stripe checkout session
// (which carries the bcrypt-hashed password and other signup fields) and
// inserts the user row.
//
// Idempotent: callable from both the webhook and the post-checkout success
// redirect. If a user with this email already exists, fulfillment is a no-op
// (no duplicate insert, no resent welcome email). This matters because the
// redirect and the webhook race — whichever arrives second must not error.
func (s *Service) fulfillCheckout(checkoutSessionID string) error {
	log.Print("webhook: fulfilling checkout")
	params := &stripepkg.CheckoutSessionParams{}
	params.AddExpand("line_items")
	sess, err := stripesession.Get(checkoutSessionID, params)
	if err != nil {
		log.Print("webhook: retrieve session metadata: ", err)
		return err
	}

	meta := sess.Metadata
	u := user.User{
		Name:       meta["name"],
		Email:      meta["email"],
		Phone:      meta["phone"],
		Active:     true,
		Password:   []byte(meta["pass"]),
		CustomerID: sess.Customer.ID,
	}
	exists, err := dbpkg.EmailExists(s.DB, u.Email)
	if err != nil {
		log.Print("webhook: EmailExists: ", err)
		return err
	}
	if exists {
		// Returning member, not a fresh signup. This covers a lapsed member
		// re-subscribing — including the self-heal case where their old customer
		// was gone and checkout minted a NEW one, so the stored customer_id is now
		// stale. Rebind the (possibly new) customer and mark active so the
		// dashboard, which looks up subscriptions by customer_id, finds the sub.
		// Idempotent: safe for the redirect/webhook race and repeated events, and
		// a no-op when nothing changed. We deliberately don't touch name/phone/
		// password (metadata carries none for a re-subscribe) or resend the
		// welcome email.
		if err := dbpkg.ReactivateReturning(s.DB, u.Email, u.CustomerID); err != nil {
			log.Print("webhook: ReactivateReturning: ", err)
			return err
		}
		log.Printf("webhook: rebound customer %s to returning member %s", u.CustomerID, u.Email)
		return nil
	}
	if err := dbpkg.AddUser(s.DB, u); err != nil {
		log.Print("webhook: AddUser: ", err)
		return err
	}

	if err := s.Mailer.Send(u.Email, u, email.Welcome, "https://discord.gg/CGBgKNwT", struct{}{}); err != nil {
		log.Print("webhook: welcome email: ", err)
		// non-fatal — user is created
	}

	n, err := dbpkg.CountActive(s.DB)
	if err != nil {
		log.Print("webhook: CountActive: ", err)
		return err
	}
	if err := s.Mailer.Send(s.Mailer.AdminAddress(), u, email.NewMember, "", struct{ Number int }{Number: n}); err != nil {
		log.Print("webhook: admin notify: ", err)
	}
	return nil
}

// handleSubscriptionEnded marks the user inactive and notifies both the user
// and the admin inbox.
func (s *Service) handleSubscriptionEnded(sub stripepkg.Subscription) error {
	if err := dbpkg.SetActive(s.DB, sub.Customer.ID, false); err != nil {
		log.Printf("webhook: SetActive: %v", err)
		return err
	}
	u, err := dbpkg.GetByCustomerID(s.DB, sub.Customer.ID)
	if err != nil {
		log.Printf("webhook: GetByCustomerID: %v", err)
		return err
	}

	n, err := dbpkg.CountActive(s.DB)
	if err != nil {
		return err
	}

	if err := s.Mailer.Send(s.Mailer.AdminAddress(), u, email.Unsubscription, "", struct{ Number int }{Number: n}); err != nil {
		log.Printf("webhook: admin unsubscription email: %v", err)
		return err
	}
	if err := s.Mailer.Send(u.Email, u, email.Goodbye, "https://discord.gg/CGBgKNwT", struct{}{}); err != nil {
		log.Printf("webhook: goodbye email: %v", err)
		return err
	}
	log.Print("webhook: subscription ended for ", u.Email)
	return nil
}
