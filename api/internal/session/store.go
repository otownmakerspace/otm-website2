// Package session owns auth cookies, login/logout, and password-reset flows.
//
// Sessions are HMAC-signed (no encryption) gorilla/sessions cookies. The
// cookie's name is configurable so the same binary can serve different
// domains without sessions colliding.
package session

import (
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/iustin94/makerspace/api/internal/i18n"
)

const (
	// MinPasswordLen is the lower bound enforced at signup and password-reset.
	MinPasswordLen = 8
	// MaxPasswordLen is bounded by bcrypt's 72-byte input cap, with safety margin
	// for multi-byte unicode characters.
	MaxPasswordLen = 31
)

// Store is the application's session store: cookie store + cookie name +
// max-age. Carries marketingBaseURL so Logout can redirect users back to the
// apex (marketing) host instead of leaving them stranded on members.*/.
type Store struct {
	cs               *sessions.CookieStore
	name             string
	marketingBaseURL string
}

// NewStore returns a Store that signs cookies with the provided private key
// and stores them under the given cookie name. MaxAge is 7 days; HttpOnly,
// SameSite=Lax.
//
// marketingBaseURL is the full URL of the apex / marketing site (e.g.
// "https://development.makerspace.olaru.dk"); Logout uses it to redirect the
// user back to the marketing site after sign-out. Empty falls back to a
// host-relative "/".
func NewStore(privateKey []byte, cookieName, marketingBaseURL string) *Store {
	cs := sessions.NewCookieStore(privateKey)
	cs.Options.MaxAge = 86400 * 7
	cs.Options.HttpOnly = true
	cs.Options.SameSite = http.SameSiteLaxMode
	return &Store{cs: cs, name: cookieName, marketingBaseURL: marketingBaseURL}
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
// The destination is language-prefixed via i18n.Path when the request came
// from a Danish page.
func (s *Store) RedirectIfLoggedIn(destination string, fallback http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.IsAuthenticated(r) {
			http.Redirect(w, r, i18n.Path(i18n.FromRequest(r), destination), http.StatusSeeOther)
			return
		}
		fallback.ServeHTTP(w, r)
	}
}

// Logout clears the authenticated flag and redirects to the marketing site
// (apex). The user came from the members subdomain; staying there post-logout
// would just dump them back at the login page. Sending them to the apex
// matches the mental model: "I signed out, take me back to where I came from".
func (s *Store) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.Get(r)
	sess.Values["authenticated"] = false
	sess.Save(r, w)
	target := i18n.Path(i18n.FromRequest(r), "/")
	if s.marketingBaseURL != "" {
		target = s.marketingBaseURL + target
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
