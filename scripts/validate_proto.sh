#!/bin/bash

# Proto 文件验证脚本
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$SCRIPT_DIR/.."
PROTO_DIR="$PROJECT_ROOT/api/proto"

echo "🔍 Validating proto files..."

# 查找所有 proto 文件
PROTO_FILES=$(find "$PROTO_DIR" -name "*.proto" -type f)

if [ -z "$PROTO_FILES" ]; then
    echo "❌ No proto files found in $PROTO_DIR"
    exit 1
fi

echo "Found proto files:"
echo "$PROTO_FILES"
echo ""

# 验证每个 proto 文件
ERROR_COUNT=0

for proto_file in $PROTO_FILES; do
    echo "Checking $proto_file..."

    # 检查语法版本
    if ! grep -q "syntax = \"proto3\"" "$proto_file"; then
        echo "  ❌ Missing or incorrect syntax declaration"
        ((ERROR_COUNT++))
    fi

    # 检查 package 声明
    if ! grep -q "^package " "$proto_file"; then
        echo "  ❌ Missing package declaration"
        ((ERROR_COUNT++))
    fi

    # 检查 go_package 选项
    if ! grep -q "option go_package" "$proto_file"; then
        echo "  ⚠️  Missing go_package option (recommended)"
    fi

    # 尝试编译检查（如果 protoc 可用）
    if command -v protoc &> /dev/null; then
        if protoc --proto_path="$PROTO_DIR" --descriptor_set_out=/dev/null "$proto_file" 2>/dev/null; then
            echo "  ✅ Syntax valid"
        else
            echo "  ❌ Syntax errors detected"
            ((ERROR_COUNT++))
        fi
    fi
done

echo ""

if [ $ERROR_COUNT -eq 0 ]; then
    echo "✅ All proto files are valid!"
    exit 0
else
    echo "❌ Found $ERROR_COUNT error(s)"
    exit 1
fi
