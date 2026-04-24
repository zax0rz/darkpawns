package auth

import (
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

// Claims holds the custom JWT payload for Dark Pawns: player identity, agent mode, and optional agent key ID.
type Claims struct {
	PlayerName string `json:"player_name"`
	IsAgent    bool   `json:"is_agent"`
	AgentKeyID int64  `json:"agent_key_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a signed HS256 JWT valid for 24 hours. The JWT_SECRET environment variable must be set.
func GenerateJWT(playerName string, isAgent bool, agentKeyID int64) (string, error) {
	// JWT secret is REQUIRED — no fallback
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable not set")
	}
	if len(secret) < 32 {
		return "", errors.New("JWT_SECRET must be at least 32 characters")
	}

	// Set token expiration
	expirationTime := time.Now().Add(24 * time.Hour) // 24-hour tokens
	
	claims := &Claims{
		PlayerName: playerName,
		IsAgent:    isAgent,
		AgentKeyID: agentKeyID,
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
