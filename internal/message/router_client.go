package message

import (
	"context"
	"fmt"

	routerpb "github.com/dollarkillerx/im-system/api/proto/router"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RouterClient Router 服务客户端接口
type RouterClient interface {
	NotifyNewMessage(ctx context.Context, convID int64, msgID string, seq int64, senderID int64, recipientIDs []int64) (int32, error)
}

// routerClient Router 服务客户端实现
type routerClient struct {
	serviceDiscovery ServiceDiscovery
}

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	GetServiceAddress(serviceName string) (string, error)
}

// NewRouterClient 创建 Router 客户端
func NewRouterClient(sd ServiceDiscovery) RouterClient {
	return &routerClient{
		serviceDiscovery: sd,
	}
}

// NotifyNewMessage 通知新消息
func (c *routerClient) NotifyNewMessage(ctx context.Context, convID int64, msgID string, seq int64, senderID int64, recipientIDs []int64) (int32, error) {
	// 从服务发现获取 Router 地址
	addr, err := c.serviceDiscovery.GetServiceAddress("router-service")
	if err != nil {
		return 0, fmt.Errorf("failed to discover router service: %w", err)
	}

	// 建立连接
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return 0, fmt.Errorf("failed to connect to router: %w", err)
	}
	defer conn.Close()

	// 创建客户端
	client := routerpb.NewRouterServiceClient(conn)

	// 调用 GetRoute 获取在线用户
	var notifiedCount int32
	for _, recipientID := range recipientIDs {
		resp, err := client.GetRoute(ctx, &routerpb.GetRouteRequest{
			UserId: recipientID,
		})

		if err != nil {
			logger.Log.Warn("Failed to get route for user",
				zap.Int64("user_id", recipientID),
				zap.Error(err),
			)
			continue
		}

		if len(resp.Routes) > 0 {
			// 用户在线，推送通知
			// 注意：实际推送由 Gateway 负责，这里只是获取路由信息
			// 在完整实现中，应该调用 Gateway 的推送接口
			notifiedCount++

			logger.Log.Debug("User online, can notify",
				zap.Int64("user_id", recipientID),
				zap.Int("device_count", len(resp.Routes)),
			)
		}
	}

	return notifiedCount, nil
}

// MockRouterClient 用于测试的 Mock 客户端
type MockRouterClient struct{}

func (m *MockRouterClient) NotifyNewMessage(ctx context.Context, convID int64, msgID string, seq int64, senderID int64, recipientIDs []int64) (int32, error) {
	return int32(len(recipientIDs)), nil
}
