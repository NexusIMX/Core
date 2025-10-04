#!/bin/bash

# 初始化 MinIO bucket
# 使用 mc (MinIO Client)

set -e

echo "🪣 Initializing MinIO bucket..."

# 等待 MinIO 启动
sleep 5

# 配置 MinIO alias
mc alias set local http://localhost:9000 minioadmin minioadmin

# 创建 bucket
mc mb local/im-files --ignore-existing

# 设置 bucket 为公开（可选，根据需求调整）
# mc anonymous set download local/im-files

echo "✅ MinIO bucket 'im-files' created successfully!"
