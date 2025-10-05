# IM System - 分布式即时通讯系统

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![gRPC](https://img.shields.io/badge/gRPC-Protocol-244c5a?style=flat&logo=grpc)](https://grpc.io)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-336791?style=flat&logo=postgresql)](https://postgresql.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

基于 Golang 的高性能分布式即时通讯系统，采用微服务架构，支持单聊、群聊、频道等多种会话模式。

## 🎯 核心特性

- **微服务架构**: Gateway、Router、Message、User、File 五大核心服务，职责清晰
- **服务发现**: 基于 Consul 的服务注册与发现，支持多实例水平扩展与负载均衡
- **实时通信**: gRPC 双向流实现 WebSocket 风格的实时消息推送
- **消息存储**: PostgreSQL 分区表设计，按天自动创建和清理（30天数据保留）
- **路由管理**: Redis 管理用户在线状态和设备路由，支持多设备同时在线
- **文件存储**: S3 兼容对象存储（garage https://github.com/deuxfleurs-org/garage），支持图片、文件、音视频上传（最大 500MB）
- **认证鉴权**: JWT Token 认证，支持设备级别的会话管理与 Token 验证
- **灵活消息**: 基于 JSONB 的消息体设计，支持文本、图片、文件、位置等多种消息类型
- **已读管理**: 基于序列号的消息已读状态跟踪，支持 @提及 和消息回复

## 🏗️ 系统架构

```
                    ┌─────────────┐
                    │   Client    │
                    └──────┬──────┘
                           │ gRPC Streaming
                    ┌──────▼──────┐
                    │   Gateway   │  (实时消息推送)
                    │   :50051    │
                    └──────┬──────┘
                           │
            ┌──────────────┼──────────────┬──────────────┐
            │              │              │              │
       ┌────▼────┐    ┌───▼────┐    ┌───▼──────┐   ┌──▼──────┐
       │  User   │    │ Router │    │ Message  │   │  File   │
       │ :50054  │    │ :50052 │    │  :50053  │   │  :8080  │
       └────┬────┘    └───┬────┘    └────┬─────┘   └────┬────┘
            │             │              │              │
            │             │              │              │
       ┌────▼─────────────▼──────────────▼──────────────▼────┐
       │                                                      │
  ┌────▼────────┐    ┌──────────┐    ┌──────────┐    ┌──────▼────┐
  │ PostgreSQL  │    │  Redis   │    │  Consul  │    │   MinIO   │
  │   :5432     │    │  :6379   │    │  :8500   │    │   :9000   │
  └─────────────┘    └──────────┘    └──────────┘    └───────────┘
  (用户/消息)        (路由/缓存)     (服务发现)        (文件存储)
```

### 架构说明

- **Gateway**: 客户端唯一入口，处理双向流连接，实现消息推送
- **Router**: 管理用户设备路由，维护在线状态，支持多设备
- **Message**: 消息持久化存储，会话管理，消息序列号生成
- **User**: 用户认证授权，JWT Token 生成与验证
- **File**: HTTP REST API，对接 S3 存储，处理文件上传下载

## 📦 服务说明

| 服务 | 端口 | 协议 | 功能说明 |
|------|------|------|----------|
| **Gateway** | 50051 | gRPC | 客户端连接网关，支持双向流通信，实时消息推送，在线状态同步 |
| **Router** | 50052 | gRPC | 用户路由管理，设备注册/注销，心跳保活，在线状态查询 |
| **Message** | 50053 | gRPC | 消息持久化，会话管理，消息拉取，已读状态更新 |
| **User** | 50054 | gRPC | 用户注册/登录，JWT Token 认证，用户信息管理 |
| **File** | 8080 | HTTP REST | 文件上传/下载，S3 对象存储，预签名 URL 生成 |
| **Consul** | 8500 | HTTP | 服务注册与发现，健康检查，配置中心 |
| **PostgreSQL** | 5432 | TCP | 用户数据、消息数据、会话数据持久化存储 |
| **Redis** | 6379 | TCP | 用户路由缓存，设备在线状态，Token 缓存 |
| **MinIO** | 9000 | S3 API | S3 兼容对象存储，文件、图片、音视频存储 |

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
git clone https://github.com/dollarkillerx/im-system.git
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

### 📖 完整 API 示例

详细的 API 调用示例请参考：**[API_EXAMPLES.md](./API_EXAMPLES.md)**

包含：
- 完整的 gRPC 和 REST API 调用示例
- 请求/响应格式说明
- 端到端使用流程示例
- 消息体格式规范
- 错误处理指南

### 🔌 Proto 定义文件

所有服务的 Proto 定义文件（包含详细中英文注释）：

| 文件 | 说明 |
|------|------|
| [common.proto](./api/proto/common/common.proto) | 通用数据结构（响应、分页、错误） |
| [user.proto](./api/proto/user/user.proto) | 用户服务 API 定义 |
| [message.proto](./api/proto/message/message.proto) | 消息服务 API 定义 |
| [router.proto](./api/proto/router/router.proto) | 路由服务 API 定义 |
| [gateway.proto](./api/proto/gateway/gateway.proto) | 网关服务 API 定义 |

### 🎯 核心 API 列表

#### User Service (gRPC - :50054)

| RPC 方法 | 功能 |
|----------|------|
| `Register` | 用户注册 |
| `Login` | 用户登录，返回 JWT Token |
| `GetUserInfo` | 获取用户信息 |
| `UpdateUserInfo` | 更新用户资料 |
| `ValidateToken` | 验证 Token 有效性 |

#### Message Service (gRPC - :50053)

| RPC 方法 | 功能 |
|----------|------|
| `CreateConversation` | 创建会话（单聊/群聊/频道） |
| `GetConversation` | 获取会话详情和成员列表 |
| `SendMessage` | 发送消息（支持 @提及、回复） |
| `PullMessages` | 增量拉取消息 |
| `UpdateReadSeq` | 更新已读序列号 |
| `NotifyNewMessage` | 通知新消息（内部调用） |

#### Router Service (gRPC - :50052)

| RPC 方法 | 功能 |
|----------|------|
| `RegisterRoute` | 注册设备路由（用户上线） |
| `UnregisterRoute` | 注销设备路由（用户下线） |
| `KeepAlive` | 心跳保活（维持在线状态） |
| `GetRoute` | 获取用户所有设备路由 |
| `GetOnlineStatus` | 查询用户在线状态 |

#### Gateway Service (gRPC - :50051)

| RPC 方法 | 功能 |
|----------|------|
| `Connect` | 建立双向流连接（实时推送） |
| `Send` | 发送消息（单次调用） |
| `Sync` | 批量同步多个会话消息 |

#### File Service (HTTP REST - :8080)

| HTTP 方法 | 路径 | 功能 |
|-----------|------|------|
| `POST` | `/v1/files` | 上传文件 |
| `GET` | `/v1/files/:id` | 获取文件信息 |
| `GET` | `/v1/files/:id/download` | 下载文件 |
| `GET` | `/v1/files/:id/url` | 获取预签名下载 URL |
| `DELETE` | `/v1/files/:id` | 删除文件 |
| `GET` | `/v1/files` | 获取用户上传的文件列表 |

### 💬 会话类型

| 类型 | 枚举值 | 说明 |
|------|--------|------|
| **DIRECT** | 0 | 单聊 - 一对一私密对话 |
| **GROUP** | 1 | 群聊 - 多人群组，成员可发消息 |
| **CHANNEL** | 2 | 频道 - 广播式，仅特定角色可发消息 |

### 👥 会话成员角色

| 角色 | 枚举值 | 权限说明 |
|------|--------|----------|
| **OWNER** | 0 | 所有者 - 完全控制权限 |
| **ADMIN** | 1 | 管理员 - 管理成员和消息 |
| **PUBLISHER** | 2 | 发布者 - 可发送消息 |
| **MEMBER** | 3 | 普通成员 - 可发消息和查看 |
| **VIEWER** | 4 | 观察者 - 只读权限 |

### 📨 消息类型

支持的消息类型：
- **文本消息**: `{"type": "text", "content": "..."}`
- **图片消息**: `{"type": "image", "file_id": "...", "width": 1920, "height": 1080}`
- **文件消息**: `{"type": "file", "file_id": "...", "file_name": "...", "file_size": 1024000}`
- **语音消息**: `{"type": "audio", "file_id": "...", "duration": 30}`
- **视频消息**: `{"type": "video", "file_id": "...", "duration": 120}`
- **位置消息**: `{"type": "location", "latitude": 39.9, "longitude": 116.4}`

### 🔔 Gateway 消息类型

| 类型 | 值 | 说明 | 方向 |
|------|-----|------|------|
| PING | 0 | 心跳请求 | C→S |
| PONG | 1 | 心跳响应 | S→C |
| AUTH | 2 | 认证消息 | C→S |
| CHAT | 3 | 聊天消息 | 双向 |
| NOTIFICATION | 4 | 系统通知 | S→C |
| ACK | 5 | 消息确认 | 双向 |
| ERROR | 6 | 错误消息 | S→C |
| TYPING | 7 | 正在输入 | 双向 |
| READ_RECEIPT | 8 | 已读回执 | C→S |
| PRESENCE | 9 | 在线状态 | S→C |

## 📁 项目结构

```
Core/
├── api/proto/              # Protocol Buffers 定义（中英文注释）
│   ├── common/            # 通用数据结构
│   ├── user/              # 用户服务 API
│   ├── message/           # 消息服务 API
│   ├── router/            # 路由服务 API
│   └── gateway/           # 网关服务 API
├── cmd/                   # 各服务入口
│   ├── gateway/
│   ├── router/
│   ├── message/
│   ├── user/
│   └── file/
├── internal/              # 内部业务逻辑
│   ├── gateway/          # Gateway 服务实现
│   ├── router/           # Router 服务实现
│   ├── message/          # Message 服务实现
│   ├── user/             # User 服务实现
│   └── file/             # File 服务实现
├── pkg/                   # 可复用的公共包
│   ├── auth/             # JWT 认证
│   ├── config/           # 配置管理
│   ├── database/         # 数据库连接
│   ├── redis/            # Redis 连接
│   ├── s3/               # S3 存储
│   ├── logger/           # 日志工具
│   ├── registry/         # Consul 服务注册
│   └── interceptor/      # gRPC 拦截器
├── configs/               # 配置文件
├── migrations/            # 数据库迁移脚本
├── deployments/docker/    # Docker 部署文件
├── scripts/               # 辅助脚本
├── Makefile              # 构建命令
├── README.md             # 本文档
└── API_EXAMPLES.md       # API 使用示例
```

## 🔐 安全特性

- ✅ **认证鉴权**: 所有 gRPC 服务强制 JWT Token 认证
- ✅ **密码安全**: bcrypt 加密存储，防止彩虹表攻击
- ✅ **Token 管理**: 设备级别 Token，支持远程登出
- ✅ **路由过期**: Redis 路由信息自动过期（60s TTL）
- ✅ **文件限制**: 上传文件大小限制（500MB），类型校验
- ✅ **配置安全**: 敏感配置通过环境变量注入
- ✅ **传输安全**: 支持 TLS 加密传输（生产环境推荐）
- ✅ **SQL 注入**: 使用参数化查询，防止 SQL 注入

## 🚀 性能优化

- ⚡ **分区表设计**: PostgreSQL 按天分区，自动清理旧数据（30天保留）
- ⚡ **缓存策略**: Redis 缓存热点数据（用户路由、在线状态）
- ⚡ **服务发现**: Consul 自动负载均衡，支持多实例水平扩展
- ⚡ **高性能通信**: gRPC HTTP/2 协议，二进制序列化
- ⚡ **连接复用**: gRPC 连接池，减少连接开销
- ⚡ **异步处理**: 消息推送采用异步机制
- ⚡ **数据库索引**: 消息表按 conv_id + seq 建立复合索引

## 📊 监控与可观测性

- 📝 **结构化日志**: 使用 Zap，JSON 格式，支持日志级别动态调整
- 📝 **日志输出**: 同时输出到 stdout 和文件，方便集中日志收集
- 💚 **健康检查**: Consul 自动健康检查，故障自动摘除
- 💚 **优雅关闭**: 监听系统信号，确保服务优雅停止
- 📈 **Metrics**: 预留 Prometheus metrics 接口
- 🔍 **分布式追踪**: 支持 gRPC Metadata 传递 Trace ID

## 🛡️ 错误处理

- 统一的 gRPC 错误码
- 详细的错误消息和堆栈信息
- 客户端友好的错误提示
- 完整的错误日志记录

## 🎓 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| **Go** | 1.24+ | 主要编程语言 |
| **gRPC** | 1.60+ | 微服务通信协议 |
| **Protocol Buffers** | v3 | 接口定义语言 |
| **PostgreSQL** | 16+ | 关系型数据库 |
| **Redis** | 7+ | 缓存与路由存储 |
| **Consul** | 1.17+ | 服务注册与发现 |
| **MinIO** | 最新 | S3 兼容对象存储 |
| **Zap** | - | 结构化日志库 |
| **JWT** | - | 身份认证 |

## 🗺️ 功能路线图

- [x] 用户注册登录
- [x] JWT Token 认证
- [x] 单聊/群聊/频道
- [x] 实时消息推送
- [x] 消息已读状态
- [x] 文件上传下载
- [x] 多设备在线
- [ ] 消息撤回
- [ ] 群组管理（踢人、禁言）
- [ ] 消息搜索
- [ ] 离线消息推送
- [ ] 端到端加密
- [ ] 音视频通话
- [ ] WebRTC 集成
- [ ] 客户端 SDK（Go/Python/JavaScript）

## 🔧 常见问题

### 如何重新生成 Proto 代码？

```bash
make proto
```

### 如何查看服务日志？

```bash
# Docker 部署
docker-compose logs -f gateway

# 本地运行
tail -f logs/gateway.log
```

### 如何添加新的服务？

1. 在 `api/proto/` 下创建新的 proto 文件
2. 在 `cmd/` 下创建服务入口
3. 在 `internal/` 下实现业务逻辑
4. 在 `Makefile` 中添加构建目标
5. 在 `docker-compose.yml` 中添加服务定义

### 如何进行数据库迁移？

```bash
psql -h localhost -U imuser -d im_system -f migrations/xxx.sql
```

### 如何测试 API？

使用 `grpcurl` 工具测试 gRPC API，详见 [API_EXAMPLES.md](./API_EXAMPLES.md)

## 📚 相关文档

- [API 使用示例](./API_EXAMPLES.md)
- [部署文档](./DEPLOYMENT.md)
- [测试文档](./TESTING.md)
- [项目总结](./PROJECT_SUMMARY.md)

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 贡献步骤

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交改动 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

### 代码规范

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 通过 `golangci-lint` 检查
- 为新功能添加测试
- 更新相关文档

## 👨‍💻 作者

[@dollarkillerx](https://github.com/dollarkillerx)

## 🙏 致谢

感谢所有为这个项目做出贡献的开发者！

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

⭐ 如果这个项目对你有帮助，请给一个 Star！
