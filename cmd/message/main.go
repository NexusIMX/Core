package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	messagepb "github.com/dollarkillerx/im-system/api/proto/message"
	"github.com/dollarkillerx/im-system/internal/message"
	"github.com/dollarkillerx/im-system/pkg/config"
	"github.com/dollarkillerx/im-system/pkg/database"
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

	// Connect to database
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Create Consul registry for service discovery
	consulRegistry, err := registry.NewConsulRegistry(&registry.ServiceConfig{
		Address:        cfg.Consul.Address,
		Scheme:         cfg.Consul.Scheme,
		ServiceName:    "message-service",
		ServicePort:    cfg.Server.Message.GRPCPort,
		CheckInterval:  cfg.Consul.HealthCheckInterval,
		DeregisterTime: cfg.Consul.DeregisterAfter,
		Tags:           []string{"grpc", "message"},
		Meta:           map[string]string{"version": "1.0.0"},
	})
	if err != nil {
		logger.Log.Fatal("Failed to create Consul registry", zap.Error(err))
	}

	// Create Router client for service discovery
	routerClient := message.NewRouterClient(consulRegistry)

	// Create service
	repo := message.NewRepository(db)
	service := message.NewService(repo, routerClient)
	grpcServer := message.NewGRPCServer(service)

	// Create interceptor config
	interceptorConfig := interceptor.ChainConfig{
		JWTManager:     nil,        // Message 服务内部调用，暂不需要 JWT 认证
		PublicMethods:  []string{}, // 所有方法都是内部方法
		EnableAuth:     false,
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

	messagepb.RegisterMessageServiceServer(server, grpcServer)

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Message.GRPCPort))
	if err != nil {
		logger.Log.Fatal("Failed to listen", zap.Error(err))
	}

	// Register with Consul
	if err := consulRegistry.Register([]string{"grpc", "message"}, map[string]string{"version": "1.0.0"}); err != nil {
		logger.Log.Fatal("Failed to register with Consul", zap.Error(err))
	}
	defer consulRegistry.Deregister()

	logger.Log.Info("Message service started",
		zap.Int("port", cfg.Server.Message.GRPCPort),
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

	logger.Log.Info("Shutting down message service...")
	server.GracefulStop()
}
