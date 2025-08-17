package middleware

import (
	"context"
	"time"

	"github.com/yourorg/test-service/pkg/logger"
	
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor provides logging middleware for gRPC
type LoggingInterceptor struct {
	logger *logger.Logger
}

// NewLoggingInterceptor creates a new logging interceptor
func NewLoggingInterceptor(logger *logger.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{logger: logger}
}

// Unary returns a server interceptor function for unary RPCs
func (i *LoggingInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		i.logger.Info("gRPC request started",
			"method", info.FullMethod,
			"request", req,
		)

		// Call the handler
		resp, err := handler(ctx, req)

		// Log the result
		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			}
		}

		i.logger.Info("gRPC request completed",
			"method", info.FullMethod,
			"code", code.String(),
			"duration", duration,
			"error", err,
		)

		return resp, err
	}
}

// Stream returns a server interceptor function for streaming RPCs
func (i *LoggingInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		i.logger.Info("gRPC stream started",
			"method", info.FullMethod,
		)

		// Call the handler
		err := handler(srv, ss)

		// Log the result
		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			}
		}

		i.logger.Info("gRPC stream completed",
			"method", info.FullMethod,
			"code", code.String(),
			"duration", duration,
			"error", err,
		)

		return err
	}
}
