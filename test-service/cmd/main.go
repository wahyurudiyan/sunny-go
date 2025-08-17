package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/yourorg/test-service/internal/config"
	"github.com/yourorg/test-service/internal/handler"
	"github.com/yourorg/test-service/internal/middleware"
	"github.com/yourorg/test-service/internal/repository"
	"github.com/yourorg/test-service/internal/service"
	"github.com/yourorg/test-service/pkg/database"
	"github.com/yourorg/test-service/pkg/logger"
	"github.com/yourorg/test-service/server"
)

func main() {
	app := fx.New(
		// Configuration
		fx.Provide(config.Load),
		
		// Infrastructure
		fx.Provide(logger.New),
		fx.Provide(database.Connect),
		
		// Repositories
		fx.Provide(repository.NewUserRepository),
		
		// Services
		fx.Provide(service.NewUserService),
		
		// Handlers
		fx.Provide(handler.NewUserHandler),
		
		// Middleware
		fx.Provide(middleware.NewAuthMiddleware),
		fx.Provide(middleware.NewLoggingMiddleware),
		
		// Servers
		fx.Provide(server.NewGRPCServer),
		fx.Provide(server.NewHTTPServer),
		
		// Lifecycle hooks
		fx.Invoke(registerHooks),
	)

	// Handle shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	if err := app.Start(ctx); err != nil {
		os.Exit(1)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	if err := app.Stop(ctx); err != nil {
		os.Exit(1)
	}
}

// registerHooks registers lifecycle hooks for the servers
func registerHooks(
	lifecycle fx.Lifecycle,
	grpcServer *server.GRPCServer,
	httpServer *server.HTTPServer,
	logger *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Start GRPC server
			go func() {
				if err := grpcServer.Start(); err != nil {
					logger.Error("GRPC server failed", zap.Error(err))
				}
			}()
			
			// Start HTTP server
			go func() {
				if err := httpServer.Start(); err != nil {
					logger.Error("HTTP server failed", zap.Error(err))
				}
			}()
			
			logger.Info("Servers started successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down servers...")
			
			// Stop servers gracefully
			grpcServer.Stop()
			httpServer.Stop(ctx)
			
			logger.Info("Servers stopped successfully")
			return nil
		},
	})
}