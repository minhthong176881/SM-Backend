package middleware

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	// "google.golang.org/grpc/metadata"
	"github.com/go-redis/redis/v8"
	"github.com/minhthong176881/Server_Management/service/serverService"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	jwtManager *JWTManager
	accesibleRoles map[string][]string
}

func NewAuthInterceptor(jwtManager *JWTManager, accesibleRoles map[string][]string) *AuthInterceptor {
	return &AuthInterceptor{
		jwtManager: jwtManager,
		accesibleRoles: accesibleRoles,
	}
}

func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func (ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Print("-> unary interceptor: ", info.FullMethod)
		err := interceptor.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func AccesibleRoles() map[string][]string {
	const path = "/server_management.SMService/"
	return map[string][]string{
		path + "GetServers": {"admin"},
		path + "GetServerById": {"admin"},
		path + "AddServer": {"admin"},
		path + "UpdateServer": {"admin"},
		path + "ExportServers": {"admin"},
		path + "DeleteServer": {"admin"},
		path + "CheckServer": {"admin"},
		path + "ValidateServer": {"admin"},
		path + "GetServerLog": {"admin"},
	}
}

func (interceptor *AuthInterceptor) authorize(ctx context.Context, method string) error {
	accessibleRoles, ok := interceptor.accesibleRoles[method]
	if !ok {
		return nil
	}

	// md, ok := metadata.FromIncomingContext(ctx)
	// if !ok {
	// 	return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	// }

	// values := md["authorization"]
	// if len(values) == 0 { // no auth header
	// 	return status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	// }

	// accessToken := values[0]
	redisClient := serverService.NewClient()
	accessToken, err := redisClient.Get(redisClient.Context(), "accessToken").Result()
	if err != nil && (err.Error() != string(redis.Nil)) {
		return err
	}
	if accessToken == "" {
		return status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}
	claims, err := interceptor.jwtManager.Verify(accessToken)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "access token is invalid: %v", err)
	}

	for _, role := range accessibleRoles {
		if role == claims.Role {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "no permission to access this RPC")
}
