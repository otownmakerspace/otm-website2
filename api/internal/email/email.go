// Package email sends transactional mail via SMTP using templates
// embedded into the binary at compile time (no runtime file lookup).
package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"

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

var (
	Welcome        = Template{File: "welcome.html", Subject: "Welcome to The O'Town Garage!"}
	Goodbye        = Template{File: "goodbye.html", Subject: "See you again soon!"}
	Reset          = Template{File: "reset.html", Subject: "Reset your password"}
	Unsubscription = Template{File: "unsubscription.html", Subject: "Canceled subscription"}
	NewMember      = Template{File: "notify_new_member.html", Subject: "New member!"}
)

// Mailer wraps SMTP credentials so handlers can call Send without
// reaching for package globals.
type Mailer struct {
	cfg config.EmailConf
}

func NewMailer(cfg config.EmailConf) *Mailer {
	return &Mailer{cfg: cfg}
}

// Send renders the template with the supplied data fields and dispatches via SMTP.
// Errors during template parse/exec are returned; SMTP errors are logged and returned.
func (m *Mailer) Send(to string, u user.User, t Template, link string, data any) error {
	tmpl, err := template.ParseFS(templatesFS, "templates/"+t.File)
	if err != nil {
		log.Print("email: parse template:", err)
		return err
	}

	var body bytes.Buffer
	err = tmpl.Execute(&body, struct {
		Name  string
		Link  string
		Email string
		Data  any
	}{Name: u.Name, Link: link, Email: u.Email, Data: data})
	if err != nil {
		log.Print("email: execute template:", err)
		return err
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.cfg.User)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", t.Subject)
	msg.SetBody("text/html", body.String())

	dialer := gomail.NewDialer(m.cfg.Host, m.cfg.Port, m.cfg.User, m.cfg.Password)
	if err := dialer.DialAndSend(msg); err != nil {
		log.Printf("email: send to %s failed: %v", to, err)
		return fmt.Errorf("email send: %w", err)
	}
	log.Printf("email: sent %q to %s", t.Subject, to)
	return nil
}

// AdminAddress returns the From-address used by the mailer; used when notifying
// the operations inbox of new members or cancellations.
func (m *Mailer) AdminAddress() string {
	return m.cfg.User
}
