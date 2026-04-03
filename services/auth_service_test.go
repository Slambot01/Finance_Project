package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Register(t *testing.T) {
	service := &AuthService{DB: testDB}

	t.Run("Register success with valid fields", func(t *testing.T) {
		cleanupTables(testDB)

		user, err := service.Register("Alice", "alice@example.com", "password123", "admin")

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "Alice", user.Name)
		assert.Equal(t, "alice@example.com", user.Email)
		assert.Equal(t, "admin", string(user.Role))
		assert.True(t, user.IsActive)
		// Password should be hashed, not plaintext.
		assert.NotEqual(t, "password123", user.Password)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123")))
	})

	t.Run("Register with empty role defaults to viewer", func(t *testing.T) {
		cleanupTables(testDB)

		user, err := service.Register("Bob", "bob@example.com", "password123", "")

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "viewer", string(user.Role))
	})

	t.Run("Register duplicate email returns error", func(t *testing.T) {
		cleanupTables(testDB)

		_, err := service.Register("User1", "dup@example.com", "password123", "viewer")
		assert.NoError(t, err)

		user, err := service.Register("User2", "dup@example.com", "password456", "viewer")

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "email already registered")
	})

	t.Run("Register invalid role returns error", func(t *testing.T) {
		cleanupTables(testDB)

		user, err := service.Register("Charlie", "charlie@example.com", "password123", "superadmin")

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid role")
	})
}

func TestAuthService_Login(t *testing.T) {
	service := &AuthService{DB: testDB}

	t.Run("Login success returns token and user", func(t *testing.T) {
		cleanupTables(testDB)
		_, _ = service.Register("LoginUser", "login@example.com", "correctpass", "analyst")

		token, user, err := service.Login("login@example.com", "correctpass")

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, user)
		assert.Equal(t, "login@example.com", user.Email)
		assert.Equal(t, "analyst", string(user.Role))
	})

	t.Run("Login wrong password", func(t *testing.T) {
		cleanupTables(testDB)
		_, _ = service.Register("WrongPass", "wrongpass@example.com", "correctpass", "viewer")

		token, user, err := service.Login("wrongpass@example.com", "wrongpassword")

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid email or password")
	})

	t.Run("Login non-existent email returns same error as wrong password", func(t *testing.T) {
		cleanupTables(testDB)

		token, user, err := service.Login("nobody@example.com", "anypassword")

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid email or password")
	})

	t.Run("Login deactivated account", func(t *testing.T) {
		cleanupTables(testDB)
		user, _ := service.Register("Deactivated", "deactivated@example.com", "password123", "viewer")
		// Deactivate the user directly in DB.
		testDB.Model(user).Update("is_active", false)

		token, returnedUser, err := service.Login("deactivated@example.com", "password123")

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Nil(t, returnedUser)
		assert.Contains(t, err.Error(), "deactivated")
	})
}
