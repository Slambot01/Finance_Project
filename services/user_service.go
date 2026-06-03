package services

import (
	apperrors "finance-dashboard/errors"
	"finance-dashboard/models"

	"gorm.io/gorm"
)

// UserService encapsulates user management business logic.
type UserService struct {
	DB           *gorm.DB
	AuditService *AuditService // optional — if nil, audit logging is skipped
}

// GetAllUsers returns every user ordered by newest first.
func (s *UserService) GetAllUsers() ([]models.User, error) {
	var users []models.User
	if err := s.DB.Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, apperrors.Internal("failed to retrieve users", err)
	}
	return users, nil
}

// GetUserByID looks up a single user by their UUID string.
func (s *UserService) GetUserByID(id string) (*models.User, error) {
	var user models.User
	if err := s.DB.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.NotFound("user", id)
		}
		return nil, apperrors.Internal("failed to retrieve user", err)
	}
	return &user, nil
}

// UpdateUser applies a partial update to the user identified by id.
// Password updates are explicitly excluded — use a dedicated flow instead.
func (s *UserService) UpdateUser(id string, updates map[string]interface{}) (*models.User, error) {
	// Fetch the existing user first.
	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// Never allow password changes through generic updates.
	delete(updates, "password")
	delete(updates, "Password")

	// Validate role if it's being changed.
	if roleVal, ok := updates["role"]; ok {
		roleStr, valid := roleVal.(string)
		if !valid {
			return nil, apperrors.Validation("role must be a string")
		}
		if _, permitted := validRoles[models.RoleType(roleStr)]; !permitted {
			return nil, apperrors.Validation("invalid role: must be one of viewer, analyst, admin")
		}
	}

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(user).Updates(updates).Error; err != nil {
			return apperrors.Internal("failed to update user", err)
		}

		// Emit audit event within the same transaction.
		if s.AuditService != nil {
			event := BuildAuditEvent("user", user.ID.String(), models.AuditUpdate, id, "", "", updates)
			if err := s.AuditService.LogEvent(tx, event); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// Re-fetch to return fresh data (Updates does not refresh all struct fields).
	if err := s.DB.Where("id = ?", id).First(user).Error; err != nil {
		return nil, apperrors.Internal("failed to retrieve updated user", err)
	}

	return user, nil
}

// DeleteUser permanently removes a user by UUID string (hard delete).
func (s *UserService) DeleteUser(id string) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Where("id = ?", id).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.NotFound("user", id)
			}
			return apperrors.Internal("failed to retrieve user", err)
		}

		if err := tx.Delete(&user).Error; err != nil {
			return apperrors.Internal("failed to delete user", err)
		}

		// Emit audit event within the same transaction.
		if s.AuditService != nil {
			event := BuildAuditEvent("user", user.ID.String(), models.AuditDelete, id, "", "", nil)
			if err := s.AuditService.LogEvent(tx, event); err != nil {
				return err
			}
		}

		return nil
	})
}
