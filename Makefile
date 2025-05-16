
# 设置 Go 编译器
GO := go

# 设置目标可执行文件的名称
BINARY_NAME := k8stools
VERSION := v0.1.0
COMMIT := $(shell git rev-parse --short HEAD)
BUILDTIME := $(shell date +%FT%T%z)
LDFLAGS := "-X 'k8stools/internal/version.Version=$(VERSION)' -X 'k8stools/internal/version.Commit=$(COMMIT)' -X 'k8stools/internal/version.BuildTime=$(BUILDTIME)'"

# 默认目标
all: build-linux build-windows build-mac

# 构建本地二进制并注入版本信息
build:
	@echo "Building local binary with version info..."
	@$(GO) build -ldflags=$(LDFLAGS) -o bin/$(BINARY_NAME)

# 构建 Linux 可执行文件
build-linux:
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 $(GO) build -ldflags=$(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64

# 构建 Windows 可执行文件
build-windows:
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 $(GO) build -ldflags=$(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe

# 构建 macOS 可执行文件
build-mac:
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 $(GO) build -ldflags=$(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64

# 清理生成的文件
clean:
	@echo "Cleaning up..."
	@rm -rf bin/ docs/

# 安装依赖
deps:
	@echo "Installing dependencies..."
	@$(GO) mod tidy

# 运行测试
test:
	@echo "Running tests..."
	@$(GO) test ./...

# 生成 CLI Markdown 帮助文档
gen-docs:
	@echo "Generating CLI docs to ./docs..."
	@mkdir -p docs
	@$(GO) run main.go gen-docs

# 帮助信息
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  all           Build binaries for all platforms"
	@echo "  build         Build local binary with version info"
	@echo "  build-linux   Build binary for Linux"
	@echo "  build-windows Build binary for Windows"
	@echo "  build-mac     Build binary for macOS"
	@echo "  clean         Clean up generated files"
	@echo "  deps          Install dependencies"
	@echo "  test          Run tests"
	@echo "  gen-docs      Generate CLI markdown documentation"
	@echo "  help          Show this help message"
