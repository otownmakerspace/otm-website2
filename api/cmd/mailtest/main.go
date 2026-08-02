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
//  1. -user / -password / -host / -port / -from / -reply-to flags
//  2. SMTP_USERNAME / SMTP_PASSWORD / SMTP_HOST / SMTP_PORT / MAIL_FROM /
//     MAIL_REPLY_TO env vars
//  3. Secret files under -secrets dir (default ../secrets/app) — username/password only
//  4. Dotenv file at -env path (default ../infra/app/.env)
package main

import (
	"bufio"
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
	secretsDir := flag.String("secrets", "../secrets/app", "directory holding smtp_username and smtp_password files")
	envFile := flag.String("env", "../infra/app/.env", "dotenv file scanned for SMTP_*/MAIL_* fallbacks (host/port live here for the dev stack)")
	user_ := flag.String("user", "", "SMTP username (overrides env / secret file / dotenv)")
	password := flag.String("password", "", "SMTP password (overrides env / secret file / dotenv)")
	host := flag.String("host", "", "SMTP host (overrides env / dotenv)")
	port := flag.Int("port", 0, "SMTP port (overrides env / dotenv)")
	from := flag.String("from", "", "visible From address — must be a Send-mail-as alias of the SMTP account or Gmail silently rewrites it (overrides env / dotenv; empty = SMTP username)")
	replyTo := flag.String("reply-to", "", "Reply-To + footer contact address (overrides env / dotenv; empty = SMTP username)")
	name := flag.String("name", "Test User", "name to render into the template")
	baseURL := flag.String("base-url", "https://otownmakerspace.dk", "public base URL for the deployed environment under test — drives both the link rendered in the template and the LogoURL fetched by the email client. Point at a staging host to test against staging.")
	link := flag.String("link", "", "fully-qualified link to render into the template (overrides -base-url-derived default)")
	flag.Parse()

	if *to == "" {
		log.Fatal("--to is required")
	}

	dotenv := readDotenv(*envFile)
	// Same resolution order as config.Load: flag, env var, secret file, dotenv.
	cfg := config.EmailConf{
		Username: pickString(*user_, os.Getenv("SMTP_USERNAME"), readFile(*secretsDir, "smtp_username"), dotenv["SMTP_USERNAME"]),
		Password: pickString(*password, os.Getenv("SMTP_PASSWORD"), readFile(*secretsDir, "smtp_password"), dotenv["SMTP_PASSWORD"]),
		Host:     pickString(*host, os.Getenv("SMTP_HOST"), dotenv["SMTP_HOST"]),
		Port:     pickInt(*port, atoiSafe(os.Getenv("SMTP_PORT")), atoiSafe(dotenv["SMTP_PORT"])),
		From:     pickString(*from, os.Getenv("MAIL_FROM"), dotenv["MAIL_FROM"]),
		ReplyTo:  pickString(*replyTo, os.Getenv("MAIL_REPLY_TO"), dotenv["MAIL_REPLY_TO"]),
	}
	if cfg.Username == "" || cfg.Password == "" || cfg.Host == "" || cfg.Port == 0 {
		log.Fatalf("missing config — user=%q host=%q port=%d password=%t",
			cfg.Username, cfg.Host, cfg.Port, cfg.Password != "")
	}

	templates := []string{*tmpl}
	if *tmpl == "all" {
		templates = []string{"welcome", "goodbye", "reset", "unsubscription", "new_member"}
	}

	base := strings.TrimRight(*baseURL, "/")
	brand := config.BrandConf{
		Name:            pickString(os.Getenv("BRAND_NAME"), "O'Town Makerspace"),
		WordmarkLeading: pickString(os.Getenv("BRAND_WORDMARK_LEADING"), "O'TOWN"),
		WordmarkAccent:  pickString(os.Getenv("BRAND_WORDMARK_ACCENT"), "MAKERSPACE"),
		LogoURL:         pickString(os.Getenv("BRAND_LOGO_URL"), base+"/images/branding/logos/wordmark-horizontal.svg"),
	}
	resolvedLink := pickString(*link, base+"/example/token-12345")
	portalURL := "https://members." + strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
	m := email.NewMailer(cfg, brand, base, portalURL)
	u := user.User{Name: *name, Email: *to}

	for _, key := range templates {
		t, ok := allTemplates[key]
		if !ok {
			log.Fatalf("unknown template: %s (valid: welcome, goodbye, reset, unsubscription, new_member, all)", key)
		}
		// new_member and unsubscription read .Data.Number; supply a fake.
		var data any = struct{ Number int }{Number: 42}
		log.Printf("→ sending %q (%s) to %s via %s:%d auth=%s from=%s reply-to=%s — link=%s logo=%s",
			t.Subject, t.File, *to, cfg.Host, cfg.Port, cfg.Username,
			pickString(cfg.From, cfg.Username), pickString(cfg.ReplyTo, cfg.Username), resolvedLink, brand.LogoURL)
		if err := m.Send(*to, u, t, resolvedLink, data); err != nil {
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

// readDotenv parses a minimal KEY=VALUE dotenv file. Comments and blank lines
// are skipped; single- or double-quoted values are unquoted. Missing file is
// silently ignored — this is a best-effort fallback, not a config source of
// truth. Anything fancier than literal "KEY=VALUE" (interpolation, multiline,
// escapes) is out of scope; reach for godotenv if those ever matter.
func readDotenv(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"' || val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
		out[key] = val
	}
	return out
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
