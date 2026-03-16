# BOSS直聘 MCP 服务 Makefile

BINARY_NAME=zhipin-mcp
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_RUN=$(GO_CMD) run
GO_TEST=$(GO_CMD) test
GO_VET=$(GO_CMD) vet
GO_LINT=golangci-lint
PKG=github.com/xpzouying/zhipin-mcp
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# 默认目标
.PHONY: all
all: build

# 清理构建产物
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf data/*.db

# 下载依赖
.PHONY: deps
deps:
	$(GO_CMD) mod download
	$(GO_CMD) mod tidy

# 构建二进制文件
.PHONY: build
build:
	$(GO_BUILD) $(LDFLAGS) -o $(BINARY_NAME) .

# 构建并支持 CGO (SQLite 需要)
.PHONY: build-cgo
build-cgo:
	CGO_ENABLED=1 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_NAME) .

# 运行服务 (默认无头模式)
.PHONY: run
run:
	$(GO_RUN) .

# 调试模式运行 (非无头)
.PHONY: run-debug
run-debug:
	$(GO_RUN) . -headless=false

# 指定端口运行
.PHONY: run-port
run-port:
	$(GO_RUN) . -port=:18061 -headless=false

# 运行并指定浏览器路径
.PHONY: run-bin
run-bin:
	$(GO_RUN) . -headless=false -bin=/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome

# 运行测试
.PHONY: test
test:
	$(GO_TEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...

# 运行测试 (简洁输出)
.PHONY: test-ci
test-ci:
	$(GO_TEST) -coverprofile=coverage.out -covermode=atomic ./...

# 运行测试并显示覆盖率
.PHONY: test-cover
test-cover:
	$(GO_TEST) -coverprofile=coverage.out ./...
	$(GO_CMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# 代码检查
.PHONY: vet
vet:
	$(GO_VET) ./...

# 代码检查 (详细)
.PHONY: vet-all
vet-all:
	$(GO_VET) -composites=false ./...

# Lint 检查
.PHONY: lint
lint:
	@if command -v $(GO_LINT) > /dev/null 2>&1; then \
		$(GO_LINT) run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# 格式化代码
.PHONY: fmt
fmt:
	$(GO_CMD) fmt ./...

# 安装依赖并构建
.PHONY: install-deps
install-deps: deps build

# 初始化数据库
.PHONY: db-init
db-init:
	@mkdir -p data
	@touch data/zhipin.db

# 查看帮助
.PHONY: help
help:
	@echo "BOSS直聘 MCP 服务 - Makefile"
	@echo ""
	@echo "可用目标:"
	@echo "  make build         构建二进制文件"
	@echo "  make build-cgo     构建二进制文件 (启用 CGO)"
	@echo "  make run           运行服务 (无头模式)"
	@echo "  make run-debug     运行服务 (调试模式，非无头)"
	@echo "  make run-port      运行服务 (指定端口)"
	@echo "  make run-bin       运行服务 (指定浏览器路径)"
	@echo "  make test          运行测试"
	@echo "  make test-cover    运行测试并生成覆盖率报告"
	@echo "  make vet           代码检查"
	@echo "  make lint          Lint 检查"
	@echo "  make fmt           格式化代码"
	@echo "  make clean         清理构建产物"
	@echo "  make deps          下载依赖"
	@echo "  make db-init       初始化数据库目录"
	@echo "  make help          显示帮助信息"

# 默认帮助
.DEFAULT_GOAL := help
