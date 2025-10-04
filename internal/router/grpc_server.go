package router

import (
	"context"

	routerpb "github.com/yourusername/im-system/api/proto/router"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	routerpb.UnimplementedRouterServiceServer
	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) RegisterRoute(ctx context.Context, req *routerpb.RegisterRouteRequest) (*routerpb.RegisterRouteResponse, error) {
	err := s.service.RegisterRoute(ctx, req.UserId, req.DeviceId, req.GatewayAddr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register route: %v", err)
	}

	return &routerpb.RegisterRouteResponse{
		Success: true,
		Message: "Route registered successfully",
	}, nil
}

func (s *GRPCServer) KeepAlive(ctx context.Context, req *routerpb.KeepAliveRequest) (*routerpb.KeepAliveResponse, error) {
	err := s.service.KeepAlive(ctx, req.UserId, req.DeviceId)
	if err != nil {
		return &routerpb.KeepAliveResponse{Success: false}, nil
	}

	return &routerpb.KeepAliveResponse{Success: true}, nil
}

func (s *GRPCServer) GetRoute(ctx context.Context, req *routerpb.GetRouteRequest) (*routerpb.GetRouteResponse, error) {
	routes, err := s.service.GetRoute(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get routes: %v", err)
	}

	var pbRoutes []*routerpb.DeviceRoute
	for _, route := range routes {
		pbRoutes = append(pbRoutes, &routerpb.DeviceRoute{
			DeviceId:    route.DeviceID,
			GatewayAddr: route.GatewayAddr,
			LastActive:  route.LastActive,
		})
	}

	return &routerpb.GetRouteResponse{Routes: pbRoutes}, nil
}

func (s *GRPCServer) UnregisterRoute(ctx context.Context, req *routerpb.UnregisterRouteRequest) (*routerpb.UnregisterRouteResponse, error) {
	err := s.service.UnregisterRoute(ctx, req.UserId, req.DeviceId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unregister route: %v", err)
	}

	return &routerpb.UnregisterRouteResponse{Success: true}, nil
}

func (s *GRPCServer) GetOnlineStatus(ctx context.Context, req *routerpb.GetOnlineStatusRequest) (*routerpb.GetOnlineStatusResponse, error) {
	online, deviceIDs, err := s.service.GetOnlineStatus(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get online status: %v", err)
	}

	return &routerpb.GetOnlineStatusResponse{
		Online:    online,
		DeviceIds: deviceIDs,
	}, nil
}
