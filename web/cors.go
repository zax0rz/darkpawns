package web

import (
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
	if os.Getenv("ENVIRONMENT") == "development" {
		return []string{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:3000"}
	}
	
	// Production defaults
	return []string{
		"https://darkpawns.example.com",
		"https://game.darkpawns.example.com",
	}
}

func isOriginAllowed(origin string, allowed []string) bool {
	// Development mode allows all origins
	if os.Getenv("ENVIRONMENT") == "development" {
		return true
	}
	
	for _, allowedOrigin := range allowed {
		if origin == allowedOrigin {
			return true
		}
		// Support wildcard subdomains
		if strings.HasPrefix(allowedOrigin, "*.") {
			domain := strings.TrimPrefix(allowedOrigin, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	
	return false
}