package gateway

import (
	"context"
	"fmt"

	messagepb "github.com/dollarkillerx/im-system/api/proto/message"
	routerpb "github.com/dollarkillerx/im-system/api/proto/router"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	GetServiceAddress(serviceName string) (string, error)
}

// ServiceClients 服务客户端集合
type ServiceClients struct {
	discovery ServiceDiscovery
}

// NewServiceClients 创建服务客户端
func NewServiceClients(discovery ServiceDiscovery) *ServiceClients {
	return &ServiceClients{
		discovery: discovery,
	}
}

// SendMessage 发送消息到 Message 服务
func (c *ServiceClients) SendMessage(ctx context.Context, convID int64, senderID int64, convType messagepb.ConversationType, body map[string]interface{}, replyTo *string, mentions []int64) (*messagepb.SendMessageResponse, error) {
	addr, err := c.discovery.GetServiceAddress("message-service")
	if err != nil {
		return nil, fmt.Errorf("failed to discover message service: %w", err)
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to message service: %w", err)
	}
	defer conn.Close()

	client := messagepb.NewMessageServiceClient(conn)

	bodyStruct, err := structpb.NewStruct(body)
	if err != nil {
		return nil, fmt.Errorf("failed to convert body: %w", err)
	}

	return client.SendMessage(ctx, &messagepb.SendMessageRequest{
		ConvId:   convID,
		SenderId: senderID,
		ConvType: convType,
		Body:     bodyStruct,
		ReplyTo:  replyTo,
		Mentions: mentions,
	})
}

// PullMessages 从 Message 服务拉取消息
func (c *ServiceClients) PullMessages(ctx context.Context, convID int64, sinceSeq int64, limit int32) (*messagepb.PullMessagesResponse, error) {
	addr, err := c.discovery.GetServiceAddress("message-service")
	if err != nil {
		return nil, fmt.Errorf("failed to discover message service: %w", err)
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to message service: %w", err)
	}
	defer conn.Close()

	client := messagepb.NewMessageServiceClient(conn)

	return client.PullMessages(ctx, &messagepb.PullMessagesRequest{
		ConvId:   convID,
		SinceSeq: sinceSeq,
		Limit:    limit,
	})
}

// RegisterRoute 注册路由到 Router 服务
func (c *ServiceClients) RegisterRoute(ctx context.Context, userID int64, deviceID string, gatewayAddr string) error {
	addr, err := c.discovery.GetServiceAddress("router-service")
	if err != nil {
		return fmt.Errorf("failed to discover router service: %w", err)
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to router service: %w", err)
	}
	defer conn.Close()

	client := routerpb.NewRouterServiceClient(conn)

	_, err = client.RegisterRoute(ctx, &routerpb.RegisterRouteRequest{
		UserId:      userID,
		DeviceId:    deviceID,
		GatewayAddr: gatewayAddr,
	})

	return err
}

// KeepAlive 发送心跳到 Router 服务
func (c *ServiceClients) KeepAlive(ctx context.Context, userID int64, deviceID string) error {
	addr, err := c.discovery.GetServiceAddress("router-service")
	if err != nil {
		return fmt.Errorf("failed to discover router service: %w", err)
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to router service: %w", err)
	}
	defer conn.Close()

	client := routerpb.NewRouterServiceClient(conn)

	_, err = client.KeepAlive(ctx, &routerpb.KeepAliveRequest{
		UserId:   userID,
		DeviceId: deviceID,
	})

	return err
}

// UnregisterRoute 从 Router 服务注销路由
func (c *ServiceClients) UnregisterRoute(ctx context.Context, userID int64, deviceID string) error {
	addr, err := c.discovery.GetServiceAddress("router-service")
	if err != nil {
		return fmt.Errorf("failed to discover router service: %w", err)
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to router service: %w", err)
	}
	defer conn.Close()

	client := routerpb.NewRouterServiceClient(conn)

	_, err = client.UnregisterRoute(ctx, &routerpb.UnregisterRouteRequest{
		UserId:   userID,
		DeviceId: deviceID,
	})

	return err
}
