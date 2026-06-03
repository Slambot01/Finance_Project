package services

import (
	"testing"

	"finance-dashboard/models"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)


func TestUserService_GetAllUsers(t *testing.T) {
	service := &UserService{DB: testDB}

	t.Run("GetAllUsers returns all users ordered by created_at DESC", func(t *testing.T) {
		cleanupTables(testDB)
		createTestUser(t, "First", "first@example.com", "viewer")
		createTestUser(t, "Second", "second@example.com", "analyst")
		createTestUser(t, "Third", "third@example.com", "admin")

		users, err := service.GetAllUsers()

		assert.NoError(t, err)
		assert.Len(t, users, 3)
		// Newest first.
		assert.Equal(t, "Third", users[0].Name)
		assert.Equal(t, "Second", users[1].Name)
		assert.Equal(t, "First", users[2].Name)
	})
}

func TestUserService_GetUserByID(t *testing.T) {
	service := &UserService{DB: testDB}

	t.Run("GetUserByID with valid ID returns correct user", func(t *testing.T) {
		cleanupTables(testDB)
		created := createTestUser(t, "FindMe", "findme@example.com", "viewer")

		user, err := service.GetUserByID(created.ID.String())

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "FindMe", user.Name)
		assert.Equal(t, created.ID, user.ID)
	})

	t.Run("GetUserByID with non-existent ID returns not found", func(t *testing.T) {
		cleanupTables(testDB)

		user, err := service.GetUserByID("00000000-0000-0000-0000-000000000000")

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetUserByID with invalid UUID string returns error", func(t *testing.T) {
		cleanupTables(testDB)

		user, err := service.GetUserByID("not-a-valid-uuid")

		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	service := &UserService{DB: testDB}

	t.Run("UpdateUser change role successfully", func(t *testing.T) {
		cleanupTables(testDB)
		created := createTestUser(t, "RoleChange", "rolechange@example.com", "viewer")

		updated, err := service.UpdateUser(created.ID.String(), map[string]interface{}{
			"role": "analyst",
		})

		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, "analyst", string(updated.Role))
	})

	t.Run("UpdateUser invalid role returns error", func(t *testing.T) {
		cleanupTables(testDB)
		created := createTestUser(t, "BadRole", "badrole@example.com", "viewer")

		updated, err := service.UpdateUser(created.ID.String(), map[string]interface{}{
			"role": "superadmin",
		})

		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "invalid role")
	})

	t.Run("UpdateUser non-existent ID returns not found", func(t *testing.T) {
		cleanupTables(testDB)

		updated, err := service.UpdateUser("00000000-0000-0000-0000-000000000000", map[string]interface{}{
			"name": "Ghost",
		})

		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("UpdateUser strips password from updates", func(t *testing.T) {
		cleanupTables(testDB)
		created := createTestUser(t, "NoPassChange", "nopass@example.com", "viewer")
		originalPassword := created.Password

		updated, err := service.UpdateUser(created.ID.String(), map[string]interface{}{
			"password": "newpassword",
			"name":     "UpdatedName",
		})

		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, "UpdatedName", updated.Name)

		// Verify password was NOT changed — original bcrypt hash should still validate.
		var dbUser models.User
		testDB.Where("id = ?", created.ID).First(&dbUser)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte("password123")))
		assert.Equal(t, originalPassword, dbUser.Password)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	service := &UserService{DB: testDB}

	t.Run("DeleteUser success", func(t *testing.T) {
		cleanupTables(testDB)
		created := createTestUser(t, "DeleteMe", "deleteme@example.com", "viewer")

		err := service.DeleteUser(created.ID.String())
		assert.NoError(t, err)

		// Verify user no longer exists in DB.
		var user models.User
		result := testDB.Where("id = ?", created.ID).First(&user)
		assert.Error(t, result.Error)
	})

	t.Run("DeleteUser non-existent ID returns error", func(t *testing.T) {
		cleanupTables(testDB)

		err := service.DeleteUser("00000000-0000-0000-0000-000000000000")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
