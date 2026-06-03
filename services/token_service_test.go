package services

import (
	"testing"

	"finance-dashboard/models"

	"github.com/stretchr/testify/assert"
)

func TestTokenService_IssueTokenPair(t *testing.T) {
	service := &TokenService{
		DB:                     testDB,
		JWTSecret:              "test-secret-key-for-unit-tests",
		AccessTokenExpiryMins:  15,
		RefreshTokenExpiryDays: 7,
	}

	t.Run("IssueTokenPair returns both tokens", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "TokenUser", "tokenuser@example.com", "viewer")

		accessToken, refreshToken, err := service.IssueTokenPair(user)

		assert.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)

		// Verify refresh token is stored in DB (hashed).
		var stored models.RefreshToken
		result := testDB.First(&stored, "user_id = ?", user.ID)
		assert.NoError(t, result.Error)
		assert.False(t, stored.IsRevoked())
		assert.False(t, stored.IsExpired())
	})
}

func TestTokenService_RefreshTokens(t *testing.T) {
	service := &TokenService{
		DB:                     testDB,
		JWTSecret:              "test-secret-key-for-unit-tests",
		AccessTokenExpiryMins:  15,
		RefreshTokenExpiryDays: 7,
	}

	t.Run("RefreshTokens rotates tokens correctly", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "RefreshUser", "refresh@example.com", "analyst")

		_, originalRefresh, err := service.IssueTokenPair(user)
		assert.NoError(t, err)

		newAccess, newRefresh, err := service.RefreshTokens(originalRefresh)

		assert.NoError(t, err)
		assert.NotEmpty(t, newAccess)
		assert.NotEmpty(t, newRefresh)
		assert.NotEqual(t, originalRefresh, newRefresh) // Different token

		// Old token should be revoked.
		var oldToken models.RefreshToken
		testDB.First(&oldToken, "token_hash = ?", hashToken(originalRefresh))
		assert.True(t, oldToken.IsRevoked())
	})

	t.Run("Using revoked token triggers family-wide revocation", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "ReplayUser", "replay@example.com", "viewer")

		_, originalRefresh, err := service.IssueTokenPair(user)
		assert.NoError(t, err)

		// Rotate once (old token is now revoked).
		_, _, err = service.RefreshTokens(originalRefresh)
		assert.NoError(t, err)

		// Try to use the old (revoked) token again — replay attack!
		_, _, err = service.RefreshTokens(originalRefresh)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reuse detected")

		// All tokens in the family should be revoked.
		var activeTokens int64
		testDB.Model(&models.RefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL", user.ID).
			Count(&activeTokens)
		assert.Equal(t, int64(0), activeTokens)
	})

	t.Run("Invalid refresh token rejected", func(t *testing.T) {
		cleanupTables(testDB)

		_, _, err := service.RefreshTokens("completely-invalid-token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token")
	})

	t.Run("Deactivated user cannot refresh tokens", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "Deactivated", "deactivated@example.com", "viewer")

		_, refreshToken, err := service.IssueTokenPair(user)
		assert.NoError(t, err)

		// Deactivate user.
		testDB.Model(user).Update("is_active", false)

		_, _, err = service.RefreshTokens(refreshToken)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deactivated")
	})
}

func TestTokenService_RevokeAllUserTokens(t *testing.T) {
	service := &TokenService{
		DB:                     testDB,
		JWTSecret:              "test-secret-key-for-unit-tests",
		AccessTokenExpiryMins:  15,
		RefreshTokenExpiryDays: 7,
	}

	t.Run("RevokeAllUserTokens revokes all active tokens", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "LogoutUser", "logout@example.com", "admin")

		// Issue multiple token pairs (simulating multiple devices).
		_, _, _ = service.IssueTokenPair(user)
		_, _, _ = service.IssueTokenPair(user)
		_, _, _ = service.IssueTokenPair(user)

		err := service.RevokeAllUserTokens(user.ID)
		assert.NoError(t, err)

		// Verify all tokens are revoked.
		var activeTokens int64
		testDB.Model(&models.RefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL", user.ID).
			Count(&activeTokens)
		assert.Equal(t, int64(0), activeTokens)
	})
}
