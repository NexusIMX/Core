# 下一步开发计划

**项目**: IM System - 分布式即时通讯系统  
**当前版本**: v1.0-alpha  
**日期**: 2025-10-05

---

## ✅ 已完成的工作

### 核心功能
- ✅ 5 个微服务实现 (Gateway, Router, Message, User, File)
- ✅ gRPC API 定义和实现
- ✅ JWT 认证系统
- ✅ PostgreSQL 数据存储
- ✅ Redis 路由管理
- ✅ MinIO/S3 文件存储
- ✅ Consul 服务注册与发现
- ✅ Docker 部署配置

### 测试和文档
- ✅ pkg 公共包单元测试 (覆盖率 85%+)
- ✅ 测试框架搭建
- ✅ 完整的 API 文档
- ✅ 测试文档和指南
- ✅ 环境配置模板

---

## 🎯 短期目标 (1-2 周)

### 1. 代码重构 - 接口化设计

**优先级**: 🔴 高

当前服务层直接依赖具体实现,不利于测试和扩展。需要引入接口层。

#### User Service
```go
// internal/user/interfaces.go
type UserRepository interface {
    CreateUser(ctx context.Context, username, password, email, nickname string) (*User, error)
    GetUserByUsername(ctx context.Context, username string) (*User, error)
    GetUserByID(ctx context.Context, userID int64) (*User, error)
    UpdateUser(ctx context.Context, userID int64, nickname, avatar, bio *string) error
    VerifyPassword(hashedPassword, password string) error
}

// internal/user/service.go 修改为
func NewService(repo UserRepository, jwtManager *auth.JWTManager) *Service
```

#### Router Service
```go
type RouteStorage interface {
    RegisterRoute(ctx context.Context, userID int64, deviceID, gatewayAddr string) error
    UnregisterRoute(ctx context.Context, userID int64, deviceID string) error
    GetRoute(ctx context.Context, userID int64) ([]*DeviceRoute, error)
    KeepAlive(ctx context.Context, userID int64, deviceID string) error
    GetOnlineStatus(ctx context.Context, userID int64) (bool, []string, error)
}
```

#### Message Service
```go
type MessageRepository interface {
    SaveMessage(ctx context.Context, msg *Message) error
    PullMessages(ctx context.Context, convID int64, sinceSeq int64, limit int32) ([]*Message, bool, error)
    GetNextSeq(ctx context.Context, convID int64) (int64, error)
    CreateConversation(ctx context.Context, convType types.ConversationType, title string, ownerID int64, memberIDs []int64) (int64, error)
    GetConversation(ctx context.Context, convID int64) (*Conversation, []*ConversationMember, error)
    UpdateReadSeq(ctx context.Context, convID int64, userID int64, seq int64) error
    GetConversationMembers(ctx context.Context, convID int64) ([]int64, error)
}

type RouterClient interface {
    NotifyNewMessage(ctx context.Context, convID int64, msgID string, seq int64, senderID int64, recipientIDs []int64) (int32, error)
}
```

#### File Service
```go
type FileRepository interface {
    Create(ctx context.Context, file *File) error
    GetByFileID(ctx context.Context, fileID string) (*File, error)
    Delete(ctx context.Context, fileID string) error
    ListByUploader(ctx context.Context, userID int64, limit, offset int32) ([]*File, error)
}

type StorageClient interface {
    Upload(ctx context.Context, key string, body io.Reader, contentType string) error
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    GetPresignedURL(ctx context.Context, key string) (string, error)
}
```

### 2. 服务层单元测试

基于接口实现 Mock 对象,添加完整的服务层测试。

**目标覆盖率**: 85%+

### 3. 限流和防刷

**优先级**: 🔴 高

添加限流中间件防止滥用:

```go
// pkg/interceptor/ratelimit.go
type RateLimiter struct {
    redis  *redis.Client
    limits map[string]int // method -> requests per minute
}

func (r *RateLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // 从 metadata 获取用户 ID
        // 检查限流
        // 允许或拒绝请求
    }
}
```

配置示例:
- 消息发送: 100 req/min
- 文件上传: 10 req/min
- 用户注册: 5 req/min

### 4. 监控指标 (Prometheus)

**优先级**: 🟡 中

添加 Prometheus metrics 采集:

```go
// pkg/metrics/metrics.go
var (
    RequestTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "im_requests_total",
            Help: "Total number of requests",
        },
        []string{"service", "method", "status"},
    )

    RequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "im_request_duration_seconds",
            Help:    "Request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"service", "method"},
    )

    ActiveConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "im_gateway_active_connections",
            Help: "Number of active gateway connections",
        },
    )
)
```

### 5. 数据库分区自动化

**优先级**: 🟡 中

实现消息表和文件表的自动分区管理:

```bash
# scripts/partition_manager.sh
#!/bin/bash
# 自动创建和清理分区表

# 创建未来 7 天的分区
# 删除 30 天前的分区
```

或使用 Go 实现:

```go
// cmd/partition-manager/main.go
// 定时任务管理分区
```

---

## 🚀 中期目标 (2-4 周)

### 1. 集成测试套件

**优先级**: 🔴 高

实现完整的集成测试:

```
test/integration/
├── user_test.go          # User Service 集成测试
├── router_test.go        # Router Service 集成测试
├── message_test.go       # Message Service 集成测试
├── file_test.go          # File Service 集成测试
├── e2e_flow_test.go      # 端到端流程测试
└── helpers/
    ├── db.go             # 数据库测试工具
    ├── redis.go          # Redis 测试工具
    └── minio.go          # MinIO 测试工具
```

使用 testcontainers:
```go
import "github.com/testcontainers/testcontainers-go"
```

### 2. 消息推送优化

**优先级**: 🟡 中

当前推送机制的改进:
- 批量推送支持
- 推送失败重试
- 离线消息队列
- 推送优先级

### 3. 群组管理功能

**优先级**: 🟡 中

实现完整的群组管理:
- 踢出成员
- 禁言管理
- 群组公告
- 入群审批

### 4. 消息搜索

**优先级**: 🟢 低

基于 PostgreSQL 全文搜索或 Elasticsearch:

```go
// internal/message/search.go
type SearchService struct {
    es *elasticsearch.Client
}

func (s *SearchService) SearchMessages(ctx context.Context, userID int64, query string, limit int) ([]*Message, error)
```

### 5. CI/CD 配置

**优先级**: 🔴 高

设置 GitHub Actions:

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
      redis:
        image: redis:7
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make test
      - run: make lint

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: make build
      - run: docker build -t im-system .

  deploy:
    if: github.ref == 'refs/heads/main'
    needs: [test, build]
    runs-on: ubuntu-latest
    steps:
      - run: echo "Deploy to staging"
```

---

## 📊 长期目标 (1-3 月)

### 1. 客户端 SDK

为常用语言提供 SDK:

#### Go SDK
```go
// sdk/go/client.go
type Client struct {
    conn *grpc.ClientConn
}

func NewClient(addr string, token string) (*Client, error)
func (c *Client) SendMessage(ctx context.Context, convID int64, content string) error
```

#### Python SDK
```python
# sdk/python/im_client.py
class IMClient:
    def __init__(self, addr: str, token: str):
        pass

    def send_message(self, conv_id: int, content: str):
        pass
```

#### JavaScript SDK
```javascript
// sdk/javascript/client.js
class IMClient {
    constructor(addr, token) {}
    async sendMessage(convId, content) {}
}
```

### 2. 消息撤回

实现消息撤回功能:
- 2 分钟内可撤回
- 撤回通知
- 撤回记录

### 3. 端到端加密 (E2EE)

**优先级**: 🟢 低 (高安全需求时为高)

实现端到端加密:
- 密钥交换 (Diffie-Hellman)
- 消息加密 (AES-256)
- 前向保密 (Forward Secrecy)

### 4. 音视频通话

**优先级**: 🟢 低

集成 WebRTC:
- 信令服务器
- STUN/TURN 服务器
- 通话状态管理

### 5. 离线推送

**优先级**: 🟡 中

集成第三方推送服务:
- APNs (iOS)
- FCM (Android)
- 推送模板管理
- 推送统计

---

## 🔧 技术债务

### 需要优化的部分

1. **错误处理标准化**
   - 统一的错误码
   - 结构化错误响应
   - 错误日志关联

2. **日志优化**
   - 统一日志格式
   - 日志级别动态调整
   - 敏感信息脱敏

3. **配置管理**
   - 支持配置中心 (Consul/etcd)
   - 配置热更新
   - 环境隔离

4. **性能优化**
   - 连接池优化
   - 缓存策略
   - 查询优化

5. **安全加固**
   - SQL 注入防护审计
   - XSS/CSRF 防护
   - 敏感数据加密

---

## 📋 优先级矩阵

| 任务 | 优先级 | 工作量 | 影响 | 建议时间 |
|------|--------|--------|------|---------|
| 接口化重构 | 🔴 高 | 中 | 高 | 第 1 周 |
| 服务层测试 | 🔴 高 | 中 | 高 | 第 1-2 周 |
| 限流防刷 | 🔴 高 | 小 | 高 | 第 1 周 |
| 集成测试 | 🔴 高 | 大 | 高 | 第 2-3 周 |
| CI/CD | 🔴 高 | 中 | 高 | 第 2 周 |
| Prometheus | 🟡 中 | 中 | 中 | 第 3 周 |
| 分区自动化 | 🟡 中 | 小 | 中 | 第 2 周 |
| 群组管理 | 🟡 中 | 中 | 中 | 第 4 周 |
| 消息推送优化 | 🟡 中 | 中 | 中 | 第 3-4 周 |
| 客户端 SDK | 🟢 低 | 大 | 中 | 第 5-8 周 |
| 消息搜索 | 🟢 低 | 大 | 中 | 第 6-8 周 |
| E2EE | 🟢 低 | 大 | 低 | 待定 |
| 音视频通话 | 🟢 低 | 大 | 中 | 待定 |

---

## 📚 学习资源

### 推荐阅读

1. **gRPC 最佳实践**
   - https://grpc.io/docs/guides/performance/
   - Production-Ready Microservices

2. **测试策略**
   - Test-Driven Development with Go
   - Go Testing By Example

3. **性能优化**
   - High Performance Go Workshop
   - Database Performance Tuning

4. **安全**
   - OWASP Top 10
   - Go Security Best Practices

---

## ✅ 检查清单模板

### 每个新功能开发

- [ ] 需求文档
- [ ] 接口设计
- [ ] 单元测试
- [ ] 集成测试
- [ ] 文档更新
- [ ] 代码审查
- [ ] 性能测试
- [ ] 安全审计

---

**维护者**: [@dollarkillerx](https://github.com/dollarkillerx)  
**最后更新**: 2025-10-05
