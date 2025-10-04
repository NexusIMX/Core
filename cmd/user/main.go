package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/im-system/internal/user"
	"github.com/yourusername/im-system/pkg/auth"
	"github.com/yourusername/im-system/pkg/config"
	"github.com/yourusername/im-system/pkg/database"
	"github.com/yourusername/im-system/pkg/logger"
	"github.com/yourusername/im-system/pkg/registry"
	userpb "github.com/yourusername/im-system/api/proto/user"
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

	// Create JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiry)

	// Create service
	repo := user.NewRepository(db)
	service := user.NewService(repo, jwtManager)
	grpcServer := user.NewGRPCServer(service)

	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.User.GRPCPort))
	if err != nil {
		logger.Log.Fatal("Failed to listen", zap.Error(err))
	}

	server := grpc.NewServer()
	userpb.RegisterUserServiceServer(server, grpcServer)

	// Register with Consul
	consulRegistry, err := registry.NewConsulRegistry(&registry.ServiceConfig{
		Address:        cfg.Consul.Address,
		Scheme:         cfg.Consul.Scheme,
		ServiceName:    "user-service",
		ServicePort:    cfg.Server.User.GRPCPort,
		CheckInterval:  cfg.Consul.HealthCheckInterval,
		DeregisterTime: cfg.Consul.DeregisterAfter,
		Tags:           []string{"grpc", "user"},
		Meta:           map[string]string{"version": "1.0.0"},
	})
	if err != nil {
		logger.Log.Fatal("Failed to create Consul registry", zap.Error(err))
	}

	if err := consulRegistry.Register([]string{"grpc", "user"}, map[string]string{"version": "1.0.0"}); err != nil {
		logger.Log.Fatal("Failed to register with Consul", zap.Error(err))
	}
	defer consulRegistry.Deregister()

	logger.Log.Info("User service started",
		zap.Int("port", cfg.Server.User.GRPCPort),
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

	logger.Log.Info("Shutting down user service...")
	server.GracefulStop()
}
