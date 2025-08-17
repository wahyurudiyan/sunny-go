package server

import (
	"net"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/yourorg/test-service/internal/config"
	"github.com/yourorg/test-service/internal/handler"
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
}