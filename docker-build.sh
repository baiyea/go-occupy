#!/bin/bash

# Docker 构建脚本
# 用于构建 go-occupy 应用的 Docker 镜像

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
IMAGE_NAME="baiyea/go-occupy"
DEFAULT_TAG="latest"
DOCKERFILE="Dockerfile"
BUILD_CONTEXT="."
DEFAULT_REGISTRY="docker.io"

# 显示帮助信息
show_help() {
    echo -e "${BLUE}Go-Occupy Docker 构建脚本${NC}"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -n, --name NAME     镜像名称 (默认: baiyea/go-occupy)"
    echo "  -t, --tag TAG       镜像标签 (默认: latest)"
    echo "  -f, --file FILE     Dockerfile 路径 (默认: Dockerfile)"
    echo "  -c, --context DIR   构建上下文目录 (默认: .)"
    echo "  --no-cache          不使用缓存构建"
    echo "  --push              构建后推送到镜像仓库 (默认: Docker Hub)"
    echo "  --registry REGISTRY 镜像仓库地址 (默认: docker.io)"
    echo "  --platform PLATFORM 目标平台 (例如: linux/amd64,linux/arm64)"
    echo "  --multi-platform    构建多平台镜像"
    echo "  --no-push           构建但不推送镜像"
    echo "  -h, --help          显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                                    # 使用默认配置构建"
    echo "  $0 -t v1.0.0                         # 构建指定标签"
    echo "  $0 --no-cache                        # 不使用缓存构建"
    echo "  $0 --push                            # 构建并推送到 Docker Hub"
    echo "  $0 --push --registry myregistry.com  # 构建并推送到指定仓库"
    echo "  $0 --multi-platform                  # 构建多平台镜像"
    echo "  $0 --no-push                         # 构建但不推送"
}

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Docker 是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装或不在 PATH 中"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker 守护进程未运行或权限不足"
        exit 1
    fi
}

# 检查 Dockerfile 是否存在
check_dockerfile() {
    if [[ ! -f "$DOCKERFILE" ]]; then
        log_error "Dockerfile 不存在: $DOCKERFILE"
        exit 1
    fi
}

# 构建镜像
build_image() {
    local build_args=""
    local platform_arg=""
    
    # 添加构建参数
    if [[ "$NO_CACHE" == "true" ]]; then
        build_args="$build_args --no-cache"
    fi
    
    if [[ -n "$PLATFORM" ]]; then
        platform_arg="--platform $PLATFORM"
    fi
    
    log_info "开始构建镜像: $IMAGE_NAME:$TAG"
    log_info "Dockerfile: $DOCKERFILE"
    log_info "构建上下文: $BUILD_CONTEXT"
    
    if [[ -n "$platform_arg" ]]; then
        log_info "目标平台: $PLATFORM"
    fi
    
    # 执行构建命令
    docker build $build_args $platform_arg \
        -f "$DOCKERFILE" \
        -t "$IMAGE_NAME:$TAG" \
        "$BUILD_CONTEXT"
    
    if [[ $? -eq 0 ]]; then
        log_info "镜像构建成功: $IMAGE_NAME:$TAG"
        
        # 显示镜像信息
        docker images "$IMAGE_NAME:$TAG" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
    else
        log_error "镜像构建失败"
        exit 1
    fi
}

# 构建多平台镜像
build_multi_platform() {
    log_info "开始构建多平台镜像: $IMAGE_NAME:$TAG"
    
    # 检查是否支持 buildx
    if ! docker buildx version &> /dev/null; then
        log_error "Docker buildx 不可用，请升级 Docker 版本"
        exit 1
    fi
    
    # 创建并使用新的构建器（如果需要）
    if ! docker buildx inspect multi-platform &> /dev/null; then
        log_info "创建多平台构建器..."
        docker buildx create --name multi-platform --use
    else
        docker buildx use multi-platform
    fi
    
    # 构建多平台镜像
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        -f "$DOCKERFILE" \
        -t "$IMAGE_NAME:$TAG" \
        --push \
        "$BUILD_CONTEXT"
    
    if [[ $? -eq 0 ]]; then
        log_info "多平台镜像构建成功: $IMAGE_NAME:$TAG"
    else
        log_error "多平台镜像构建失败"
        exit 1
    fi
}

# 推送镜像
push_image() {
    # 设置默认仓库
    local registry=${REGISTRY:-$DEFAULT_REGISTRY}
    
    # 构建完整的镜像名称
    local full_image_name
    if [[ "$registry" == "docker.io" ]]; then
        # Docker Hub 不需要前缀
        full_image_name="$IMAGE_NAME:$TAG"

        # 检查是否已登录 Docker Hub
        if ! docker info | grep -q "Username"; then
            echo "请先登录 Docker Hub:"
            docker login
        fi
    else
        full_image_name="$registry/$IMAGE_NAME:$TAG"
    fi
    
    log_info "标记镜像: $IMAGE_NAME:$TAG -> $full_image_name"
    docker tag "$IMAGE_NAME:$TAG" "$full_image_name"
    
    log_info "推送镜像到: $full_image_name"
    
    # 检查是否已登录到目标仓库
    if [[ "$registry" == "docker.io" ]]; then
        if ! docker info | grep -q "Username"; then
            log_warn "未检测到 Docker Hub 登录信息"
            log_info "请先运行: docker login"
            log_info "或者使用: docker login -u YOUR_USERNAME"
        fi
    fi
    
    docker push "$full_image_name"
    
    if [[ $? -eq 0 ]]; then
        log_info "镜像推送成功: $full_image_name"
        
        # 显示推送后的镜像信息
        if [[ "$registry" == "docker.io" ]]; then
            log_info "镜像已推送到 Docker Hub: https://hub.docker.com/r/$IMAGE_NAME"
        fi
    else
        log_error "镜像推送失败"
        log_error "请检查:"
        log_error "1. 网络连接是否正常"
        log_error "2. 是否已登录到目标仓库 (docker login)"
        log_error "3. 是否有推送权限"
        exit 1
    fi
}

# 运行测试容器
test_container() {
    log_info "运行测试容器..."
    
    # 创建临时容器进行测试
    local container_id=$(docker run -d --rm "$IMAGE_NAME:$TAG" --help)
    
    if [[ $? -eq 0 ]]; then
        log_info "测试容器启动成功，容器ID: $container_id"
        
        # 等待几秒钟让容器完全启动
        sleep 3
        
        # 检查容器状态
        if docker ps | grep -q "$container_id"; then
            log_info "容器运行正常"
            docker logs "$container_id"
        fi
        
        # 停止并删除测试容器
        docker stop "$container_id" &> /dev/null || true
        log_info "测试完成"
    else
        log_error "测试容器启动失败"
        exit 1
    fi
}

# 主函数
main() {
    # 默认启用推送
    PUSH=${PUSH:-true}
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--name)
                IMAGE_NAME="$2"
                shift 2
                ;;
            -t|--tag)
                TAG="$2"
                shift 2
                ;;
            -f|--file)
                DOCKERFILE="$2"
                shift 2
                ;;
            -c|--context)
                BUILD_CONTEXT="$2"
                shift 2
                ;;
            --no-cache)
                NO_CACHE="true"
                shift
                ;;
            --push)
                PUSH="true"
                shift
                ;;
            --no-push)
                PUSH="false"
                shift
                ;;
            --registry)
                REGISTRY="$2"
                shift 2
                ;;
            --platform)
                PLATFORM="$2"
                shift 2
                ;;
            --multi-platform)
                MULTI_PLATFORM="true"
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 设置默认标签
    TAG=${TAG:-$DEFAULT_TAG}
    
    # 检查前置条件
    check_docker
    check_dockerfile
    
    # 显示构建信息
    log_info "=== Go-Occupy Docker 构建 ==="
    log_info "镜像名称: $IMAGE_NAME"
    log_info "镜像标签: $TAG"
    log_info "Dockerfile: $DOCKERFILE"
    log_info "构建上下文: $BUILD_CONTEXT"
    
    if [[ "$PUSH" == "true" ]]; then
        local registry=${REGISTRY:-$DEFAULT_REGISTRY}
        log_info "推送目标: $registry"
    else
        log_info "推送: 禁用"
    fi
    
    if [[ "$MULTI_PLATFORM" == "true" ]]; then
        build_multi_platform
    else
        build_image
    fi
    
    # 运行测试（可选）
    if [[ "$TEST" != "false" ]]; then
        test_container
    fi
    
    # 推送镜像（如果需要）
    if [[ "$PUSH" == "true" ]]; then
        push_image
    fi
    
    log_info "构建流程完成！"
}

# 运行主函数
main "$@" 