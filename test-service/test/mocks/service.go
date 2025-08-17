package mocks

import (
	"github.com/yourorg/test-service/internal/models"

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
