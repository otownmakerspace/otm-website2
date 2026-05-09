package stripe

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"

	stripepkg "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"

	"github.com/iustin94/makerspace/api/internal/i18n"
)

// priceOption is one row in the price selector — pre-formatted for the template.
type priceOption struct {
	ID       string // price_xxx
	Name     string // product name (with nickname suffix when distinct)
	Amount   string // pre-formatted "200 DKK / month"
	Selected bool   // first active recurring price gets selected by default
}

type pricesView struct {
	Options        []priceOption
	PublishableKey string
	Empty          bool
	S              i18n.Strings
}

// ServePrices returns the GET /checkout/prices handler — htmx fragment listing
// active recurring Stripe prices as radio buttons. Public (no auth) since it's
// part of the signup flow.
//
// Stripe's Price.Product is an ID by default; we expand it so the template can
// show the product name.
func (s *Service) ServePrices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := &stripepkg.PriceListParams{
			Active: stripepkg.Bool(true),
			Type:   stripepkg.String("recurring"),
		}
		params.AddExpand("data.product")
		params.Limit = stripepkg.Int64(20)

		view := pricesView{
			PublishableKey: s.Cfg.Stripe.PublishableKey,
			S:              i18n.For(i18n.FromRequest(r)),
		}

		iter := price.List(params)
		first := true
		for iter.Next() {
			p := iter.Price()
			// Only show prices whose parent product is active.
			if p.Product == nil || !p.Product.Active {
				continue
			}
			if p.Recurring == nil {
				continue
			}
			name := p.Product.Name
			if p.Nickname != "" && p.Nickname != name {
				name = name + " — " + p.Nickname
			}
			view.Options = append(view.Options, priceOption{
				ID:       p.ID,
				Name:     name,
				Amount:   formatAmount(p.UnitAmount, p.Currency, p.Recurring.Interval),
				Selected: first,
			})
			first = false
		}
		if err := iter.Err(); err != nil {
			log.Print("prices: stripe list: ", err)
		}
		view.Empty = len(view.Options) == 0

		tmpl, err := template.ParseFS(templatesFS, "templates/prices.html")
		if err != nil {
			log.Print("prices: parse template: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		var out bytes.Buffer
		if err := tmpl.Execute(&out, view); err != nil {
			log.Print("prices: render: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, out.String())
	}
}
