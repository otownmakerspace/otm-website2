// mailtest sends a single transactional email through the same Mailer the
// real server uses. Lets you exercise each template against your real SMTP
// config without spinning up the full stack or going through Stripe.
//
// Usage from the api/ directory:
//
//	go run ./cmd/mailtest -to you@example.com                    # default: reset
//	go run ./cmd/mailtest -to you@example.com -template welcome
//	go run ./cmd/mailtest -to you@example.com -template all      # send every template in turn
//
// Reads SMTP credentials from (in priority order):
//  1. -user / -password / -host / -port flags
//  2. EMAIL_ADDRESS / EMAIL_PASSWORD / EMAIL_HOST / EMAIL_PORT env vars
//  3. Secret files under -secrets dir (default ../secrets/app)
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/iustin94/makerspace/api/internal/config"
	"github.com/iustin94/makerspace/api/internal/email"
	"github.com/iustin94/makerspace/api/internal/user"
)

var allTemplates = map[string]email.Template{
	"welcome":        email.Welcome,
	"goodbye":        email.Goodbye,
	"reset":          email.Reset,
	"unsubscription": email.Unsubscription,
	"new_member":     email.NewMember,
}

func main() {
	to := flag.String("to", "", "recipient address (required)")
	tmpl := flag.String("template", "reset", "template name: welcome|goodbye|reset|unsubscription|new_member|all")
	secretsDir := flag.String("secrets", "../secrets/app", "directory holding email_address and email_password files")
	user_ := flag.String("user", "", "SMTP username (overrides env / secret file)")
	password := flag.String("password", "", "SMTP password (overrides env / secret file)")
	host := flag.String("host", "", "SMTP host (overrides env)")
	port := flag.Int("port", 0, "SMTP port (overrides env)")
	name := flag.String("name", "Test User", "name to render into the template")
	link := flag.String("link", "http://localhost:4242/example/token-12345", "link to render into the template")
	flag.Parse()

	if *to == "" {
		log.Fatal("--to is required")
	}

	cfg := config.EmailConf{
		User:     pickString(*user_, os.Getenv("EMAIL_ADDRESS"), readFile(*secretsDir, "email_address")),
		Password: pickString(*password, os.Getenv("EMAIL_PASSWORD"), readFile(*secretsDir, "email_password")),
		Host:     pickString(*host, os.Getenv("EMAIL_HOST")),
		Port:     pickInt(*port, atoiSafe(os.Getenv("EMAIL_PORT"))),
	}
	if cfg.User == "" || cfg.Password == "" || cfg.Host == "" || cfg.Port == 0 {
		log.Fatalf("missing config — user=%q host=%q port=%d password=%t",
			cfg.User, cfg.Host, cfg.Port, cfg.Password != "")
	}

	templates := []string{*tmpl}
	if *tmpl == "all" {
		templates = []string{"welcome", "goodbye", "reset", "unsubscription", "new_member"}
	}

	brand := config.BrandConf{
		Name:            pickString(os.Getenv("BRAND_NAME"), "O'Town Makerspace"),
		WordmarkLeading: pickString(os.Getenv("BRAND_WORDMARK_LEADING"), "O'TOWN"),
		WordmarkAccent:  pickString(os.Getenv("BRAND_WORDMARK_ACCENT"), "MAKERSPACE"),
		LogoURL:         pickString(os.Getenv("BRAND_LOGO_URL"), "https://makerspace.olaru.dk/images/branding/logos/wordmark-consolidated.svg"),
	}
	m := email.NewMailer(cfg, brand)
	u := user.User{Name: *name, Email: *to}

	for _, key := range templates {
		t, ok := allTemplates[key]
		if !ok {
			log.Fatalf("unknown template: %s (valid: welcome, goodbye, reset, unsubscription, new_member, all)", key)
		}
		// new_member and unsubscription read .Data.Number; supply a fake.
		var data any = struct{ Number int }{Number: 42}
		log.Printf("→ sending %q (%s) to %s via %s:%d as %s",
			t.Subject, t.File, *to, cfg.Host, cfg.Port, cfg.User)
		if err := m.Send(*to, u, t, *link, data); err != nil {
			log.Fatalf("✗ send failed: %v", err)
		}
		log.Printf("✓ sent")
	}
	log.Printf("done — %d message(s) sent", len(templates))
}

func readFile(dir, name string) string {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func pickString(opts ...string) string {
	for _, s := range opts {
		if s != "" {
			return s
		}
	}
	return ""
}

func pickInt(opts ...int) int {
	for _, n := range opts {
		if n != 0 {
			return n
		}
	}
	return 0
}

func atoiSafe(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
