package middleware

import (
	"context"

	"github.com/yourorg/test-service/pkg/logger"
	
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor provides authentication middleware for gRPC
type AuthInterceptor struct {
	logger *logger.Logger
}

// NewAuthInterceptor creates a new auth interceptor
func NewAuthInterceptor(logger *logger.Logger) *AuthInterceptor {
	return &AuthInterceptor{logger: logger}
}

// Unary returns a server interceptor function for unary RPCs
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for health check and other public endpoints
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract token from metadata
		token, err := i.extractToken(ctx)
		if err != nil {
			i.logger.Warn("Authentication failed", "method", info.FullMethod, "error", err)
			return nil, status.Errorf(codes.Unauthenticated, "authentication required")
		}

		// Validate token (implement your token validation logic)
		userID, err := i.validateToken(token)
		if err != nil {
			i.logger.Warn("Token validation failed", "method", info.FullMethod, "error", err)
			return nil, status.Errorf(codes.Unauthenticated, "invalid token")
		}

		// Add user ID to context
		ctx = context.WithValue(ctx, "user_id", userID)

		return handler(ctx, req)
	}
}

// Stream returns a server interceptor function for streaming RPCs
func (i *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Skip auth for public methods
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		// Extract token from metadata
		token, err := i.extractToken(ss.Context())
		if err != nil {
			i.logger.Warn("Authentication failed", "method", info.FullMethod, "error", err)
			return status.Errorf(codes.Unauthenticated, "authentication required")
		}

		// Validate token
		userID, err := i.validateToken(token)
		if err != nil {
			i.logger.Warn("Token validation failed", "method", info.FullMethod, "error", err)
			return status.Errorf(codes.Unauthenticated, "invalid token")
		}

		// Create new context with user ID
		ctx := context.WithValue(ss.Context(), "user_id", userID)
		
		// Wrap the stream with new context
		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrapped)
	}
}

// extractToken extracts the bearer token from gRPC metadata
func (i *AuthInterceptor) extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "metadata not found")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization header not found")
	}

	token := values[0]
	if len(token) < 7 || token[:7] != "Bearer " {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header format")
	}

	return token[7:], nil
}

// validateToken validates the JWT token and returns the user ID
func (i *AuthInterceptor) validateToken(token string) (int64, error) {
	// TODO: Implement JWT token validation
	// This is a placeholder implementation
	// You should implement proper JWT validation here
	
	if token == "" {
		return 0, status.Error(codes.Unauthenticated, "empty token")
	}

	// Placeholder: return user ID 1 for valid token
	// In real implementation, decode JWT and extract user ID
	return 1, nil
}

// isPublicMethod checks if a method should skip authentication
func isPublicMethod(method string) bool {
	publicMethods := []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
		// Add other public methods here
	}

	for _, publicMethod := range publicMethods {
		if method == publicMethod {
			return true
		}
	}

	return false
}

// wrappedStream wraps grpc.ServerStream with a new context
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapper's context
func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
