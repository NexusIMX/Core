#!/bin/bash

# Garage 初始化脚本
# 用于创建 bucket 和 access key

set -e

echo "🚀 Initializing Garage S3 storage..."

# 等待 Garage 启动
echo "⏳ Waiting for Garage to be ready..."
sleep 5

# Garage 容器名称
GARAGE_CONTAINER="im-garage"

# 检查容器是否运行
if ! docker ps | grep -q $GARAGE_CONTAINER; then
    echo "❌ Error: Garage container is not running"
    exit 1
fi

echo "✅ Garage container is running"

# 创建节点配置
echo "📝 Configuring Garage node..."
docker exec $GARAGE_CONTAINER garage node id > /tmp/garage_node_id.txt
NODE_ID=$(cat /tmp/garage_node_id.txt | grep -oP '(?<=: )[a-f0-9]+')

echo "Node ID: $NODE_ID"

# 设置节点角色
docker exec $GARAGE_CONTAINER garage layout assign \
    -z dc1 \
    -c 1 \
    $NODE_ID

# 应用布局
docker exec $GARAGE_CONTAINER garage layout apply --version 1

echo "✅ Garage layout configured"

# 创建 access key
echo "🔑 Creating access key..."
docker exec $GARAGE_CONTAINER garage key create im-system-key

# 获取 access key 信息
docker exec $GARAGE_CONTAINER garage key info im-system-key > /tmp/garage_key_info.txt

ACCESS_KEY=$(grep "Key ID" /tmp/garage_key_info.txt | awk '{print $3}')
SECRET_KEY=$(grep "Secret key" /tmp/garage_key_info.txt | awk '{print $3}')

echo "Access Key: $ACCESS_KEY"
echo "Secret Key: $SECRET_KEY"

# 创建 bucket
echo "📦 Creating bucket 'im-files'..."
docker exec $GARAGE_CONTAINER garage bucket create im-files

# 允许 key 访问 bucket
docker exec $GARAGE_CONTAINER garage bucket allow \
    --read \
    --write \
    --owner \
    im-files \
    --key im-system-key

echo "✅ Bucket 'im-files' created and permissions set"

# 输出配置信息
echo ""
echo "================================================"
echo "✅ Garage initialization complete!"
echo "================================================"
echo ""
echo "S3 Configuration:"
echo "  Endpoint: http://localhost:3900"
echo "  Region: garage"
echo "  Bucket: im-files"
echo "  Access Key: $ACCESS_KEY"
echo "  Secret Key: $SECRET_KEY"
echo ""
echo "Update your .env file with these credentials:"
echo "  S3_ENDPOINT=http://garage:3900"
echo "  S3_ACCESS_KEY=$ACCESS_KEY"
echo "  S3_SECRET_KEY=$SECRET_KEY"
echo "================================================"

# 清理临时文件
rm -f /tmp/garage_node_id.txt /tmp/garage_key_info.txt

echo ""
echo "🎉 Done! You can now use Garage for file storage."
