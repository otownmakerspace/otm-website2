package stripe

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	stripepkg "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"

	dbpkg "github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/i18n"
)

// profileView is the data passed to templates/profile.html.
//
// Field names that overlap with Strings field names are suffixed with "Val"
// to keep template lookups unambiguous (e.g. .EmailVal vs .S.Email).
type profileView struct {
	Name      string
	EmailVal  string
	Phone     string
	Edit      bool
	Error     string
	SavedFlag bool
	NameMax   int
	PhoneMax  int
	S         i18n.Strings
}

// renderProfile renders the profile fragment in either view or edit mode.
func renderProfile(view profileView) (bytes.Buffer, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/profile.html")
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("profile: parse template: %w", err)
	}
	view.NameMax = 100
	view.PhoneMax = 30
	var out bytes.Buffer
	if err := tmpl.Execute(&out, view); err != nil {
		return bytes.Buffer{}, err
	}
	return out, nil
}

// authedEmail returns the logged-in user's email or empty string if not auth'd.
// Caller writes the 403 + HX-Redirect when empty.
func (s *Service) authedEmail(w http.ResponseWriter, r *http.Request) string {
	sess, _ := s.Sessions.Get(r)
	if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
		lang := i18n.FromRequest(r)
		w.Header().Add("HX-Redirect", i18n.Path(lang, "/login"))
		http.Error(w, "Forbidden", http.StatusForbidden)
		return ""
	}
	emailAddr, _ := sess.Values["email"].(string)
	return emailAddr
}

// ServeProfile returns the GET /profile handler — htmx fragment in view mode.
func (s *Service) ServeProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emailAddr := s.authedEmail(w, r)
		if emailAddr == "" {
			return
		}
		u, err := dbpkg.GetByEmail(s.DB, emailAddr)
		if err != nil {
			log.Print("profile: db lookup: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		out, err := renderProfile(profileView{
			Name: u.Name, EmailVal: u.Email, Phone: u.Phone, Edit: false,
			S: i18n.For(i18n.FromRequest(r)),
		})
		if err != nil {
			log.Print("profile: render: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}

// ServeProfileEdit returns the GET /profile/edit handler — same fragment in edit mode.
func (s *Service) ServeProfileEdit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emailAddr := s.authedEmail(w, r)
		if emailAddr == "" {
			return
		}
		u, err := dbpkg.GetByEmail(s.DB, emailAddr)
		if err != nil {
			log.Print("profile/edit: db lookup: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		out, err := renderProfile(profileView{
			Name: u.Name, EmailVal: u.Email, Phone: u.Phone, Edit: true,
			S: i18n.For(i18n.FromRequest(r)),
		})
		if err != nil {
			log.Print("profile/edit: render: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}

// UpdateProfile returns the POST /profile handler.
// Updates name + phone in our DB and (best-effort) on the Stripe customer so
// they stay in sync. Email is intentionally NOT editable here: it's the login
// identifier and would require coordinated DB + Stripe + session cookie updates.
func (s *Service) UpdateProfile() http.HandlerFunc {
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
		name := strings.TrimSpace(r.Form.Get("name"))
		phone := strings.TrimSpace(r.Form.Get("phone"))

		fail := func(msg string) {
			out, _ := renderProfile(profileView{
				Name: name, EmailVal: emailAddr, Phone: phone, Edit: true,
				Error: msg, S: strs,
			})
			fmt.Fprint(w, out.String())
		}

		if name == "" {
			fail(strs.ErrNameEmpty)
			return
		}
		if len(name) > 100 || len(phone) > 30 {
			fail(strs.ErrTooLong)
			return
		}

		if err := dbpkg.UpdateProfile(s.DB, emailAddr, name, phone); err != nil {
			log.Print("profile: db update: ", err)
			fail(strs.ErrSaveFailed)
			return
		}

		// Sync to Stripe so customer-portal / receipts show the new name/phone.
		// Failure here is non-fatal — local DB is the source of truth.
		u, err := dbpkg.GetByEmail(s.DB, emailAddr)
		if err == nil && u.CustomerID != "" {
			_, sErr := customer.Update(u.CustomerID, &stripepkg.CustomerParams{
				Name:  stripepkg.String(name),
				Phone: stripepkg.String(phone),
			})
			if sErr != nil {
				log.Print("profile: stripe customer.Update: ", sErr)
			}
		}

		out, err := renderProfile(profileView{
			Name: name, EmailVal: emailAddr, Phone: phone, Edit: false, SavedFlag: true,
			S: strs,
		})
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}
