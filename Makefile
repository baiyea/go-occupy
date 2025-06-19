.PHONY: build run clean test help test-unit test-integration test-coverage test-benchmark

# 默认目标
.DEFAULT_GOAL := help

# 项目名称
BINARY_NAME=go-occupy

# 构建目标
build: ## 构建可执行文件
	@echo "构建 $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) main.go
	@echo "构建完成: $(BINARY_NAME)"

# 运行目标
run: ## 运行程序（使用默认配置）
	@echo "运行 $(BINARY_NAME)..."
	go run main.go

# 开发模式运行
dev: ## 开发模式运行（内存30%，CPU20%，磁盘60%）
	@echo "开发模式运行..."
	go run main.go -m 30 -c 20 -d 60

# 测试模式运行
test-run: ## 测试模式运行（内存50%，CPU30%，磁盘70%）
	@echo "测试模式运行..."
	go run main.go -m 50 -c 30 -d 70

# 高负载模式运行
high-load: ## 高负载模式运行（内存80%，CPU70%，磁盘90%）
	@echo "高负载模式运行..."
	go run main.go -m 80 -c 70 -d 90

# 清理目标
clean: ## 清理构建文件
	@echo "清理构建文件..."
	rm -f $(BINARY_NAME)
	@echo "清理完成"

# 安装依赖
deps: ## 安装项目依赖
	@echo "安装依赖..."
	go mod tidy
	@echo "依赖安装完成"

# 运行所有测试
test: ## 运行所有测试
	@echo "运行所有测试..."
	go test -v ./...
	@echo "测试完成"

# 运行单元测试
test-unit: ## 运行单元测试
	@echo "运行单元测试..."
	go test -v -run "^Test[A-Z]" ./...
	@echo "单元测试完成"

# 运行集成测试
test-integration: ## 运行集成测试
	@echo "运行集成测试..."
	go test -v -run "^TestFull|^TestResource|^TestGraceful|^TestError|^TestPerformance" ./...
	@echo "集成测试完成"

# 运行测试覆盖率
test-coverage: ## 运行测试覆盖率
	@echo "运行测试覆盖率..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 运行基准测试
test-benchmark: ## 运行基准测试
	@echo "运行基准测试..."
	go test -v -bench=. -benchmem ./...
	@echo "基准测试完成"

# 格式化代码
fmt: ## 格式化代码
	@echo "格式化代码..."
	go fmt ./...
	@echo "代码格式化完成"

# 代码检查
lint: ## 运行代码检查
	@echo "运行代码检查..."
	golangci-lint run
	@echo "代码检查完成"

# 代码检查（如果安装了golangci-lint）
lint-check: ## 检查是否安装了golangci-lint
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint 未安装，跳过代码检查"; \
	else \
		echo "运行代码检查..."; \
		golangci-lint run; \
		echo "代码检查完成"; \
	fi

# 完整构建和测试流程
all: deps fmt lint-check test build ## 完整构建和测试流程

# 帮助信息
help: ## 显示帮助信息
	@echo "Go-Occupy 项目管理工具"
	@echo ""
	@echo "可用命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "测试命令:"
	@echo "  make test           # 运行所有测试"
	@echo "  make test-unit      # 运行单元测试"
	@echo "  make test-integration # 运行集成测试"
	@echo "  make test-coverage  # 运行测试覆盖率"
	@echo "  make test-benchmark # 运行基准测试"
	@echo ""
	@echo "示例:"
	@echo "  make build          # 构建可执行文件"
	@echo "  make run            # 运行程序"
	@echo "  make dev            # 开发模式运行"
	@echo "  make clean          # 清理构建文件"
	@echo "  make all            # 完整构建和测试流程" 