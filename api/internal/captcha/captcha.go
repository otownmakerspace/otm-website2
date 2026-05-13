// Package captcha wraps altcha-lib-go for the signup + password-reset forms
// and provides a simple honeypot helper.
//
// Two layers, ordered cheap → expensive:
//
//  1. Honeypot: a hidden form field that humans don't see but naive bots
//     auto-fill. Tripped honeypot → silent drop, no further work.
//  2. Altcha: client-side proof-of-work, HMAC-verified server-side. Self-
//     hosted, no external service, no tracking, no Klaro consent entry
//     needed.
//
// The Service holds the HMAC key (loaded from secrets/app/altcha_hmac_key).
// When the key is empty, Verify is a no-op and ServeChallenge returns 503 —
// useful for local dev and tests that don't want to set up the secret.
package captcha

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/altcha-org/altcha-lib-go"
)

// HoneypotField is the input name humans never see and bots tend to autofill.
// "website" is a common autofill target. The form template renders it
// off-screen with aria-hidden and tabindex=-1.
const HoneypotField = "website"

// AltchaField is the form value the Altcha widget submits after solving.
const AltchaField = "altcha"

// challengeTTL bounds how long a fresh challenge stays valid. Long enough
// for a slow user to finish typing the form; short enough to limit replay.
const challengeTTL = 10 * time.Minute

// Service holds dependencies for captcha operations.
type Service struct {
	hmacKey string
}

// NewService returns a Service. Pass an empty key to disable verification
// (Verify becomes a no-op, ServeChallenge returns 503). Use only in dev/test.
func NewService(hmacKey string) *Service {
	return &Service{hmacKey: hmacKey}
}

// Enabled reports whether captcha verification is configured.
func (s *Service) Enabled() bool {
	return s.hmacKey != ""
}

// ServeChallenge is the GET /captcha/challenge handler. Returns a fresh
// Altcha challenge as JSON for the client widget to solve.
func (s *Service) ServeChallenge() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.Enabled() {
			http.Error(w, "captcha disabled", http.StatusServiceUnavailable)
			return
		}
		expires := time.Now().Add(challengeTTL)
		challenge, err := altcha.CreateChallenge(altcha.ChallengeOptions{
			HMACKey: s.hmacKey,
			Expires: &expires,
		})
		if err != nil {
			log.Printf("captcha: CreateChallenge: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(challenge); err != nil {
			log.Printf("captcha: encode: %v", err)
		}
	}
}

// Verify reads the altcha payload from form values and validates it against
// the HMAC key. Returns nil if the captcha is correctly solved, an error
// describing the failure otherwise.
//
// When the service is disabled (empty key), Verify is a no-op returning nil.
func (s *Service) Verify(r *http.Request) error {
	if !s.Enabled() {
		return nil
	}
	token := r.FormValue(AltchaField)
	if token == "" {
		return errors.New("captcha: missing solution")
	}
	ok, err := altcha.VerifySolution(token, s.hmacKey, true)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("captcha: invalid solution")
	}
	return nil
}

// HoneypotTriggered reports whether the hidden honeypot field has a value,
// indicating a likely bot. The caller should treat a triggered honeypot
// silently — fake-success the request rather than tell the bot it failed.
func HoneypotTriggered(r *http.Request) bool {
	return r.FormValue(HoneypotField) != ""
}
