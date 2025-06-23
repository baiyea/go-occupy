.PHONY: build run clean help docker-build docker-build-multi docker-push

# 默认目标
.DEFAULT_GOAL := help

# 项目名称
BINARY_NAME=go-occupy

# Docker 配置
DOCKER_IMAGE_NAME=baiyea/go-occupy
DOCKER_TAG=latest

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
	docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .
	@echo "Docker 镜像构建完成: $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"

# Docker 多平台构建
docker-build-multi: ## 构建多平台 Docker 镜像
	@echo "构建多平台 Docker 镜像..."
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) --push .
	@echo "多平台 Docker 镜像构建完成: $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"



# Docker 推送
docker-push: ## 推送 Docker 镜像到 Docker Hub
	@echo "推送 Docker 镜像到 Docker Hub..."
	docker push $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)
	@echo "Docker 镜像推送完成"

# 完整构建流程
all: deps fmt lint-check build ## 完整构建流程

# 帮助信息
help: ## 显示帮助信息
	@echo "Go-Occupy 项目管理工具"
	@echo ""
	@echo "可用命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Docker 命令:"
	@echo "  make docker-build       # 构建 Docker 镜像"
	@echo "  make docker-build-multi # 构建多平台 Docker 镜像"
	@echo "  make docker-push        # 推送 Docker 镜像到 Docker Hub"
	@echo ""
	@echo "示例:"
	@echo "  make build          # 构建可执行文件"
	@echo "  make run            # 运行程序"
	@echo "  make dev            # 开发模式运行"
	@echo "  make clean          # 清理构建文件" 