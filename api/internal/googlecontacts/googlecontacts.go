// Package googlecontacts syncs the membership mailing list into a Google
// Contacts label (contact group) on the admin's personal Gmail account, via
// the People API.
//
// Gmail has no native "mailing list" object — the idiom for a personal
// account is a Contacts label: typing the label name into To/BCC in the Gmail
// composer expands it to every contact carrying the label. So "add to the
// mailing list" means "ensure a contact with this email exists and carries
// the label", and "remove" means "take the label off" (the contact itself is
// left in place — we never delete contacts we may not own).
//
// Auth is a pre-minted OAuth2 refresh token (see cmd/googleauth for the
// one-time interactive flow that produces it). A service account would not
// work here: personal Gmail accounts don't support domain-wide delegation.
package googlecontacts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/iustin94/makerspace/api/internal/config"
)

const apiBase = "https://people.googleapis.com/v1"

// Scope is the People API read/write-contacts scope the refresh token must
// have been granted (cmd/googleauth requests exactly this).
const Scope = "https://www.googleapis.com/auth/contacts"

// Endpoint is Google's OAuth2 endpoint pair. Declared here (not pulled from
// golang.org/x/oauth2/google) to keep the dependency surface at x/oauth2 core.
var Endpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.google.com/o/oauth2/auth",
	TokenURL: "https://oauth2.googleapis.com/token",
}

// ErrDisabled is returned by AddMember/RemoveMember when the client was built
// without credentials. Callers normally gate on Enabled() instead; the error
// exists so a missed gate is loud rather than a silent no-op.
var ErrDisabled = errors.New("googlecontacts: disabled (missing client id/secret/refresh token)")

// Client talks to the People API on behalf of the admin Google account.
// Zero-credential configs produce a disabled client (Enabled() == false) so
// deployments without the Google secrets keep working unchanged.
type Client struct {
	httpc *http.Client
	group string // label display name, e.g. "Makerspace Members"

	mu            sync.Mutex
	groupResource string // cached "contactGroups/<id>", resolved lazily
}

// New builds a Client from config. The oauth2 transport self-refreshes access
// tokens from the long-lived refresh token, so the client needs no further
// interactive auth for its lifetime.
func New(cfg config.GoogleConf) *Client {
	c := &Client{group: cfg.ContactGroup}
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RefreshToken == "" {
		return c // disabled
	}
	oc := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     Endpoint,
		Scopes:       []string{Scope},
	}
	// Background context: the token source outlives any single request.
	c.httpc = oc.Client(context.Background(), &oauth2.Token{RefreshToken: cfg.RefreshToken})
	return c
}

// Enabled reports whether credentials were configured.
func (c *Client) Enabled() bool { return c != nil && c.httpc != nil }

// AddMember ensures a contact with this email exists and carries the mailing
// list label. Idempotent: adding an already-labelled contact is a no-op on
// Google's side.
func (c *Client) AddMember(ctx context.Context, email, name string) error {
	if !c.Enabled() {
		return ErrDisabled
	}
	group, err := c.ensureGroup(ctx)
	if err != nil {
		return err
	}
	person, err := c.findContactByEmail(ctx, email)
	if err != nil {
		return err
	}
	if person == "" {
		return c.createContact(ctx, group, email, name)
	}
	return c.modifyGroupMembers(ctx, group, []string{person}, nil)
}

// RemoveMember takes the mailing list label off the contact with this email.
// The contact itself is preserved. Unknown emails are a no-op (the member may
// have been added before the integration existed, or removed by hand).
func (c *Client) RemoveMember(ctx context.Context, email string) error {
	if !c.Enabled() {
		return ErrDisabled
	}
	group, err := c.ensureGroup(ctx)
	if err != nil {
		return err
	}
	person, err := c.findContactByEmail(ctx, email)
	if err != nil {
		return err
	}
	if person == "" {
		return nil
	}
	return c.modifyGroupMembers(ctx, group, nil, []string{person})
}

// Member is one desired mailing-list entry, as SyncMembers consumes them.
type Member struct {
	Email string
	Name  string
}

// syncCreateDelay paces contact creations during SyncMembers so a large
// backfill stays inside the People API per-minute write quota. Labelling
// existing contacts is exempt — that is a single batched members:modify call.
const syncCreateDelay = 700 * time.Millisecond

// modifyBatchMax caps resource names per members:modify request, comfortably
// under the People API's documented maximum of 1000 per mutate call.
const modifyBatchMax = 200

// SyncMembers ensures every member is a contact carrying the mailing-list
// label. Duplicate-safe by construction: the full contact list is fetched
// ONCE up front, so re-running (e.g. on every server start) re-labels rather
// than re-creates, and repeated emails in the input collapse to one contact.
// Contacts are never removed from the label — the cancellation webhook owns
// removal.
//
// Returns how many contacts were created, how many existing contacts were
// newly labelled, and how many were already in place. A per-member create
// failure is collected rather than fatal — the rest of the list still syncs —
// and the joined error reports every failure.
func (c *Client) SyncMembers(ctx context.Context, members []Member) (created, labelled, already int, err error) {
	if !c.Enabled() {
		return 0, 0, 0, ErrDisabled
	}
	group, err := c.ensureGroup(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	byEmail, inGroup, err := c.listContacts(ctx, group)
	if err != nil {
		return 0, 0, 0, err
	}

	var errs []error
	var toLabel []string
	seen := make(map[string]bool, len(members))
	paced := false
	for _, m := range members {
		email := strings.ToLower(strings.TrimSpace(m.Email))
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true

		person, exists := byEmail[email]
		switch {
		case !exists:
			if paced {
				select {
				case <-time.After(syncCreateDelay):
				case <-ctx.Done():
					return created, labelled, already, errors.Join(append(errs, ctx.Err())...)
				}
			}
			paced = true
			if cerr := c.createContact(ctx, group, email, m.Name); cerr != nil {
				errs = append(errs, cerr)
				continue
			}
			created++
		case !inGroup[person]:
			toLabel = append(toLabel, person)
			inGroup[person] = true // two input emails can map to one contact — label it once
			labelled++
		default:
			already++
		}
	}
	for start := 0; start < len(toLabel); start += modifyBatchMax {
		batch := toLabel[start:min(start+modifyBatchMax, len(toLabel))]
		if merr := c.modifyGroupMembers(ctx, group, batch, nil); merr != nil {
			labelled -= len(batch)
			errs = append(errs, merr)
		}
	}
	return created, labelled, already, errors.Join(errs...)
}

// listContacts fetches every connection once, returning each email address
// mapped to its person resource name (a multi-email contact appears under all
// its addresses) plus the set of people already carrying the label. One
// paginated sweep serves a whole SyncMembers run — the per-member lookup that
// findContactByEmail does would cost one full listing PER member.
func (c *Client) listContacts(ctx context.Context, group string) (byEmail map[string]string, inGroup map[string]bool, err error) {
	byEmail = make(map[string]string)
	inGroup = make(map[string]bool)
	pageToken := ""
	for {
		var out struct {
			Connections []struct {
				ResourceName   string `json:"resourceName"`
				EmailAddresses []struct {
					Value string `json:"value"`
				} `json:"emailAddresses"`
				Memberships []struct {
					ContactGroupMembership struct {
						ContactGroupResourceName string `json:"contactGroupResourceName"`
					} `json:"contactGroupMembership"`
				} `json:"memberships"`
			} `json:"connections"`
			NextPageToken string `json:"nextPageToken"`
		}
		q := url.Values{
			"personFields": {"emailAddresses,memberships"},
			"pageSize":     {"1000"},
		}
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		if err := c.call(ctx, http.MethodGet, "/people/me/connections?"+q.Encode(), nil, &out); err != nil {
			return nil, nil, fmt.Errorf("list connections: %w", err)
		}
		for _, p := range out.Connections {
			for _, e := range p.EmailAddresses {
				addr := strings.ToLower(strings.TrimSpace(e.Value))
				if addr == "" {
					continue
				}
				if _, dup := byEmail[addr]; !dup {
					byEmail[addr] = p.ResourceName
				}
			}
			for _, m := range p.Memberships {
				if m.ContactGroupMembership.ContactGroupResourceName == group {
					inGroup[p.ResourceName] = true
				}
			}
		}
		pageToken = out.NextPageToken
		if pageToken == "" {
			return byEmail, inGroup, nil
		}
	}
}

// ensureGroup resolves the label's resource name ("contactGroups/<id>"),
// creating the label on first use. Cached for the process lifetime — labels
// keep their id even if renamed in the Contacts UI, so the cache can't go
// stale short of the label being deleted (which surfaces as a 404 on modify).
func (c *Client) ensureGroup(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.groupResource != "" {
		return c.groupResource, nil
	}

	pageToken := ""
	for {
		var out struct {
			ContactGroups []struct {
				ResourceName  string `json:"resourceName"`
				FormattedName string `json:"formattedName"`
			} `json:"contactGroups"`
			NextPageToken string `json:"nextPageToken"`
		}
		q := url.Values{"pageSize": {"200"}}
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		if err := c.call(ctx, http.MethodGet, "/contactGroups?"+q.Encode(), nil, &out); err != nil {
			return "", fmt.Errorf("list contact groups: %w", err)
		}
		for _, g := range out.ContactGroups {
			if strings.EqualFold(g.FormattedName, c.group) {
				c.groupResource = g.ResourceName
				return c.groupResource, nil
			}
		}
		pageToken = out.NextPageToken
		if pageToken == "" {
			break
		}
	}

	var created struct {
		ResourceName string `json:"resourceName"`
	}
	body := map[string]any{"contactGroup": map[string]string{"name": c.group}}
	if err := c.call(ctx, http.MethodPost, "/contactGroups", body, &created); err != nil {
		return "", fmt.Errorf("create contact group %q: %w", c.group, err)
	}
	c.groupResource = created.ResourceName
	return c.groupResource, nil
}

// findContactByEmail returns the resource name ("people/<id>") of the first
// contact whose email matches, or "" if none. Implemented as a full
// connections listing rather than people:searchContacts because search
// requires an out-of-band cache-warmup request and can serve stale results —
// unacceptable inside a webhook. A makerspace's contact list is small; the
// paginated listing is one or two requests.
func (c *Client) findContactByEmail(ctx context.Context, email string) (string, error) {
	pageToken := ""
	for {
		var out struct {
			Connections []struct {
				ResourceName   string `json:"resourceName"`
				EmailAddresses []struct {
					Value string `json:"value"`
				} `json:"emailAddresses"`
			} `json:"connections"`
			NextPageToken string `json:"nextPageToken"`
		}
		q := url.Values{
			"personFields": {"emailAddresses"},
			"pageSize":     {"1000"},
		}
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		if err := c.call(ctx, http.MethodGet, "/people/me/connections?"+q.Encode(), nil, &out); err != nil {
			return "", fmt.Errorf("list connections: %w", err)
		}
		for _, p := range out.Connections {
			for _, e := range p.EmailAddresses {
				if strings.EqualFold(strings.TrimSpace(e.Value), strings.TrimSpace(email)) {
					return p.ResourceName, nil
				}
			}
		}
		pageToken = out.NextPageToken
		if pageToken == "" {
			return "", nil
		}
	}
}

// createContact inserts a new contact already carrying the label.
func (c *Client) createContact(ctx context.Context, group, email, name string) error {
	person := map[string]any{
		"emailAddresses": []map[string]string{{"value": email}},
		"memberships": []map[string]any{
			{"contactGroupMembership": map[string]string{"contactGroupResourceName": group}},
		},
	}
	if name != "" {
		person["names"] = []map[string]string{{"unstructuredName": name}}
	}
	if err := c.call(ctx, http.MethodPost, "/people:createContact", person, nil); err != nil {
		return fmt.Errorf("create contact %s: %w", email, err)
	}
	return nil
}

// modifyGroupMembers adds/removes contacts to/from the label in one call.
func (c *Client) modifyGroupMembers(ctx context.Context, group string, add, remove []string) error {
	body := map[string]any{}
	if len(add) > 0 {
		body["resourceNamesToAdd"] = add
	}
	if len(remove) > 0 {
		body["resourceNamesToRemove"] = remove
	}
	if err := c.call(ctx, http.MethodPost, "/"+group+"/members:modify", body, nil); err != nil {
		return fmt.Errorf("modify members of %s: %w", group, err)
	}
	return nil
}

// call performs one authorized JSON request. Non-2xx responses become errors
// carrying a snippet of Google's error body (which names the failing field —
// invaluable when a field mask or scope is wrong).
func (c *Client) call(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, apiBase+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(snippet)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
