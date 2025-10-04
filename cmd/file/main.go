package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dollarkillerx/im-system/internal/file"
	"github.com/dollarkillerx/im-system/pkg/auth"
	"github.com/dollarkillerx/im-system/pkg/config"
	"github.com/dollarkillerx/im-system/pkg/database"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/dollarkillerx/im-system/pkg/registry"
	"github.com/dollarkillerx/im-system/pkg/s3"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

	// Initialize database
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize S3 client
	s3Client, err := s3.NewClient(&s3.Config{
		Endpoint:        cfg.S3.Endpoint,
		AccessKeyID:     cfg.S3.AccessKeyID,
		SecretAccessKey: cfg.S3.SecretAccessKey,
		Bucket:          cfg.S3.Bucket,
		Region:          cfg.S3.Region,
	})
	if err != nil {
		logger.Log.Fatal("Failed to create S3 client", zap.Error(err))
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiry)

	// Create repository and service
	repo := file.NewRepository(db)
	service := file.NewService(repo, s3Client, cfg.Server.File.MaxFileSize)

	// Create HTTP handler
	handler := file.NewHandler(service)

	// Create Gin router
	if cfg.Server.File.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	// Apply middlewares
	router.Use(file.CORSMiddleware())

	// API routes
	v1 := router.Group("/v1")
	{
		// 需要认证的路由
		files := v1.Group("/files")
		files.Use(file.AuthMiddleware(jwtManager))
		{
			files.POST("", handler.UploadFile)               // 上传文件
			files.GET("", handler.ListUserFiles)             // 获取文件列表
			files.GET("/:id", handler.GetFileInfo)           // 获取文件信息
			files.GET("/:id/download", handler.DownloadFile) // 下载文件
			files.GET("/:id/url", handler.GetDownloadURL)    // 获取下载链接
			files.DELETE("/:id", handler.DeleteFile)         // 删除文件
		}
	}

	// Health check (无需认证)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Register with Consul
	consulRegistry, err := registry.NewConsulRegistry(&registry.ServiceConfig{
		Address:        cfg.Consul.Address,
		Scheme:         cfg.Consul.Scheme,
		ServiceName:    "file-service",
		ServicePort:    cfg.Server.File.HTTPPort,
		CheckInterval:  cfg.Consul.HealthCheckInterval,
		DeregisterTime: cfg.Consul.DeregisterAfter,
		Tags:           []string{"http", "file"},
		Meta:           map[string]string{"version": "1.0.0"},
	})
	if err != nil {
		logger.Log.Fatal("Failed to create Consul registry", zap.Error(err))
	}

	if err := consulRegistry.Register([]string{"http", "file"}, map[string]string{"version": "1.0.0"}); err != nil {
		logger.Log.Fatal("Failed to register with Consul", zap.Error(err))
	}
	defer consulRegistry.Deregister()

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.File.HTTPPort)
	logger.Log.Info("File service started",
		zap.Int("port", cfg.Server.File.HTTPPort),
		zap.String("mode", cfg.Server.File.Mode),
		zap.Int64("max_file_size", cfg.Server.File.MaxFileSize),
	)

	// Create server with graceful shutdown
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down file service...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Log.Info("File service exited")
}
