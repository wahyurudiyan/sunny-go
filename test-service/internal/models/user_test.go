package models

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
