package web

import (
	"log"
	"net"
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
		if origin != "" && isOriginAllowed(origin, allowedOrigins, r) {
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
		parts := strings.Split(envOrigins, ",")
		origins := make([]string, 0, len(parts))
		for _, o := range parts {
			o = strings.TrimSpace(o)
			if o != "" {
				origins = append(origins, o)
			}
		}
		return origins
	}

	// Default development origins
	if isDevMode(nil) {
		return []string{"http://localhost:3000", "http://localhost:4350", "http://127.0.0.1:3000"}
	}

	// Production defaults — explicit list only, no wildcards
	return []string{
		"https://darkpawns.labz0rz.com",
	}
}

// allowedSubdomains lists the specific subdomains permitted for CORS.
// M-12: No wildcard matching — only explicitly listed subdomains are allowed.
var allowedSubdomains = map[string][]string{}

func isDevMode(r *http.Request) bool {
	if os.Getenv("ENVIRONMENT") != "development" {
		return false
	}
	if r == nil {
		return true
	}
	if peerIsLocal(r.RemoteAddr) {
		return true
	}
	log.Printf("[CORS] WARNING: dev mode CORS rejected for non-local peer %q (host header %q)", r.RemoteAddr, r.Host)
	return false
}

// peerIsLocal reports whether the connection's remote peer is a loopback
// address. The decision is based on the actual TCP peer (RemoteAddr), not the
// client-controlled Host header.
func peerIsLocal(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(host, "localhost")
}

func isOriginAllowed(origin string, allowed []string, r *http.Request) bool {
	// M-13: Development mode allows all origins, but only when the request is
	// directed at a local address. This must NEVER activate in production.
	// The guard checks ENVIRONMENT and the request host and cannot be overridden
	// via CORS_ALLOWED_ORIGINS.
	if isDevMode(r) {
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
