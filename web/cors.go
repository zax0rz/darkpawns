package web

import (
	"log"
	"net/http"
	"os"
	"strings"
)

// CORSMiddleware provides configurable CORS headers for HTTP requests
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment or use defaults
		allowedOrigins := getAllowedOrigins()
		
		origin := r.Header.Get("Origin")
		
		// Check if origin is allowed
		if origin != "" && isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		}
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func getAllowedOrigins() []string {
	// Read from environment variable
	if envOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); envOrigins != "" {
		return strings.Split(envOrigins, ",")
	}

	// Default development origins
	if isDevMode() {
		return []string{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:3000"}
	}

	// Production defaults — explicit list only, no wildcards
	return []string{
		"https://darkpawns.example.com",
		"https://game.darkpawns.example.com",
	}
}

// allowedSubdomains lists the specific subdomains permitted for CORS.
// M-12: No wildcard matching — only explicitly listed subdomains are allowed.
var allowedSubdomains = map[string][]string{
	"darkpawns.example.com": {"game", "www", "api"},
}

func isDevMode() bool {
	return os.Getenv("ENVIRONMENT") == "development"
}

func isOriginAllowed(origin string, allowed []string) bool {
	// M-13: Development mode allows all origins.
	// This must NEVER activate in production. The guard is explicitly
	// checking ENVIRONMENT and cannot be overridden via CORS_ALLOWED_ORIGINS.
	if isDevMode() {
		log.Printf("[CORS] WARNING: dev mode — allowing origin %q (NEVER ship this config)", origin)
		return true
	}

	// Production: only exact matches or explicitly listed subdomains
	for _, allowedOrigin := range allowed {
		if origin == allowedOrigin {
			return true
		}
	}

	// Check against explicit subdomain allowlists per base domain
	for baseDomain, subs := range allowedSubdomains {
		suffix := "." + baseDomain
		if strings.HasSuffix(origin, suffix) {
			prefix := strings.TrimSuffix(origin, suffix)
			for _, sub := range subs {
				if prefix == sub {
					return true
				}
			}
		}
	}

	return false
}