package templates

// Test fixtures and helpers

// User fixture template
const UserFixtureTemplate = `package fixtures

import (
	"time"

	"{{.ModulePath}}/internal/models"
)

// NewUser creates a new user fixture for testing
func NewUser() *models.User {
	now := time.Now()
	return &models.User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewUserWithID creates a new user fixture with a specific ID
func NewUserWithID(id int64) *models.User {
	user := NewUser()
	user.ID = id
	return user
}

// NewUserWithEmail creates a new user fixture with a specific email
func NewUserWithEmail(email string) *models.User {
	user := NewUser()
	user.Email = email
	return user
}

// NewUserWithUsername creates a new user fixture with a specific username
func NewUserWithUsername(username string) *models.User {
	user := NewUser()
	user.Username = username
	return user
}

// NewUsers creates multiple user fixtures for testing
func NewUsers(count int) []*models.User {
	users := make([]*models.User, count)
	for i := 0; i < count; i++ {
		user := NewUser()
		user.ID = int64(i + 1)
		user.Username = fmt.Sprintf("testuser%d", i+1)
		user.Email = fmt.Sprintf("test%d@example.com", i+1)
		user.FirstName = fmt.Sprintf("Test%d", i+1)
		user.LastName = fmt.Sprintf("User%d", i+1)
		users[i] = user
	}
	return users
}

// NewCreateUserRequest creates a create user request fixture
func NewCreateUserRequest() *models.CreateUserRequest {
	return &models.CreateUserRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
	}
}

// NewCreateUserRequestWithEmail creates a create user request with specific email
func NewCreateUserRequestWithEmail(email string) *models.CreateUserRequest {
	req := NewCreateUserRequest()
	req.Email = email
	return req
}

// NewCreateUserRequestWithUsername creates a create user request with specific username
func NewCreateUserRequestWithUsername(username string) *models.CreateUserRequest {
	req := NewCreateUserRequest()
	req.Username = username
	return req
}

// NewUpdateUserRequest creates an update user request fixture
func NewUpdateUserRequest() *models.UpdateUserRequest {
	return &models.UpdateUserRequest{
		FirstName: stringPtr("UpdatedFirst"),
		LastName:  stringPtr("UpdatedLast"),
	}
}

// NewUpdateUserRequestWithFirstName creates an update user request with specific first name
func NewUpdateUserRequestWithFirstName(firstName string) *models.UpdateUserRequest {
	return &models.UpdateUserRequest{
		FirstName: stringPtr(firstName),
	}
}

// NewUpdateUserRequestWithLastName creates an update user request with specific last name
func NewUpdateUserRequestWithLastName(lastName string) *models.UpdateUserRequest {
	return &models.UpdateUserRequest{
		LastName: stringPtr(lastName),
	}
}

// NewUserFilter creates a user filter fixture
func NewUserFilter() *models.UserFilter {
	return &models.UserFilter{
		Limit:  10,
		Offset: 0,
	}
}

// NewUserFilterWithEmail creates a user filter with email
func NewUserFilterWithEmail(email string) *models.UserFilter {
	filter := NewUserFilter()
	filter.Email = &email
	return filter
}

// NewUserFilterWithUsername creates a user filter with username
func NewUserFilterWithUsername(username string) *models.UserFilter {
	filter := NewUserFilter()
	filter.Username = &username
	return filter
}

// NewUserFilterWithLimit creates a user filter with specific limit
func NewUserFilterWithLimit(limit int) *models.UserFilter {
	filter := NewUserFilter()
	filter.Limit = limit
	return filter
}

// NewUserFilterWithOffset creates a user filter with specific offset
func NewUserFilterWithOffset(offset int) *models.UserFilter {
	filter := NewUserFilter()
	filter.Offset = offset
	return filter
}

// InvalidCreateUserRequests returns a slice of invalid create user requests for testing
func InvalidCreateUserRequests() []*models.CreateUserRequest {
	return []*models.CreateUserRequest{
		{
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
			// Missing username
		},
		{
			Username:  "testuser",
			FirstName: "Test",
			LastName:  "User",
			// Missing email
		},
		{
			Username:  "testuser",
			Email:     "invalid-email",
			FirstName: "Test",
			LastName:  "User",
		},
		{
			Username:  "",
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
		},
		{
			Username:  "testuser",
			Email:     "",
			FirstName: "Test",
			LastName:  "User",
		},
	}
}

// InvalidUpdateUserRequests returns a slice of invalid update user requests for testing
func InvalidUpdateUserRequests() []*models.UpdateUserRequest {
	return []*models.UpdateUserRequest{
		{
			FirstName: stringPtr(""),
		},
		{
			LastName: stringPtr(""),
		},
	}
}

// Helper functions

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

// intPtr returns a pointer to the given int
func intPtr(i int) *int {
	return &i
}

// int64Ptr returns a pointer to the given int64
func int64Ptr(i int64) *int64 {
	return &i
}

// timePtr returns a pointer to the given time
func timePtr(t time.Time) *time.Time {
	return &t
}
`

// Test helper template
const TestHelpersTemplate = `package fixtures

import (
	"context"
	"fmt"
	"os"
	"testing"

	"{{.ModulePath}}/internal/models"
	"{{.ModulePath}}/pkg/database"
	"{{.ModulePath}}/pkg/logger"

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
`
