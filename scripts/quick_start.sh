#!/bin/bash

# 快速启动脚本

set -e

echo "🚀 Starting IM System..."

# 1. 启动基础设施
echo "📦 Starting infrastructure (PostgreSQL, Redis, Consul, MinIO)..."
docker-compose -f deployments/docker/docker-compose.yml up -d postgres redis consul minio

# 2. 等待基础设施就绪
echo "⏳ Waiting for infrastructure to be ready..."
sleep 10

# 3. 运行数据库迁移
echo "🗄️  Running database migrations..."
PGPASSWORD=impassword psql -h localhost -U imuser -d im_system -f migrations/001_init_schema.sql

# 4. 初始化 MinIO bucket
echo "🪣 Initializing MinIO bucket..."
docker exec im-minio mc alias set local http://localhost:9000 minioadmin minioadmin || true
docker exec im-minio mc mb local/im-files --ignore-existing || true

# 5. 启动服务
echo "🎯 Starting services..."
docker-compose -f deployments/docker/docker-compose.yml up -d user-service router-service message-service gateway-service file-service

echo "✅ All services started successfully!"
echo ""
echo "📍 Service endpoints:"
echo "  - User Service:    localhost:50054 (gRPC)"
echo "  - Router Service:  localhost:50052 (gRPC)"
echo "  - Message Service: localhost:50053 (gRPC)"
echo "  - Gateway Service: localhost:50051 (gRPC)"
echo "  - File Service:    localhost:8080  (HTTP)"
echo ""
echo "📍 Infrastructure:"
echo "  - PostgreSQL:      localhost:5432"
echo "  - Redis:           localhost:6379"
echo "  - Consul UI:       http://localhost:8500"
echo "  - MinIO Console:   http://localhost:9001 (minioadmin/minioadmin)"
echo ""
echo "🔍 Check service status:"
echo "  docker-compose -f deployments/docker/docker-compose.yml ps"
echo ""
echo "📝 View logs:"
echo "  docker-compose -f deployments/docker/docker-compose.yml logs -f"
