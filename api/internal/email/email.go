// Package email sends transactional mail via SMTP using templates
// embedded into the binary at compile time (no runtime file lookup).
package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"strings"

	gomail "gopkg.in/mail.v2"

	"github.com/iustin94/makerspace/api/internal/config"
	"github.com/iustin94/makerspace/api/internal/user"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Template names exactly match the file basenames in templates/.
type Template struct {
	File    string
	Subject string
}

// Subject lines may include the literal placeholder "{brand}" — it gets
// replaced with cfg.Brand.Name at send time. Placeholder absence is fine,
// the replacement is a no-op.
var (
	Welcome        = Template{File: "welcome.html", Subject: "Welcome to {brand}!"}
	Goodbye        = Template{File: "goodbye.html", Subject: "See you again soon!"}
	Reset          = Template{File: "reset.html", Subject: "Reset your password"}
	Unsubscription = Template{File: "unsubscription.html", Subject: "Canceled subscription"}
	NewMember      = Template{File: "notify_new_member.html", Subject: "New member!"}
)

// Mailer wraps SMTP credentials and brand identity so handlers can call
// Send without reaching for package globals.
//
// marketingURL (apex site) and portalURL (member backend) are passed in by
// the caller from config so templates never hardcode a deployment's host —
// staging mail links to staging, production to production. Sender identity:
// mail goes out From cfg.From (noreply-style alias) with Reply-To and the
// rendered contact address set to cfg.ReplyTo (the human-answered inbox);
// both fall back to cfg.Username (the SMTP login) when unset.
type Mailer struct {
	cfg          config.EmailConf
	brand        config.BrandConf
	marketingURL string
	portalURL    string
}

func NewMailer(cfg config.EmailConf, brand config.BrandConf, marketingURL, portalURL string) *Mailer {
	return &Mailer{cfg: cfg, brand: brand, marketingURL: marketingURL, portalURL: portalURL}
}

// fromAddr is the visible sender of automated mail. Falls back to the SMTP
// login so a deployment without MAIL_FROM behaves exactly as before the
// transport/identity split.
func (m *Mailer) fromAddr() string {
	if m.cfg.From != "" {
		return m.cfg.From
	}
	return m.cfg.Username
}

// replyToAddr is the human-answered address: Reply-To header, footer contact
// link, and admin-notification recipient. Falls back to the SMTP login.
func (m *Mailer) replyToAddr() string {
	if m.cfg.ReplyTo != "" {
		return m.cfg.ReplyTo
	}
	return m.cfg.Username
}

// host strips the scheme and any trailing slash from a URL so templates can
// show "otownmakerspace.dk" as link text while linking to the full URL.
func host(rawURL string) string {
	s := strings.TrimPrefix(rawURL, "https://")
	s = strings.TrimPrefix(s, "http://")
	return strings.TrimRight(s, "/")
}

// Send renders the template with the supplied data fields and dispatches via SMTP.
// Errors during template parse/exec are returned; SMTP errors are logged and returned.
//
// Each template defines an "email_content" block (and optionally "email_preheader"),
// then references the shared chrome via {{template "_layout.html" .}}. We parse the
// layout alongside the content template so block overrides resolve correctly.
func (m *Mailer) Send(to string, u user.User, t Template, link string, data any) error {
	tmpl, err := template.New(t.File).Funcs(template.FuncMap{"host": host}).
		ParseFS(templatesFS, "templates/_layout.html", "templates/"+t.File)
	if err != nil {
		log.Print("email: parse template:", err)
		return err
	}

	subject := strings.ReplaceAll(t.Subject, "{brand}", m.brand.Name)

	var body bytes.Buffer
	err = tmpl.ExecuteTemplate(&body, t.File, struct {
		Name         string
		Link         string
		Email        string
		Subject      string
		Brand        config.BrandConf
		MarketingURL string
		PortalURL    string
		ContactEmail string
		Data         any
	}{
		Name: u.Name, Link: link, Email: u.Email, Subject: subject, Brand: m.brand,
		MarketingURL: m.marketingURL, PortalURL: m.portalURL, ContactEmail: m.replyToAddr(), Data: data,
	})
	if err != nil {
		log.Print("email: execute template:", err)
		return err
	}

	msg := gomail.NewMessage()
	// Brand name as the display name — otherwise the receiving client (or
	// Gmail's SMTP layer) decorates the bare address with the sending
	// account's profile name, which is an ops detail, not the brand.
	msg.SetAddressHeader("From", m.fromAddr(), m.brand.Name)
	msg.SetHeader("To", to)
	// Replies to noreply-style automated mail should reach a human. Omitted
	// when From already is the contact address (redundant header).
	if m.replyToAddr() != m.fromAddr() {
		msg.SetAddressHeader("Reply-To", m.replyToAddr(), m.brand.Name)
	}
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body.String())

	dialer := gomail.NewDialer(m.cfg.Host, m.cfg.Port, m.cfg.Username, m.cfg.Password)
	if err := dialer.DialAndSend(msg); err != nil {
		log.Printf("email: send to %s failed: %v", to, err)
		return fmt.Errorf("email send: %w", err)
	}
	log.Printf("email: sent %q to %s", subject, to)
	return nil
}

// AdminAddress returns the operations inbox notified of new members and
// cancellations — the human-answered contact address, not the noreply sender.
func (m *Mailer) AdminAddress() string {
	return m.replyToAddr()
}
