package services

import (
	"errors"
	"fmt"

	"finance-dashboard/models"

	"gorm.io/gorm"
)

// UserService encapsulates user management business logic.
type UserService struct {
	DB *gorm.DB
}

// GetAllUsers returns every user ordered by newest first.
func (s *UserService) GetAllUsers() ([]models.User, error) {
	var users []models.User
	if err := s.DB.Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, errors.New("failed to retrieve users")
	}
	return users, nil
}

// GetUserByID looks up a single user by their UUID string.
func (s *UserService) GetUserByID(id string) (*models.User, error) {
	var user models.User
	if err := s.DB.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user with id %s not found", id)
		}
		return nil, errors.New("failed to retrieve user")
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
			return nil, errors.New("role must be a string")
		}
		if _, permitted := validRoles[models.RoleType(roleStr)]; !permitted {
			return nil, errors.New("invalid role: must be one of viewer, analyst, admin")
		}
	}

	if err := s.DB.Model(user).Updates(updates).Error; err != nil {
		return nil, errors.New("failed to update user")
	}

	return user, nil
}

// DeleteUser permanently removes a user by UUID string (hard delete).
func (s *UserService) DeleteUser(id string) error {
	var user models.User
	if err := s.DB.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user with id %s not found", id)
		}
		return errors.New("failed to retrieve user")
	}

	if err := s.DB.Delete(&user).Error; err != nil {
		return errors.New("failed to delete user")
	}

	return nil
}
