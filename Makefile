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

# =============================================================================
# 构建相关
# =============================================================================

# 清理构建产物
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf data/*.db
	rm -rf dist/

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

# 多平台构建
.PHONY: build-all
build-all:
	@echo "==> 构建多平台版本..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .
	@echo "==> 构建完成: dist/"

# =============================================================================
# 运行相关
# =============================================================================

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

# 指定浏览器路径运行
.PHONY: run-bin
run-bin:
	$(GO_RUN) . -headless=false -bin=/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome

# 扫码登录模式运行 (显示浏览器窗口供扫码)
.PHONY: run-qrcode
run-qrcode:
	$(GO_RUN) . -headless=false

# 后台运行服务
.PHONY: daemon
daemon:
	@echo "==> 启动服务 (后台)..."
	@mkdir -p logs
	nohup ./$(BINARY_NAME) > logs/$(BINARY_NAME).log 2>&1 &
	@echo "==> 服务已启动, PID: $$(pgrep -f $(BINARY_NAME))"

# 停止后台服务
.PHONY: stop
stop:
	@pkill -f $(BINARY_NAME) && echo "==> 服务已停止" || echo "==> 服务未运行"

# =============================================================================
# 测试相关
# =============================================================================

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

# 单元测试 (不含集成测试)
.PHONY: test-unit
test-unit:
	$(GO_TEST) -v -short ./...

# 测试并监听变化
.PHONY: test-watch
test-watch:
	@if command -v fswatch > /dev/null 2>&1; then \
		fswatch -o . | xargs -n1 $(GO_TEST) -short ./...; \
	else \
		echo "fswatch not installed, skipping..."; \
	fi

# =============================================================================
# 代码质量
# =============================================================================

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

# 完整代码质量检查
.PHONY: check
check: vet lint test

# 格式化代码
.PHONY: fmt
fmt:
	$(GO_CMD) fmt ./...

# 安装依赖并构建
.PHONY: install-deps
install-deps: deps build

# =============================================================================
# 数据库相关
# =============================================================================

# 初始化数据库
.PHONY: db-init
db-init:
	@mkdir -p data
	@touch data/zhipin.db

# 查看数据库大小
.PHONY: db-size
db-size:
	@if [ -f data/zhipin.db ]; then \
		du -h data/zhipin.db; \
	else \
		echo "数据库文件不存在"; \
	fi

# =============================================================================
# 开发相关
# =============================================================================

# 设置开发环境
.PHONY: env
env:
	@echo "==> 设置开发环境..."
	@mkdir -p data logs
	@echo "==> 环境设置完成"

# 查看服务状态
.PHONY: status
status:
	@PID=$$(pgrep -f $(BINARY_NAME)); \
	if [ -n "$$PID" ]; then \
		echo "服务运行中, PID: $$PID"; \
		ps -p $$PID -o pid,ppid,cmd,etime; \
	else \
		echo "服务未运行"; \
	fi

# 查看日志
.PHONY: logs
logs:
	@if [ -f logs/$(BINARY_NAME).log ]; then \
		tail -f logs/$(BINARY_NAME).log; \
	else \
		echo "日志文件不存在, 请先使用 make daemon 启动服务"; \
	fi

# 查看最近日志
.PHONY: logs-recent
logs-recent:
	@if [ -f logs/$(BINARY_NAME).log ]; then \
		tail -n 50 logs/$(BINARY_NAME).log; \
	else \
		echo "日志文件不存在"; \
	fi

# =============================================================================
# 帮助信息
# =============================================================================

# 查看帮助
.PHONY: help
help:
	@echo "BOSS直聘 MCP 服务 - Makefile"
	@echo ""
	@echo "用法: make <目标>"
	@echo ""
	@echo "=== 构建 ==="
	@echo "  build          构建二进制文件"
	@echo "  build-cgo      构建二进制文件 (启用 CGO)"
	@echo "  build-all      多平台构建"
	@echo "  clean          清理构建产物"
	@echo ""
	@echo "=== 运行 ==="
	@echo "  run            运行服务 (无头模式)"
	@echo "  run-debug      运行服务 (调试模式，非无头)"
	@echo "  run-port       运行服务 (指定端口)"
	@echo "  run-bin        运行服务 (指定浏览器路径)"
	@echo "  run-qrcode     扫码登录模式 (显示浏览器窗口)"
	@echo "  daemon         后台运行服务"
	@echo "  stop           停止后台服务"
	@echo ""
	@echo "=== 测试 ==="
	@echo "  test           运行测试"
	@echo "  test-ci        运行测试 (CI 模式)"
	@echo "  test-cover     运行测试并生成覆盖率报告"
	@echo "  test-unit      单元测试"
	@echo ""
	@echo "=== 代码质量 ==="
	@echo "  vet            代码检查"
	@echo "  lint           Lint 检查"
	@echo "  check          完整代码质量检查"
	@echo "  fmt            格式化代码"
	@echo ""
	@echo "=== 数据库 ==="
	@echo "  db-init        初始化数据库目录"
	@echo "  db-size        查看数据库大小"
	@echo ""
	@echo "=== 开发工具 ==="
	@echo "  env            设置开发环境"
	@echo "  status         查看服务状态"
	@echo "  logs           查看日志 (实时)"
	@echo "  logs-recent    查看最近日志"
	@echo ""
	@echo "=== API 测试 ==="
	@echo "  api-health           健康检查"
	@echo "  api-login-status     登录状态"
	@echo "  api-login-qrcode     获取登录二维码 (Base64)"
	@echo "  api-login-qrcode-br  扫码登录 (显示浏览器窗口)"
	@echo "  api-login-delete     删除 cookies"
	@echo "  api-search           搜索职位"
	@echo "  api-job-detail       职位详情"
	@echo "  api-deliver          投递简历"
	@echo "  api-batch-deliver   批量投递"
	@echo "  api-delivered        已投递列表"
	@echo "  api-stats            投递统计"
	@echo "  api-config-get       获取配置"
	@echo "  api-config-update    更新配置"
	@echo "  api-cron-start       启动定时任务"
	@echo "  api-cron-stop        停止定时任务"
	@echo "  api-all              运行所有 API 测试"

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

# 获取登录二维码 (Base64)
.PHONY: api-login-qrcode
api-login-qrcode:
	@echo "==> 获取登录二维码 (Base64)"
	curl -s $(API_BASE)/api/login/qrcode

# 扫码登录 (显示浏览器窗口)
.PHONY: api-login-qrcode-br
api-login-qrcode-br:
	@echo "==> 扫码登录 (显示浏览器窗口)"
	@echo "请在弹出的浏览器窗口中扫码登录..."
	curl -s $(API_BASE)/api/login/qrcode/browser

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
		-d '{"keyword": "Go开发"}'

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
	@$(MAKE) api-delivered
	@echo ""
	@$(MAKE) api-stats
	@echo ""
	@$(MAKE) api-config-get
	@echo ""
	@echo "========================================"
	@echo "API 测试完成"
	@echo "========================================"
