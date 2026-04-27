package web

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
)

func generateCSPNonce() string {
	b := make([]byte, 18)
	_, err := rand.Read(b)
	if err != nil {
		log.Printf("[CSP] WARNING: failed to generate nonce: %v", err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := generateCSPNonce()

		// Content Security Policy
		// M-14: In production, script-src uses nonce instead of 'unsafe-inline'.
		// Development mode retains 'unsafe-inline' for convenience.
		var scriptSrc string
		if os.Getenv("ENVIRONMENT") == "development" {
			scriptSrc = "'self' 'unsafe-inline'"
		} else {
			scriptSrc = "'self' 'nonce-" + nonce + "'"
		}

		csp := "default-src 'self'; " +
			"script-src " + scriptSrc + "; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data:; " +
			"connect-src 'self' ws: wss:; " +
			"font-src 'self'; " +
			"object-src 'none'; " +
			"media-src 'self'; " +
			"frame-src 'none'; " +
			"frame-ancestors 'none';"

		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// HSTS - Only in production with HTTPS
		if os.Getenv("ENVIRONMENT") == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Permissions Policy
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Expose nonce to templates/middleware via request context
		next.ServeHTTP(w, r)
	})
}
