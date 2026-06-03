package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// JWTIssuer identifies tokens issued by this service.
	JWTIssuer = "immutablecore"
	// JWTAudience identifies the intended consumer of the token.
	JWTAudience = "immutablecore-api"
)

// Claims holds the custom JWT payload fields alongside standard registered claims.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed HS256 JWT carrying the user's ID, email, and role.
// The secret and expiry must be provided explicitly — no environment variable reads.
func GenerateToken(userID, email, role, secret string, expiry time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret must not be empty")
	}

	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    JWTIssuer,
			Audience:  jwt.ClaimStrings{JWTAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken parses the given token string, verifies the HMAC signing method,
// issuer, audience, and returns the embedded claims. Returns a descriptive error on any failure.
func ValidateToken(tokenString, secret string) (*Claims, error) {
	if secret == "" {
		return nil, errors.New("JWT secret must not be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	},
		jwt.WithIssuer(JWTIssuer),
		jwt.WithAudience(JWTAudience),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
