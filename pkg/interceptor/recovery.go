package interceptor

import (
	"context"
	"runtime/debug"

	"github.com/yourusername/im-system/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryUnaryInterceptor 一元 RPC 恢复拦截器
func RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// 记录 panic 信息和堆栈
				logger.Log.Error("Panic recovered in gRPC handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)

				// 返回内部错误
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// RecoveryStreamInterceptor 流式 RPC 恢复拦截器
func RecoveryStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// 记录 panic 信息和堆栈
				logger.Log.Error("Panic recovered in gRPC stream handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)

				// 返回内部错误
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, stream)
	}
}
