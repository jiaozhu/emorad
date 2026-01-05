# Emorad Makefile
# 支持 Windows、macOS (Intel/Arm)、Linux (x86/arm) 多平台编译

# 项目信息
APP_NAME := emorad
VERSION := 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 编译参数
GO := go
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -s -w"

# 输出目录
BUILD_DIR := build

# 平台列表
PLATFORMS := darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows-amd64

# 默认目标
.DEFAULT_GOAL := build

# 帮助信息
.PHONY: help
help:
	@echo "🎯 Emorad 构建工具"
	@echo ""
	@echo "使用方法:"
	@echo "  make build          编译当前平台"
	@echo "  make all            编译所有平台"
	@echo "  make install        安装到系统 (/usr/local/bin)"
	@echo "  make clean          清理构建产物"
	@echo "  make test           运行测试"
	@echo "  make vet            运行代码检查"
	@echo "  make fmt            格式化代码"
	@echo ""
	@echo "平台编译:"
	@echo "  make darwin-amd64   macOS Intel"
	@echo "  make darwin-arm64   macOS Apple Silicon"
	@echo "  make linux-amd64    Linux x86_64"
	@echo "  make linux-arm64    Linux ARM64"
	@echo "  make windows-amd64  Windows x86_64"
	@echo ""
	@echo "版本信息: $(VERSION) ($(GIT_COMMIT))"

# 编译当前平台
.PHONY: build
build:
	@echo "📦 编译当前平台..."
	$(GO) build $(LDFLAGS) -o $(APP_NAME)
	@echo "✅ 编译完成: $(APP_NAME)"

# 安装到系统
.PHONY: install
install: build
	@echo "📦 安装到 /usr/local/bin..."
	@sudo cp $(APP_NAME) /usr/local/bin/$(APP_NAME)
	@echo "✅ 安装完成"

# 清理
.PHONY: clean
clean:
	@echo "🧹 清理构建产物..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(APP_NAME) $(APP_NAME).exe
	@echo "✅ 清理完成"

# 运行测试
.PHONY: test
test:
	@echo "🧪 运行测试..."
	$(GO) test -v ./...

# 代码检查
.PHONY: vet
vet:
	@echo "🔍 运行代码检查..."
	$(GO) vet ./...

# 格式化代码
.PHONY: fmt
fmt:
	@echo "✨ 格式化代码..."
	$(GO) fmt ./...

# 编译所有平台
.PHONY: all
all: clean $(PLATFORMS)
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ 所有平台编译完成!"
	@echo ""
	@echo "📁 构建文件位置: $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/
	@echo ""

# macOS Intel
.PHONY: darwin-amd64
darwin-amd64:
	@echo "📦 编译 macOS Intel (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64
	@echo "✅ macOS Intel 编译完成"

# macOS Apple Silicon
.PHONY: darwin-arm64
darwin-arm64:
	@echo "📦 编译 macOS Apple Silicon (arm64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64
	@echo "✅ macOS Apple Silicon 编译完成"

# Linux x86_64
.PHONY: linux-amd64
linux-amd64:
	@echo "📦 编译 Linux x86_64 (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64
	@echo "✅ Linux x86_64 编译完成"

# Linux ARM64
.PHONY: linux-arm64
linux-arm64:
	@echo "📦 编译 Linux ARM64 (arm64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64
	@echo "✅ Linux ARM64 编译完成"

# Windows x86_64
.PHONY: windows-amd64
windows-amd64:
	@echo "📦 编译 Windows x86_64 (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe
	@echo "✅ Windows x86_64 编译完成"

# 打包发布
.PHONY: release
release: all
	@echo "📦 打包发布文件..."
	@cd $(BUILD_DIR) && for f in $(APP_NAME)-*; do \
		if [ -f "$$f" ]; then \
			tar -czvf "$$f.tar.gz" "$$f" 2>/dev/null || zip "$$f.zip" "$$f"; \
		fi \
	done
	@echo "✅ 打包完成"

# 显示版本
.PHONY: version
version:
	@echo "$(APP_NAME) v$(VERSION) ($(GIT_COMMIT))"
