package web

import (
	"net/http"
	"path/filepath"
	"strings"
)

// ContentNegotiationMiddleware handles Accept header-based content negotiation
func ContentNegotiationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only handle onboarding requests
		if strings.HasPrefix(r.URL.Path, "/onboarding") {
			accept := r.Header.Get("Accept")

			// Check for markdown request
			if strings.Contains(accept, "text/markdown") {
				http.ServeFile(w, r, filepath.Join("web", "onboarding", "onboarding.md")) // #nosec G703 — hardcoded path
				return
			}

			// Check for JSON request
			if strings.Contains(accept, "application/json") {
				http.ServeFile(w, r, filepath.Join("web", "onboarding", "onboarding.json")) // #nosec G703 — hardcoded path
				return
			}

			// Default to HTML
			http.ServeFile(w, r, filepath.Join("web", "onboarding", "index.html")) // #nosec G703 — hardcoded path
			return
		}

		// API routes are served by the registered API router; do not short-circuit them here.
		if strings.HasPrefix(r.URL.Path, "/api") {
			next.ServeHTTP(w, r)
			return
		}

		// Pass through to next handler
		next.ServeHTTP(w, r)
	})
}
