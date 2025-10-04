package interceptor

import (
	"github.com/yourusername/im-system/pkg/auth"
	"google.golang.org/grpc"
)

// ChainConfig 拦截器链配置
type ChainConfig struct {
	JWTManager    *auth.JWTManager
	PublicMethods []string
	EnableAuth    bool
	EnableLogging bool
	EnableRecovery bool
}

// ChainUnaryInterceptors 创建一元拦截器链
func ChainUnaryInterceptors(config ChainConfig) []grpc.UnaryServerInterceptor {
	var interceptors []grpc.UnaryServerInterceptor

	// Recovery 应该在最外层，确保能捕获所有 panic
	if config.EnableRecovery {
		interceptors = append(interceptors, RecoveryUnaryInterceptor())
	}

	// Logging 在 Recovery 之后
	if config.EnableLogging {
		interceptors = append(interceptors, LoggingUnaryInterceptor())
	}

	// Auth 在最内层，最后执行
	if config.EnableAuth && config.JWTManager != nil {
		authInterceptor := NewAuthInterceptor(config.JWTManager, config.PublicMethods)
		interceptors = append(interceptors, authInterceptor.Unary())
	}

	return interceptors
}

// ChainStreamInterceptors 创建流式拦截器链
func ChainStreamInterceptors(config ChainConfig) []grpc.StreamServerInterceptor {
	var interceptors []grpc.StreamServerInterceptor

	// Recovery 应该在最外层
	if config.EnableRecovery {
		interceptors = append(interceptors, RecoveryStreamInterceptor())
	}

	// Logging 在 Recovery 之后
	if config.EnableLogging {
		interceptors = append(interceptors, LoggingStreamInterceptor())
	}

	// Auth 在最内层
	if config.EnableAuth && config.JWTManager != nil {
		authInterceptor := NewAuthInterceptor(config.JWTManager, config.PublicMethods)
		interceptors = append(interceptors, authInterceptor.Stream())
	}

	return interceptors
}
