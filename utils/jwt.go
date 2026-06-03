package utils

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the custom JWT payload fields alongside standard registered claims.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed HS256 JWT carrying the user's ID, email, and role.
// It reads JWT_SECRET and JWT_EXPIRY_HOURS from environment variables.
// If JWT_EXPIRY_HOURS is missing or not parseable, it defaults to 24 hours.
func GenerateToken(userID, email, role string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable is not set")
	}

	expiryHours := 24
	if raw := os.Getenv("JWT_EXPIRY_HOURS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			expiryHours = parsed
		}
	}

	return GenerateTokenWithExpiry(userID, email, role, secret, time.Duration(expiryHours)*time.Hour)
}

// GenerateTokenWithExpiry creates a signed HS256 JWT with an explicit secret
// and expiry duration. This is used by the TokenService to generate short-lived
// access tokens (15 min) while keeping the legacy GenerateToken for backward compatibility.
func GenerateTokenWithExpiry(userID, email, role, secret string, expiry time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret must not be empty")
	}

	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken parses the given token string, verifies the HMAC signing method,
// and returns the embedded claims. Returns a descriptive error on any failure.
func ValidateToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET environment variable is not set")
	}

	return ValidateTokenWithSecret(tokenString, secret)
}

// ValidateTokenWithSecret parses and validates a token using an explicit secret.
// Used by the TokenService for consistent secret management.
func ValidateTokenWithSecret(tokenString, secret string) (*Claims, error) {
	if secret == "" {
		return nil, errors.New("JWT secret must not be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
