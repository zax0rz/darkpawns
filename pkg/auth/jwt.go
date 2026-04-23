package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type Claims struct {
	PlayerName string `json:"player_name"`
	IsAgent    bool   `json:"is_agent"`
	AgentKeyID int64  `json:"agent_key_id,omitempty"`
	jwt.RegisteredClaims
}

func GenerateJWT(playerName string, isAgent bool, agentKeyID int64) (string, error) {
	// Get JWT secret from environment
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a random secret if not set (for development only)
		if os.Getenv("ENVIRONMENT") == "development" {
			secret = generateRandomSecret()
		} else {
			return "", errors.New("JWT_SECRET environment variable not set")
		}
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

func ValidateJWT(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET environment variable not set")
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

func generateRandomSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based secret
		return fmt.Sprintf("dev-secret-%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(bytes)
}