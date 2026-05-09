package session

import (
	"database/sql"
	"log"
	"net/http"
	"net/url"

	"golang.org/x/crypto/bcrypt"

	dbpkg "github.com/iustin94/makerspace/api/internal/db"
)

// Login returns the POST /login/ handler. Bad credentials redirect to
// /login/?reason=combo_fail without revealing whether the email exists.
func (s *Store) Login(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Print("Sign-in request")
		if err := r.ParseForm(); err != nil || !validateSignInInput(r.Form) {
			log.Print("login: malformed request")
			http.Redirect(w, r, "/login/?reason=misc", http.StatusSeeOther)
			return
		}

		email := r.Form.Get("email")
		passwordHash, customerID, err := dbpkg.LookupForLogin(database, email)
		if err == sql.ErrNoRows {
			log.Print("login: user not found")
			http.Redirect(w, r, "/login/?reason=combo_fail", http.StatusSeeOther)
			return
		} else if err != nil {
			log.Print("login: db error: ", err)
			http.Redirect(w, r, "/login/?reason=misc", http.StatusSeeOther)
			return
		}

		if err := bcrypt.CompareHashAndPassword(passwordHash, []byte(r.Form.Get("password"))); err != nil {
			if err == bcrypt.ErrMismatchedHashAndPassword {
				log.Print("login: password mismatch")
				http.Redirect(w, r, "/login/?reason=combo_fail", http.StatusFound)
				return
			}
			log.Print("login: bcrypt error: ", err)
			http.Redirect(w, r, "/login/?reason=failed_crypt", http.StatusFound)
			return
		}

		sess, _ := s.Get(r)
		sess.Values["authenticated"] = true
		sess.Values["customer_id"] = customerID
		sess.Values["email"] = email
		sess.Save(r, w)
		log.Print("login: ok")
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// validateSignInInput rejects forms missing the required fields.
func validateSignInInput(form url.Values) bool {
	for _, id := range []string{"email", "password"} {
		if !form.Has(id) {
			return false
		}
	}
	return true
}

// HashAndSalt returns a bcrypt hash of password using DefaultCost.
// Used at signup (before sending to Stripe metadata) and at password reset.
func HashAndSalt(password string) ([]byte, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Print("bcrypt: ", err)
		return nil, err
	}
	return hashed, nil
}
