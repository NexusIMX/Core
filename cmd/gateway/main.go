package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	gatewaypb "github.com/dollarkillerx/im-system/api/proto/gateway"
	"github.com/dollarkillerx/im-system/internal/gateway"
	"github.com/dollarkillerx/im-system/pkg/auth"
	"github.com/dollarkillerx/im-system/pkg/config"
	"github.com/dollarkillerx/im-system/pkg/interceptor"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/dollarkillerx/im-system/pkg/registry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	if err := logger.Init(cfg.Log.Level, cfg.Log.Encoding, cfg.Log.OutputPaths); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Create JWT manager for authentication
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiry)

	// Create Consul registry for service discovery
	consulRegistry, err := registry.NewConsulRegistry(&registry.ServiceConfig{
		Address:        cfg.Consul.Address,
		Scheme:         cfg.Consul.Scheme,
		ServiceName:    "gateway-service",
		ServicePort:    cfg.Server.Gateway.GRPCPort,
		CheckInterval:  cfg.Consul.HealthCheckInterval,
		DeregisterTime: cfg.Consul.DeregisterAfter,
		Tags:           []string{"grpc", "gateway"},
		Meta:           map[string]string{"version": "1.0.0"},
	})
	if err != nil {
		logger.Log.Fatal("Failed to create Consul registry", zap.Error(err))
	}

	// Create connection manager
	connMgr := gateway.NewConnectionManager()

	// Create service clients
	clients := gateway.NewServiceClients(consulRegistry)

	// Create message handler
	handler := gateway.NewHandler(connMgr, clients)

	// Get gateway address
	gatewayAddr := gateway.GetGatewayAddr(cfg.Server.Gateway.GRPCPort)

	// Create gRPC server
	grpcServerImpl := gateway.NewGRPCServer(connMgr, handler, clients, gatewayAddr)

	// Create interceptor config
	// Gateway 需要认证，所有方法都需要 Token
	interceptorConfig := interceptor.ChainConfig{
		JWTManager:     jwtManager,
		PublicMethods:  []string{}, // Gateway 没有公开方法
		EnableAuth:     true,
		EnableLogging:  true,
		EnableRecovery: true,
	}

	// Create gRPC server with interceptors
	unaryInterceptors := interceptor.ChainUnaryInterceptors(interceptorConfig)
	streamInterceptors := interceptor.ChainStreamInterceptors(interceptorConfig)

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	)

	gatewaypb.RegisterGatewayServiceServer(server, grpcServerImpl)

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Gateway.GRPCPort))
	if err != nil {
		logger.Log.Fatal("Failed to listen", zap.Error(err))
	}

	// Register with Consul
	if err := consulRegistry.Register([]string{"grpc", "gateway"}, map[string]string{"version": "1.0.0"}); err != nil {
		logger.Log.Fatal("Failed to register with Consul", zap.Error(err))
	}
	defer consulRegistry.Deregister()

	// Start cleanup goroutine for inactive connections
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go connMgr.CleanupInactive(ctx, 5*time.Minute)

	logger.Log.Info("Gateway service started",
		zap.Int("port", cfg.Server.Gateway.GRPCPort),
		zap.String("gateway_addr", gatewayAddr),
	)

	// Start server in goroutine
	go func() {
		if err := server.Serve(lis); err != nil {
			logger.Log.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down gateway service...")
	server.GracefulStop()
}
