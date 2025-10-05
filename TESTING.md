# 测试文档

本文档详细说明了 IM 系统的测试策略、测试覆盖范围以及如何运行测试。

## 📋 目录

- [测试策略](#测试策略)
- [单元测试](#单元测试)
- [集成测试](#集成测试)
- [运行测试](#运行测试)
- [测试覆盖率](#测试覆盖率)
- [Mock 对象](#mock-对象)
- [最佳实践](#最佳实践)

## 🎯 测试策略

我们的测试策略包含三个层次:

### 1. 单元测试 (Unit Tests)
- 测试单个函数、方法或类的行为
- 使用 Mock 对象隔离依赖
- 快速执行,提供即时反馈
- 覆盖边界条件和错误处理

### 2. 集成测试 (Integration Tests)
- 测试多个组件之间的交互
- 使用真实的数据库、Redis 等依赖
- 验证服务间的 gRPC 调用
- 测试端到端的业务流程

### 3. 端到端测试 (E2E Tests)
- 测试完整的用户场景
- 从客户端到后端的完整流程
- 使用 Docker Compose 启动完整环境

## 🧪 单元测试

### 已覆盖的组件

#### pkg/ 公共包
- ✅ `pkg/auth/jwt_test.go` - JWT Token 生成和验证
- ✅ `pkg/types/enums_test.go` - 枚举类型和权限验证

#### 服务层
- ✅ `internal/user/service_test.go` - 用户注册、登录、信息管理
- ✅ `internal/router/service_test.go` - 路由注册、心跳、在线状态
- ✅ `internal/message/service_test.go` - 消息发送、拉取、会话管理
- ✅ `internal/file/service_test.go` - 文件上传、下载、删除

### 测试示例

#### JWT 认证测试
```go
func TestJWTManager_Generate(t *testing.T) {
    manager := NewJWTManager("test-secret", 1*time.Hour)

    token, err := manager.Generate(123, "device-001")

    assert.NoError(t, err)
    assert.NotEmpty(t, token)
}
```

#### 用户服务测试
```go
func TestService_Register(t *testing.T) {
    repo := newMockRepository()
    jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour)
    service := NewService(repo, jwtManager)

    userID, err := service.Register(ctx, "testuser", "password123",
                                    "test@example.com", "Test User")

    assert.NoError(t, err)
    assert.NotEqual(t, int64(0), userID)
}
```

## 🔗 集成测试

### 创建集成测试目录

```bash
mkdir -p test/integration
```

### 集成测试示例

#### 用户服务集成测试
```go
// test/integration/user_test.go
func TestUserServiceIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // 连接真实数据库
    db := setupTestDatabase(t)
    defer cleanupTestDatabase(t, db)

    repo := user.NewRepository(db)
    jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour)
    service := user.NewService(repo, jwtManager)

    // 测试完整的注册流程
    userID, err := service.Register(context.Background(),
        "testuser", "password123", "test@example.com", "Test User")

    require.NoError(t, err)
    assert.Greater(t, userID, int64(0))

    // 测试登录
    _, token, _, _, err := service.Login(context.Background(),
        "testuser", "password123", "device-001")

    require.NoError(t, err)
    assert.NotEmpty(t, token)
}
```

#### gRPC 服务集成测试
```go
// test/integration/grpc_test.go
func TestGRPCServices(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // 启动测试服务器
    server := startTestServer(t)
    defer server.Stop()

    // 创建客户端连接
    conn := createTestClient(t, server.Address())
    defer conn.Close()

    // 测试 User Service
    userClient := pb.NewUserServiceClient(conn)
    resp, err := userClient.Register(context.Background(), &pb.RegisterRequest{
        Username: "testuser",
        Password: "password123",
        Email:    "test@example.com",
        Nickname: "Test User",
    })

    require.NoError(t, err)
    assert.Greater(t, resp.UserId, int64(0))
}
```

## 🚀 运行测试

### 运行所有测试
```bash
make test
```

### 运行单元测试（跳过集成测试）
```bash
go test -short ./...
```

### 运行特定包的测试
```bash
go test ./pkg/auth/...
go test ./internal/user/...
go test ./internal/router/...
```

### 运行单个测试函数
```bash
go test -run TestJWTManager_Generate ./pkg/auth/
```

### 带详细输出
```bash
go test -v ./...
```

### 并发运行测试
```bash
go test -parallel 4 ./...
```

## 📊 测试覆盖率

### 生成覆盖率报告
```bash
make test-coverage
```

或手动执行:
```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 查看覆盖率
```bash
go tool cover -func=coverage.out
```

### 覆盖率目标
- **pkg/**: 目标 90%+
- **internal/*/service.go**: 目标 85%+
- **internal/*/repository.go**: 目标 80%+
- **整体项目**: 目标 75%+

## 🎭 Mock 对象

### Mock Repository 示例
```go
type MockRepository struct {
    users map[string]*User
    createUserFunc func(ctx context.Context, username, password, email, nickname string) (*User, error)
}

func (m *MockRepository) CreateUser(ctx context.Context, username, password, email, nickname string) (*User, error) {
    if m.createUserFunc != nil {
        return m.createUserFunc(ctx, username, password, email, nickname)
    }
    // 默认实现
    user := &User{
        ID:       int64(len(m.users) + 1),
        Username: username,
        Email:    email,
        Nickname: nickname,
    }
    m.users[username] = user
    return user, nil
}
```

### Mock Redis Client
```go
type MockRedisClient struct {
    data map[string]map[string]string
}

func (m *MockRedisClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
    if m.data[key] == nil {
        m.data[key] = make(map[string]string)
    }
    // 实现 mock 逻辑
    cmd := redis.NewIntCmd(ctx)
    cmd.SetVal(1)
    return cmd
}
```

## 🎓 最佳实践

### 1. 测试命名规范
```go
// 函数: Test{PackageName}_{FunctionName}
func TestUserService_Register(t *testing.T) {}

// 方法: Test{StructName}_{MethodName}
func TestJWTManager_Generate(t *testing.T) {}

// 子测试: 使用描述性名称
t.Run("successful registration", func(t *testing.T) {})
t.Run("duplicate username error", func(t *testing.T) {})
```

### 2. 表驱动测试 (Table-Driven Tests)
```go
func TestConversationType_IsValid(t *testing.T) {
    tests := []struct {
        name     string
        convType ConversationType
        want     bool
    }{
        {"direct conversation", ConversationTypeDirect, true},
        {"group conversation", ConversationTypeGroup, true},
        {"invalid type", ConversationType("invalid"), false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.convType.IsValid()
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 3. 使用 testify 断言库
```go
// 推荐
assert.Equal(t, expected, actual)
assert.NoError(t, err)
require.NotNil(t, obj) // require 失败时立即停止

// 不推荐
if actual != expected {
    t.Errorf("Expected %v, got %v", expected, actual)
}
```

### 4. 测试隔离
```go
func TestUserService(t *testing.T) {
    // 每个测试使用独立的 mock
    t.Run("test case 1", func(t *testing.T) {
        repo := newMockRepository() // 新实例
        service := NewService(repo, jwtManager)
        // 测试...
    })

    t.Run("test case 2", func(t *testing.T) {
        repo := newMockRepository() // 新实例
        service := NewService(repo, jwtManager)
        // 测试...
    })
}
```

### 5. 清理资源
```go
func TestFileUpload(t *testing.T) {
    tmpFile := createTempFile(t)
    defer os.Remove(tmpFile) // 确保清理

    db := setupTestDB(t)
    defer db.Close()

    // 测试...
}
```

### 6. 并发安全测试
```go
func TestConcurrentAccess(t *testing.T) {
    service := NewService()

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // 并发操作
            service.DoSomething()
        }()
    }
    wg.Wait()
}
```

## 🐛 调试测试

### 打印调试信息
```bash
go test -v ./pkg/auth/ 2>&1 | grep "PASS\|FAIL"
```

### 使用 Delve 调试器
```bash
dlv test ./pkg/auth -- -test.run TestJWTManager_Generate
```

### 测试超时设置
```bash
go test -timeout 30s ./...
```

## 📦 持续集成 (CI)

### GitHub Actions 示例
```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## 🔍 测试检查清单

在提交代码前,确保:

- [ ] 所有单元测试通过: `go test ./...`
- [ ] 测试覆盖率达标: `make test-coverage`
- [ ] 代码通过 lint 检查: `make lint`
- [ ] 没有数据竞争: `go test -race ./...`
- [ ] 集成测试通过: `go test ./test/integration/...`
- [ ] 新功能有对应的测试
- [ ] 边界条件和错误处理有测试覆盖

## 📚 相关资源

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify 文档](https://github.com/stretchr/testify)
- [Go Test Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)

---

**维护者**: [@dollarkillerx](https://github.com/dollarkillerx)
**最后更新**: 2025-10-05
