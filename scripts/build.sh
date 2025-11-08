#!/bin/bash

# 设置编译环境变量
export GO111MODULE=on
export GOOS=$(go env GOOS)
export GOARCH=$(go env GOARCH)

# 创建构建目录
BUILD_DIR="build"
mkdir -p $BUILD_DIR

# 获取版本信息（如果是 Git 仓库）
VERSION=$(git describe --tags 2>/dev/null || echo "v1.0.0")
BUILD_TIME=$(date "+%F %T")

# 编译参数
LDFLAGS="-X main.Version=$VERSION -X 'main.BuildTime=$BUILD_TIME'"

# 执行编译
echo "🚀 开始编译 Emorad..."
go build -ldflags "$LDFLAGS" -o $BUILD_DIR/emorad

# 检查编译结果
if [ $? -eq 0 ]; then
    echo "✅ 编译成功！"
    echo "📦 二进制文件位置: $BUILD_DIR/emorad"
    echo "📊 文件大小: $(du -h $BUILD_DIR/emorad | cut -f1)"
else
    echo "❌ 编译失败！"
    exit 1
fi 