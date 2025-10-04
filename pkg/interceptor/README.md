# gRPC 拦截器

本目录包含所有 gRPC 服务使用的拦截器。

## 📦 拦截器列表

### 1. Auth Interceptor - 认证拦截器

验证 JWT Token 并注入用户信息到 context。

**功能:**
- 从 metadata 中提取 Bearer Token
- 验证 Token 有效性
- 注入 user_id 和 device_id 到 context
- 支持公开方法白名单

**使用示例:**

```go
import (
    "github.com/dollarkillerx/im-system/pkg/auth"
    "github.com/dollarkillerx/im-system/pkg/interceptor"
)

// 创建 JWT Manager
jwtManager := auth.NewJWTManager(secretKey, expiry)

// 定义公开方法（不需要认证）
publicMethods := []string{
    "/user.UserService/Register",
    "/user.UserService/Login",
}

// 创建认证拦截器
authInterceptor := interceptor.NewAuthInterceptor(jwtManager, publicMethods)

// 应用到 gRPC 服务器
server := grpc.NewServer(
    grpc.UnaryInterceptor(authInterceptor.Unary()),
    grpc.StreamInterceptor(authInterceptor.Stream()),
)
```

**从 context 获取用户信息:**

```go
import "github.com/dollarkillerx/im-system/pkg/interceptor"

func (s *MyService) SomeMethod(ctx context.Context, req *SomeRequest) (*SomeResponse, error) {
    // 获取用户 ID
    userID, ok := interceptor.GetUserID(ctx)
    if !ok {
        return nil, fmt.Errorf("user not authenticated")
    }

    // 获取设备 ID
    deviceID, ok := interceptor.GetDeviceID(ctx)

    // 使用用户信息处理业务逻辑
    // ...
}
```

### 2. Logging Interceptor - 日志拦截器

记录所有 RPC 调用的日志。

**功能:**
- 记录方法名、耗时、状态码
- 记录用户 ID 和设备 ID（如果有）
- 区分成功和失败请求
- 支持一元和流式 RPC

**使用示例:**

```go
import "github.com/dollarkillerx/im-system/pkg/interceptor"

server := grpc.NewServer(
    grpc.UnaryInterceptor(interceptor.LoggingUnaryInterceptor()),
    grpc.StreamInterceptor(interceptor.LoggingStreamInterceptor()),
)
```

**日志输出示例:**

```json
{
  "level": "info",
  "timestamp": "2024-01-01T12:00:00Z",
  "message": "gRPC request completed",
  "method": "/user.UserService/GetUserInfo",
  "duration": "15ms",
  "status": "OK",
  "user_id": 12345,
  "device_id": "device-abc-123"
}
```

### 3. Recovery Interceptor - 恢复拦截器

捕获 panic 并返回错误，防止服务崩溃。

**功能:**
- 捕获 panic
- 记录堆栈信息
- 返回 Internal 错误给客户端
- 保持服务稳定运行

**使用示例:**

```go
import "github.com/dollarkillerx/im-system/pkg/interceptor"

server := grpc.NewServer(
    grpc.UnaryInterceptor(interceptor.RecoveryUnaryInterceptor()),
    grpc.StreamInterceptor(interceptor.RecoveryStreamInterceptor()),
)
```

## 🔗 拦截器链

使用 `ChainConfig` 组合多个拦截器：

```go
import (
    "github.com/dollarkillerx/im-system/pkg/auth"
    "github.com/dollarkillerx/im-system/pkg/interceptor"
    "google.golang.org/grpc"
    grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
)

// 创建配置
config := interceptor.ChainConfig{
    JWTManager: jwtManager,
    PublicMethods: []string{
        "/user.UserService/Register",
        "/user.UserService/Login",
    },
    EnableAuth:     true,
    EnableLogging:  true,
    EnableRecovery: true,
}

// 创建拦截器链
unaryInterceptors := interceptor.ChainUnaryInterceptors(config)
streamInterceptors := interceptor.ChainStreamInterceptors(config)

// 创建 gRPC 服务器
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(unaryInterceptors...),
    grpc.ChainStreamInterceptor(streamInterceptors...),
)
```

## 📋 拦截器执行顺序

**一元 RPC 执行顺序:**
1. Recovery (最外层) - 捕获所有 panic
2. Logging - 记录请求日志
3. Auth (最内层) - 验证认证
4. 实际的 Handler

**流式 RPC 执行顺序:**
同一元 RPC

## 🔐 客户端调用示例

### 添加认证 Token

```go
import (
    "context"
    "google.golang.org/grpc/metadata"
)

// 创建带 Token 的 context
func createAuthContext(ctx context.Context, token string) context.Context {
    md := metadata.Pairs("authorization", "Bearer "+token)
    return metadata.NewOutgoingContext(ctx, md)
}

// 调用 RPC
ctx := createAuthContext(context.Background(), "your-jwt-token")
resp, err := client.GetUserInfo(ctx, &GetUserInfoRequest{UserId: 123})
```

### Go 客户端完整示例

```go
package main

import (
    "context"
    "log"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"

    userpb "github.com/dollarkillerx/im-system/api/proto/user"
)

func main() {
    // 连接服务器
    conn, err := grpc.Dial(
        "localhost:50054",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // 创建客户端
    client := userpb.NewUserServiceClient(conn)

    // 1. 登录获取 Token
    loginResp, err := client.Login(context.Background(), &userpb.LoginRequest{
        Username: "testuser",
        Password: "password123",
        DeviceId: "device-001",
    })
    if err != nil {
        log.Fatal(err)
    }

    token := loginResp.Token
    log.Printf("Logged in, token: %s", token)

    // 2. 使用 Token 调用需要认证的接口
    md := metadata.Pairs("authorization", "Bearer "+token)
    ctx := metadata.NewOutgoingContext(context.Background(), md)

    userResp, err := client.GetUserInfo(ctx, &userpb.GetUserInfoRequest{
        UserId: loginResp.UserId,
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("User info: %+v", userResp.UserInfo)
}
```

## 📚 最佳实践

1. **拦截器顺序**: Recovery → Logging → Auth
2. **公开方法**: 只将必要的方法标记为公开（如登录、注册）
3. **错误处理**: 使用 gRPC status 返回规范的错误
4. **日志级别**: 成功用 Info，失败用 Error
5. **Token 传递**: 使用 metadata 而不是请求体

## 🧪 测试

创建测试文件 `interceptor_test.go`:

```go
package interceptor_test

import (
    "context"
    "testing"

    "github.com/dollarkillerx/im-system/pkg/auth"
    "github.com/dollarkillerx/im-system/pkg/interceptor"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
)

func TestAuthInterceptor(t *testing.T) {
    jwtManager := auth.NewJWTManager("test-secret", time.Hour)
    authInt := interceptor.NewAuthInterceptor(jwtManager, []string{"/public.Method"})

    // 测试未提供 token
    ctx := context.Background()
    _, err := authInt.Unary()(ctx, nil, &grpc.UnaryServerInfo{
        FullMethod: "/protected.Method",
    }, func(ctx context.Context, req interface{}) (interface{}, error) {
        return "ok", nil
    })

    if err == nil {
        t.Error("Expected authentication error")
    }

    // 测试有效 token
    token, _ := jwtManager.Generate(123, "device-1")
    md := metadata.Pairs("authorization", "Bearer "+token)
    ctx = metadata.NewIncomingContext(context.Background(), md)

    _, err = authInt.Unary()(ctx, nil, &grpc.UnaryServerInfo{
        FullMethod: "/protected.Method",
    }, func(ctx context.Context, req interface{}) (interface{}, error) {
        userID, ok := interceptor.GetUserID(ctx)
        if !ok || userID != 123 {
            t.Error("User ID not found in context")
        }
        return "ok", nil
    })

    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
}
```
