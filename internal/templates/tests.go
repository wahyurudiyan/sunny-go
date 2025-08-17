package templates

// Test templates for unit testing

// User model test template
const UserModelTestTemplate = `package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name      string
		user      User
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid user",
			user: User{
				Username:  "testuser",
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			},
			wantError: false,
		},
		{
			name: "missing username",
			user: User{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			},
			wantError: true,
			errorMsg:  "username is required",
		},
		{
			name: "invalid email",
			user: User{
				Username:  "testuser",
				Email:     "invalid-email",
				FirstName: "Test",
				LastName:  "User",
			},
			wantError: true,
			errorMsg:  "invalid email format",
		},
		{
			name: "empty email",
			user: User{
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
			},
			wantError: true,
			errorMsg:  "email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateUserRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       CreateUserRequest
		wantError bool
	}{
		{
			name: "valid request",
			req: CreateUserRequest{
				Username:  "testuser",
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			},
			wantError: false,
		},
		{
			name: "missing username",
			req: CreateUserRequest{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			},
			wantError: true,
		},
		{
			name: "invalid email",
			req: CreateUserRequest{
				Username:  "testuser",
				Email:     "invalid",
				FirstName: "Test",
				LastName:  "User",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateUserRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UpdateUserRequest
		wantError bool
	}{
		{
			name: "valid partial update",
			req: UpdateUserRequest{
				FirstName: stringPtr("NewFirst"),
			},
			wantError: false,
		},
		{
			name: "empty update",
			req:  UpdateUserRequest{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
`

// User repository test template
const UserRepositoryTestTemplate = `package repository

import (
	"context"
	"testing"

	"{{.ModulePath}}/internal/models"
	"{{.ModulePath}}/test/fixtures"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUserRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	repo := NewUserRepository(gormDB)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		user := fixtures.NewCreateUserRequest()
		
		mock.ExpectBegin()
		mock.ExpectQuery(` + "`" + `INSERT INTO "users"` + "`" + `).
			WithArgs(user.Username, user.Email, user.FirstName, user.LastName, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		createdUser, err := repo.Create(ctx, user)
		
		assert.NoError(t, err)
		assert.Equal(t, int64(1), createdUser.ID)
		assert.Equal(t, user.Username, createdUser.Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		user := fixtures.NewCreateUserRequest()
		
		mock.ExpectBegin()
		mock.ExpectQuery(` + "`" + `INSERT INTO "users"` + "`" + `).
			WithArgs(user.Username, user.Email, user.FirstName, user.LastName, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		_, err := repo.Create(ctx, user)
		
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	repo := NewUserRepository(gormDB)
	ctx := context.Background()

	t.Run("user found", func(t *testing.T) {
		expectedUser := fixtures.NewUser()
		expectedUser.ID = 1

		rows := sqlmock.NewRows([]string{"id", "username", "email", "first_name", "last_name", "created_at", "updated_at"}).
			AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Email, expectedUser.FirstName, expectedUser.LastName,
				   expectedUser.CreatedAt, expectedUser.UpdatedAt)

		mock.ExpectQuery(` + "`" + `SELECT \* FROM "users" WHERE "users"\."id" = \$1` + "`" + `).
			WithArgs(1).
			WillReturnRows(rows)

		user, err := repo.GetByID(ctx, 1)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Username, user.Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		mock.ExpectQuery(` + "`" + `SELECT \* FROM "users" WHERE "users"\."id" = \$1` + "`" + `).
			WithArgs(1).
			WillReturnError(gorm.ErrRecordNotFound)

		_, err := repo.GetByID(ctx, 1)
		
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
`

// User service test template
const UserServiceTestTemplate = `package service

import (
	"testing"

	"{{.ModulePath}}/internal/models"
	"{{.ModulePath}}/test/fixtures"
	"{{.ModulePath}}/test/mocks"

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
`

// User handler test template
const UserHandlerTestTemplate = `package handler

import (
	"context"
	"testing"

	"{{.ModulePath}}/gen/user/v1"
	"{{.ModulePath}}/internal/models"
	"{{.ModulePath}}/test/fixtures"
	"{{.ModulePath}}/test/mocks"

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
`

// API user handler test template
const APIUserHandlerTestTemplate = `package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"{{.ModulePath}}/internal/models"
	"{{.ModulePath}}/test/fixtures"
	"{{.ModulePath}}/test/mocks"

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
`

// Database test template
const DatabaseTestTemplate = `package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	tests := []struct {
		name      string
		dbURL     string
		wantError bool
	}{
		{
			name:      "invalid connection string",
			dbURL:     "invalid://connection",
			wantError: true,
		},
		{
			name:      "sqlite in-memory",
			dbURL:     "sqlite://file::memory:?cache=shared",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := Connect(tt.dbURL)
			
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				
				// Clean up
				if db != nil {
					sqlDB, _ := db.DB()
					if sqlDB != nil {
						sqlDB.Close()
					}
				}
			}
		})
	}
}
`

// Logger test template
const LoggerTestTemplate = `package logger

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		wantType string
	}{
		{
			name:     "info level",
			level:    "info",
			wantType: "*logger.Logger",
		},
		{
			name:     "debug level",
			level:    "debug",
			wantType: "*logger.Logger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.level)
			assert.NotNil(t, logger)
		})
	}
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger: slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}

	logger.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
	assert.Contains(t, output, "INFO")
}
`
