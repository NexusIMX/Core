package interceptor

import (
	"context"
	"time"

	"github.com/dollarkillerx/im-system/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingUnaryInterceptor 一元 RPC 日志拦截器
func LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// 调用实际的 handler
		resp, err := handler(ctx, req)

		// 记录日志
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("status", statusCode.String()),
		}

		// 添加用户信息（如果有）
		if userID, ok := GetUserID(ctx); ok {
			fields = append(fields, zap.Int64("user_id", userID))
		}
		if deviceID, ok := GetDeviceID(ctx); ok {
			fields = append(fields, zap.String("device_id", deviceID))
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Log.Error("gRPC request failed", fields...)
		} else {
			logger.Log.Info("gRPC request completed", fields...)
		}

		return resp, err
	}
}

// LoggingStreamInterceptor 流式 RPC 日志拦截器
func LoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// 调用实际的 handler
		err := handler(srv, stream)

		// 记录日志
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("status", statusCode.String()),
			zap.Bool("is_client_stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
		}

		ctx := stream.Context()
		if userID, ok := GetUserID(ctx); ok {
			fields = append(fields, zap.Int64("user_id", userID))
		}
		if deviceID, ok := GetDeviceID(ctx); ok {
			fields = append(fields, zap.String("device_id", deviceID))
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Log.Error("gRPC stream failed", fields...)
		} else {
			logger.Log.Info("gRPC stream completed", fields...)
		}

		return err
	}
}
