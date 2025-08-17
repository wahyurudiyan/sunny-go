package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourorg/test-service/internal/models"
	"github.com/yourorg/test-service/test/fixtures"
	"github.com/yourorg/test-service/test/mocks"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserAPIHandler_CreateUser(t *testing.T) {
	mockService := new(mocks.MockUserService)
	mockLogger := new(mocks.MockLogger)
	handler := NewUserAPIHandler(mockService, mockLogger)

	t.Run("successful creation", func(t *testing.T) {
		req := fixtures.NewCreateUserRequest()

		expectedUser := fixtures.NewUser()
		expectedUser.ID = 1
		expectedUser.Username = req.Username
		expectedUser.Email = req.Email
		expectedUser.FirstName = req.FirstName
		expectedUser.LastName = req.LastName

		mockService.On("CreateUser", req).Return(expectedUser, nil)
		mockLogger.On("Info", "CreateUser API request received").Return()

		body, _ := json.Marshal(req)
		request := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		handler.CreateUser(response, request)

		assert.Equal(t, http.StatusCreated, response.Code)
		
		var responseUser models.User
		err := json.Unmarshal(response.Body.Bytes(), &responseUser)
		require.NoError(t, err)
		assert.Equal(t, expectedUser.ID, responseUser.ID)
		assert.Equal(t, expectedUser.Username, responseUser.Username)

		mockService.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockLogger.On("Info", "CreateUser API request received").Return()
		mockLogger.On("Error", "Failed to decode request", mock.Anything).Return()

		request := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer([]byte("invalid json")))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		handler.CreateUser(response, request)

		assert.Equal(t, http.StatusBadRequest, response.Code)
		
		var errorResp map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.Equal(t, true, errorResp["error"])
		assert.Equal(t, "Invalid request body", errorResp["message"])

		mockService.AssertNotCalled(t, "CreateUser")
		mockLogger.AssertExpectations(t)
	})
}
