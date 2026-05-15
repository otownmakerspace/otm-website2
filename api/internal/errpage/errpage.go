// Package errpage serves the Hugo-built styled error pages (404, 500, 403,
// etc.) with the right HTTP status code. Shared between the http router
// (for static-fallback 404s) and the stripe service (for page-level
// handlers that hit a 500 condition).
//
// Fragment handlers — htmx-loaded snippets like /dashboard/hero or
// /invoices — should NOT use this. Their errors are swapped into small
// regions of a parent page; a full-styled error page would replace the
// fragment region with header/footer/hero chrome and look broken. Those
// stay on plain stdhttp.Error.
package errpage

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Render writes the Hugo-built error page for `code` (e.g. "404", "500",
// "403") to w with the given HTTP status. Tries both URL conventions Hugo
// emits:
//
//   - <staticDir>/<code>.html       (Hugo's auto-generated 404.html)
//   - <staticDir>/<code>/index.html (content/<code>.md with url: "/<code>/")
//
// Picks the Danish localized variant under da/ when the request path
// begins with /da/. Falls back to plain status text if nothing matches —
// better than a silent 200 OK on an error.
func Render(w http.ResponseWriter, r *http.Request, staticDir, code string, status int) {
	candidates := []string{}
	if strings.HasPrefix(r.URL.Path, "/da/") {
		candidates = append(candidates,
			"da/"+code+".html",
			"da/"+code+"/index.html",
		)
	}
	candidates = append(candidates,
		code+".html",
		code+"/index.html",
	)
	for _, p := range candidates {
		data, err := os.ReadFile(filepath.Join(staticDir, p))
		if err != nil {
			continue
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		w.Write(data)
		return
	}
	http.Error(w, http.StatusText(status), status)
}
