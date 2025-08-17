package server

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/yourorg/test-service/internal/config"
	"github.com/yourorg/test-service/internal/handler"
	"github.com/yourorg/test-service/internal/middleware"
)

// HTTPServerParams defines the dependencies for the HTTP server
type HTTPServerParams struct {
	fx.In
	
	Config      *config.Config
	Logger      *zap.Logger
	UserHandler *handler.UserHandler
}

// HTTPServer represents the HTTP server
type HTTPServer struct {
	server *http.Server
	config *config.Config
	logger *zap.Logger
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(params HTTPServerParams) *HTTPServer {
	// TODO: Configure your chosen HTTP framework (gin/echo/fiber)
	// For now, using standard net/http
	
	mux := http.NewServeMux()
	
	// Apply middleware
	handler := middleware.LoggingMiddleware(params.Logger)(mux)
	handler = middleware.AuthMiddleware()(handler)
	
	// Register routes
	// TODO: Register your API routes here
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	address := params.Config.HTTP.Address
	if address == "" {
		address = ":8080"
	}
	
	server := &http.Server{
		Addr:         address,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	return &HTTPServer{
		server: server,
		config: params.Config,
		logger: params.Logger,
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server", zap.String("address", s.server.Addr))
	return s.server.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	return s.server.Shutdown(ctx)
}