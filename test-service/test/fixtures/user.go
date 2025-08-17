package fixtures

import (
	"time"

	"github.com/yourorg/test-service/internal/models"
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
