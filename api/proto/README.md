# Protocol Buffers 定义

本目录包含所有 gRPC 服务的 Protocol Buffers 定义。

## 📁 目录结构

```
api/proto/
├── common/              # 通用类型定义
│   ├── common.proto
│   └── common.pb.go
├── user/                # 用户服务
│   ├── user.proto
│   ├── user.pb.go
│   └── user_grpc.pb.go
├── router/              # 路由服务
│   ├── router.proto
│   ├── router.pb.go
│   └── router_grpc.pb.go
├── message/             # 消息服务
│   ├── message.proto
│   ├── message.pb.go
│   └── message_grpc.pb.go
└── gateway/             # 网关服务
    ├── gateway.proto
    ├── gateway.pb.go
    └── gateway_grpc.pb.go
```

## 🔨 生成代码

### 使用 Make 命令

```bash
make proto
```

### 使用脚本

```bash
bash scripts/generate_proto.sh
```

### 手动生成

```bash
protoc --proto_path=api/proto \
       --go_out=api/proto --go_opt=paths=source_relative \
       --go-grpc_out=api/proto --go-grpc_opt=paths=source_relative \
       api/proto/user/user.proto
```

## 🔍 验证 Proto 文件

```bash
bash scripts/validate_proto.sh
```

## 📝 服务说明

### Common - 通用类型

通用的响应、分页、错误等类型定义。

**消息类型:**
- `Response` - 通用响应
- `PaginationRequest` - 分页请求
- `PaginationResponse` - 分页响应
- `Error` - 错误信息

### User Service - 用户服务

用户注册、登录、认证相关功能。

**RPC 方法:**
- `Register(RegisterRequest) -> RegisterResponse` - 用户注册
- `Login(LoginRequest) -> LoginResponse` - 用户登录
- `GetUserInfo(GetUserInfoRequest) -> GetUserInfoResponse` - 获取用户信息
- `UpdateUserInfo(UpdateUserInfoRequest) -> UpdateUserInfoResponse` - 更新用户信息
- `ValidateToken(ValidateTokenRequest) -> ValidateTokenResponse` - 验证 Token

**端口:** 50054

### Router Service - 路由服务

管理用户路由、在线状态、多设备连接。

**RPC 方法:**
- `RegisterRoute(RegisterRouteRequest) -> RegisterRouteResponse` - 注册路由
- `KeepAlive(KeepAliveRequest) -> KeepAliveResponse` - 心跳保活
- `GetRoute(GetRouteRequest) -> GetRouteResponse` - 获取路由
- `UnregisterRoute(UnregisterRouteRequest) -> UnregisterRouteResponse` - 注销路由
- `GetOnlineStatus(GetOnlineStatusRequest) -> GetOnlineStatusResponse` - 获取在线状态

**端口:** 50052

### Message Service - 消息服务

消息发送、存储、拉取、会话管理。

**RPC 方法:**
- `SendMessage(SendMessageRequest) -> SendMessageResponse` - 发送消息
- `PullMessages(PullMessagesRequest) -> PullMessagesResponse` - 拉取消息
- `GetConversation(GetConversationRequest) -> GetConversationResponse` - 获取会话
- `CreateConversation(CreateConversationRequest) -> CreateConversationResponse` - 创建会话
- `UpdateReadSeq(UpdateReadSeqRequest) -> UpdateReadSeqResponse` - 更新已读位置
- `NotifyNewMessage(NotifyNewMessageRequest) -> NotifyNewMessageResponse` - 通知新消息

**枚举类型:**
- `ConversationType` - 会话类型 (DIRECT, GROUP, CHANNEL)
- `ConversationRole` - 会话角色 (OWNER, ADMIN, PUBLISHER, MEMBER, VIEWER)

**端口:** 50053

### Gateway Service - 网关服务

客户端接入、消息推送、实时通信。

**RPC 方法:**
- `Connect(stream GatewayMessage) -> stream GatewayMessage` - 建立双向流连接
- `Send(SendRequest) -> SendResponse` - 发送消息
- `Sync(SyncRequest) -> SyncResponse` - 同步消息

**消息类型:**
- `PING/PONG` - 心跳
- `AUTH` - 认证
- `CHAT` - 聊天消息
- `NOTIFICATION` - 通知
- `ACK` - 确认
- `ERROR` - 错误
- `TYPING` - 输入状态
- `READ_RECEIPT` - 已读回执
- `PRESENCE` - 在线状态

**端口:** 50051

## 🎯 消息体设计

### 图文混排消息

消息正文使用 `google.protobuf.Struct` 类型，支持富文本：

```json
{
  "type": "rich_text",
  "content": [
    {"type": "text", "text": "你好，这是一张图片："},
    {
      "type": "image",
      "url": "https://cdn.example.com/img/123.jpg",
      "width": 800,
      "height": 600
    },
    {"type": "text", "text": "图片后面还有文字"}
  ]
}
```

### 纯文本消息

```json
{
  "type": "text",
  "text": "Hello, world!"
}
```

### 图片消息

```json
{
  "type": "image",
  "url": "https://cdn.example.com/img/123.jpg",
  "width": 800,
  "height": 600,
  "thumbnail": "https://cdn.example.com/img/123_thumb.jpg"
}
```

### 文件消息

```json
{
  "type": "file",
  "url": "https://cdn.example.com/files/doc.pdf",
  "filename": "document.pdf",
  "size": 1024000,
  "mime_type": "application/pdf"
}
```

## 🔄 版本管理

- 使用 `proto3` 语法
- 所有字段使用显式编号
- 使用 `optional` 标记可选字段
- 不删除已使用的字段编号
- 新增字段追加到末尾

## 📚 参考资料

- [Protocol Buffers 官方文档](https://protobuf.dev/)
- [gRPC Go 快速开始](https://grpc.io/docs/languages/go/quickstart/)
- [Proto3 语言指南](https://protobuf.dev/programming-guides/proto3/)

## ⚙️ 开发工具

### 安装 protoc

macOS:
```bash
brew install protobuf
```

Linux:
```bash
apt-get install protobuf-compiler
```

### 安装 Go 插件

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

或使用 Make:
```bash
make install-tools
```
