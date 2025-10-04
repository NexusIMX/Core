# API 使用示例

本文档提供 IM 系统各服务的 API 调用示例。

## 目录
- [User Service](#user-service)
- [Message Service](#message-service)
- [Router Service](#router-service)
- [Gateway Service](#gateway-service)
- [File Service](#file-service)

---

## User Service

### 1. 用户注册

```bash
grpcurl -plaintext -d '{
  "username": "alice",
  "password": "password123",
  "email": "alice@example.com",
  "nickname": "Alice"
}' localhost:50054 user.UserService/Register
```

**响应示例：**
```json
{
  "userId": "1",
  "message": "user registered successfully"
}
```

### 2. 用户登录

```bash
grpcurl -plaintext -d '{
  "username": "alice",
  "password": "password123",
  "device_id": "device-001"
}' localhost:50054 user.UserService/Login
```

**响应示例：**
```json
{
  "userId": "1",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "1696586400",
  "userInfo": {
    "userId": "1",
    "username": "alice",
    "email": "alice@example.com",
    "nickname": "Alice",
    "avatar": "",
    "bio": "",
    "createdAt": "1696500000"
  }
}
```

### 3. 获取用户信息

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{"user_id": "1"}' \
  localhost:50054 user.UserService/GetUserInfo
```

**响应示例：**
```json
{
  "userInfo": {
    "userId": "1",
    "username": "alice",
    "email": "alice@example.com",
    "nickname": "Alice",
    "avatar": "",
    "bio": "",
    "createdAt": "1696500000"
  }
}
```

### 4. 更新用户信息

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{
    "user_id": "1",
    "nickname": "Alice Updated",
    "avatar": "https://example.com/avatar.jpg",
    "bio": "Hello, I am Alice!"
  }' localhost:50054 user.UserService/UpdateUserInfo
```

**响应示例：**
```json
{
  "success": true,
  "message": "user info updated successfully"
}
```

### 5. 验证 Token

```bash
grpcurl -plaintext \
  -d '{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }' localhost:50054 user.UserService/ValidateToken
```

**响应示例：**
```json
{
  "valid": true,
  "userId": "1",
  "deviceId": "device-001"
}
```

---

## Message Service

Message Service 负责消息的存储、检索和会话管理。

### 1. 创建会话

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{
    "type": "DIRECT",
    "title": "Alice and Bob",
    "owner_id": "1",
    "member_ids": ["1", "2"]
  }' localhost:50053 message.MessageService/CreateConversation
```

**响应示例：**
```json
{
  "convId": "1",
  "message": "conversation created successfully"
}
```

### 2. 发送消息

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{
    "conv_id": "1",
    "sender_id": "1",
    "conv_type": "DIRECT",
    "body": {
      "type": "text",
      "content": "Hello, Bob!"
    }
  }' localhost:50053 message.MessageService/SendMessage
```

**响应示例：**
```json
{
  "msgId": "msg-uuid-123",
  "seq": "1",
  "createdAt": "1696500100"
}
```

### 3. 拉取消息

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{
    "conv_id": "1",
    "since_seq": "0",
    "limit": 50
  }' localhost:50053 message.MessageService/PullMessages
```

**响应示例：**
```json
{
  "messages": [
    {
      "msgId": "msg-uuid-123",
      "convId": "1",
      "seq": "1",
      "senderId": "1",
      "convType": "DIRECT",
      "body": {
        "type": "text",
        "content": "Hello, Bob!"
      },
      "visibility": "public",
      "createdAt": "1696500100"
    }
  ],
  "hasMore": false
}
```

### 4. 获取会话信息

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{
    "conv_id": "1"
  }' localhost:50053 message.MessageService/GetConversation
```

**响应示例：**
```json
{
  "conversation": {
    "id": "1",
    "type": "DIRECT",
    "title": "Alice and Bob",
    "ownerId": "1",
    "createdAt": "1696500000",
    "members": [
      {
        "userId": "1",
        "role": "OWNER",
        "muted": false,
        "lastReadSeq": "5",
        "joinedAt": "1696500000"
      },
      {
        "userId": "2",
        "role": "MEMBER",
        "muted": false,
        "lastReadSeq": "3",
        "joinedAt": "1696500000"
      }
    ]
  }
}
```

### 5. 更新已读序列号

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{
    "conv_id": "1",
    "user_id": "2",
    "seq": "5"
  }' localhost:50053 message.MessageService/UpdateReadSeq
```

**响应示例：**
```json
{
  "success": true
}
```

### 6. 发送带 @ 和回复的消息

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{
    "conv_id": "1",
    "sender_id": "1",
    "conv_type": "GROUP",
    "body": {
      "type": "text",
      "content": "@Bob Check this out!"
    },
    "reply_to": "msg-uuid-100",
    "mentions": ["2"]
  }' localhost:50053 message.MessageService/SendMessage
```

---

## Router Service

Router Service 负责管理用户连接路由和在线状态。

### 1. 注册路由（用户上线）

```bash
grpcurl -plaintext \
  -d '{
    "user_id": "1",
    "device_id": "device-001",
    "gateway_addr": "gateway-1:50051"
  }' localhost:50052 router.RouterService/RegisterRoute
```

**响应示例：**
```json
{
  "success": true,
  "message": "route registered successfully"
}
```

### 2. 心跳保活

```bash
grpcurl -plaintext \
  -d '{
    "user_id": "1",
    "device_id": "device-001"
  }' localhost:50052 router.RouterService/KeepAlive
```

**响应示例：**
```json
{
  "success": true
}
```

### 3. 获取用户路由信息

```bash
grpcurl -plaintext \
  -d '{
    "user_id": "1"
  }' localhost:50052 router.RouterService/GetRoute
```

**响应示例：**
```json
{
  "routes": [
    {
      "deviceId": "device-001",
      "gatewayAddr": "gateway-1:50051",
      "lastActive": "1696500200"
    },
    {
      "deviceId": "device-002",
      "gatewayAddr": "gateway-2:50051",
      "lastActive": "1696500150"
    }
  ]
}
```

### 4. 获取在线状态

```bash
grpcurl -plaintext \
  -d '{
    "user_id": "2"
  }' localhost:50052 router.RouterService/GetOnlineStatus
```

**响应示例：**
```json
{
  "online": true,
  "deviceIds": ["device-003", "device-004"]
}
```

### 5. 注销路由（用户下线）

```bash
grpcurl -plaintext \
  -d '{
    "user_id": "1",
    "device_id": "device-001"
  }' localhost:50052 router.RouterService/UnregisterRoute
```

**响应示例：**
```json
{
  "success": true
}
```

---

## Gateway Service

Gateway Service 提供实时双向通信，支持 WebSocket 风格的消息推送。

### 1. 建立双向流连接

使用 gRPC 客户端建立双向流连接：

```python
import grpc
import gateway_pb2
import gateway_pb2_grpc
import time
from google.protobuf.struct_pb2 import Struct

# 创建 channel
channel = grpc.insecure_channel('localhost:50051')
stub = gateway_pb2_grpc.GatewayServiceStub(channel)

# 添加认证 metadata
metadata = [('authorization', 'Bearer YOUR_TOKEN')]

def message_generator():
    # 发送认证消息
    auth_payload = Struct()
    auth_payload.update({"token": "YOUR_TOKEN"})

    yield gateway_pb2.GatewayMessage(
        type=gateway_pb2.MessageType.AUTH,
        payload=auth_payload,
        timestamp=int(time.time())
    )

    # 发送心跳
    while True:
        time.sleep(30)
        yield gateway_pb2.GatewayMessage(
            type=gateway_pb2.MessageType.PING,
            timestamp=int(time.time())
        )

# 建立双向流
responses = stub.Connect(message_generator(), metadata=metadata)

# 接收消息
for response in responses:
    print(f"Received: Type={response.type}, Timestamp={response.timestamp}")
    if response.type == gateway_pb2.MessageType.CHAT:
        print(f"Chat message: {response.payload}")
```

### 2. 发送聊天消息（通过流）

```python
from google.protobuf.struct_pb2 import Struct

# 构造消息负载
chat_payload = Struct()
chat_payload.update({
    "conv_id": 1,
    "conv_type": "direct",
    "body": {
        "type": "text",
        "content": "Hello, Bob!"
    }
})

# 发送聊天消息
chat_message = gateway_pb2.GatewayMessage(
    type=gateway_pb2.MessageType.CHAT,
    payload=chat_payload,
    timestamp=int(time.time()),
    msg_id="client-generated-id-123"
)

# 在双向流中发送
# (需要在 message_generator 函数中 yield 这条消息)
```

### 3. 发送消息（单次 RPC 调用）

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{
    "conv_id": "1",
    "conv_type": "direct",
    "body": {
      "type": "text",
      "content": "Hello!"
    }
  }' localhost:50051 gateway.GatewayService/Send
```

**响应示例：**
```json
{
  "msgId": "msg-uuid-456",
  "seq": "10",
  "createdAt": "1696500300"
}
```

### 4. 同步消息

批量同步多个会话的消息：

```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_TOKEN" \
  -d '{
    "conversations": [
      {
        "conv_id": "1",
        "since_seq": "0"
      },
      {
        "conv_id": "2",
        "since_seq": "5"
      }
    ]
  }' localhost:50051 gateway.GatewayService/Sync
```

**响应示例：**
```json
{
  "convMessages": [
    {
      "convId": "1",
      "messages": [
        {
          "msgId": "msg-uuid-1",
          "convId": "1",
          "seq": "1",
          "senderId": "2",
          "convType": "direct",
          "body": {
            "type": "text",
            "content": "Hi Alice!"
          },
          "createdAt": "1696500100"
        }
      ],
      "hasMore": false
    },
    {
      "convId": "2",
      "messages": [
        {
          "msgId": "msg-uuid-10",
          "convId": "2",
          "seq": "6",
          "senderId": "3",
          "convType": "group",
          "body": {
            "type": "text",
            "content": "Welcome to the group!"
          },
          "createdAt": "1696500200"
        }
      ],
      "hasMore": true
    }
  ]
}
```

### 5. Gateway 消息类型说明

Gateway 支持以下消息类型：

| 类型 | 值 | 说明 | 方向 |
|------|-----|------|------|
| PING | 0 | 心跳请求 | 客户端 → 服务端 |
| PONG | 1 | 心跳响应 | 服务端 → 客户端 |
| AUTH | 2 | 认证消息 | 客户端 → 服务端 |
| CHAT | 3 | 聊天消息 | 双向 |
| NOTIFICATION | 4 | 系统通知 | 服务端 → 客户端 |
| ACK | 5 | 消息确认 | 双向 |
| ERROR | 6 | 错误消息 | 服务端 → 客户端 |
| TYPING | 7 | 正在输入状态 | 双向 |
| READ_RECEIPT | 8 | 已读回执 | 客户端 → 服务端 |
| PRESENCE | 9 | 在线状态变更 | 服务端 → 客户端 |

---

## File Service

File Service 使用 HTTP REST API。

### 1. 上传文件

```bash
curl -X POST http://localhost:8080/v1/files \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@/path/to/your/image.jpg"
```

**响应示例：**
```json
{
  "file_id": "f123e456-7890-12ab-cd34-56ef7890abcd",
  "file_name": "image.jpg",
  "file_size": 1024000,
  "content_type": "image/jpeg",
  "created_at": 1696500200
}
```

### 2. 获取文件信息

```bash
curl http://localhost:8080/v1/files/f123e456-7890-12ab-cd34-56ef7890abcd \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应示例：**
```json
{
  "file_id": "f123e456-7890-12ab-cd34-56ef7890abcd",
  "file_name": "image.jpg",
  "file_size": 1024000,
  "content_type": "image/jpeg",
  "uploader_id": 1,
  "status": "active",
  "created_at": 1696500200
}
```

### 3. 下载文件

```bash
curl http://localhost:8080/v1/files/f123e456-7890-12ab-cd34-56ef7890abcd/download \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o downloaded_file.jpg
```

### 4. 获取预签名下载链接

```bash
curl http://localhost:8080/v1/files/f123e456-7890-12ab-cd34-56ef7890abcd/url \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应示例：**
```json
{
  "url": "http://localhost:9000/im-files/uploads/2024/10/05/f123e456-7890-12ab-cd34-56ef7890abcd.jpg?X-Amz-Algorithm=..."
}
```

### 5. 删除文件

```bash
curl -X DELETE http://localhost:8080/v1/files/f123e456-7890-12ab-cd34-56ef7890abcd \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应示例：**
```json
{
  "message": "file deleted successfully"
}
```

### 6. 获取用户上传的文件列表

```bash
curl "http://localhost:8080/v1/files?limit=20&offset=0" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应示例：**
```json
{
  "files": [
    {
      "file_id": "f123e456-7890-12ab-cd34-56ef7890abcd",
      "file_name": "image.jpg",
      "file_size": 1024000,
      "content_type": "image/jpeg",
      "status": "active",
      "created_at": 1696500200
    }
  ],
  "total": 1
}
```

---

## 完整使用流程示例

### 场景：Alice 给 Bob 发送带图片的消息

```bash
# 1. Alice 注册
grpcurl -plaintext -d '{
  "username": "alice",
  "password": "password123",
  "email": "alice@example.com",
  "nickname": "Alice"
}' localhost:50054 user.UserService/Register

# 2. Bob 注册
grpcurl -plaintext -d '{
  "username": "bob",
  "password": "password456",
  "email": "bob@example.com",
  "nickname": "Bob"
}' localhost:50054 user.UserService/Register

# 3. Alice 登录获取 token
ALICE_TOKEN=$(grpcurl -plaintext -d '{
  "username": "alice",
  "password": "password123",
  "device_id": "alice-device-1"
}' localhost:50054 user.UserService/Login | jq -r '.token')

# 获取 Alice 的 user_id
ALICE_ID=$(grpcurl -plaintext -d '{
  "username": "alice",
  "password": "password123",
  "device_id": "alice-device-1"
}' localhost:50054 user.UserService/Login | jq -r '.userId')

# 4. Bob 登录
BOB_TOKEN=$(grpcurl -plaintext -d '{
  "username": "bob",
  "password": "password456",
  "device_id": "bob-device-1"
}' localhost:50054 user.UserService/Login | jq -r '.token')

# 获取 Bob 的 user_id
BOB_ID=$(grpcurl -plaintext -d '{
  "username": "bob",
  "password": "password456",
  "device_id": "bob-device-1"
}' localhost:50054 user.UserService/Login | jq -r '.userId')

# 5. Alice 创建与 Bob 的会话
CONV_ID=$(grpcurl -plaintext \
  -H "authorization: Bearer $ALICE_TOKEN" \
  -d "{
    \"type\": \"DIRECT\",
    \"title\": \"Alice and Bob\",
    \"owner_id\": \"$ALICE_ID\",
    \"member_ids\": [\"$ALICE_ID\", \"$BOB_ID\"]
  }" localhost:50053 message.MessageService/CreateConversation | jq -r '.convId')

# 6. Alice 上传图片
FILE_ID=$(curl -X POST http://localhost:8080/v1/files \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -F "file=@/path/to/image.jpg" | jq -r '.file_id')

# 7. Alice 通过 Message Service 发送带图片的消息
grpcurl -plaintext \
  -H "authorization: Bearer $ALICE_TOKEN" \
  -d "{
    \"conv_id\": \"$CONV_ID\",
    \"sender_id\": \"$ALICE_ID\",
    \"conv_type\": \"DIRECT\",
    \"body\": {
      \"type\": \"image\",
      \"file_id\": \"$FILE_ID\",
      \"width\": 1920,
      \"height\": 1080
    }
  }" localhost:50053 message.MessageService/SendMessage

# 8. Bob 通过 Gateway 同步消息
grpcurl -plaintext \
  -H "authorization: Bearer $BOB_TOKEN" \
  -d "{
    \"conversations\": [
      {
        \"conv_id\": \"$CONV_ID\",
        \"since_seq\": \"0\"
      }
    ]
  }" localhost:50051 gateway.GatewayService/Sync

# 9. Bob 下载图片
curl "http://localhost:8080/v1/files/$FILE_ID/download" \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -o received_image.jpg

# 10. Bob 标记消息为已读
grpcurl -plaintext \
  -H "authorization: Bearer $BOB_TOKEN" \
  -d "{
    \"conv_id\": \"$CONV_ID\",
    \"user_id\": \"$BOB_ID\",
    \"seq\": \"1\"
  }" localhost:50053 message.MessageService/UpdateReadSeq
```

---

## 健康检查

所有服务都提供健康检查端点：

```bash
# File Service
curl http://localhost:8080/health

# 其他 gRPC 服务可以通过 Consul 查看
curl http://localhost:8500/v1/health/service/user-service
curl http://localhost:8500/v1/health/service/router-service
curl http://localhost:8500/v1/health/service/message-service
curl http://localhost:8500/v1/health/service/gateway-service
curl http://localhost:8500/v1/health/service/file-service
```

---

## 错误处理

所有 API 都遵循统一的错误格式：

**gRPC 错误：**
```json
{
  "code": 16,
  "message": "user not authenticated",
  "details": []
}
```

**HTTP 错误：**
```json
{
  "error": "invalid or expired token"
}
```

常见错误码：
- `UNAUTHENTICATED` (16): 未认证或 token 无效
- `PERMISSION_DENIED` (7): 权限不足
- `NOT_FOUND` (5): 资源不存在
- `INVALID_ARGUMENT` (3): 参数错误
- `INTERNAL` (13): 服务器内部错误
- `ALREADY_EXISTS` (6): 资源已存在
- `UNAVAILABLE` (14): 服务不可用

---

## 附录

### A. 服务端口列表

| 服务 | 端口 | 协议 | 说明 |
|------|------|------|------|
| Gateway | 50051 | gRPC | 网关服务（双向流） |
| Router | 50052 | gRPC | 路由服务 |
| Message | 50053 | gRPC | 消息服务 |
| User | 50054 | gRPC | 用户服务 |
| File | 8080 | HTTP | 文件服务（REST API） |
| Consul | 8500 | HTTP | 服务发现与健康检查 |
| PostgreSQL | 5432 | TCP | 数据库 |
| Redis | 6379 | TCP | 缓存 |
| MinIO | 9000 | HTTP | 对象存储 |
| MinIO Console | 9001 | HTTP | MinIO 管理控制台 |

### B. 消息体格式示例

#### 文本消息
```json
{
  "type": "text",
  "content": "Hello, world!"
}
```

#### 图片消息
```json
{
  "type": "image",
  "file_id": "f123e456-7890-12ab-cd34-56ef7890abcd",
  "width": 1920,
  "height": 1080,
  "thumbnail_url": "https://example.com/thumb.jpg"
}
```

#### 文件消息
```json
{
  "type": "file",
  "file_id": "f789abcd-1234-56ef-7890-abcdef123456",
  "file_name": "document.pdf",
  "file_size": 2048000
}
```

#### 语音消息
```json
{
  "type": "audio",
  "file_id": "fabc1234-5678-90de-f123-456789abcdef",
  "duration": 30
}
```

#### 视频消息
```json
{
  "type": "video",
  "file_id": "fdef5678-90ab-cdef-1234-567890abcdef",
  "duration": 120,
  "width": 1280,
  "height": 720,
  "thumbnail_url": "https://example.com/video_thumb.jpg"
}
```

#### 位置消息
```json
{
  "type": "location",
  "latitude": 39.9042,
  "longitude": 116.4074,
  "address": "Beijing, China"
}
```

### C. 会话类型说明

| 类型 | 枚举值 | 说明 | 特点 |
|------|--------|------|------|
| DIRECT | 0 | 单聊 | 一对一私密对话 |
| GROUP | 1 | 群聊 | 多人群组对话，成员可发消息 |
| CHANNEL | 2 | 频道 | 广播式频道，仅特定角色可发消息 |

### D. 会话成员角色

| 角色 | 枚举值 | 权限 |
|------|--------|------|
| OWNER | 0 | 完全控制权限（创建、删除、管理成员） |
| ADMIN | 1 | 管理权限（管理成员、删除消息） |
| PUBLISHER | 2 | 发布权限（可发送消息） |
| MEMBER | 3 | 普通成员（可发消息、查看消息） |
| VIEWER | 4 | 观察者（只读，不可发消息） |

### E. Proto 定义文件

所有 Proto 定义文件位于 `api/proto/` 目录：

- `api/proto/common/common.proto` - 通用数据结构
- `api/proto/user/user.proto` - 用户服务定义
- `api/proto/message/message.proto` - 消息服务定义
- `api/proto/router/router.proto` - 路由服务定义
- `api/proto/gateway/gateway.proto` - 网关服务定义

详细的字段说明请参考各 proto 文件中的注释（包含中英文双语）。
