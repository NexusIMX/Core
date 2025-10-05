# 🚀 IM System 启动指南

完整的项目启动步骤和常见问题解决方案。

---

## 📋 目录

1. [方式一：Docker Compose 一键启动（推荐）](#方式一docker-compose-一键启动推荐)
2. [方式二：本地开发启动](#方式二本地开发启动)
3. [验证服务](#验证服务)
4. [常见问题](#常见问题)
5. [API 使用示例](#api-使用示例)

---

## 方式一：Docker Compose 一键启动（推荐）

### 1. 前置要求

确保已安装：
- Docker (20.10+)
- Docker Compose (v2.0+)

检查安装：
```bash
docker --version
docker-compose --version
```

### 2. 启动所有服务

```bash
# 进入 docker 目录
cd deployments/docker

# 启动所有服务（基础设施 + 应用服务）
docker-compose up -d
```

这将启动：
- ✅ PostgreSQL (5432)
- ✅ Redis (6379)
- ✅ Consul (8500) - UI: http://localhost:8500
- ✅ Garage (3900, 3902, 3903) - S3 兼容存储
- ✅ User Service (50054)
- ✅ Router Service (50052)
- ✅ Message Service (50053)
- ✅ Gateway Service (50051)
- ✅ File Service (8080)

### 3. 查看服务状态

```bash
# 查看所有容器状态
docker-compose ps

# 查看服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f user-service
docker-compose logs -f gateway-service
```

### 4. 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷（谨慎！会删除所有数据）
docker-compose down -v
```

---

## 方式二：本地开发启动

适合需要调试代码或快速迭代的场景。

### 1. 前置要求

- Go 1.24+
- PostgreSQL 16+
- Redis 7+
- Consul 1.17+
- MinIO / S3 兼容存储

### 2. 启动基础设施

#### 选项 A：使用 Docker 启动基础设施

```bash
cd deployments/docker

# 只启动基础设施服务
docker-compose up -d postgres redis consul garage
```

#### 选项 B：手动启动基础设施

**PostgreSQL:**
```bash
# macOS
brew services start postgresql@16

# Linux
sudo systemctl start postgresql
```

**Redis:**
```bash
# macOS
brew services start redis

# Linux
sudo systemctl start redis
```

**Consul:**
```bash
# 开发模式
consul agent -dev
```

**Garage:**
```bash
# 使用 Docker 运行 Garage（推荐）
docker run -d \
  --name garage \
  -p 3900:3900 \
  -p 3902:3902 \
  -p 3903:3903 \
  -v /tmp/garage/data:/var/lib/garage/data \
  -v /tmp/garage/meta:/var/lib/garage/meta \
  dxflrs/garage:v1.0.0 \
  server
```

### 3. 配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑 .env 文件（可选，使用默认值即可）
vim .env
```

### 4. 初始化数据库

```bash
# 创建数据库
psql -U postgres -c "CREATE DATABASE im_system;"
psql -U postgres -c "CREATE USER imuser WITH PASSWORD 'impassword';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE im_system TO imuser;"

# 运行迁移脚本
psql -h localhost -U imuser -d im_system -f migrations/001_init_schema.sql
```

### 5. 初始化 Garage

```bash
# 返回项目根目录
cd ../..

# 运行 Garage 初始化脚本
bash scripts/init_garage.sh
```

这将自动：
- 配置 Garage 节点
- 创建 access key
- 创建 bucket: `im-files`
- 设置权限

**手动初始化（可选）：**
```bash
# 进入 Garage 容器
docker exec -it im-garage sh

# 获取节点 ID
garage node id

# 配置节点（将 <NODE_ID> 替换为实际 ID）
garage layout assign -z dc1 -c 1 <NODE_ID>
garage layout apply --version 1

# 创建 key
garage key create im-system-key

# 创建 bucket
garage bucket create im-files

# 授权
garage bucket allow --read --write --owner im-files --key im-system-key
```

### 6. 编译服务

```bash
# 下载依赖
make deps

# 编译所有服务
make build
```

编译后的二进制文件位于 `bin/` 目录：
```
bin/
├── user
├── router
├── message
├── gateway
└── file
```

### 7. 启动服务（按顺序）

**终端 1 - User Service:**
```bash
make run-user
# 或
./bin/user
```

**终端 2 - Router Service:**
```bash
make run-router
# 或
./bin/router
```

**终端 3 - Message Service:**
```bash
make run-message
# 或
./bin/message
```

**终端 4 - Gateway Service:**
```bash
make run-gateway
# 或
./bin/gateway
```

**终端 5 - File Service:**
```bash
make run-file
# 或
./bin/file
```

---

## 验证服务

### 1. 检查 Consul 服务注册

访问 Consul UI: http://localhost:8500

应该看到以下服务：
- ✅ gateway
- ✅ router
- ✅ message
- ✅ user
- ✅ file

### 2. 健康检查

```bash
# 检查 Consul
curl http://localhost:8500/v1/status/leader

# 检查 Garage（S3 API）
curl http://localhost:3900/

# 检查 PostgreSQL
psql -h localhost -U imuser -d im_system -c "SELECT 1;"

# 检查 Redis
redis-cli ping
```

### 3. 测试 API

#### 注册用户

```bash
# 使用 grpcurl 测试（需要安装 grpcurl）
grpcurl -plaintext \
  -d '{
    "username": "testuser",
    "password": "password123",
    "email": "test@example.com",
    "nickname": "Test User"
  }' \
  localhost:50054 \
  user.UserService/Register
```

#### 用户登录

```bash
grpcurl -plaintext \
  -d '{
    "username": "testuser",
    "password": "password123",
    "device_id": "device-001"
  }' \
  localhost:50054 \
  user.UserService/Login
```

响应会包含 JWT Token，用于后续请求。

---

## 常见问题

### 1. 端口被占用

**问题**: `bind: address already in use`

**解决**:
```bash
# 查找占用端口的进程
lsof -i :5432  # PostgreSQL
lsof -i :6379  # Redis
lsof -i :50051 # Gateway

# 停止进程
kill -9 <PID>
```

### 2. 数据库连接失败

**问题**: `could not connect to database`

**解决**:
```bash
# 检查 PostgreSQL 是否运行
pg_isready -h localhost -p 5432

# 检查用户和密码
psql -h localhost -U imuser -d im_system

# 重置数据库
make db-reset
```

### 3. Consul 服务注册失败

**问题**: `failed to register service`

**解决**:
```bash
# 检查 Consul 是否运行
curl http://localhost:8500/v1/status/leader

# 重启 Consul（开发模式）
consul agent -dev

# Docker 模式
docker-compose restart consul
```

### 4. Garage 连接失败

**问题**: `failed to connect to S3`

**解决**:
```bash
# 检查 Garage 是否运行
docker ps | grep garage

# 重新初始化 Garage
bash scripts/init_garage.sh

# 检查 Garage 状态
docker exec im-garage garage status

# 查看 bucket 列表
docker exec im-garage garage bucket list
```

### 5. Docker 构建失败

**问题**: `failed to build image`

**解决**:
```bash
# 清理 Docker 缓存
docker system prune -a

# 重新构建
cd deployments/docker
docker-compose build --no-cache

# 查看详细日志
docker-compose up --build
```

### 6. 服务无法相互通信

**问题**: `connection refused` 或 `no such host`

**解决 (Docker)**:
- 检查所有服务在同一网络: `im-network`
- 使用服务名而非 localhost (如 `postgres` 而非 `localhost`)

**解决 (本地)**:
- 确保所有服务使用 `localhost`
- 检查防火墙设置

---

## API 使用示例

详细的 API 示例请查看 [API_EXAMPLES.md](./API_EXAMPLES.md)

### 完整流程示例

```bash
# 1. 注册用户
curl -X POST localhost:50054/register \
  -d '{"username":"alice","password":"pass123"}'

# 2. 登录获取 Token
TOKEN=$(curl -X POST localhost:50054/login \
  -d '{"username":"alice","password":"pass123"}' | jq -r '.token')

# 3. 创建会话
CONV_ID=$(curl -X POST localhost:50053/conversation \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"type":"direct","members":[100,200]}' | jq -r '.conv_id')

# 4. 发送消息
curl -X POST localhost:50053/message \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"conv_id\":$CONV_ID,\"content\":\"Hello World\"}"

# 5. 上传文件
curl -X POST localhost:8080/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.jpg"
```

---

## 性能优化建议

### 生产环境配置

1. **数据库连接池**
   ```env
   POSTGRES_MAX_CONNS=100
   POSTGRES_MIN_CONNS=10
   ```

2. **Redis 持久化**
   ```bash
   # 启用 AOF
   redis-cli CONFIG SET appendonly yes
   ```

3. **Consul 集群**
   ```bash
   # 至少 3 个节点
   consul agent -server -bootstrap-expect=3
   ```

4. **服务扩容**
   ```yaml
   # docker-compose.yml
   deploy:
     replicas: 3  # Router Service
   ```

---

## 监控和调试

### 查看服务日志

```bash
# Docker 模式
docker-compose logs -f user-service
docker-compose logs -f --tail=100 gateway-service

# 本地模式
# 服务日志会直接输出到终端
```

### 性能监控

```bash
# 查看容器资源使用
docker stats

# 查看 PostgreSQL 连接
psql -h localhost -U imuser -d im_system -c "SELECT * FROM pg_stat_activity;"

# 查看 Redis 状态
redis-cli INFO
```

---

## 下一步

- 📖 阅读 [API_EXAMPLES.md](./API_EXAMPLES.md) 了解详细 API 用法
- 🧪 运行测试: `make test`
- 📊 查看测试覆盖率: `make test-coverage`
- 🔧 配置 CI/CD: 参考 NEXT_STEPS.md
- 📝 查看开发计划: [NEXT_STEPS.md](./NEXT_STEPS.md)

---

## 获取帮助

- 📧 Email: support@example.com
- 💬 Issues: https://github.com/dollarkillerx/im-system/issues
- 📚 文档: 查看 `docs/` 目录

**祝你使用愉快！** 🎉
