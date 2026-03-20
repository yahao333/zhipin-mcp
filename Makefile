# BOSS直聘 MCP 服务 Makefile

BINARY_NAME=zhipin-mcp
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_RUN=$(GO_CMD) run
GO_TEST=$(GO_CMD) test
GO_VET=$(GO_CMD) vet
GO_LINT=golangci-lint
PKG=github.com/yahao333/zhipin-mcp
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
	@echo ""
	@echo "API 测试命令 (需要先启动服务):"
	@echo "  make api-health           健康检查"
	@echo "  make api-login-status     登录状态"
	@echo "  make api-login-qrcode     获取登录二维码"
	@echo "  make api-login-delete     删除 cookies"
	@echo "  make api-search           搜索职位"
	@echo "  make api-job-detail       职位详情"
	@echo "  make api-deliver          投递简历"
	@echo "  make api-batch-deliver   批量投递"
	@echo "  make api-delivered        已投递列表"
	@echo "  make api-stats            投递统计"
	@echo "  make api-config-get       获取配置"
	@echo "  make api-config-update    更新配置"
	@echo "  make api-cron-start       启动定时任务"
	@echo "  make api-cron-stop        停止定时任务"
	@echo "  make api-all              运行所有 API 测试"

# 默认帮助
.DEFAULT_GOAL := help

# =============================================================================
# API 测试命令 (需要先启动服务)
# =============================================================================

API_BASE := http://localhost:18061

# 健康检查
.PHONY: api-health
api-health:
	@echo "==> 健康检查"
	curl -s $(API_BASE)/api/health

# 登录状态
.PHONY: api-login-status
api-login-status:
	@echo "==> 检查登录状态"
	curl -s $(API_BASE)/api/login/status

# 获取登录二维码
.PHONY: api-login-qrcode
api-login-qrcode:
	@echo "==> 获取登录二维码"
	curl -s $(API_BASE)/api/login/qrcode

# 删除 cookies
.PHONY: api-login-delete
api-login-delete:
	@echo "==> 删除 cookies"
	curl -s -X DELETE $(API_BASE)/api/login/cookies

# 搜索职位
.PHONY: api-search
api-search:
	@echo "==> 搜索职位"
	curl -s -X POST $(API_BASE)/api/jobs/search \
		-H "Content-Type: application/json" \
		-d '{"keyword": "Go开发", "city": "北京"}'

# 职位详情
.PHONY: api-job-detail
api-job-detail:
	@echo "==> 职位详情"
	@echo "注意: job_id 需要替换为真实 ID"
	curl -s $(API_BASE)/api/jobs/job123456

# 投递简历
.PHONY: api-deliver
api-deliver:
	@echo "==> 投递简历"
	@echo "注意: job_id 需要替换为真实 ID"
	curl -s -X POST $(API_BASE)/api/deliver \
		-H "Content-Type: application/json" \
		-d '{"job_id": "job123"}'

# 批量投递
.PHONY: api-batch-deliver
api-batch-deliver:
	@echo "==> 批量投递"
	@echo "注意: job_id 需要替换为真实 ID"
	curl -s -X POST $(API_BASE)/api/batch/deliver \
		-H "Content-Type: application/json" \
		-d '{"job_ids": ["job123", "job456", "job789"]}'

# 已投递列表
.PHONY: api-delivered
api-delivered:
	@echo "==> 已投递列表"
	curl -s "$(API_BASE)/api/delivered?limit=20&offset=0"

# 投递统计
.PHONY: api-stats
api-stats:
	@echo "==> 投递统计"
	curl -s $(API_BASE)/api/stats

# 获取配置
.PHONY: api-config-get
api-config-get:
	@echo "==> 获取配置"
	curl -s $(API_BASE)/api/config

# 更新配置
.PHONY: api-config-update
api-config-update:
	@echo "==> 更新配置 (每日上限 20)"
	curl -s -X PUT $(API_BASE)/api/config \
		-H "Content-Type: application/json" \
		-d '{"max_daily": 20}'

# 启动定时任务
.PHONY: api-cron-start
api-cron-start:
	@echo "==> 启动定时任务"
	curl -s -X POST $(API_BASE)/api/cron/start \
		-H "Content-Type: application/json" \
		-d '{
			"task_name": "每日求职",
			"cron_expression": "0 9 * * *",
			"keyword": "Go后端开发",
			"city": "北京"
		}'

# 停止定时任务
.PHONY: api-cron-stop
api-cron-stop:
	@echo "==> 停止定时任务"
	@echo "注意: task_id 需要替换为真实 ID"
	curl -s -X POST $(API_BASE)/api/cron/stop \
		-H "Content-Type: application/json" \
		-d '{"task_id": 1}'

# 运行所有 API 测试
.PHONY: api-all
api-all:
	@echo "========================================"
	@echo "BOSS直聘 MCP 服务 API 测试"
	@echo "========================================"
	@echo ""
	@$(MAKE) api-health
	@echo ""
	@$(MAKE) api-login-status
	@echo ""
	@$(MAKE) api-login-qrcode
	@echo ""
	@$(MAKE) api-search
	@echo ""
	@$(MAKE) api-delivered
	@echo ""
	@$(MAKE) api-stats
	@echo ""
	@$(MAKE) api-config-get
	@echo ""
	@echo "========================================"
	@echo "API 测试完成"
	@echo "========================================"
