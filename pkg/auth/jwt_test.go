package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func setTestJWTSecret(t *testing.T) {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-characters-long")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })
}

func TestValidateJWT_AcceptsHS256(t *testing.T) {
	setTestJWTSecret(t)

	token, err := GenerateJWT("Hero", false, 0, "player")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT failed for HS256 token: %v", err)
	}
	if claims.PlayerName != "Hero" {
		t.Errorf("PlayerName = %q, want %q", claims.PlayerName, "Hero")
	}
}

func TestValidateJWT_RejectsHS384(t *testing.T) {
	setTestJWTSecret(t)

	secret := os.Getenv("JWT_SECRET")
	claims := &Claims{
		PlayerName: "Hero",
		Role:       "player",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "darkpawns",
			Subject:   "Hero",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign HS384 token: %v", err)
	}

	if _, err := ValidateJWT(token); err == nil {
		t.Fatal("ValidateJWT accepted an HS384 token")
	}
}

func TestValidateJWT_RejectsHS512(t *testing.T) {
	setTestJWTSecret(t)

	secret := os.Getenv("JWT_SECRET")
	claims := &Claims{
		PlayerName: "Hero",
		Role:       "player",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "darkpawns",
			Subject:   "Hero",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign HS512 token: %v", err)
	}

	if _, err := ValidateJWT(token); err == nil {
		t.Fatal("ValidateJWT accepted an HS512 token")
	}
}

// TestValidateJWT_RejectsWrongIssuer verifies that a token signed with the
// correct HS256 secret but a wrong/empty issuer is rejected. GenerateJWT sets
// Issuer = "darkpawns"; ValidateJWT must enforce it so a secret reused across
// services can't accept tokens issued elsewhere (DP-795).
func TestValidateJWT_RejectsWrongIssuer(t *testing.T) {
	setTestJWTSecret(t)
	secret := os.Getenv("JWT_SECRET")

	cases := []struct {
		name   string
		issuer string
	}{
		{"empty", ""},
		{"wrong service", "other-service"},
		{"close typo", "DarkPawns"}, // case-sensitive
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims := &Claims{
				PlayerName: "Hero",
				Role:       "player",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					Issuer:    tc.issuer,
					Subject:   "Hero",
				},
			}
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
			if err != nil {
				t.Fatalf("sign token: %v", err)
			}

			if _, err := ValidateJWT(token); err == nil {
				t.Fatalf("ValidateJWT accepted token with issuer %q (want rejection)", tc.issuer)
			}
		})
	}
}

// TestValidateJWT_AcceptsCorrectIssuer verifies the happy path still works
// after WithIssuer is added to ValidateJWT.
func TestValidateJWT_AcceptsCorrectIssuer(t *testing.T) {
	setTestJWTSecret(t)

	token, err := GenerateJWT("Hero", false, 0, "player")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT failed for token with issuer %q: %v", JWTIssuer, err)
	}
	if claims.Issuer != JWTIssuer {
		t.Errorf("claims.Issuer = %q, want %q", claims.Issuer, JWTIssuer)
	}
}

func TestValidateJWTSecret(t *testing.T) {
	cases := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"unset", "", true},
		{"too short", "short", true},
		{"just under minimum", "0123456789012345678901234567890", true}, // 30 chars
		{"exactly minimum", "01234567890123456789012345678901", false},  // 32 chars
		{"long", "test-secret-that-is-at-least-32-characters-long", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.secret == "" {
				os.Unsetenv("JWT_SECRET")
			} else {
				os.Setenv("JWT_SECRET", tc.secret)
			}
			t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

			err := ValidateJWTSecret()
			if tc.wantErr && err == nil {
				t.Errorf("ValidateJWTSecret() = nil, want error (secret %q)", tc.secret)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidateJWTSecret() = %v, want nil (secret %q)", err, tc.secret)
			}
		})
	}
}
