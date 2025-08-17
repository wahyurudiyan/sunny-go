package templates

// Mock implementations for testing

// Mock user repository template
const MockUserRepositoryTemplate = `package mocks

import (
	"context"

	"{{.ModulePath}}/internal/models"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of repository.UserRepositoryInterface
type MockUserRepository struct {
	mock.Mock
}

// Create mocks the Create method
func (m *MockUserRepository) Create(ctx context.Context, user *models.CreateUserRequest) (*models.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// GetByID mocks the GetByID method
func (m *MockUserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// GetByEmail mocks the GetByEmail method
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// GetByUsername mocks the GetByUsername method
func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// Update mocks the Update method
func (m *MockUserRepository) Update(ctx context.Context, user *models.User) (*models.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// Delete mocks the Delete method
func (m *MockUserRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// List mocks the List method
func (m *MockUserRepository) List(ctx context.Context, filter *models.UserFilter) ([]*models.User, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

// Count mocks the Count method
func (m *MockUserRepository) Count(ctx context.Context, filter *models.UserFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}
`

// Mock user service template
const MockUserServiceTemplate = `package mocks

import (
	"{{.ModulePath}}/internal/models"

	"github.com/stretchr/testify/mock"
)

// MockUserService is a mock implementation of service.UserServiceInterface
type MockUserService struct {
	mock.Mock
}

// CreateUser mocks the CreateUser method
func (m *MockUserService) CreateUser(req *models.CreateUserRequest) (*models.User, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// GetUser mocks the GetUser method
func (m *MockUserService) GetUser(id int64) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// GetUserByEmail mocks the GetUserByEmail method
func (m *MockUserService) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// GetUserByUsername mocks the GetUserByUsername method
func (m *MockUserService) GetUserByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// UpdateUser mocks the UpdateUser method
func (m *MockUserService) UpdateUser(id int64, req *models.UpdateUserRequest) (*models.User, error) {
	args := m.Called(id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// DeleteUser mocks the DeleteUser method
func (m *MockUserService) DeleteUser(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

// ListUsers mocks the ListUsers method
func (m *MockUserService) ListUsers(filter *models.UserFilter) ([]*models.User, error) {
	args := m.Called(filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

// CountUsers mocks the CountUsers method
func (m *MockUserService) CountUsers(filter *models.UserFilter) (int64, error) {
	args := m.Called(filter)
	return args.Get(0).(int64), args.Error(1)
}
`

// Mock logger template
const MockLoggerTemplate = `package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of logger.Logger
type MockLogger struct {
	mock.Mock
}

// Info mocks the Info method
func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// Error mocks the Error method
func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// Debug mocks the Debug method
func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// Warn mocks the Warn method
func (m *MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// Fatal mocks the Fatal method
func (m *MockLogger) Fatal(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// With mocks the With method
func (m *MockLogger) With(keysAndValues ...interface{}) *MockLogger {
	m.Called(keysAndValues...)
	return m
}
`
