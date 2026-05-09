// Package i18n provides language detection and string translations for the
// dashboard / member-portal htmx fragments and redirects.
//
// The two supported languages are English ("en", default) and Danish ("da").
// Frontend pages live at /<page> for English and /da/<page> for Danish; this
// package's helpers pick the right path/strings based on the request's
// HX-Current-URL or Referer header.
package i18n

import (
	"net/http"
	"net/url"
	"strings"
)

// Lang is the canonical language code: "en" or "da".
type Lang string

const (
	EN Lang = "en"
	DA Lang = "da"
)

// FromRequest returns the language for any incoming request.
//
// Sources, in priority order:
//
//  1. HX-Current-URL header — set by htmx on every fragment request, contains
//     the URL of the page the user is viewing. Most reliable for fragments.
//  2. Referer header — for form POSTs, the URL of the page the form was on.
//  3. The request's own URL path — for direct page loads (browser navigation
//     to /da/dashboard). Catches cases where neither header is present.
//
// Returns DA if any source has a /da/ prefix; EN otherwise.
func FromRequest(r *http.Request) Lang {
	if hxURL := r.Header.Get("HX-Current-URL"); hxURL != "" {
		if u, err := url.Parse(hxURL); err == nil && hasDaPrefix(u.Path) {
			return DA
		}
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil && hasDaPrefix(u.Path) {
			return DA
		}
	}
	if hasDaPrefix(r.URL.Path) {
		return DA
	}
	return EN
}

// hasDaPrefix reports whether path starts with /da or /da/.
func hasDaPrefix(path string) bool {
	return path == "/da" || strings.HasPrefix(path, "/da/")
}

// Path prefixes p with /da when lang is Danish.
// Use for redirect targets after form POSTs. Examples:
//
//	Path(EN, "/dashboard")          -> "/dashboard"
//	Path(DA, "/dashboard")          -> "/da/dashboard"
//	Path(DA, "/login/?reason=misc") -> "/da/login/?reason=misc"
//	Path(DA, "/")                   -> "/da/"
func Path(lang Lang, p string) string {
	if lang != DA {
		return p
	}
	if p == "" || p == "/" {
		return "/da/"
	}
	return "/da" + p
}
