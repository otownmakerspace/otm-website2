package stripe

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	dbpkg "github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/i18n"
	"github.com/iustin94/makerspace/api/internal/session"
)

// passwordView is the data passed to templates/password.html.
type passwordView struct {
	Error     string
	SavedFlag bool
	MinLen    int
	MaxLen    int
	S         i18n.Strings
}

func renderPassword(view passwordView) (bytes.Buffer, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/password.html")
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("password: parse template: %w", err)
	}
	view.MinLen = session.MinPasswordLen
	view.MaxLen = session.MaxPasswordLen
	var out bytes.Buffer
	if err := tmpl.Execute(&out, view); err != nil {
		return bytes.Buffer{}, err
	}
	return out, nil
}

// ServePassword returns GET /profile/password — renders the empty form.
func (s *Service) ServePassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authedEmail(w, r) == "" {
			return
		}
		out, err := renderPassword(passwordView{S: i18n.For(i18n.FromRequest(r))})
		if err != nil {
			log.Print("password: render: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}

// UpdatePassword returns POST /profile/password.
// Requires the current password (defence in depth — even though the user is
// already authenticated, this prevents a stolen-cookie attacker from locking
// the real owner out by changing the password).
func (s *Service) UpdatePassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emailAddr := s.authedEmail(w, r)
		if emailAddr == "" {
			return
		}
		strs := i18n.For(i18n.FromRequest(r))
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		current := r.Form.Get("current")
		next := r.Form.Get("next")
		confirm := r.Form.Get("confirm")

		fail := func(msg string) {
			out, _ := renderPassword(passwordView{Error: msg, S: strs})
			fmt.Fprint(w, out.String())
		}

		if current == "" || next == "" || confirm == "" {
			fail(strs.ErrAllRequired)
			return
		}
		if next != confirm {
			fail(strs.ErrPwMismatch)
			return
		}
		if len(next) < session.MinPasswordLen || len(next) > session.MaxPasswordLen {
			fail(fmt.Sprintf(strs.ErrPwLength, session.MinPasswordLen, session.MaxPasswordLen))
			return
		}
		if next == current {
			fail(strs.ErrPwSame)
			return
		}

		u, err := dbpkg.GetByEmail(s.DB, emailAddr)
		if err != nil {
			log.Print("password: db lookup: ", err)
			fail(strs.ErrVerifyFailed)
			return
		}
		if err := bcrypt.CompareHashAndPassword(u.Password, []byte(current)); err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				fail(strs.ErrCurrentWrong)
				return
			}
			log.Print("password: bcrypt compare: ", err)
			fail(strs.ErrVerifyFailed)
			return
		}

		newHash, err := session.HashAndSalt(next)
		if err != nil {
			fail(strs.ErrPwSaveFailed)
			return
		}
		if err := dbpkg.UpdatePassword(s.DB, emailAddr, newHash); err != nil {
			log.Print("password: db update: ", err)
			fail(strs.ErrPwSaveFailed)
			return
		}

		out, err := renderPassword(passwordView{SavedFlag: true, S: strs})
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}
