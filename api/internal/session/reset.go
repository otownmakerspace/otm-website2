package session

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/iustin94/makerspace/api/internal/email"
	"github.com/iustin94/makerspace/api/internal/i18n"
	"github.com/iustin94/makerspace/api/internal/user"
)

// ResetMailer is the subset of *email.Mailer used by the password-reset flow.
// Defined as an interface here so handlers don't take a concrete *email.Mailer
// dependency in their signatures (and to make tests easier).
type ResetMailer interface {
	Send(to string, u user.User, t email.Template, link string, data any) error
}

// generateToken returns 32 bytes of entropy hex-encoded (64 chars).
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken returns the SHA-256 hex digest of token. The DB stores only the
// hash; the raw token travels through the email link only.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// RequestReset returns the POST /request-reset handler. To prevent email
// enumeration, the user is always redirected to /reset-sent regardless of
// whether the email exists.
//
// publicURL is the absolute base for the email's reset link; lang-aware paths
// only apply to the redirects shown in the user's browser, not the link
// embedded in the email (which the user clicks from their inbox and may have
// opened in a different language session).
func RequestReset(database *sql.DB, mailer ResetMailer, publicURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Print("password reset requested")
		lang := i18n.FromRequest(r)
		emailAddr := r.FormValue("email")
		if emailAddr == "" {
			log.Print("reset: empty email")
			http.Redirect(w, r, i18n.Path(lang, "/login/?reason=misc"), http.StatusSeeOther)
			return
		}

		// Anti-enumeration: always redirect to /reset-sent.
		var n int
		err := database.QueryRow("SELECT 1 FROM user WHERE email = ?", emailAddr).Scan(&n)
		if err == sql.ErrNoRows {
			log.Print("reset: requested for non-existent email")
			http.Redirect(w, r, i18n.Path(lang, "/reset-sent"), http.StatusSeeOther)
			return
		} else if err != nil {
			log.Print("reset: db error: ", err)
			http.Redirect(w, r, i18n.Path(lang, "/reset-sent"), http.StatusSeeOther)
			return
		}

		token, err := generateToken()
		if err != nil {
			log.Print("reset: token generation failed: ", err)
			http.Redirect(w, r, i18n.Path(lang, "/reset-sent"), http.StatusSeeOther)
			return
		}

		expiry := time.Now().Add(1 * time.Hour)
		if _, err := database.Exec(
			"INSERT INTO password_reset_tokens (email, token, expires_at) VALUES (?, ?, ?)",
			emailAddr, hashToken(token), expiry,
		); err != nil {
			log.Print("reset: db insert: ", err)
			http.Redirect(w, r, i18n.Path(lang, "/reset-sent"), http.StatusSeeOther)
			return
		}

		// The reset link drops the user on /reset-password (or /da/reset-password)
		// based on the language they were on when they requested the reset.
		resetLink := fmt.Sprintf("%s%s?token=%s", publicURL, i18n.Path(lang, "/reset-password"), token)
		if err := mailer.Send(emailAddr, user.User{}, email.Reset, resetLink, struct{}{}); err != nil {
			log.Print("reset: email send: ", err)
		}

		http.Redirect(w, r, i18n.Path(lang, "/reset-sent"), http.StatusSeeOther)
	}
}

// ResetPassword returns the POST /reset-password/ handler. Validates the
// token + expiry, hashes the new password, deletes the token row.
func ResetPassword(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.FromRequest(r)
		token := r.URL.Query().Get("token")
		if token == "" {
			log.Print("reset: token missing in query: ", r.URL.Query())
			http.Redirect(w, r, i18n.Path(lang, "/login/?reason=token_invalid"), http.StatusSeeOther)
			return
		}
		log.Print("reset: applying new password")

		newPassword := r.FormValue("password")
		if len(newPassword) < MinPasswordLen || len(newPassword) > MaxPasswordLen {
			http.Redirect(w, r, i18n.Path(lang, "/login/?reason=misc"), http.StatusSeeOther)
			return
		}

		tokenHash := hashToken(token)
		var emailAddr string
		var expiresAt time.Time
		err := database.QueryRow(
			"SELECT email, expires_at FROM password_reset_tokens WHERE token = ?",
			tokenHash,
		).Scan(&emailAddr, &expiresAt)
		if err == sql.ErrNoRows {
			log.Print("reset: unknown token")
			http.Redirect(w, r, i18n.Path(lang, "/login/?reason=token_invalid"), http.StatusSeeOther)
			return
		}
		if err != nil {
			log.Print("reset: db error: ", err)
			http.Redirect(w, r, i18n.Path(lang, "/login/?reason=db_query"), http.StatusSeeOther)
			return
		}

		if time.Now().After(expiresAt) {
			http.Error(w, "Token expired", http.StatusBadRequest)
			return
		}

		hashed, err := HashAndSalt(newPassword)
		if err != nil {
			http.Redirect(w, r, i18n.Path(lang, "/login/?reason=failed_crypt"), http.StatusSeeOther)
			return
		}
		if _, err := database.Exec("UPDATE user SET password = ? WHERE email = ?", hashed, emailAddr); err != nil {
			http.Redirect(w, r, i18n.Path(lang, "/login/?reason=db_update"), http.StatusSeeOther)
			return
		}

		database.Exec("DELETE FROM password_reset_tokens WHERE token = ?", tokenHash)
		log.Print("reset: success")
		http.Redirect(w, r, i18n.Path(lang, "/reset-success"), http.StatusSeeOther)
	}
}
