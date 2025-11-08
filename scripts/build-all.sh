#!/bin/bash

# 构建多平台版本脚本
set -e

VERSION="1.0.0"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
OUTPUT_DIR="build"

echo "🚀 开始构建 Emorad v${VERSION}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 清理旧的构建
rm -rf ${OUTPUT_DIR}
mkdir -p ${OUTPUT_DIR}

# 构建参数
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -s -w"

# 构建 Windows 版本
echo ""
echo "📦 构建 Windows amd64..."
GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o ${OUTPUT_DIR}/emorad-windows-amd64.exe
echo "✓ Windows amd64 构建完成"

# 构建 macOS 版本
echo ""
echo "📦 构建 macOS amd64..."
GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o ${OUTPUT_DIR}/emorad-darwin-amd64
echo "✓ macOS amd64 构建完成"

echo ""
echo "📦 构建 macOS arm64 (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o ${OUTPUT_DIR}/emorad-darwin-arm64
echo "✓ macOS arm64 构建完成"

# 构建 Linux 版本
echo ""
echo "📦 构建 Linux amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o ${OUTPUT_DIR}/emorad-linux-amd64
echo "✓ Linux amd64 构建完成"

echo ""
echo "📦 构建 Linux arm64..."
GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o ${OUTPUT_DIR}/emorad-linux-arm64
echo "✓ Linux arm64 构建完成"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 所有平台构建完成！"
echo ""
echo "📁 构建文件位置: ${OUTPUT_DIR}/"
ls -lh ${OUTPUT_DIR}/
echo ""
echo "📊 文件大小统计:"
du -sh ${OUTPUT_DIR}/*
