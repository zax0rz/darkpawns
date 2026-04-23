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
				http.ServeFile(w, r, filepath.Join("web", "onboarding", "onboarding.md"))
				return
			}
			
			// Check for JSON request
			if strings.Contains(accept, "application/json") {
				http.ServeFile(w, r, filepath.Join("web", "onboarding", "onboarding.json"))
				return
			}
			
			// Default to HTML
			http.ServeFile(w, r, filepath.Join("web", "onboarding", "index.html"))
			return
		}
		
		// Handle API requests
		if strings.HasPrefix(r.URL.Path, "/api") {
			if r.URL.Path == "/api/openapi.json" {
				http.ServeFile(w, r, filepath.Join("web", "api", "openapi.json"))
				return
			}
			
			// Default API response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "API endpoint not found", "docs": "/api/openapi.json"}`))
			return
		}
		
		// Pass through to next handler
		next.ServeHTTP(w, r)
	})
}