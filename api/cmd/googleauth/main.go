// Command googleauth performs the one-time interactive OAuth2 flow that mints
// the long-lived Google refresh token the backend uses for the Contacts
// mailing-list sync (internal/googlecontacts).
//
// Prerequisites (one-time, in https://console.cloud.google.com):
//  1. Create a project (any name) and enable the "People API".
//  2. Configure the OAuth consent screen. IMPORTANT: set Publishing status to
//     "In production" — while it is "Testing", Google expires refresh tokens
//     after 7 days, which silently kills the sync.
//  3. Create an OAuth client ID of type "Desktop app"; note the client id and
//     secret.
//
// Then run, on a machine with a browser:
//
//	GOOGLE_CLIENT_ID=... GOOGLE_CLIENT_SECRET=... go run ./cmd/googleauth
//
// Sign in with the Gmail account that should own the mailing list, approve
// the Contacts permission, and copy the printed refresh token into the
// google_refresh_token secret file (plus the id/secret into their files).
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/oauth2"

	"github.com/iustin94/makerspace/api/internal/config"
	"github.com/iustin94/makerspace/api/internal/googlecontacts"
)

func main() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET (from a Desktop-app OAuth client in Google Cloud console)")
	}

	// Loopback redirect on an OS-assigned port. Google's "Desktop app" client
	// type accepts any localhost port, so nothing needs registering.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("listen: ", err)
	}
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     googlecontacts.Endpoint,
		Scopes:       []string{googlecontacts.Scope},
		RedirectURL:  fmt.Sprintf("http://%s/callback", ln.Addr().String()),
	}

	// Random state guards the loopback listener against a stray/forged hit.
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		log.Fatal("rand: ", err)
	}
	state := hex.EncodeToString(stateBytes)

	// AccessTypeOffline requests a refresh token; ApprovalForce guarantees one
	// is issued even if this account already granted the scope before (Google
	// only returns a refresh token on the first consent otherwise).
	authURL := conf.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("Open this URL in your browser and sign in with the Gmail account that owns the mailing list:")
	fmt.Println()
	fmt.Println("  " + authURL)
	fmt.Println()

	codeCh := make(chan string, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" || r.URL.Query().Get("state") != state {
			http.NotFound(w, r)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			fmt.Fprintf(w, "Authorization failed: %s. You can close this tab.", errMsg)
			log.Fatal("authorization denied: ", errMsg)
		}
		fmt.Fprint(w, "Authorized — you can close this tab and return to the terminal.")
		codeCh <- r.URL.Query().Get("code")
	})}
	go srv.Serve(ln)

	code := <-codeCh
	ctx := context.Background()
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal("token exchange: ", err)
	}
	if tok.RefreshToken == "" {
		log.Fatal("Google returned no refresh token — revoke the app's access at https://myaccount.google.com/permissions and run again")
	}

	fmt.Println()
	fmt.Println("Refresh token (store as the google_refresh_token secret):")
	fmt.Println()
	fmt.Println("  " + tok.RefreshToken)
	fmt.Println()

	// Optional end-to-end smoke test: add a contact to the label exactly the
	// way the server will, so a scope or consent-screen mistake surfaces now,
	// not at the first webhook.
	smokeEmail := os.Getenv("GOOGLE_SMOKE_TEST_EMAIL")
	if smokeEmail == "" {
		fmt.Println("Tip: set GOOGLE_SMOKE_TEST_EMAIL=<some address> to also run an end-to-end add-to-label test.")
		return
	}
	group := os.Getenv("GOOGLE_CONTACT_GROUP")
	if group == "" {
		group = "Makerspace Members"
	}
	client := googlecontacts.New(config.GoogleConf{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: tok.RefreshToken,
		ContactGroup: group,
	})
	if err := client.AddMember(ctx, smokeEmail, "Smoke Test"); err != nil {
		log.Fatal("smoke test: ", err)
	}
	fmt.Printf("Smoke test OK: %s added to label %q — remove it by hand if unwanted.\n", smokeEmail, group)
}
