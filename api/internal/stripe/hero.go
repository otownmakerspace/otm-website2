package stripe

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	dbpkg "github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/i18n"
)

// heroView feeds templates/hero.html — the personalized greeting at the top of
// the dashboard. All fields are pre-formatted strings.
type heroView struct {
	FirstName      string
	StatusHeadline string // "Membership renews 9 June 2026" / "No active membership"
	StatusDetail   string // optional secondary line, e.g. "Cancel scheduled for X"
	Active         bool
	CancelAtEnd    bool
	S              i18n.Strings
}

// firstNameOf returns the first whitespace-separated token from full.
// Falls back to the full string if there's no space, or "" if empty.
func firstNameOf(full string) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return ""
	}
	if i := strings.Index(full, " "); i > 0 {
		return full[:i]
	}
	return full
}

// nextOpenHouseDate returns the next Thursday at 17:00 from now, formatted
// per language. Locale-friendly date format ("2 January 2006" English style;
// "2. januar 2006" Danish style).
func nextOpenHouseDate(lang i18n.Lang) string {
	now := time.Now()
	// time.Thursday == 4. Days until next Thursday (today counts if it's Thu and before 17:00).
	daysAhead := (int(time.Thursday) - int(now.Weekday()) + 7) % 7
	if daysAhead == 0 && now.Hour() >= 17 {
		daysAhead = 7
	}
	d := now.AddDate(0, 0, daysAhead)
	if lang == i18n.DA {
		return formatDanishDate(d)
	}
	return d.Format("Monday 2 January")
}

// formatDanishDate renders a date as "torsdag 14. maj" — Danish weekday +
// day + month (no year, since Open House is recurring).
func formatDanishDate(d time.Time) string {
	weekdays := map[time.Weekday]string{
		time.Sunday:    "søndag",
		time.Monday:    "mandag",
		time.Tuesday:   "tirsdag",
		time.Wednesday: "onsdag",
		time.Thursday:  "torsdag",
		time.Friday:    "fredag",
		time.Saturday:  "lørdag",
	}
	months := []string{
		"januar", "februar", "marts", "april", "maj", "juni",
		"juli", "august", "september", "oktober", "november", "december",
	}
	return fmt.Sprintf("%s %d. %s", weekdays[d.Weekday()], d.Day(), months[d.Month()-1])
}

// ServeHero returns the GET /dashboard/hero handler — htmx fragment with the
// greeting + headline status that anchors the dashboard.
func (s *Service) ServeHero() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emailAddr := s.authedEmail(w, r)
		if emailAddr == "" {
			return
		}
		strs := i18n.For(i18n.FromRequest(r))
		view := heroView{S: strs}

		u, err := dbpkg.GetByEmail(s.DB, emailAddr)
		if err == nil {
			view.FirstName = firstNameOf(u.Name)
		}

		// Headline status comes from the active subscription (if any).
		// Reusing listSubscriptions keeps Stripe-API plumbing in one place.
		statusUnknown := false
		if u.CustomerID != "" {
			subs := listSubscriptions(u.CustomerID)
			if subs.Next() {
				sub := subs.Subscription()
				view.Active = true
				view.CancelAtEnd = sub.CancelAtPeriodEnd
				if sub.Items != nil && len(sub.Items.Data) > 0 {
					end := sub.Items.Data[0].CurrentPeriodEnd
					if end != 0 {
						formatted := time.Unix(end, 0).Format("2 January 2006")
						if view.CancelAtEnd {
							view.StatusHeadline = fmt.Sprintf(strs.HeroEnding, formatted)
						} else {
							view.StatusHeadline = fmt.Sprintf(strs.HeroRenewsOn, formatted)
						}
					}
				}
			}
			// See renderSubscriptions: an errored list is not an empty one, and a
			// missing customer is not a transient failure. Only the latter should
			// read as "status unknown"; a gone customer just means "not currently a
			// member", which HeroInactive already covers.
			if err := subs.Err(); err != nil && !isMissingResource(err) {
				log.Print("hero: could not determine membership status: ", err)
				statusUnknown = true
			}
		}
		if view.StatusHeadline == "" {
			if statusUnknown {
				view.StatusHeadline = strs.StatusUnavailable
			} else {
				view.StatusHeadline = strs.HeroInactive
			}
		}

		// Mention next Open House when the user is active and not winding down,
		// linking to /events/ if they want details.
		if view.Active && !view.CancelAtEnd {
			lang := i18n.FromRequest(r)
			view.StatusDetail = fmt.Sprintf(strs.HeroOpenHouse, nextOpenHouseDate(lang))
		}

		tmpl, err := template.ParseFS(templatesFS, "templates/hero.html")
		if err != nil {
			log.Print("hero: parse template: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		var out bytes.Buffer
		if err := tmpl.Execute(&out, view); err != nil {
			log.Print("hero: render: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}
