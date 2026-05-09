package stripe

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"

	stripepkg "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/subscription"

	"github.com/iustin94/makerspace/api/internal/user"
)

//go:embed templates/*.html
var templatesFS embed.FS

// ErrNoSubs is returned when a customer has no active subscriptions.
var ErrNoSubs = errors.New("no subscriptions available")

// listSubscriptions returns the iterator of active subs for a customer (max 1).
func listSubscriptions(customerID string) *subscription.Iter {
	params := &stripepkg.SubscriptionListParams{}
	params.Customer = stripepkg.String(customerID)
	params.Status = stripepkg.String("active")
	params.Limit = stripepkg.Int64(1)
	return subscription.List(params)
}

// renderSubscriptions renders the htmx subscriptions partial for a user.
func renderSubscriptions(u user.User) (bytes.Buffer, error) {
	log.Print("subs: lookup for customer ", u.CustomerID)
	subs := listSubscriptions(u.CustomerID)
	u.Active = subs.Next()
	subID := ""
	if u.Active {
		s := subs.Subscription()
		subID = s.ID
		log.Printf("subs: id=%s status=%s", s.ID, s.Status)
	}

	tmpl, err := template.ParseFS(templatesFS, "templates/subscriptions.html")
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("subs: parse template: %w", err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, struct {
		User  user.User
		SubID string
	}{User: u, SubID: subID}); err != nil {
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
			w.Header().Add("HX-Redirect", "/login")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		out, err := renderSubscriptions(user.FromSession(*sess))
		if err != nil && err != ErrNoSubs {
			log.Print("subs: ", err)
			fmt.Fprintln(w, "<p>Internal server error -", err, "</p>")
		}
		fmt.Fprint(w, out.String())
	}
}

// CancelSubscription returns the POST /cancel-subscription handler. Verifies
// the subscription belongs to the logged-in user before cancelling immediately.
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

		if _, err := subscription.Cancel(subID, &stripepkg.SubscriptionCancelParams{}); err != nil {
			log.Printf("cancel: stripe error: %v", err)
			http.Error(w, "Failed to cancel subscription", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "Registered event: Subscription canceled")
	}
}
