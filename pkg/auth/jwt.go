package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken indicates the JWT could not be parsed or has an invalid signature.
var (
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken indicates the JWT has passed its expiration time.
	ErrExpiredToken = errors.New("token expired")
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const claimsContextKey contextKey = "jwt_claims"

// SetClaimsOnContext stores validated JWT claims on the request context.
// Use this in auth middleware so downstream handlers can retrieve claims.
func SetClaimsOnContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// GetClaimsFromContext retrieves validated JWT claims from the request context.
// Returns nil, false if no claims are present.
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	v := ctx.Value(claimsContextKey)
	if v == nil {
		return nil, false
	}
	claims, ok := v.(*Claims)
	return claims, ok
}

// Claims holds the custom JWT payload for Dark Pawns: player identity, agent mode, and optional agent key ID.
type Claims struct {
	PlayerName string `json:"player_name"`
	IsAgent    bool   `json:"is_agent"`
	AgentKeyID int64  `json:"agent_key_id,omitempty"`
	Role       string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// HasRole returns true if the claim's role meets or exceeds the required role in the hierarchy.
// Hierarchy: player(0) < research(1) < builder(2) < admin(3).
// Returns false if either role is unknown (defense-in-depth against typos).
func (c *Claims) HasRole(required string) bool {
	hierarchy := map[string]int{"player": 0, "research": 1, "builder": 2, "admin": 3}
	reqLevel, reqOK := hierarchy[required]
	roleLevel, roleOK := hierarchy[c.Role]
	if !reqOK || !roleOK {
		return false
	}
	return roleLevel >= reqLevel
}

// GenerateJWT creates a signed HS256 JWT valid for 24 hours. The JWT_SECRET environment variable must be set.
// role is the optional RBAC role ("player", "research", "builder", "admin"); empty string defaults to "player".
func GenerateJWT(playerName string, isAgent bool, agentKeyID int64, role string) (string, error) {
	// JWT secret is REQUIRED — no fallback
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable not set")
	}
	if len(secret) < 32 {
		return "", errors.New("JWT_SECRET must be at least 32 characters")
	}

	// Default role to "player" if empty
	if role == "" {
		role = "player"
	}

	// Set token expiration
	expirationTime := time.Now().Add(24 * time.Hour) // 24-hour tokens
	
	claims := &Claims{
		PlayerName: playerName,
		IsAgent:    isAgent,
		AgentKeyID: agentKeyID,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "darkpawns",
			Subject:   playerName,
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateJWT parses and validates a JWT string, returning the embedded Claims on success.
func ValidateJWT(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET environment variable not set")
	}
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 characters")
	}
	
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, ErrInvalidToken
}
