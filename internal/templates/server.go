package templates

// Server templates for GRPC and HTTP servers

// GRPC server template
const GRPCServerTemplate = `package server

import (
	"net"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/handler"
)

// GRPCServerParams defines the dependencies for the GRPC server
type GRPCServerParams struct {
	fx.In
	
	Config      *config.Config
	Logger      *zap.Logger
	UserHandler *handler.UserHandler
}

// GRPCServer represents the GRPC server
type GRPCServer struct {
	server  *grpc.Server
	config  *config.Config
	logger  *zap.Logger
}

// NewGRPCServer creates a new GRPC server
func NewGRPCServer(params GRPCServerParams) *GRPCServer {
	server := grpc.NewServer()
	
	// Register services
	// TODO: Register your generated protobuf services here
	// pb.RegisterUserServiceServer(server, params.UserHandler)
	
	// Enable reflection in development
	if params.Config.Environment == "development" {
		reflection.Register(server)
	}
	
	return &GRPCServer{
		server: server,
		config: params.Config,
		logger: params.Logger,
	}
}

// Start starts the GRPC server
func (s *GRPCServer) Start() error {
	address := s.config.GRPC.Address
	if address == "" {
		address = ":50051"
	}
	
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	
	s.logger.Info("Starting GRPC server", zap.String("address", address))
	return s.server.Serve(listener)
}

// Stop gracefully stops the GRPC server
func (s *GRPCServer) Stop() {
	s.logger.Info("Stopping GRPC server")
	s.server.GracefulStop()
}`

// HTTP server template
const HTTPServerTemplate = `package server

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/handler"
	"{{.ModulePath}}/internal/middleware"
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
}`
