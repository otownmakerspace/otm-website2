package stripe

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	stripepkg "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/invoice"

	"github.com/iustin94/makerspace/api/internal/i18n"
)

// invoiceRow is one row in the recent-invoices list.
type invoiceRow struct {
	Title     string // human-friendly: "May 2026 membership"
	Number    string // technical Stripe number: "61F2990E-0107"  — shown as tooltip
	Date      string
	Amount    string // pre-formatted "200 DKK"
	Status    string // "paid", "open", etc.
	HostedURL string
	PDFURL    string
}

type invoicesView struct {
	Rows  []invoiceRow
	Empty bool
	S     i18n.Strings
}

// ServeInvoices returns the GET /invoices handler — htmx fragment with the
// last 5 invoices for the logged-in customer.
func (s *Service) ServeInvoices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r)
		if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
			lang := i18n.FromRequest(r)
			w.Header().Add("HX-Redirect", i18n.Path(lang, "/login"))
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		customerID, _ := sess.Values["customer_id"].(string)

		strs := i18n.For(i18n.FromRequest(r))
		view := invoicesView{S: strs}
		if customerID != "" {
			params := &stripepkg.InvoiceListParams{}
			params.Customer = stripepkg.String(customerID)
			params.Limit = stripepkg.Int64(5)

			iter := invoice.List(params)
			for iter.Next() {
				inv := iter.Invoice()
				created := time.Unix(inv.Created, 0)
				view.Rows = append(view.Rows, invoiceRow{
					Title:     fmt.Sprintf(strs.InvoiceTitleMonth, created.Format("January 2006")),
					Number:    inv.Number,
					Date:      created.Format("2 Jan 2006"),
					Amount:    formatMinorAmount(inv.AmountPaid, inv.Currency),
					Status:    string(inv.Status),
					HostedURL: inv.HostedInvoiceURL,
					PDFURL:    inv.InvoicePDF,
				})
			}
			if err := iter.Err(); err != nil {
				log.Print("invoices: stripe list: ", err)
			}
		}
		view.Empty = len(view.Rows) == 0

		tmpl, err := template.ParseFS(templatesFS, "templates/invoices.html")
		if err != nil {
			log.Print("invoices: parse template: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		var out bytes.Buffer
		if err := tmpl.Execute(&out, view); err != nil {
			log.Print("invoices: render: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}

// formatMinorAmount renders a Stripe minor-unit amount as "<amount> <CCY>".
func formatMinorAmount(minor int64, currency stripepkg.Currency) string {
	major := minor / 100
	rem := minor % 100
	ccy := stripeUpper(string(currency))
	if ccy == "" {
		ccy = "DKK"
	}
	if rem == 0 {
		return fmt.Sprintf("%d %s", major, ccy)
	}
	return fmt.Sprintf("%d.%02d %s", major, rem, ccy)
}
