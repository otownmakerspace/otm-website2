// Command googlebackfill seeds the Google Contacts mailing-list label from
// the membership database: every user with active = true is added to the
// label (existing contacts just gain the label; unknown emails become new
// contacts). One-time catch-up for members who joined before the webhook
// sync existed — from then on the webhook keeps the label current.
//
// Idempotent: re-running is safe, Google treats re-adding a labelled contact
// as a no-op. Members who cancelled before the sync existed are NOT removed —
// this tool only adds; prune the label by hand if needed.
//
// Usage from the api/ directory (against the local dev stack):
//
//	go run ./cmd/googlebackfill -dry-run   # print who would be added
//	go run ./cmd/googlebackfill            # do it
//
// Credentials resolve like cmd/mailtest: flags beat GOOGLE_* env vars, which
// beat secret files under -secrets (default ../secrets/app). The -db default
// is the dev stack's SQLite file (repo-root db/ is bind-mounted to /app/db in
// the container); point -db elsewhere to backfill from another environment's
// data, e.g. a copy of the VPS database.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iustin94/makerspace/api/internal/config"
	dbpkg "github.com/iustin94/makerspace/api/internal/db"
	"github.com/iustin94/makerspace/api/internal/googlecontacts"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "list the members that would be added, change nothing")
	secretsDir := flag.String("secrets", "../secrets/app", "directory holding google_client_id / google_client_secret / google_refresh_token files")
	clientID := flag.String("client-id", "", "Google OAuth client ID (overrides env / secret file)")
	clientSecret := flag.String("client-secret", "", "Google OAuth client secret (overrides env / secret file)")
	refreshToken := flag.String("refresh-token", "", "Google OAuth refresh token (overrides env / secret file)")
	group := flag.String("group", "", "Contacts label name (overrides GOOGLE_CONTACT_GROUP env; default \"Makerspace Members\")")
	dbPath := flag.String("db", "../db/production.db", "SQLite/libsql database file (default: the dev stack's bind-mounted file)")
	delay := flag.Duration("delay", 700*time.Millisecond, "pause between contact writes, to stay inside the People API per-minute write quota")
	flag.Parse()

	gcfg := config.GoogleConf{
		ClientID:     resolve(*clientID, "GOOGLE_CLIENT_ID", *secretsDir, "google_client_id"),
		ClientSecret: resolve(*clientSecret, "GOOGLE_CLIENT_SECRET", *secretsDir, "google_client_secret"),
		RefreshToken: resolve(*refreshToken, "GOOGLE_REFRESH_TOKEN", *secretsDir, "google_refresh_token"),
		ContactGroup: resolve(*group, "GOOGLE_CONTACT_GROUP", "", ""),
	}
	if gcfg.ContactGroup == "" {
		gcfg.ContactGroup = "Makerspace Members"
	}
	client := googlecontacts.New(gcfg)
	if !client.Enabled() {
		log.Fatal("google credentials incomplete — provide client id/secret/refresh token via flags, GOOGLE_* env vars, or files in ", *secretsDir)
	}

	if _, err := os.Stat(*dbPath); err != nil {
		log.Fatalf("db file %s: %v (pass -db pointing at the SQLite file)", *dbPath, err)
	}
	database, err := dbpkg.Open(config.BackendConf{DBPath: *dbPath})
	if err != nil {
		log.Fatal("db open: ", err)
	}
	defer database.Close()

	members, err := dbpkg.ListActive(database)
	if err != nil {
		log.Fatal("list active members: ", err)
	}
	if len(members) == 0 {
		fmt.Println("no active members in the database — nothing to do")
		return
	}

	fmt.Printf("%d active member(s) in %s → label %q\n\n", len(members), *dbPath, gcfg.ContactGroup)
	if *dryRun {
		for _, m := range members {
			fmt.Printf("  would add: %s (%s)\n", m.Email, m.Name)
		}
		fmt.Println("\ndry run — nothing written. Re-run without -dry-run to apply.")
		return
	}

	added, failed := 0, 0
	for i, m := range members {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := client.AddMember(ctx, m.Email, m.Name)
		cancel()
		if err != nil {
			failed++
			log.Printf("FAILED %s: %v", m.Email, err)
		} else {
			added++
			fmt.Printf("  added: %s (%s)\n", m.Email, m.Name)
		}
		if i < len(members)-1 {
			time.Sleep(*delay)
		}
	}

	fmt.Printf("\ndone: %d added, %d failed\n", added, failed)
	if failed > 0 {
		fmt.Println("re-run to retry failures — already-added members are no-ops")
		os.Exit(1)
	}
}

// resolve returns the first non-empty of: explicit flag value, env var,
// trimmed content of secretsDir/file (skipped when either part is empty).
func resolve(flagVal, envKey, secretsDir, file string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if secretsDir == "" || file == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(secretsDir, file))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
