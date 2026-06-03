package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testSecret = "test-secret-key-for-unit-tests"

func TestGenerateToken(t *testing.T) {
	t.Run("GenerateToken_success_returns_non_empty_string", func(t *testing.T) {
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, 24*time.Hour)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("GenerateToken_with_empty_secret_returns_error", func(t *testing.T) {
		token, err := GenerateToken("user-123", "test@example.com", "admin", "", 24*time.Hour)
		assert.Error(t, err)
		assert.Empty(t, token)
	})
}

func TestValidateToken(t *testing.T) {
	t.Run("ValidateToken_with_valid_token_returns_correct_claims", func(t *testing.T) {
		token, err := GenerateToken("user-abc", "alice@example.com", "viewer", testSecret, 24*time.Hour)
		assert.NoError(t, err)

		claims, err := ValidateToken(token, testSecret)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, "user-abc", claims.UserID)
		assert.Equal(t, "alice@example.com", claims.Email)
		assert.Equal(t, "viewer", claims.Role)
	})

	t.Run("GenerateToken_then_ValidateToken_round_trip_claims_match", func(t *testing.T) {
		userID := "uuid-round-trip"
		email := "roundtrip@example.com"
		role := "analyst"

		token, err := GenerateToken(userID, email, role, testSecret, 24*time.Hour)
		assert.NoError(t, err)

		claims, err := ValidateToken(token, testSecret)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("ValidateToken_with_garbage_string_returns_error", func(t *testing.T) {
		claims, err := ValidateToken("this.is.garbage", testSecret)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("ValidateToken_with_tampered_token_returns_error", func(t *testing.T) {
		token, err := GenerateToken("user-tamper", "tamper@example.com", "admin", testSecret, 24*time.Hour)
		assert.NoError(t, err)

		// Tamper with the token by flipping a character in the signature (last segment).
		runes := []rune(token)
		lastIdx := len(runes) - 1
		if runes[lastIdx] == 'a' {
			runes[lastIdx] = 'b'
		} else {
			runes[lastIdx] = 'a'
		}
		tampered := string(runes)

		claims, err := ValidateToken(tampered, testSecret)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("ValidateToken_with_empty_string_returns_error", func(t *testing.T) {
		claims, err := ValidateToken("", testSecret)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("ValidateToken_with_wrong_secret_returns_error", func(t *testing.T) {
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, 24*time.Hour)
		assert.NoError(t, err)

		claims, err := ValidateToken(token, "wrong-secret")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}
