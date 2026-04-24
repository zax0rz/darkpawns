package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/auth"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// ContextKeyPlayerName is the context key used to store the authenticated
	// player name extracted from the JWT.
	ContextKeyPlayerName contextKey = "player_name"
	// ContextKeyIsAgent is the context key used to store whether the
	// authenticated session is an agent.
	ContextKeyIsAgent contextKey = "is_agent"
)

// AuthMiddleware protects HTTP endpoints with JWT bearer token authentication.
// It extracts and validates a Bearer token from the Authorization header,
// then stores the validated claims on the request context for downstream handlers.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateJWT(token)
		if err != nil {
			http.Error(w, `{"error":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Store claims on context for downstream handlers
		ctx := context.WithValue(r.Context(), ContextKeyPlayerName, claims.PlayerName)
		ctx = context.WithValue(ctx, ContextKeyIsAgent, claims.IsAgent)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetPlayerNameFromContext retrieves the authenticated player name from a request context.
func GetPlayerNameFromContext(r *http.Request) (string, bool) {
	v := r.Context().Value(ContextKeyPlayerName)
	if v == nil {
		return "", false
	}
	name, ok := v.(string)
	return name, ok
}

// IsAgentFromContext retrieves the agent flag from a request context.
func IsAgentFromContext(r *http.Request) bool {
	v := r.Context().Value(ContextKeyIsAgent)
	if v == nil {
		return false
	}
	isAgent, ok := v.(bool)
	return ok && isAgent
}
