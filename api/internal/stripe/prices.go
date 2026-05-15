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

// priceOption is the single membership tier rendered into the signup form.
// Pre-formatted for the template. All fields originate from Stripe so the
// card is data-driven — to edit the description or the "what's included"
// list, change the Product in the Stripe dashboard, not this code.
//
// Amount, Currency, and Interval are kept as separate fields so the
// template can give each its own typographic weight (large amount, smaller
// currency code, smaller "per month") — matching the homepage pricing
// card design from layouts/_partials/memberships.html.
type priceOption struct {
	ID          string   // price_xxx — kept for client-side Stripe.js use if needed
	Name        string   // product name (with nickname suffix when distinct)
	Description string   // product description (Stripe Product.description)
	Features    []string // marketing features (Stripe Product.marketing_features[].name)
	Amount      string   // major currency unit, e.g. "200" or "199.50"
	Currency    string   // uppercase ISO code, e.g. "DKK"
	Interval    string   // "per month", "per year", etc.
}

type pricesView struct {
	Option         priceOption
	PublishableKey string
	Empty          bool // true when STRIPE_PRICE_ID is unset/invalid/archived
	S              i18n.Strings
}

// ServePrices returns the GET /checkout/prices handler — htmx fragment that
// displays the configured membership tier as a read-only card. Public (no
// auth) since it's part of the signup flow.
//
// The displayed tier is whatever STRIPE_PRICE_ID points to. There is no
// user-side selection: the canonical price is set by deployment config and
// rendered for confirmation only. This avoids the tampering surface of
// accepting a price_id from the form and matches the single-tier business
// model the site actually operates.
//
// Stripe's Price.Product is an ID by default; we expand it so the template
// can show the product name.
func (s *Service) ServePrices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		view := pricesView{
			PublishableKey: s.Cfg.Stripe.PublishableKey,
			S:              i18n.For(i18n.FromRequest(r)),
		}

		if s.Cfg.Stripe.PriceID == "" {
			log.Print("prices: STRIPE_PRICE_ID is not configured")
		} else {
			params := &stripepkg.PriceParams{}
			params.AddExpand("product")
			p, err := price.Get(s.Cfg.Stripe.PriceID, params)
			switch {
			case err != nil:
				log.Printf("prices: price.Get(%s): %v", s.Cfg.Stripe.PriceID, err)
			case p == nil, !p.Active, p.Recurring == nil:
				log.Printf("prices: %s is not an active recurring price", s.Cfg.Stripe.PriceID)
			case p.Product == nil, !p.Product.Active:
				log.Printf("prices: %s parent product is missing or inactive", s.Cfg.Stripe.PriceID)
			default:
				name := p.Product.Name
				if p.Nickname != "" && p.Nickname != name {
					name = name + " — " + p.Nickname
				}
				features := make([]string, 0, len(p.Product.MarketingFeatures))
				for _, f := range p.Product.MarketingFeatures {
					if f != nil && f.Name != "" {
						features = append(features, f.Name)
					}
				}
				// Split major/minor + currency + interval so the template can
				// give each part its own typographic weight (matches the
				// homepage pricing card layout).
				major := p.UnitAmount / 100
				minor := p.UnitAmount % 100
				amount := fmt.Sprintf("%d", major)
				if minor != 0 {
					amount = fmt.Sprintf("%d.%02d", major, minor)
				}
				ccy := string(p.Currency)
				if ccy == "" {
					ccy = "DKK"
				}
				view.Option = priceOption{
					ID:          p.ID,
					Name:        name,
					Description: p.Product.Description,
					Features:    features,
					Amount:      amount,
					Currency:    stripeUpper(ccy),
					Interval:    "per " + string(p.Recurring.Interval),
				}
			}
		}
		view.Empty = view.Option.ID == ""

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
