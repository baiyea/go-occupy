.PHONY: build run clean test help test-unit test-integration test-coverage test-benchmark docker-build docker-run docker-clean docker-push docker-push-custom

# 默认目标
.DEFAULT_GOAL := help

# 项目名称
BINARY_NAME=go-occupy

# Docker 配置
DOCKER_IMAGE_NAME=baiyea/go-occupy
DOCKER_TAG=latest
DOCKER_REGISTRY=docker.io

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

# Docker 构建
docker-build: ## 构建 Docker 镜像
	@echo "构建 Docker 镜像..."
	@chmod +x docker-build.sh
	./docker-build.sh -n $(DOCKER_IMAGE_NAME) -t $(DOCKER_TAG)
	@echo "Docker 镜像构建完成: $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"

# Docker 构建（无缓存）
docker-build-no-cache: ## 构建 Docker 镜像（无缓存）
	@echo "构建 Docker 镜像（无缓存）..."
	@chmod +x docker-build.sh
	./docker-build.sh -n $(DOCKER_IMAGE_NAME) -t $(DOCKER_TAG) --no-cache
	@echo "Docker 镜像构建完成: $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"

# Docker 构建（不推送）
docker-build-local: ## 构建 Docker 镜像（本地构建，不推送）
	@echo "构建 Docker 镜像（本地构建）..."
	@chmod +x docker-build.sh
	./docker-build.sh -n $(DOCKER_IMAGE_NAME) -t $(DOCKER_TAG) --no-push
	@echo "Docker 镜像构建完成: $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"

# Docker 多平台构建
docker-build-multi: ## 构建多平台 Docker 镜像
	@echo "构建多平台 Docker 镜像..."
	@chmod +x docker-build.sh
	./docker-build.sh -n $(DOCKER_IMAGE_NAME) -t $(DOCKER_TAG) --multi-platform
	@echo "多平台 Docker 镜像构建完成: $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"

# Docker 运行
docker-run: ## 运行 Docker 容器
	@echo "运行 Docker 容器..."
	docker run --rm -it $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)

# Docker 开发模式运行
docker-run-dev: ## 开发模式运行 Docker 容器
	@echo "开发模式运行 Docker 容器..."
	docker run --rm -it $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) -m 30 -c 20 -d 60

# Docker 测试模式运行
docker-run-test: ## 测试模式运行 Docker 容器
	@echo "测试模式运行 Docker 容器..."
	docker run --rm -it $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) -m 50 -c 30 -d 70

# Docker 高负载模式运行
docker-run-high: ## 高负载模式运行 Docker 容器
	@echo "高负载模式运行 Docker 容器..."
	docker run --rm -it $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) -m 80 -c 70 -d 90

# Docker Compose 开发模式
docker-compose-dev: ## 使用 Docker Compose 开发模式运行
	@echo "启动开发模式..."
	docker-compose --profile dev up -d
	@echo "开发模式已启动，使用 'docker-compose logs -f' 查看日志"

# Docker Compose 测试模式
docker-compose-test: ## 使用 Docker Compose 测试模式运行
	@echo "启动测试模式..."
	docker-compose --profile test up -d
	@echo "测试模式已启动，使用 'docker-compose logs -f' 查看日志"

# Docker Compose 高负载模式
docker-compose-high: ## 使用 Docker Compose 高负载模式运行
	@echo "启动高负载模式..."
	docker-compose --profile high-load up -d
	@echo "高负载模式已启动，使用 'docker-compose logs -f' 查看日志"

# Docker Compose 监控模式
docker-compose-monitor: ## 使用 Docker Compose 启动监控服务
	@echo "启动监控服务..."
	docker-compose --profile monitor up -d
	@echo "监控服务已启动:"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Grafana: http://localhost:3000 (admin/admin)"

# Docker Compose 停止
docker-compose-down: ## 停止 Docker Compose 服务
	@echo "停止 Docker Compose 服务..."
	docker-compose down
	@echo "服务已停止"

# Docker 清理
docker-clean: ## 清理 Docker 镜像和容器
	@echo "清理 Docker 资源..."
	docker rmi $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) 2>/dev/null || true
	docker container prune -f
	docker image prune -f
	@echo "Docker 清理完成"

# Docker 推送
docker-push: ## 推送 Docker 镜像到仓库
	@echo "推送 Docker 镜像到 $(DOCKER_REGISTRY)..."
	@chmod +x docker-build.sh
	./docker-build.sh -n $(DOCKER_IMAGE_NAME) -t $(DOCKER_TAG) --push --registry $(DOCKER_REGISTRY)
	@echo "Docker 镜像推送完成"

# Docker 推送（指定仓库）
docker-push-custom: ## 推送 Docker 镜像到指定仓库
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "错误: 请设置 DOCKER_REGISTRY 环境变量"; \
		echo "示例: make docker-push-custom DOCKER_REGISTRY=myregistry.com"; \
		exit 1; \
	fi
	@echo "推送 Docker 镜像到 $(DOCKER_REGISTRY)..."
	@chmod +x docker-build.sh
	./docker-build.sh -n $(DOCKER_IMAGE_NAME) -t $(DOCKER_TAG) --push --registry $(DOCKER_REGISTRY)
	@echo "Docker 镜像推送完成"

# 完整构建和测试流程
all: deps fmt lint-check test build ## 完整构建和测试流程

# Docker 完整流程
docker-all: docker-build docker-run ## Docker 完整构建和运行流程

# 帮助信息
help: ## 显示帮助信息
	@echo "Go-Occupy 项目管理工具"
	@echo ""
	@echo "可用命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "测试命令:"
	@echo "  make test           # 运行所有测试"
	@echo "  make test-unit      # 运行单元测试"
	@echo "  make test-integration # 运行集成测试"
	@echo "  make test-coverage  # 运行测试覆盖率"
	@echo "  make test-benchmark # 运行基准测试"
	@echo ""
	@echo "Docker 命令:"
	@echo "  make docker-build      # 构建 Docker 镜像（默认推送到 Docker Hub）"
	@echo "  make docker-build-local # 构建 Docker 镜像（本地构建，不推送）"
	@echo "  make docker-run        # 运行 Docker 容器"
	@echo "  make docker-clean      # 清理 Docker 资源"
	@echo "  make docker-push       # 推送 Docker 镜像到 Docker Hub"
	@echo "  make docker-push-custom # 推送 Docker 镜像到指定仓库"
	@echo ""
	@echo "Docker Compose 命令:"
	@echo "  make docker-compose-dev    # 开发模式"
	@echo "  make docker-compose-test   # 测试模式"
	@echo "  make docker-compose-high   # 高负载模式"
	@echo "  make docker-compose-monitor # 监控模式"
	@echo ""
	@echo "示例:"
	@echo "  make build          # 构建可执行文件"
	@echo "  make run            # 运行程序"
	@echo "  make dev            # 开发模式运行"
	@echo "  make clean          # 清理构建文件"
	@echo "  make all            # 完整构建和测试流程"
	@echo "  make docker-all     # Docker 完整流程" 