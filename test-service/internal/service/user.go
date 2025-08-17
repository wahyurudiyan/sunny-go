package service

import (
	"fmt"

	"github.com/yourorg/test-service/internal/models"
	"github.com/yourorg/test-service/internal/repository"
	"github.com/yourorg/test-service/pkg/logger"
)

// UserService handles user business logic
type UserService struct {
	repo   *repository.UserRepository
	logger *logger.Logger
}

// NewUserService creates a new user service
func NewUserService(repo *repository.UserRepository, logger *logger.Logger) *UserService {
	return &UserService{
		repo:   repo,
		logger: logger,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(req *models.CreateUserRequest) (*models.User, error) {
	s.logger.Info("Creating user", "email", req.Email, "username", req.Username)

	// Check if user already exists
	existingUser, err := s.repo.GetByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Create the user
	user, err := s.repo.Create(req)
	if err != nil {
		s.logger.Error("Failed to create user", "error", err, "email", req.Email)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User created successfully", "id", user.ID, "email", user.Email)
	return user, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(id int64) (*models.User, error) {
	s.logger.Debug("Getting user", "id", id)

	user, err := s.repo.GetByID(id)
	if err != nil {
		s.logger.Error("Failed to get user", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	s.logger.Debug("Getting user by email", "email", email)

	user, err := s.repo.GetByEmail(email)
	if err != nil {
		s.logger.Error("Failed to get user by email", "error", err, "email", email)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
