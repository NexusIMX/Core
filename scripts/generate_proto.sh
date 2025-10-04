#!/bin/bash

# Proto 代码生成脚本
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$SCRIPT_DIR/.."
PROTO_DIR="$PROJECT_ROOT/api/proto"
GO_OUT_DIR="$PROJECT_ROOT/api/proto"

echo "🔨 Generating proto code..."

# 检查 protoc 是否安装
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc not found. Please install protobuf compiler."
    echo "   macOS: brew install protobuf"
    echo "   Linux: apt-get install protobuf-compiler"
    exit 1
fi

# 检查 protoc-gen-go 是否安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo "❌ protoc-gen-go not found. Installing..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# 检查 protoc-gen-go-grpc 是否安装
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "❌ protoc-gen-go-grpc not found. Installing..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 生成 proto 文件
SERVICES="common user router message gateway"

for service in $SERVICES; do
    PROTO_FILE="$PROTO_DIR/$service/$service.proto"

    if [ -f "$PROTO_FILE" ]; then
        echo "📦 Generating $service proto..."

        protoc \
            --proto_path="$PROTO_DIR" \
            --go_out="$GO_OUT_DIR" \
            --go_opt=paths=source_relative \
            --go-grpc_out="$GO_OUT_DIR" \
            --go-grpc_opt=paths=source_relative \
            "$PROTO_FILE"

        echo "✅ Generated $service proto"
    else
        echo "⚠️  $PROTO_FILE not found, skipping..."
    fi
done

echo ""
echo "✅ All proto files generated successfully!"
echo ""
echo "Generated files:"
find "$GO_OUT_DIR" -name "*.pb.go" -type f
