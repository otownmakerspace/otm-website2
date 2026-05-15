package stripe

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	stripepkg "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/subscription"

	"github.com/iustin94/makerspace/api/internal/i18n"
	"github.com/iustin94/makerspace/api/internal/user"
)

// maxReactivationsPerPeriod caps in-period cancel/reactivate ping-pong.
// One reactivation per billing cycle lets a user undo an accidental cancel
// but not loop through the flow repeatedly to test or game the system.
const maxReactivationsPerPeriod = 1

// reactivationsMetadataKey is the Stripe subscription metadata field that
// tracks how many times the current period has been reactivated. Reset to "0"
// by the customer.subscription.updated webhook on cycle advance.
const reactivationsMetadataKey = "reactivations_this_period"

//go:embed templates/*.html
var templatesFS embed.FS

// ErrNoSubs is returned when a customer has no active subscriptions.
var ErrNoSubs = errors.New("no subscriptions available")

// subscriptionView is the shape passed to templates/subscriptions.html.
// All fields are pre-formatted strings so the template stays logic-light.
type subscriptionView struct {
	Active           bool
	SubID            string
	Status           string
	PlanLabel        string // e.g. "200 DKK / month"
	StartedOn        string // e.g. "March 2024"
	NextRenewal      string // e.g. "5 June 2026"
	MemberSince      string // from Customer.Created
	CancelAtEnd      bool
	CancelEndsMsg    string // localized "Your membership ends on X..." with NextRenewal substituted
	OpenHouseDate    string // pre-formatted next Thursday for the localized hint
	OpenHouseLinkURL string // /events/ or /da/events/
	CanReactivate    bool   // false when reactivations_this_period >= cap; hides the reactivate button
	S                i18n.Strings
}

// reactivationsCount reads the per-period reactivation counter from a
// subscription's metadata. Missing or unparseable values are treated as 0.
func reactivationsCount(sub *stripepkg.Subscription) int {
	if sub == nil || sub.Metadata == nil {
		return 0
	}
	raw, ok := sub.Metadata[reactivationsMetadataKey]
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// listSubscriptions returns the iterator of active subs for a customer (max 1).
func listSubscriptions(customerID string) *subscription.Iter {
	params := &stripepkg.SubscriptionListParams{}
	params.Customer = stripepkg.String(customerID)
	params.Status = stripepkg.String("active")
	params.Limit = stripepkg.Int64(1)
	return subscription.List(params)
}

// formatAmount renders a Stripe minor-unit price as "<amount> <CCY> / <interval>".
// Stripe uses minor units (cents/øre); we divide by 100. DKK is the expected currency.
func formatAmount(unitAmount int64, currency stripepkg.Currency, interval stripepkg.PriceRecurringInterval) string {
	major := unitAmount / 100
	rem := unitAmount % 100
	ccy := string(currency)
	if ccy == "" {
		ccy = "DKK"
	}
	if rem == 0 {
		return fmt.Sprintf("%d %s / %s", major, stripeUpper(ccy), string(interval))
	}
	return fmt.Sprintf("%d.%02d %s / %s", major, rem, stripeUpper(ccy), string(interval))
}

// stripeUpper uppercases an ASCII currency code without depending on strings.ToUpper
// to avoid a tiny import for one call site. (Currency codes are always ASCII.)
func stripeUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

// renderSubscriptions renders the htmx subscriptions partial for a user.
func renderSubscriptions(u user.User, lang i18n.Lang) (bytes.Buffer, error) {
	log.Print("subs: lookup for customer ", u.CustomerID)

	view := subscriptionView{
		S:                i18n.For(lang),
		OpenHouseDate:    nextOpenHouseDate(lang),
		OpenHouseLinkURL: i18n.Path(lang, "/events/"),
	}

	// Customer (for "member since"). Soft-fail: a missing Stripe customer just
	// means we omit the date — the rest of the card still renders.
	if u.CustomerID != "" {
		if cus, err := customer.Get(u.CustomerID, nil); err == nil && cus != nil {
			view.MemberSince = time.Unix(cus.Created, 0).Format("January 2006")
		} else if err != nil {
			log.Print("subs: customer.Get: ", err)
		}
	}

	// Subscription (active, max 1). The current API exposes period boundaries
	// on the SubscriptionItem, not the Subscription itself.
	subs := listSubscriptions(u.CustomerID)
	if subs.Next() {
		s := subs.Subscription()
		view.Active = true
		view.SubID = s.ID
		view.Status = string(s.Status)
		view.CancelAtEnd = s.CancelAtPeriodEnd
		if s.StartDate != 0 {
			view.StartedOn = time.Unix(s.StartDate, 0).Format("January 2006")
		}
		if s.Items != nil && len(s.Items.Data) > 0 {
			item := s.Items.Data[0]
			if item.CurrentPeriodEnd != 0 {
				view.NextRenewal = time.Unix(item.CurrentPeriodEnd, 0).Format("2 January 2006")
			}
			if item.Price != nil && item.Price.Recurring != nil {
				view.PlanLabel = formatAmount(item.Price.UnitAmount, item.Price.Currency, item.Price.Recurring.Interval)
			}
		}
		if view.CancelAtEnd {
			view.CancelEndsMsg = fmt.Sprintf(view.S.CancelEndsOn, view.NextRenewal)
		}
		// Only meaningful when CancelAtEnd is true (the only state that shows
		// the reactivate button), but harmless to set unconditionally.
		view.CanReactivate = reactivationsCount(s) < maxReactivationsPerPeriod
		log.Printf("subs: id=%s status=%s", s.ID, s.Status)
	}

	tmpl, err := template.ParseFS(templatesFS, "templates/subscriptions.html")
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("subs: parse template: %w", err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, view); err != nil {
		log.Print("subs: execute template: ", err)
		return bytes.Buffer{}, err
	}
	return out, nil
}

// ServeSubscriptions returns the GET /subscriptions handler. Renders the htmx
// partial for the logged-in user. Unauthenticated requests get an HX-Redirect
// to /login.
func (s *Service) ServeSubscriptions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			lang := i18n.FromRequest(r)
			w.Header().Add("HX-Redirect", i18n.Path(lang, "/login"))
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		out, err := renderSubscriptions(user.FromSession(*sess), i18n.FromRequest(r))
		if err != nil && err != ErrNoSubs {
			log.Print("subs: ", err)
			fmt.Fprintln(w, "<p>Internal server error -", err, "</p>")
		}
		fmt.Fprint(w, out.String())
	}
}

// cancelModalView feeds templates/cancel-modal.html.
type cancelModalView struct {
	SubID       string
	EndsOn      string // "9 June 2026" — the user's current period end
	S           i18n.Strings
}

// DismissModal returns an empty body — used by Keep-membership to clear
// #modal-mount via htmx innerHTML swap. Auth-gated to keep behaviour uniform.
func (s *Service) DismissModal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		// Empty 200 — htmx swaps it into #modal-mount, removing the modal.
	}
}

// ServeCancelModal returns the GET /cancel-modal handler — htmx fragment with
// the cancellation confirmation modal. Replaces the prior browser-level
// hx-confirm with a designed dialog explaining what happens after cancel.
func (s *Service) ServeCancelModal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		subID := r.URL.Query().Get("sub_id")
		if subID == "" {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Verify ownership and read current period end so the modal can show the
		// concrete "you keep access until X" date.
		customerID, _ := sess.Values["customer_id"].(string)
		sub, err := subscription.Get(subID, nil)
		if err != nil || sub.Customer.ID != customerID {
			log.Printf("cancel-modal: ownership check failed for sub %s by customer %s", subID, customerID)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		view := cancelModalView{
			SubID: subID,
			S:     i18n.For(i18n.FromRequest(r)),
		}
		if sub.Items != nil && len(sub.Items.Data) > 0 {
			if end := sub.Items.Data[0].CurrentPeriodEnd; end != 0 {
				view.EndsOn = time.Unix(end, 0).Format("2 January 2006")
			}
		}

		tmpl, err := template.ParseFS(templatesFS, "templates/cancel-modal.html")
		if err != nil {
			log.Print("cancel-modal: parse template: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		var out bytes.Buffer
		if err := tmpl.Execute(&out, view); err != nil {
			log.Print("cancel-modal: render: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}

// ServeReactivateModal returns the GET /reactivate-modal handler — htmx
// fragment with the reactivation confirmation modal. Mirrors ServeCancelModal
// for friction symmetry; we want reactivating to be a deliberate two-step
// choice just like cancelling, even though it's non-destructive.
//
// Reuses cancelModalView since the shape (SubID, EndsOn, S) is identical.
func (s *Service) ServeReactivateModal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		subID := r.URL.Query().Get("sub_id")
		if subID == "" {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Verify ownership and read current period end so the modal can show
		// the concrete "we'll keep billing you on X" date.
		customerID, _ := sess.Values["customer_id"].(string)
		sub, err := subscription.Get(subID, nil)
		if err != nil || sub.Customer.ID != customerID {
			log.Printf("reactivate-modal: ownership check failed for sub %s by customer %s", subID, customerID)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		view := cancelModalView{
			SubID: subID,
			S:     i18n.For(i18n.FromRequest(r)),
		}
		if sub.Items != nil && len(sub.Items.Data) > 0 {
			if end := sub.Items.Data[0].CurrentPeriodEnd; end != 0 {
				view.EndsOn = time.Unix(end, 0).Format("2 January 2006")
			}
		}

		tmpl, err := template.ParseFS(templatesFS, "templates/reactivate-modal.html")
		if err != nil {
			log.Print("reactivate-modal: parse template: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var out bytes.Buffer
		if err := tmpl.Execute(&out, view); err != nil {
			log.Print("reactivate-modal: render: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}

// CancelSubscription returns the POST /cancel-subscription handler. Verifies
// the subscription belongs to the logged-in user before cancelling immediately.
//
// On success, emits an empty body that clears #modal-mount and an HX-Trigger
// for 'membership-changed' so the membership card reloads itself with the
// new state ('Cancel scheduled for X' or 'No active membership').
func (s *Service) CancelSubscription() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		subID := r.URL.Query().Get("sub_id")
		if subID == "" {
			log.Print("cancel: missing sub_id query: ", r.URL.Query())
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		customerID, _ := sess.Values["customer_id"].(string)
		sub, err := subscription.Get(subID, nil)
		if err != nil || sub.Customer.ID != customerID {
			log.Printf("cancel: ownership check failed for sub %s by customer %s", subID, customerID)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Cancel at period end rather than immediately — the user keeps access
		// until the date they've already paid for. This matches what the modal
		// promises ("you keep access until X").
		if _, err := subscription.Update(subID, &stripepkg.SubscriptionParams{
			CancelAtPeriodEnd: stripepkg.Bool(true),
		}); err != nil {
			log.Printf("cancel: stripe error: %v", err)
			http.Error(w, "Failed to cancel subscription", http.StatusInternalServerError)
			return
		}

		// Empty body clears #modal-mount; HX-Trigger reloads the membership card.
		w.Header().Set("HX-Trigger", "membership-changed")
	}
}

// ReactivateSubscription returns the POST /reactivate-subscription handler.
// Undoes a pending cancel-at-period-end by flipping the Stripe flag back to
// false. Non-destructive — no modal needed.
//
// On success, emits an HX-Trigger for 'membership-changed' so the membership
// card reloads itself and 204 No Content (the button's hx-target already
// triggers a refetch via the event, but the 204 keeps the immediate swap
// neutral).
func (s *Service) ReactivateSubscription() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		subID := r.URL.Query().Get("sub_id")
		if subID == "" {
			log.Print("reactivate: missing sub_id query: ", r.URL.Query())
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		customerID, _ := sess.Values["customer_id"].(string)
		sub, err := subscription.Get(subID, nil)
		if err != nil || sub.Customer.ID != customerID {
			log.Printf("reactivate: ownership check failed for sub %s by customer %s", subID, customerID)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Anti-gaming guard: cap reactivations per billing period. The counter
		// is reset by the customer.subscription.updated webhook when the cycle
		// rolls over (current_period_end advances).
		count := reactivationsCount(sub)
		if count >= maxReactivationsPerPeriod {
			log.Printf("reactivate: limit reached for sub %s (count=%d, cap=%d)", subID, count, maxReactivationsPerPeriod)
			http.Error(w, "Reactivation limit reached for this billing period", http.StatusConflict)
			return
		}

		if _, err := subscription.Update(subID, &stripepkg.SubscriptionParams{
			CancelAtPeriodEnd: stripepkg.Bool(false),
			Params: stripepkg.Params{
				Metadata: map[string]string{
					reactivationsMetadataKey: strconv.Itoa(count + 1),
				},
			},
		}); err != nil {
			log.Printf("reactivate: stripe error: %v", err)
			http.Error(w, "Failed to reactivate subscription", http.StatusInternalServerError)
			return
		}

		w.Header().Set("HX-Trigger", "membership-changed")
		w.WriteHeader(http.StatusNoContent)
	}
}
