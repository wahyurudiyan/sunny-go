package handler

import (
	"context"
	"testing"

	"github.com/yourorg/test-service/gen/user/v1"
	"github.com/yourorg/test-service/internal/models"
	"github.com/yourorg/test-service/test/fixtures"
	"github.com/yourorg/test-service/test/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUserHandler_CreateUser(t *testing.T) {
	mockService := new(mocks.MockUserService)
	mockLogger := new(mocks.MockLogger)
	handler := NewUserHandler(mockService, mockLogger)

	t.Run("successful creation", func(t *testing.T) {
		req := &user.CreateUserRequest{
			Username:  "testuser",
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
		}

		expectedUser := fixtures.NewUser()
		expectedUser.ID = 1
		expectedUser.Username = req.Username
		expectedUser.Email = req.Email
		expectedUser.FirstName = req.FirstName
		expectedUser.LastName = req.LastName

		mockService.On("CreateUser", mock.MatchedBy(func(createReq *models.CreateUserRequest) bool {
			return createReq.Username == req.Username && createReq.Email == req.Email
		})).Return(expectedUser, nil)

		mockLogger.On("Info", "CreateUser gRPC request received", mock.Anything).Return()

		resp, err := handler.CreateUser(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, expectedUser.ID, resp.User.Id)
		assert.Equal(t, expectedUser.Username, resp.User.Username)
		assert.Equal(t, expectedUser.Email, resp.User.Email)
		mockService.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		req := &user.CreateUserRequest{
			Username:  "testuser",
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
		}

		mockService.On("CreateUser", mock.Anything).Return(nil, assert.AnError)
		mockLogger.On("Info", "CreateUser gRPC request received", mock.Anything).Return()
		mockLogger.On("Error", "Failed to create user", mock.Anything).Return()

		resp, err := handler.CreateUser(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Internal, status.Code(err))
		mockService.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})
}
