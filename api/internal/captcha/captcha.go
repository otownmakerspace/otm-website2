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
// The Service holds the HMAC key (loaded from secrets/app/altcha_hmac_key)
// and an in-memory set of recently-redeemed challenge signatures for replay
// protection. A solution that verifies cryptographically can still be
// rejected if its signature was already used within the challenge TTL.
//
// When the HMAC key is empty, Verify is a no-op and ServeChallenge returns
// 503 — useful for local dev and tests that don't want to set up the secret.
package captcha

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
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
//
// The seen map records signatures of solutions that have already been
// verified once. Stateless verification of the same token would otherwise
// accept a replay during the challenge's TTL window. The map is bounded by
// challengeTTL — expired entries are dropped on every verify, so unbounded
// memory growth is impossible.
type Service struct {
	hmacKey string

	mu   sync.Mutex
	seen map[string]time.Time // signature → expiry
}

// NewService returns a Service. Pass an empty key to disable verification
// (Verify becomes a no-op, ServeChallenge returns 503). Use only in dev/test.
func NewService(hmacKey string) *Service {
	return &Service{
		hmacKey: hmacKey,
		seen:    make(map[string]time.Time),
	}
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
// the HMAC key, then checks the signature hasn't been previously redeemed.
//
// Returns nil if the captcha is correctly solved and not a replay, an error
// describing the failure otherwise. When the service is disabled (empty
// key), Verify is a no-op returning nil.
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

	// Solution math + HMAC are valid. Replay check: has this exact signature
	// been redeemed before within the TTL window?
	sig, err := signatureFromToken(token)
	if err != nil {
		return err
	}
	if !s.recordVerified(sig) {
		return errors.New("captcha: replay detected")
	}
	return nil
}

// HoneypotTriggered reports whether the hidden honeypot field has a value,
// indicating a likely bot. The caller should treat a triggered honeypot
// silently — fake-success the request rather than tell the bot it failed.
func HoneypotTriggered(r *http.Request) bool {
	return r.FormValue(HoneypotField) != ""
}

// signatureFromToken extracts the signature out of a base64-encoded Altcha
// payload, matching the encoding the library itself uses (StdEncoding base64
// of JSON, see altcha-lib-go/altcha.go:parsePayload).
func signatureFromToken(token string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	var p altcha.Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", err
	}
	if p.Signature == "" {
		return "", errors.New("captcha: token has no signature")
	}
	return p.Signature, nil
}

// recordVerified inserts a signature into the seen set with an expiry one
// challengeTTL in the future. Returns false if the signature was already
// present (i.e. this is a replay).
//
// Expired entries are dropped opportunistically on every call. Since the
// map is bounded by the number of unique signatures redeemed within the
// last challengeTTL minutes, growth is naturally limited.
func (s *Service) recordVerified(signature string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for sig, expiry := range s.seen {
		if now.After(expiry) {
			delete(s.seen, sig)
		}
	}
	if _, exists := s.seen[signature]; exists {
		return false
	}
	s.seen[signature] = now.Add(challengeTTL)
	return true
}
