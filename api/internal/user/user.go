// Package user defines the application's User domain type and helpers.
//
// User is a leaf type intentionally — other packages (db, stripe, session,
// email) import this; this package imports only the Stripe SDK to attach
// metadata onto checkout sessions.
package user

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/stripe/stripe-go/v82"
)

type User struct {
	Email         string
	Name          string
	Phone         string
	Password      []byte
	PlainPassword string
	Active        bool
	CustomerID    string
}

// FromSession reconstructs a User from a session's stored values.
// Only the fields persisted into the cookie are populated.
func FromSession(s sessions.Session) User {
	email, _ := s.Values["email"].(string)
	customerID, _ := s.Values["customer_id"].(string)
	return User{Email: email, CustomerID: customerID}
}

// FromForm reads a User from an HTTP form submission (signup).
// Caller is responsible for hashing PlainPassword before persisting.
func FromForm(r *http.Request) User {
	return User{
		Name:          r.Form.Get("name"),
		Email:         r.Form.Get("email"),
		PlainPassword: r.Form.Get("pass"),
		Phone:         r.Form.Get("phone"),
	}
}

// AddToStripeMetadata attaches user fields to a Stripe checkout session
// so the post-payment webhook can recover them and create the DB row.
// Password is the bcrypt hash (already salted) — never the plaintext.
func (u *User) AddToStripeMetadata(params *stripe.CheckoutSessionParams) {
	params.AddMetadata("email", u.Email)
	if u.Name != "" {
		params.AddMetadata("name", u.Name)
	}
	if u.Phone != "" {
		params.AddMetadata("phone", u.Phone)
	}
	if len(u.Password) > 0 {
		params.AddMetadata("pass", string(u.Password))
	}
}
