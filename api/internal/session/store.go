// Package session owns auth cookies, login/logout, and password-reset flows.
//
// Sessions are HMAC-signed (no encryption) gorilla/sessions cookies. The
// cookie's name is configurable so the same binary can serve different
// domains without sessions colliding.
package session

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	// MinPasswordLen is the lower bound enforced at signup and password-reset.
	MinPasswordLen = 8
	// MaxPasswordLen is bounded by bcrypt's 72-byte input cap, with safety margin
	// for multi-byte unicode characters.
	MaxPasswordLen = 31
)

// Store is the application's session store: cookie store + cookie name + max-age.
type Store struct {
	cs   *sessions.CookieStore
	name string
}

// NewStore returns a Store that signs cookies with the provided private key
// and stores them under the given cookie name. MaxAge is 7 days; HttpOnly,
// SameSite=Lax.
func NewStore(privateKey []byte, cookieName string) *Store {
	cs := sessions.NewCookieStore(privateKey)
	cs.Options.MaxAge = 86400 * 7
	cs.Options.HttpOnly = true
	cs.Options.SameSite = http.SameSiteLaxMode
	return &Store{cs: cs, name: cookieName}
}

// Get returns the session for the request, creating it if missing.
func (s *Store) Get(r *http.Request) (*sessions.Session, error) {
	return s.cs.Get(r, s.name)
}

// IsAuthenticated returns true if the request carries a valid logged-in cookie.
func (s *Store) IsAuthenticated(r *http.Request) bool {
	sess, _ := s.Get(r)
	if sess == nil {
		return false
	}
	auth, ok := sess.Values["authenticated"].(bool)
	return ok && auth
}

// RedirectIfLoggedIn returns a handler that redirects authenticated requests
// to destination, or otherwise serves the static fallback (typically the Hugo
// page for /login or /checkout).
func (s *Store) RedirectIfLoggedIn(destination string, fallback http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.IsAuthenticated(r) {
			http.Redirect(w, r, destination, http.StatusSeeOther)
			return
		}
		fallback.ServeHTTP(w, r)
	}
}

// Logout clears the authenticated flag and redirects to /.
func (s *Store) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.Get(r)
	sess.Values["authenticated"] = false
	sess.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
