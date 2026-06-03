package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	apperrors "finance-dashboard/errors"
	"finance-dashboard/models"
	"finance-dashboard/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TokenService manages the full token lifecycle: issuance of short-lived
// access tokens paired with long-lived refresh tokens, automatic rotation
// on refresh, and family-based revocation for replay attack detection.
type TokenService struct {
	DB                     *gorm.DB
	JWTSecret              string
	AccessTokenExpiryMins  int
	RefreshTokenExpiryDays int
}

// IssueTokenPair generates a short-lived access token (15 min default) and a
// refresh token (7 day default), stores the refresh token hash in the database,
// and returns both tokens as raw strings.
func (s *TokenService) IssueTokenPair(user *models.User) (accessToken, refreshToken string, err error) {
	// Generate access token.
	expiryMins := s.AccessTokenExpiryMins
	if expiryMins <= 0 {
		expiryMins = 15
	}

	accessToken, err = utils.GenerateToken(
		user.ID.String(), user.Email, string(user.Role),
		s.JWTSecret, time.Duration(expiryMins)*time.Minute,
	)
	if err != nil {
		return "", "", apperrors.Internal("failed to generate access token", err)
	}

	// Generate refresh token (cryptographically random).
	refreshToken, err = generateSecureToken()
	if err != nil {
		return "", "", apperrors.Internal("failed to generate refresh token", err)
	}

	// Store hashed refresh token in database.
	familyID := uuid.New()
	expiryDays := s.RefreshTokenExpiryDays
	if expiryDays <= 0 {
		expiryDays = 7
	}

	refreshTokenRecord := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour),
	}

	if err := s.DB.Create(&refreshTokenRecord).Error; err != nil {
		return "", "", apperrors.Internal("failed to store refresh token", err)
	}

	return accessToken, refreshToken, nil
}

// RefreshTokens validates the provided refresh token, revokes it, issues a new
// token pair, and returns them. If a revoked token is presented, all tokens
// in the family are revoked (replay attack detection).
func (s *TokenService) RefreshTokens(rawRefreshToken string) (accessToken, newRefreshToken string, err error) {
	tokenHash := hashToken(rawRefreshToken)

	var existingToken models.RefreshToken
	if err := s.DB.Where("token_hash = ?", tokenHash).First(&existingToken).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", "", apperrors.Unauthorized("invalid refresh token")
		}
		return "", "", apperrors.Internal("failed to look up refresh token", err)
	}

	// If the token was already revoked, this is a replay attack.
	// Revoke ALL tokens in this family as a security measure.
	if existingToken.IsRevoked() {
		_ = s.RevokeTokenFamily(existingToken.FamilyID)
		return "", "", apperrors.Unauthorized("refresh token reuse detected — all sessions revoked for security")
	}

	if existingToken.IsExpired() {
		return "", "", apperrors.Unauthorized("refresh token has expired")
	}

	// Look up the user.
	var user models.User
	if err := s.DB.Where("id = ?", existingToken.UserID).First(&user).Error; err != nil {
		return "", "", apperrors.Internal("failed to look up user for token refresh", err)
	}

	if !user.IsActive {
		return "", "", apperrors.Unauthorized("account is deactivated")
	}

	// Issue new token pair.
	accessToken, newRefreshToken, err = s.IssueTokenPair(&user)
	if err != nil {
		return "", "", err
	}

	// Revoke the old token and link it to the new one.
	now := time.Now()
	newTokenHash := hashToken(newRefreshToken)

	var newTokenRecord models.RefreshToken
	if err := s.DB.Where("token_hash = ?", newTokenHash).First(&newTokenRecord).Error; err != nil {
		return "", "", apperrors.Internal("failed to find new refresh token", err)
	}

	// Update the new token to inherit the family from the old token.
	s.DB.Model(&newTokenRecord).Update("family_id", existingToken.FamilyID)

	// Mark old token as revoked and replaced.
	s.DB.Model(&existingToken).Updates(map[string]interface{}{
		"revoked_at":  &now,
		"replaced_by": &newTokenRecord.ID,
	})

	return accessToken, newRefreshToken, nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user (logout from all devices).
func (s *TokenService) RevokeAllUserTokens(userID uuid.UUID) error {
	now := time.Now()
	result := s.DB.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", &now)

	if result.Error != nil {
		return apperrors.Internal("failed to revoke user tokens", result.Error)
	}

	return nil
}

// RevokeTokenFamily revokes all tokens in a given family. Called when a
// revoked token is presented, indicating a potential replay attack.
func (s *TokenService) RevokeTokenFamily(familyID uuid.UUID) error {
	now := time.Now()
	result := s.DB.Model(&models.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", &now)

	if result.Error != nil {
		return apperrors.Internal("failed to revoke token family", result.Error)
	}

	return nil
}

// hashToken returns the SHA-256 hex digest of a token string.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// generateSecureToken returns a cryptographically secure random 32-byte
// hex-encoded token string.
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
