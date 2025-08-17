package service

import (
	"testing"

	"github.com/yourorg/test-service/internal/models"
	"github.com/yourorg/test-service/test/fixtures"
	"github.com/yourorg/test-service/test/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_CreateUser(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	mockLogger := new(mocks.MockLogger)
	service := NewUserService(mockRepo, mockLogger)

	t.Run("successful creation", func(t *testing.T) {
		req := fixtures.NewCreateUserRequest()

		expectedUser := fixtures.NewUser()
		expectedUser.ID = 1
		expectedUser.Username = req.Username
		expectedUser.Email = req.Email
		expectedUser.FirstName = req.FirstName
		expectedUser.LastName = req.LastName

		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(user *models.CreateUserRequest) bool {
			return user.Username == req.Username && user.Email == req.Email
		})).Return(expectedUser, nil)

		mockLogger.On("Info", "Creating user", mock.Anything).Return()

		user, err := service.CreateUser(req)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, req.Username, user.Username)
		mockRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		req := fixtures.NewCreateUserRequest()

		mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		mockLogger.On("Info", "Creating user", mock.Anything).Return()
		mockLogger.On("Error", "Failed to create user", mock.Anything).Return()

		user, err := service.CreateUser(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})
}

func TestUserService_GetUser(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	mockLogger := new(mocks.MockLogger)
	service := NewUserService(mockRepo, mockLogger)

	t.Run("user found", func(t *testing.T) {
		expectedUser := fixtures.NewUser()
		expectedUser.ID = 1

		mockRepo.On("GetByID", mock.Anything, int64(1)).Return(expectedUser, nil)
		mockLogger.On("Debug", "Getting user", mock.Anything).Return()

		user, err := service.GetUser(1)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		mockRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, int64(1)).Return(nil, assert.AnError)
		mockLogger.On("Debug", "Getting user", mock.Anything).Return()
		mockLogger.On("Error", "Failed to get user", mock.Anything).Return()

		user, err := service.GetUser(1)

		assert.Error(t, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})
}
