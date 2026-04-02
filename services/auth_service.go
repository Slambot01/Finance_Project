package services

import (
	"errors"

	"finance-dashboard/models"
	"finance-dashboard/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService encapsulates authentication and registration business logic.
type AuthService struct {
	DB *gorm.DB
}

// validRoles is the canonical set of allowed roles.
var validRoles = map[models.RoleType]struct{}{
	models.RoleViewer:  {},
	models.RoleAnalyst: {},
	models.RoleAdmin:   {},
}

// Register creates a new user after validating the role, checking for
// duplicate emails, and hashing the password with bcrypt.
func (s *AuthService) Register(name, email, password, role string) (*models.User, error) {
	// Default to viewer if no role provided.
	if role == "" {
		role = string(models.RoleViewer)
	}

	if _, ok := validRoles[models.RoleType(role)]; !ok {
		return nil, errors.New("invalid role: must be one of viewer, analyst, admin")
	}

	// Check for existing email.
	var count int64
	if err := s.DB.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return nil, errors.New("failed to check existing email")
	}
	if count > 0 {
		return nil, errors.New("email already registered")
	}

	// Hash the password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	user := models.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
		Role:     models.RoleType(role),
		IsActive: true,
	}

	if err := s.DB.Create(&user).Error; err != nil {
		return nil, errors.New("failed to create user")
	}

	return &user, nil
}

// Login authenticates a user by email and password, returning a signed JWT
// and the user record on success.
func (s *AuthService) Login(email, password string) (string, *models.User, error) {
	var user models.User
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("invalid email or password")
		}
		return "", nil, errors.New("failed to query user")
	}

	if !user.IsActive {
		return "", nil, errors.New("account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	token, err := utils.GenerateToken(user.ID.String(), user.Email, string(user.Role))
	if err != nil {
		return "", nil, errors.New("failed to generate authentication token")
	}

	return token, &user, nil
}
