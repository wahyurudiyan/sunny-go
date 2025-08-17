package fixtures

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/yourorg/test-service/internal/models"
	"github.com/yourorg/test-service/pkg/database"
	"github.com/yourorg/test-service/pkg/logger"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestHelper provides utilities for integration testing
type TestHelper struct {
	DB     *gorm.DB
	Logger *logger.Logger
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	// Use test database or in-memory SQLite
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = "sqlite://file::memory:?cache=shared"
	}

	db, err := database.Connect(testDBURL)
	require.NoError(t, err)

	// Auto-migrate tables
	err = db.AutoMigrate(&models.User{})
	require.NoError(t, err)

	logger := logger.New("test", "info")

	return &TestHelper{
		DB:     db,
		Logger: logger,
	}
}

// SeedUsers seeds the database with test users
func (h *TestHelper) SeedUsers(t *testing.T, count int) []*models.User {
	users := make([]*models.User, count)
	
	for i := 0; i < count; i++ {
		user := &models.User{
			Username:  fmt.Sprintf("testuser%d", i+1),
			Email:     fmt.Sprintf("test%d@example.com", i+1),
			FirstName: fmt.Sprintf("Test%d", i+1),
			LastName:  fmt.Sprintf("User%d", i+1),
		}
		
		result := h.DB.Create(user)
		require.NoError(t, result.Error)
		users[i] = user
	}
	
	return users
}

// CleanDatabase cleans the database by deleting all records
func (h *TestHelper) CleanDatabase(t *testing.T) {
	result := h.DB.Exec("DELETE FROM users")
	require.NoError(t, result.Error)
}

// CreateUser creates a user in the database
func (h *TestHelper) CreateUser(t *testing.T, username, email, firstName, lastName string) *models.User {
	user := &models.User{
		Username:  username,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}
	
	result := h.DB.Create(user)
	require.NoError(t, result.Error)
	
	return user
}

// GetUserByID gets a user by ID from the database
func (h *TestHelper) GetUserByID(t *testing.T, id int64) *models.User {
	var user models.User
	result := h.DB.First(&user, id)
	require.NoError(t, result.Error)
	return &user
}

// GetUserByEmail gets a user by email from the database
func (h *TestHelper) GetUserByEmail(t *testing.T, email string) *models.User {
	var user models.User
	result := h.DB.Where("email = ?", email).First(&user)
	require.NoError(t, result.Error)
	return &user
}

// GetUserByUsername gets a user by username from the database
func (h *TestHelper) GetUserByUsername(t *testing.T, username string) *models.User {
	var user models.User
	result := h.DB.Where("username = ?", username).First(&user)
	require.NoError(t, result.Error)
	return &user
}

// CountUsers counts the number of users in the database
func (h *TestHelper) CountUsers(t *testing.T) int64 {
	var count int64
	result := h.DB.Model(&models.User{}).Count(&count)
	require.NoError(t, result.Error)
	return count
}

// UserExists checks if a user exists by ID
func (h *TestHelper) UserExists(t *testing.T, id int64) bool {
	var count int64
	result := h.DB.Model(&models.User{}).Where("id = ?", id).Count(&count)
	require.NoError(t, result.Error)
	return count > 0
}

// UserExistsByEmail checks if a user exists by email
func (h *TestHelper) UserExistsByEmail(t *testing.T, email string) bool {
	var count int64
	result := h.DB.Model(&models.User{}).Where("email = ?", email).Count(&count)
	require.NoError(t, result.Error)
	return count > 0
}

// UserExistsByUsername checks if a user exists by username
func (h *TestHelper) UserExistsByUsername(t *testing.T, username string) bool {
	var count int64
	result := h.DB.Model(&models.User{}).Where("username = ?", username).Count(&count)
	require.NoError(t, result.Error)
	return count > 0
}

// TruncateTable truncates a specific table
func (h *TestHelper) TruncateTable(t *testing.T, tableName string) {
	result := h.DB.Exec(fmt.Sprintf("DELETE FROM %s", tableName))
	require.NoError(t, result.Error)
}

// Close closes the database connection
func (h *TestHelper) Close() error {
	sqlDB, err := h.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// WithContext returns a new context for testing
func (h *TestHelper) WithContext() context.Context {
	return context.Background()
}

// WithTestTransaction runs a function within a database transaction that is rolled back
func (h *TestHelper) WithTestTransaction(t *testing.T, fn func(*gorm.DB)) {
	tx := h.DB.Begin()
	require.NoError(t, tx.Error)
	
	defer func() {
		tx.Rollback()
	}()
	
	fn(tx)
}
