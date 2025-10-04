# IM System - 分布式即时通讯系统

基于 Golang 的高性能分布式 IM 系统，采用微服务架构，支持单聊、群聊、频道等多种会话类型。

## 🎯 核心特性

- **微服务架构**: Gateway、Router、Message、User、File 五大核心服务
- **服务发现**: 基于 Consul 的服务注册与发现，支持多实例水平扩展
- **消息存储**: PostgreSQL 分区表，按天自动创建和清理（30天数据保留）
- **实时路由**: Redis 管理用户在线状态和设备路由，支持多设备在线
- **文件存储**: S3 兼容存储（MinIO），支持最大 500MB 文件上传
- **认证鉴权**: JWT Token 认证，支持设备级别的会话管理
- **图文混排**: 基于 JSONB 的消息体设计，支持富文本消息

## 🏗️ 系统架构

```
┌─────────┐
│ Client  │
└────┬────┘
     │ gRPC
┌────▼──────────┐      ┌──────────┐
│   Gateway     │◄────►│  Consul  │
└────┬──────────┘      └──────────┘
     │
     ├────────┬────────┬────────┐
     │        │        │        │
┌────▼────┐ ┌▼─────┐ ┌▼──────┐ ┌▼────┐
│  User   │ │Router│ │Message│ │File │
└─────────┘ └──────┘ └───────┘ └─────┘
     │         │         │         │
     └─────┬───┴────┬────┴────┬────┘
           │        │         │
      ┌────▼────┐  ┌▼─────┐  ┌▼────┐
      │PostgreSQL│ │Redis │  │MinIO│
      └─────────┘  └──────┘  └─────┘
```

## 📦 服务说明

| 服务 | 端口 | 功能 | 技术栈 |
|------|------|------|--------|
| Gateway | 50051 | 消息收发入口、推送通知 | gRPC |
| Router | 50052 | 用户路由与在线状态管理 | gRPC + Redis |
| Message | 50053 | 消息存储、分发、增量拉取 | gRPC + PostgreSQL |
| User | 50054 | 用户注册、登录、JWT 鉴权 | gRPC + PostgreSQL |
| File | 8080 | 文件上传、S3 存储 | REST + S3 |
| Consul | 8500 | 服务注册与发现 | - |
| PostgreSQL | 5432 | 关系型数据库 | - |
| Redis | 6379 | 缓存与路由 | - |
| MinIO | 9000 | S3 兼容存储 | - |

## 🚀 快速开始

### 前置要求

- Go 1.24+
- Docker & Docker Compose
- PostgreSQL 16+
- Redis 7+
- Consul 1.17+
- Protocol Buffers 编译器

### 1. 克隆项目

```bash
git clone https://github.com/yourusername/im-system.git
cd im-system
```

### 2. 安装开发工具

```bash
make install-tools
```

### 3. 生成 Proto 代码

```bash
make proto
```

### 4. 下载依赖

```bash
make deps
```

### 5. 使用 Docker Compose 启动所有服务

```bash
cd deployments/docker
docker-compose up -d
```

### 6. 查看服务状态

```bash
docker-compose ps
```

### 7. 查看日志

```bash
docker-compose logs -f [service-name]
```

## 🛠️ 本地开发

### 编译服务

```bash
# 编译所有服务
make build

# 编译单个服务
make build-user
make build-router
make build-message
make build-gateway
make build-file
```

### 运行服务

```bash
# 确保 PostgreSQL、Redis、Consul 已启动
# 运行单个服务
make run-user
make run-router
make run-message
make run-gateway
make run-file
```

### 测试

```bash
# 运行所有测试
make test

# 测试覆盖率
make test-coverage

# 代码检查
make lint

# 代码格式化
make fmt
```

## 📊 数据库迁移

### 初始化数据库

```bash
# 使用 psql 运行迁移脚本
psql -h localhost -U imuser -d im_system -f migrations/001_init_schema.sql
```

### 重置数据库（开发环境）

```bash
make db-reset
```

## 🔧 配置说明

配置文件位于 `configs/config.yaml`，支持环境变量覆盖：

```yaml
server:
  gateway:
    grpc_port: 50051
  router:
    grpc_port: 50052
  # ...

consul:
  address: localhost:8500
  scheme: http
  # ...

database:
  host: localhost
  port: 5432
  # ...
```

环境变量示例（`.env`）：

```bash
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=imuser
POSTGRES_PASSWORD=impassword
POSTGRES_DB=im_system

REDIS_HOST=localhost
REDIS_PORT=6379

JWT_SECRET=your-secret-key-change-in-production

CONSUL_ADDRESS=localhost:8500
```

## 📝 API 文档

### User Service (gRPC)

- `Register`: 用户注册
- `Login`: 用户登录
- `GetUserInfo`: 获取用户信息
- `UpdateUserInfo`: 更新用户信息
- `ValidateToken`: 验证 Token

### Router Service (gRPC)

- `RegisterRoute`: 注册路由
- `KeepAlive`: 心跳保活
- `GetRoute`: 获取路由
- `UnregisterRoute`: 注销路由
- `GetOnlineStatus`: 获取在线状态

### Message Service (gRPC)

- `SendMessage`: 发送消息
- `PullMessages`: 拉取消息
- `GetConversation`: 获取会话信息
- `CreateConversation`: 创建会话
- `UpdateReadSeq`: 更新已读位置

### Gateway Service (gRPC)

- `Connect`: 建立长连接（双向流）
- `Send`: 发送消息
- `Sync`: 同步消息

### File Service (REST)

- `POST /v1/files`: 上传文件
- `GET /v1/files/:id`: 获取文件信息

## 🔐 安全考虑

- 所有 gRPC 服务使用 JWT 认证
- 密码使用 bcrypt 加密存储
- Redis 路由信息自动过期（60s TTL）
- 文件上传大小限制（500MB）
- 敏感配置使用环境变量

## 🎯 性能优化

- PostgreSQL 按天分区表，自动清理旧数据
- Redis 缓存热点数据，减少数据库压力
- Consul 服务发现，支持负载均衡
- gRPC 高性能通信
- 多实例水平扩展

## 📈 监控与日志

- 结构化日志（Zap）
- 支持多输出（stdout + 文件）
- Consul 健康检查
- 服务优雅关闭

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

## 📞 联系方式

- Issues: https://github.com/yourusername/im-system/issues
- Email: your@email.com
