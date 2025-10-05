#!/bin/bash

# IM System 快速启动脚本

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting IM System...${NC}"

# 检查 docker 和 docker-compose
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    exit 1
fi

# 进入 docker 目录
cd deployments/docker

echo -e "${YELLOW}📦 Starting infrastructure services...${NC}"

# 启动基础设施
docker-compose up -d postgres redis consul garage

echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"
sleep 10

# 检查 PostgreSQL
echo -e "${YELLOW}🔍 Checking PostgreSQL...${NC}"
until docker exec im-postgres pg_isready -U imuser > /dev/null 2>&1; do
    echo "Waiting for PostgreSQL..."
    sleep 2
done
echo -e "${GREEN}✅ PostgreSQL is ready${NC}"

# 检查 Redis
echo -e "${YELLOW}🔍 Checking Redis...${NC}"
until docker exec im-redis redis-cli ping > /dev/null 2>&1; do
    echo "Waiting for Redis..."
    sleep 2
done
echo -e "${GREEN}✅ Redis is ready${NC}"

# 检查 Consul
echo -e "${YELLOW}🔍 Checking Consul...${NC}"
until curl -s http://localhost:8500/v1/status/leader > /dev/null 2>&1; do
    echo "Waiting for Consul..."
    sleep 2
done
echo -e "${GREEN}✅ Consul is ready${NC}"

# 初始化 Garage
echo -e "${YELLOW}🔧 Initializing Garage...${NC}"
sleep 5
cd ../..
bash scripts/init_garage.sh

# 返回 docker 目录
cd deployments/docker

# 启动应用服务
echo -e "${YELLOW}🚀 Starting application services...${NC}"
docker-compose up -d user-service router-service message-service gateway-service file-service

echo -e "${YELLOW}⏳ Waiting for services to register with Consul...${NC}"
sleep 10

# 显示服务状态
echo ""
echo -e "${GREEN}================================================${NC}"
echo -e "${GREEN}✅ IM System is now running!${NC}"
echo -e "${GREEN}================================================${NC}"
echo ""
echo "📊 Service Status:"
docker-compose ps

echo ""
echo "🌐 Web UIs:"
echo "  - Consul: http://localhost:8500"
echo ""
echo "🔌 Service Endpoints:"
echo "  - Gateway:  localhost:50051 (gRPC)"
echo "  - Router:   localhost:50052 (gRPC)"
echo "  - Message:  localhost:50053 (gRPC)"
echo "  - User:     localhost:50054 (gRPC)"
echo "  - File:     localhost:8080  (HTTP)"
echo ""
echo "💾 Storage:"
echo "  - PostgreSQL: localhost:5432"
echo "  - Redis:      localhost:6379"
echo "  - Garage S3:  localhost:3900"
echo ""
echo "📝 Logs:"
echo "  docker-compose logs -f [service-name]"
echo ""
echo "🛑 Stop:"
echo "  docker-compose down"
echo ""
echo -e "${GREEN}🎉 Happy coding!${NC}"
