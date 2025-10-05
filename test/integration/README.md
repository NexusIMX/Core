# 集成测试

本目录包含系统的集成测试,用于验证多个组件之间的交互。

## 前置条件

### 1. 启动测试依赖服务

使用 Docker Compose 启动所需的基础设施:

```bash
cd deployments/docker
docker-compose up -d postgres redis consul minio
```

### 2. 运行数据库迁移

```bash
# 设置环境变量
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=imuser
export POSTGRES_PASSWORD=impassword
export POSTGRES_DB=im_system_test

# 创建测试数据库
psql -h localhost -U postgres -c "CREATE DATABASE im_system_test;"

# 运行迁移
psql -h localhost -U imuser -d im_system_test -f ../../migrations/001_init_schema.sql
```

## 运行集成测试

### 运行所有集成测试
```bash
go test ./test/integration/... -v
```

### 运行特定测试文件
```bash
go test ./test/integration/user_test.go -v
```

### 跳过集成测试(只运行单元测试)
```bash
go test -short ./...
```

## 测试环境配置

集成测试使用以下环境变量:

```bash
# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=imuser
POSTGRES_PASSWORD=impassword
POSTGRES_DB=im_system_test

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Consul
CONSUL_ADDRESS=localhost:8500

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=im-files-test
MINIO_USE_SSL=false

# JWT
JWT_SECRET=test-secret-key-for-integration-tests
```

## 测试覆盖范围

### 当前实现的集成测试

- [ ] **用户服务集成测试** (`user_test.go`)
  - 用户注册 → 数据库存储 → 登录 → Token 验证

- [ ] **路由服务集成测试** (`router_test.go`)
  - 设备注册 → Redis 存储 → 在线状态查询 → 心跳保活

- [ ] **消息服务集成测试** (`message_test.go`)
  - 创建会话 → 发送消息 → 消息存储 → 拉取消息 → 已读状态更新

- [ ] **文件服务集成测试** (`file_test.go`)
  - 文件上传 → S3 存储 → 数据库记录 → 下载 → 预签名 URL → 删除

- [ ] **端到端流程测试** (`e2e_test.go`)
  - 用户注册 → 登录 → 创建会话 → 发送消息(含文件) → 接收消息

### 计划添加的测试

- [ ] gRPC 服务间通信测试
- [ ] Gateway 实时推送测试
- [ ] 消息路由和分发测试
- [ ] 并发场景测试
- [ ] 故障恢复测试

## 最佳实践

### 1. 测试数据隔离

每个测试应该创建和清理自己的测试数据:

```go
func TestUserFlow(t *testing.T) {
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    // 测试逻辑...
}
```

### 2. 使用测试专用数据库

不要使用开发或生产数据库进行集成测试。

### 3. 并发测试

集成测试可能较慢,使用并发执行:

```bash
go test -parallel 4 ./test/integration/...
```

### 4. 跳过条件

使用 `-short` 标志允许快速测试:

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    // 测试逻辑...
}
```

## 故障排查

### 数据库连接失败

```bash
# 检查 PostgreSQL 是否运行
docker-compose ps postgres

# 查看日志
docker-compose logs postgres
```

### Redis 连接失败

```bash
# 检查 Redis 是否运行
docker-compose ps redis

# 测试连接
redis-cli -h localhost -p 6379 ping
```

### MinIO 连接失败

```bash
# 检查 MinIO 是否运行
docker-compose ps minio

# 访问控制台
open http://localhost:9001
```

## CI/CD 集成

在 CI 环境中运行集成测试:

```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: impassword
          POSTGRES_DB: im_system_test
        ports:
          - 5432:5432

      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run integration tests
        env:
          POSTGRES_HOST: localhost
          POSTGRES_PORT: 5432
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: impassword
          POSTGRES_DB: im_system_test
          REDIS_HOST: localhost
          REDIS_PORT: 6379
        run: |
          go test -v ./test/integration/...
```

## 清理测试环境

```bash
# 停止所有服务
cd deployments/docker
docker-compose down

# 删除测试数据和卷
docker-compose down -v

# 删除测试数据库
psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS im_system_test;"
```

---

**注意**: 集成测试需要完整的基础设施环境,确保在运行测试前启动所有依赖服务。
