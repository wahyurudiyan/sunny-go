package templates

// API layer templates

// API user handler template (HTTP/REST via grpc-gateway)
const APIUserTemplate = `package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"{{.ModulePath}}/internal/models"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/pkg/logger"

	"github.com/gorilla/mux"
)

// UserAPIHandler handles HTTP requests for user operations via REST API
type UserAPIHandler struct {
	service *service.UserService
	logger  *logger.Logger
}

// NewUserAPIHandler creates a new user API handler
func NewUserAPIHandler(service *service.UserService, logger *logger.Logger) *UserAPIHandler {
	return &UserAPIHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers HTTP routes for user API
func (h *UserAPIHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/users", h.CreateUser).Methods("POST")
	router.HandleFunc("/api/v1/users/{id:[0-9]+}", h.GetUser).Methods("GET")
	router.HandleFunc("/api/v1/users/{id:[0-9]+}", h.UpdateUser).Methods("PUT")
	router.HandleFunc("/api/v1/users/{id:[0-9]+}", h.DeleteUser).Methods("DELETE")
	router.HandleFunc("/api/v1/users", h.ListUsers).Methods("GET")
}

// CreateUser handles POST /api/v1/users
func (h *UserAPIHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("CreateUser API request received")

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.service.CreateUser(&req)
	if err != nil {
		h.logger.Error("Failed to create user", "error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	h.writeJSONResponse(w, http.StatusCreated, user)
}

// GetUser handles GET /api/v1/users/{id}
func (h *UserAPIHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Invalid user ID", "id", idStr, "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	h.logger.Info("GetUser API request received", "id", id)

	user, err := h.service.GetUser(id)
	if err != nil {
		h.logger.Error("Failed to get user", "error", err, "id", id)
		h.writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	h.writeJSONResponse(w, http.StatusOK, user)
}

// UpdateUser handles PUT /api/v1/users/{id}
func (h *UserAPIHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Invalid user ID", "id", idStr, "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	h.logger.Info("UpdateUser API request received", "id", id)

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.service.UpdateUser(id, &req)
	if err != nil {
		h.logger.Error("Failed to update user", "error", err, "id", id)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	h.writeJSONResponse(w, http.StatusOK, user)
}

// DeleteUser handles DELETE /api/v1/users/{id}
func (h *UserAPIHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Invalid user ID", "id", idStr, "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	h.logger.Info("DeleteUser API request received", "id", id)

	err = h.service.DeleteUser(id)
	if err != nil {
		h.logger.Error("Failed to delete user", "error", err, "id", id)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListUsers handles GET /api/v1/users
func (h *UserAPIHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("ListUsers API request received")

	query := r.URL.Query()
	filter := &models.UserFilter{}

	if email := query.Get("email"); email != "" {
		filter.Email = &email
	}
	if username := query.Get("username"); username != "" {
		filter.Username = &username
	}
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	users, err := h.service.ListUsers(filter)
	if err != nil {
		h.logger.Error("Failed to list users", "error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

// writeJSONResponse writes a JSON response
func (h *UserAPIHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// writeErrorResponse writes an error response
func (h *UserAPIHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   true,
		"message": message,
		"status":  statusCode,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
	}
}
`

// gRPC Gateway YAML configuration template
const GRPCGatewayYamlTemplate = `type: google.api.Service
config_version: 3

http:
  rules:
    - selector: user.v1.UserService.CreateUser
      post: "/v1/users"
      body: "*"
    - selector: user.v1.UserService.GetUser
      get: "/v1/users/{id}"
    - selector: user.v1.UserService.UpdateUser
      put: "/v1/users/{id}"
      body: "*"
    - selector: user.v1.UserService.DeleteUser
      delete: "/v1/users/{id}"
    - selector: user.v1.UserService.ListUsers
      get: "/v1/users"
`
