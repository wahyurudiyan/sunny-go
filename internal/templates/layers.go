package templates

// Application layer templates (models, services, handlers, repositories)

// Configuration template
const ConfigTemplate = `package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
	LogLevel    string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/{{.ServiceName}}_db?sslmode=disable"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	return config, nil
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt gets environment variable as int with fallback
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

// getEnvBool gets environment variable as bool with fallback
func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}
`

// User model template
const UserModelTemplate = `package models

import (
	"time"
)

// User represents a user entity
type User struct {
	ID        int64     ` + "`json:\"id\" db:\"id\"`" + `
	Email     string    ` + "`json:\"email\" db:\"email\"`" + `
	Username  string    ` + "`json:\"username\" db:\"username\"`" + `
	FirstName string    ` + "`json:\"first_name\" db:\"first_name\"`" + `
	LastName  string    ` + "`json:\"last_name\" db:\"last_name\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\" db:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\" db:\"updated_at\"`" + `
}

// CreateUserRequest represents request to create a new user
type CreateUserRequest struct {
	Email     string ` + "`json:\"email\" validate:\"required,email\"`" + `
	Username  string ` + "`json:\"username\" validate:\"required,min=3,max=50\"`" + `
	FirstName string ` + "`json:\"first_name\" validate:\"required,min=1,max=100\"`" + `
	LastName  string ` + "`json:\"last_name\" validate:\"required,min=1,max=100\"`" + `
}

// UpdateUserRequest represents request to update a user
type UpdateUserRequest struct {
	FirstName *string ` + "`json:\"first_name,omitempty\" validate:\"omitempty,min=1,max=100\"`" + `
	LastName  *string ` + "`json:\"last_name,omitempty\" validate:\"omitempty,min=1,max=100\"`" + `
}

// UserFilter represents filters for user queries
type UserFilter struct {
	Email    *string
	Username *string
	Limit    int
	Offset   int
}
`

// User repository template
const UserRepositoryTemplate = `package repository

import (
	"database/sql"
	"fmt"

	"{{.ModulePath}}/internal/models"
	"{{.ModulePath}}/pkg/database"
)

// UserRepository handles user data operations
type UserRepository struct {
	db *database.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *models.CreateUserRequest) (*models.User, error) {
	query := ` + "`" + `
		INSERT INTO users (email, username, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, email, username, first_name, last_name, created_at, updated_at
	` + "`" + `

	var result models.User
	err := r.db.QueryRow(query, user.Email, user.Username, user.FirstName, user.LastName).Scan(
		&result.ID,
		&result.Email,
		&result.Username,
		&result.FirstName,
		&result.LastName,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &result, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id int64) (*models.User, error) {
	query := ` + "`" + `
		SELECT id, email, username, first_name, last_name, created_at, updated_at
		FROM users
		WHERE id = $1
	` + "`" + `

	var user models.User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := ` + "`" + `
		SELECT id, email, username, first_name, last_name, created_at, updated_at
		FROM users
		WHERE email = $1
	` + "`" + `

	var user models.User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
`

// User service template
const UserServiceTemplate = `package service

import (
	"fmt"

	"{{.ModulePath}}/internal/models"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/pkg/logger"
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
`

// User handler template
const UserHandlerTemplate = `package handler

import (
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/pkg/logger"
)

// UserHandler handles gRPC requests for user operations
type UserHandler struct {
	// Uncomment when protobuf is generated
	// pb.UnimplementedUserServiceServer
	
	service *service.UserService
	logger  *logger.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(service *service.UserService, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger,
	}
}

// Example gRPC handler methods - uncomment and modify when protobuf is generated
// Also uncomment the additional imports when implementing these methods:
//   "context"
//   "{{.ModulePath}}/internal/models"
//   "google.golang.org/grpc/codes"
//   "google.golang.org/grpc/status"
//   "google.golang.org/protobuf/types/known/timestamppb"
`
