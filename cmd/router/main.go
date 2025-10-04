package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	routerpb "github.com/yourusername/im-system/api/proto/router"
	"github.com/yourusername/im-system/internal/router"
	"github.com/yourusername/im-system/pkg/config"
	"github.com/yourusername/im-system/pkg/logger"
	redisutil "github.com/yourusername/im-system/pkg/redis"
	"github.com/yourusername/im-system/pkg/registry"
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

	// Connect to Redis
	redisClient, err := redisutil.NewRedisClient(&cfg.Redis)
	if err != nil {
		logger.Log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Create service
	service := router.NewService(redisClient)
	grpcServer := router.NewGRPCServer(service)

	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Router.GRPCPort))
	if err != nil {
		logger.Log.Fatal("Failed to listen", zap.Error(err))
	}

	server := grpc.NewServer()
	routerpb.RegisterRouterServiceServer(server, grpcServer)

	// Register with Consul
	consulRegistry, err := registry.NewConsulRegistry(&registry.ServiceConfig{
		Address:        cfg.Consul.Address,
		Scheme:         cfg.Consul.Scheme,
		ServiceName:    "router-service",
		ServicePort:    cfg.Server.Router.GRPCPort,
		CheckInterval:  cfg.Consul.HealthCheckInterval,
		DeregisterTime: cfg.Consul.DeregisterAfter,
		Tags:           []string{"grpc", "router"},
		Meta:           map[string]string{"version": "1.0.0"},
	})
	if err != nil {
		logger.Log.Fatal("Failed to create Consul registry", zap.Error(err))
	}

	if err := consulRegistry.Register([]string{"grpc", "router"}, map[string]string{"version": "1.0.0"}); err != nil {
		logger.Log.Fatal("Failed to register with Consul", zap.Error(err))
	}
	defer consulRegistry.Deregister()

	logger.Log.Info("Router service started",
		zap.Int("port", cfg.Server.Router.GRPCPort),
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

	logger.Log.Info("Shutting down router service...")
	server.GracefulStop()
}
