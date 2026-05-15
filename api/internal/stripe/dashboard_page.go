package stripe

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	dbpkg "github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/errpage"
	"github.com/iustin94/makerspace/api/internal/i18n"
)

// dashboardPageView feeds templates/dashboard-page.html — a complete HTML
// document with its own auth-aware chrome.
type dashboardPageView struct {
	Lang      i18n.Lang
	Head      template.HTML // extracted from docs/index.html — Tailwind, fonts, theme tokens
	LogoLight template.HTML // inline SVG of the marketing-site light-mode logo
	LogoDark  template.HTML // inline SVG of the marketing-site dark-mode logo
	UserEmail string
	HomeURL   string // /  or  /da/
	LogoutURL string // /logout
	S         i18n.Strings
}

// loadHugoHead reads <head>...</head> out of docs/index.html so the dashboard
// page gets the same Tailwind, fonts, and theme-token setup as the rest of
// the site without duplicating the boilerplate.
//
// Returns empty HTML if the file is missing — the dashboard then renders with
// a minimal Tailwind CDN fallback (defined in dashboard-page.html). That's
// good enough to debug without the static site present.
func loadHugoHead(staticDir string) template.HTML {
	path := filepath.Join(staticDir, "index.html")
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("dashboard: couldn't read %s for head extraction: %v", path, err)
		return ""
	}
	s := string(data)
	start := strings.Index(s, "<head>")
	end := strings.Index(s, "</head>")
	if start == -1 || end == -1 {
		log.Print("dashboard: no <head>...</head> in static index — using fallback")
		return ""
	}
	// Skip past "<head>" itself; the template emits the open tag explicitly so
	// we can prepend a per-page <title> before this content.
	inner := s[start+len("<head>") : end]
	return template.HTML(inner)
}

// loadInlineSVG reads an SVG file from the static dir so it can be emitted
// inline in the rendered HTML. Inlining is required for the horizontal-lockup
// because it references gear + wordmark SVGs via <image href="...">; browsers
// sandbox SVGs loaded through <img> and refuse to fetch those inner resources
// (Opaque Response Blocking). Inline SVG bypasses the sandbox.
//
// Returns empty HTML if the file is missing — callers should render gracefully
// without a logo rather than 500'ing the dashboard.
func loadInlineSVG(staticDir, relPath string) template.HTML {
	path := filepath.Join(staticDir, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("dashboard: couldn't read %s for inline SVG: %v", path, err)
		return ""
	}
	return template.HTML(data)
}

// ServeDashboardPage returns the GET /dashboard handler. Auth-gated. Renders a
// complete HTML page using a Go template — independent of Hugo's static output
// for the dashboard URL itself, so the chrome (header, profile menu) can read
// the request's session and render auth-aware content directly.
func (s *Service) ServeDashboardPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		lang := i18n.FromRequest(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			http.Redirect(w, r, i18n.Path(lang, "/login/"), http.StatusSeeOther)
			return
		}
		emailAddr, _ := sess.Values["email"].(string)

		// Lookup the user for first-name display in the menu.
		var displayName string
		if emailAddr != "" {
			if u, err := dbpkg.GetByEmail(s.DB, emailAddr); err == nil {
				displayName = u.Name
				if displayName == "" {
					displayName = u.Email
				}
			}
		}
		if displayName == "" {
			displayName = emailAddr
		}

		// "Back to site" goes to the marketing host (apex). If MarketingBaseURL
		// isn't set (tests / local), fall back to a relative path so the link
		// still works on the same host.
		homeURL := i18n.Path(lang, "/")
		if s.Cfg.Backend.MarketingBaseURL != "" {
			homeURL = s.Cfg.Backend.MarketingBaseURL + homeURL
		}
		view := dashboardPageView{
			Lang:      lang,
			Head:      s.HugoHead,
			LogoLight: s.LogoLight,
			LogoDark:  s.LogoDark,
			UserEmail: displayName,
			HomeURL:   homeURL,
			LogoutURL: "/logout",
			S:         i18n.For(lang),
		}

		tmpl, err := template.ParseFS(templatesFS, "templates/dashboard-page.html")
		if err != nil {
			log.Print("dashboard: parse template: ", err)
			errpage.Render(w, r, s.Cfg.Backend.StaticDir, "500", http.StatusInternalServerError)
			return
		}
		var out bytes.Buffer
		if err := tmpl.Execute(&out, view); err != nil {
			log.Print("dashboard: render: ", err)
			errpage.Render(w, r, s.Cfg.Backend.StaticDir, "500", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store") // user-specific, never cache
		fmt.Fprint(w, out.String())
	}
}
