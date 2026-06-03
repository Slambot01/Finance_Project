package services

import (
	"context"

	apperrors "finance-dashboard/errors"
	"finance-dashboard/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService encapsulates authentication and registration business logic.
type AuthService struct {
	DB             *gorm.DB
	TokenService   *TokenService
	AccountService *AccountService
	AuditService   *AuditService
}

// validRoles is the canonical set of allowed roles, shared across the services package.
var validRoles = map[models.RoleType]struct{}{
	models.RoleViewer:  {},
	models.RoleAnalyst: {},
	models.RoleAdmin:   {},
}

// Register creates a new user after validating the role, checking for
// duplicate emails, and hashing the password with bcrypt.
// Admin role cannot be self-assigned — new users are always created as viewers
// unless an admin explicitly promotes them afterwards.
// On success, default accounts (Cash, Revenue, Expenses) are auto-created and
// a user.registered audit event is emitted — all within the same transaction.
func (s *AuthService) Register(name, email, password, role string) (*models.User, error) {
	// Default to viewer if no role provided.
	if role == "" {
		role = string(models.RoleViewer)
	}

	// Prevent self-assignment of admin role during registration.
	// Admin accounts must be promoted by an existing admin through the
	// user management endpoint. This prevents privilege escalation.
	if models.RoleType(role) == models.RoleAdmin {
		return nil, apperrors.Forbidden("admin role cannot be self-assigned during registration; register as viewer or analyst and request promotion from an admin")
	}

	if _, ok := validRoles[models.RoleType(role)]; !ok {
		return nil, apperrors.Validation("invalid role: must be one of viewer, analyst, admin")
	}

	var user models.User

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		// Check for existing email inside the transaction for atomicity.
		var count int64
		if err := tx.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
			return apperrors.Internal("failed to check existing email", err)
		}
		if count > 0 {
			return apperrors.Conflict("email already registered")
		}

		// Hash the password.
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return apperrors.Internal("failed to hash password", err)
		}

		user = models.User{
			Name:     name,
			Email:    email,
			Password: string(hashedPassword),
			Role:     models.RoleType(role),
			IsActive: true,
		}

		if err := tx.Create(&user).Error; err != nil {
			return apperrors.Internal("failed to create user", err)
		}

		// Auto-create default ledger accounts (Cash, Revenue, Expenses) for the new user.
		if s.AccountService != nil {
			if err := s.AccountService.CreateDefaultAccounts(tx, user.ID); err != nil {
				return err
			}
		}

		// Emit audit event inside the same transaction (atomicity guaranteed).
		if s.AuditService != nil {
			event := BuildAuditEvent("user", user.ID.String(), models.AuditCreate, "system", "", "", map[string]string{"role": role})
			if err := s.AuditService.LogEvent(tx, event); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Login authenticates a user by email and password. On success it issues a
// token pair (short-lived access token + long-lived refresh token), emits a
// user.login audit event, and returns both tokens with the user record.
// requestID and ipAddress are used for audit logging only; they may be empty.
func (s *AuthService) Login(_ context.Context, email, password, requestID, ipAddress string) (accessToken, refreshToken string, user *models.User, err error) {
	var u models.User
	if dbErr := s.DB.Where("email = ?", email).First(&u).Error; dbErr != nil {
		if dbErr == gorm.ErrRecordNotFound {
			return "", "", nil, apperrors.Unauthorized("invalid email or password")
		}
		return "", "", nil, apperrors.Internal("failed to query user", dbErr)
	}

	if !u.IsActive {
		return "", "", nil, apperrors.Unauthorized("account is deactivated")
	}

	if bcryptErr := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); bcryptErr != nil {
		return "", "", nil, apperrors.Unauthorized("invalid email or password")
	}

	// Issue a full token pair (access + refresh) via the TokenService.
	if s.TokenService != nil {
		accessToken, refreshToken, err = s.TokenService.IssueTokenPair(&u)
		if err != nil {
			return "", "", nil, err
		}
	} else {
		return "", "", nil, apperrors.Internal("token service is not configured", nil)
	}

	// Emit a login audit event (best-effort — don't fail login on audit error).
	if s.AuditService != nil {
		event := BuildAuditEvent("user", u.ID.String(), models.AuditLogin, u.ID.String(), requestID, ipAddress, nil)
		_ = s.DB.Transaction(func(tx *gorm.DB) error {
			return s.AuditService.LogEvent(tx, event)
		})
	}


	return accessToken, refreshToken, &u, nil
}
