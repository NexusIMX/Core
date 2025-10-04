package gateway

import (
	"context"
	"fmt"
	"io"
	"time"

	gatewaypb "github.com/dollarkillerx/im-system/api/proto/gateway"
	messagepb "github.com/dollarkillerx/im-system/api/proto/message"
	"github.com/dollarkillerx/im-system/pkg/interceptor"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServer Gateway gRPC 服务器
type GRPCServer struct {
	gatewaypb.UnimplementedGatewayServiceServer
	connMgr     *ConnectionManager
	handler     *Handler
	clients     *ServiceClients
	gatewayAddr string
}

// NewGRPCServer 创建 gRPC 服务器
func NewGRPCServer(connMgr *ConnectionManager, handler *Handler, clients *ServiceClients, gatewayAddr string) *GRPCServer {
	return &GRPCServer{
		connMgr:     connMgr,
		handler:     handler,
		clients:     clients,
		gatewayAddr: gatewayAddr,
	}
}

// Connect 建立双向流连接
func (s *GRPCServer) Connect(stream gatewaypb.GatewayService_ConnectServer) error {
	ctx := stream.Context()

	// 从 context 获取用户信息（由拦截器注入）
	userID, ok := interceptor.GetUserID(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	deviceID, ok := interceptor.GetDeviceID(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "device not identified")
	}

	logger.Log.Info("Client connecting",
		zap.Int64("user_id", userID),
		zap.String("device_id", deviceID),
	)

	// 创建连接
	conn := NewConnection(userID, deviceID, stream)
	s.connMgr.AddConnection(conn)
	defer s.connMgr.RemoveConnection(userID, deviceID)

	// 注册路由到 Router 服务
	if err := s.clients.RegisterRoute(ctx, userID, deviceID, s.gatewayAddr); err != nil {
		logger.Log.Error("Failed to register route",
			zap.Int64("user_id", userID),
			zap.String("device_id", deviceID),
			zap.Error(err),
		)
	}
	defer s.clients.UnregisterRoute(context.Background(), userID, deviceID)

	// 启动发送 goroutine
	sendDone := make(chan struct{})
	go s.sendLoop(conn, sendDone)

	// 启动心跳 goroutine
	keepAliveDone := make(chan struct{})
	go s.keepAliveLoop(ctx, conn, keepAliveDone)

	// 接收客户端消息
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			logger.Log.Info("Client disconnected (EOF)",
				zap.Int64("user_id", userID),
				zap.String("device_id", deviceID),
			)
			break
		}
		if err != nil {
			logger.Log.Error("Receive error",
				zap.Int64("user_id", userID),
				zap.String("device_id", deviceID),
				zap.Error(err),
			)
			break
		}

		// 处理消息
		s.handler.HandleClientMessage(ctx, conn, msg)
	}

	// 等待发送和心跳 goroutine 结束
	conn.Close()
	<-sendDone
	<-keepAliveDone

	logger.Log.Info("Client connection closed",
		zap.Int64("user_id", userID),
		zap.String("device_id", deviceID),
	)

	return nil
}

// Send 发送消息（一元 RPC）
func (s *GRPCServer) Send(ctx context.Context, req *gatewaypb.SendRequest) (*gatewaypb.SendResponse, error) {
	userID, ok := interceptor.GetUserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	// 转换 conv_type
	var convType messagepb.ConversationType
	switch req.ConvType {
	case "direct":
		convType = messagepb.ConversationType_DIRECT
	case "group":
		convType = messagepb.ConversationType_GROUP
	case "channel":
		convType = messagepb.ConversationType_CHANNEL
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid conv_type")
	}

	// 调用 Message 服务
	resp, err := s.clients.SendMessage(ctx, req.ConvId, userID, convType, req.Body.AsMap(), req.ReplyTo, req.Mentions)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send message: %v", err)
	}

	return &gatewaypb.SendResponse{
		MsgId:     resp.MsgId,
		Seq:       resp.Seq,
		CreatedAt: resp.CreatedAt,
	}, nil
}

// Sync 同步消息
func (s *GRPCServer) Sync(ctx context.Context, req *gatewaypb.SyncRequest) (*gatewaypb.SyncResponse, error) {
	userID, ok := interceptor.GetUserID(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	var convMessages []*gatewaypb.ConvMessages

	for _, convSync := range req.Conversations {
		// 从 Message 服务拉取消息
		resp, err := s.clients.PullMessages(ctx, convSync.ConvId, convSync.SinceSeq, 100)
		if err != nil {
			logger.Log.Error("Failed to pull messages",
				zap.Int64("user_id", userID),
				zap.Int64("conv_id", convSync.ConvId),
				zap.Error(err),
			)
			continue
		}

		// 转换消息格式
		var chatMessages []*gatewaypb.ChatMessage
		for _, msg := range resp.Messages {
			chatMessages = append(chatMessages, &gatewaypb.ChatMessage{
				MsgId:     msg.MsgId,
				ConvId:    msg.ConvId,
				Seq:       msg.Seq,
				SenderId:  msg.SenderId,
				ConvType:  msg.ConvType.String(),
				Body:      msg.Body,
				ReplyTo:   msg.ReplyTo,
				Mentions:  msg.Mentions,
				CreatedAt: msg.CreatedAt,
			})
		}

		convMessages = append(convMessages, &gatewaypb.ConvMessages{
			ConvId:   convSync.ConvId,
			Messages: chatMessages,
			HasMore:  resp.HasMore,
		})
	}

	return &gatewaypb.SyncResponse{
		ConvMessages: convMessages,
	}, nil
}

// sendLoop 发送循环
func (s *GRPCServer) sendLoop(conn *Connection, done chan struct{}) {
	defer close(done)

	for {
		select {
		case msg, ok := <-conn.SendChan:
			if !ok {
				return
			}

			if err := conn.Stream.Send(msg); err != nil {
				logger.Log.Error("Failed to send message to client",
					zap.Int64("user_id", conn.UserID),
					zap.String("device_id", conn.DeviceID),
					zap.Error(err),
				)
				return
			}

		case <-conn.CloseChan:
			return
		}
	}
}

// keepAliveLoop 心跳循环
func (s *GRPCServer) keepAliveLoop(ctx context.Context, conn *Connection, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 发送心跳到 Router 服务
			if err := s.clients.KeepAlive(ctx, conn.UserID, conn.DeviceID); err != nil {
				logger.Log.Warn("Failed to send keep-alive",
					zap.Int64("user_id", conn.UserID),
					zap.String("device_id", conn.DeviceID),
					zap.Error(err),
				)
			}

		case <-conn.CloseChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// GetGatewayAddr 获取网关地址
func GetGatewayAddr(port int) string {
	// 这里简化实现，实际应该获取外网 IP
	return fmt.Sprintf("gateway:%d", port)
}
