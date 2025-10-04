package interceptor

import (
	"context"
	"strings"

	"github.com/dollarkillerx/im-system/pkg/auth"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authorizationHeader = "authorization"
	bearerPrefix        = "bearer "
)

// AuthInterceptor JWT 认证拦截器
type AuthInterceptor struct {
	jwtManager    *auth.JWTManager
	publicMethods map[string]bool // 不需要认证的方法
}

// NewAuthInterceptor 创建认证拦截器
func NewAuthInterceptor(jwtManager *auth.JWTManager, publicMethods []string) *AuthInterceptor {
	methodsMap := make(map[string]bool)
	for _, method := range publicMethods {
		methodsMap[method] = true
	}

	return &AuthInterceptor{
		jwtManager:    jwtManager,
		publicMethods: methodsMap,
	}
}

// Unary 一元 RPC 拦截器
func (a *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		logger.Log.Debug("Unary interceptor",
			zap.String("method", info.FullMethod),
		)

		// 检查是否为公开方法
		if a.publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// 验证 Token
		claims, err := a.authorize(ctx)
		if err != nil {
			return nil, err
		}

		// 将用户信息注入 context
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "device_id", claims.DeviceID)

		return handler(ctx, req)
	}
}

// Stream 流式 RPC 拦截器
func (a *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		logger.Log.Debug("Stream interceptor",
			zap.String("method", info.FullMethod),
		)

		// 检查是否为公开方法
		if a.publicMethods[info.FullMethod] {
			return handler(srv, stream)
		}

		// 验证 Token
		claims, err := a.authorize(stream.Context())
		if err != nil {
			return err
		}

		// 创建包装的 stream，注入用户信息
		wrappedStream := &authServerStream{
			ServerStream: stream,
			userID:       claims.UserID,
			deviceID:     claims.DeviceID,
		}

		return handler(srv, wrappedStream)
	}
}

// authorize 从 metadata 中提取并验证 token
func (a *AuthInterceptor) authorize(ctx context.Context) (*auth.Claims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	values := md[authorizationHeader]
	if len(values) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	accessToken := values[0]
	// 去掉 "Bearer " 前缀
	if strings.HasPrefix(strings.ToLower(accessToken), bearerPrefix) {
		accessToken = accessToken[len(bearerPrefix):]
	}

	claims, err := a.jwtManager.Validate(accessToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	return claims, nil
}

// authServerStream 包装的 ServerStream，携带用户信息
type authServerStream struct {
	grpc.ServerStream
	userID   int64
	deviceID string
}

// Context 返回携带用户信息的 context
func (s *authServerStream) Context() context.Context {
	ctx := s.ServerStream.Context()
	ctx = context.WithValue(ctx, "user_id", s.userID)
	ctx = context.WithValue(ctx, "device_id", s.deviceID)
	return ctx
}

// GetUserID 从 context 中获取用户 ID
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value("user_id").(int64)
	return userID, ok
}

// GetDeviceID 从 context 中获取设备 ID
func GetDeviceID(ctx context.Context) (string, bool) {
	deviceID, ok := ctx.Value("device_id").(string)
	return deviceID, ok
}
