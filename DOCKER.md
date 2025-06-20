# Docker 使用说明

本文档介绍如何使用 Docker 来构建和运行 go-occupy 应用。

## 快速开始

### 1. 构建 Docker 镜像

```bash
# 使用默认配置构建（自动推送到 Docker Hub）
make docker-build

# 本地构建（不推送）
make docker-build-local

# 或者直接使用构建脚本
./docker-build.sh

# 构建指定标签
./docker-build.sh -t v1.0.0

# 不使用缓存构建
./docker-build.sh --no-cache

# 构建但不推送
./docker-build.sh --no-push
```

### 2. 运行 Docker 容器

```bash
# 运行容器（显示帮助信息）
make docker-run

# 开发模式运行
make docker-run-dev

# 测试模式运行
make docker-run-test

# 高负载模式运行
make docker-run-high
```

### 3. 使用 Docker Compose

```bash
# 开发模式
make docker-compose-dev

# 测试模式
make docker-compose-test

# 高负载模式
make docker-compose-high

# 启动监控服务
make docker-compose-monitor

# 停止服务
make docker-compose-down
```

## 详细说明

### Dockerfile 特性

- **多阶段构建**: 使用 golang:1.21-alpine 作为构建环境，alpine:latest 作为运行环境
- **安全优化**: 使用非 root 用户运行应用
- **镜像优化**: 最小化镜像大小，只包含必要的运行时依赖
- **健康检查**: 内置健康检查机制
- **时区支持**: 支持亚洲/上海时区

### 构建脚本功能

`docker-build.sh` 脚本提供以下功能：

- 自动检查 Docker 环境
- 支持自定义镜像名称和标签
- 支持多平台构建（linux/amd64, linux/arm64）
- **默认推送到 Docker Hub**
- 支持推送到其他镜像仓库
- 内置容器测试功能
- 彩色日志输出

### 常用命令示例

```bash
# 构建并推送到 Docker Hub（默认）
./docker-build.sh -t v1.0.0

# 构建并推送到指定仓库
./docker-build.sh -t v1.0.0 --push --registry myregistry.com

# 构建多平台镜像
./docker-build.sh --multi-platform

# 构建指定平台镜像
./docker-build.sh --platform linux/arm64

# 本地构建（不推送）
./docker-build.sh --no-push

# 查看帮助信息
./docker-build.sh --help
```

### Docker Hub 推送

默认情况下，镜像会推送到 Docker Hub：

- **镜像名称**: `baiyea/go-occupy`
- **默认标签**: `latest`
- **仓库地址**: `docker.io`

推送前请确保：

1. **已登录 Docker Hub**:
   ```bash
   docker login
   # 或者
   docker login -u YOUR_USERNAME
   ```

2. **有推送权限**: 确保您有权限推送到 `baiyea/go-occupy` 仓库

3. **网络连接正常**: 确保能够访问 Docker Hub

### Docker Compose 配置

`docker-compose.yml` 提供了多种运行模式：

- **开发模式**: 内存30%，CPU20%，磁盘60%
- **测试模式**: 内存50%，CPU30%，磁盘70%
- **高负载模式**: 内存80%，CPU70%，磁盘90%
- **监控模式**: 包含 Prometheus 和 Grafana

### 环境变量

可以通过环境变量自定义配置：

```bash
# 设置镜像名称
export DOCKER_IMAGE_NAME=my-go-occupy

# 设置镜像标签
export DOCKER_TAG=v1.0.0

# 设置镜像仓库（默认为 docker.io）
export DOCKER_REGISTRY=myregistry.com
```

### 资源限制

Docker Compose 配置中设置了资源限制：

- 内存限制: 1GB
- CPU 限制: 1.0 核心
- 内存预留: 512MB
- CPU 预留: 0.5 核心

### 日志管理

- 日志驱动: json-file
- 最大文件大小: 10MB
- 最大文件数量: 3个

### 监控集成

监控模式包含：

- **Prometheus**: 端口 9090，用于指标收集
- **Grafana**: 端口 3000，用于数据可视化
  - 默认用户名: admin
  - 默认密码: admin

## 故障排除

### 常见问题

1. **Docker Hub 登录问题**
   ```bash
   # 检查登录状态
   docker info | grep Username
   
   # 重新登录
   docker logout
   docker login
   ```

2. **推送权限问题**
   ```bash
   # 检查是否有权限推送到目标仓库
   # 确保您有权限推送到 baiyea/go-occupy
   ```

3. **权限问题**
   ```bash
   # 确保当前用户在 docker 组中
   sudo usermod -aG docker $USER
   # 重新登录或重启系统
   ```

4. **端口冲突**
   ```bash
   # 检查端口占用
   netstat -tulpn | grep :9090
   # 修改 docker-compose.yml 中的端口映射
   ```

5. **镜像构建失败**
   ```bash
   # 清理 Docker 缓存
   docker system prune -a
   # 重新构建
   make docker-build-no-cache
   ```

### 调试命令

```bash
# 查看容器日志
docker-compose logs -f go-occupy

# 进入容器调试
docker exec -it go-occupy-app /bin/sh

# 查看容器资源使用
docker stats go-occupy-app

# 检查镜像层
docker history baiyea/go-occupy:latest

# 检查 Docker Hub 镜像
docker pull baiyea/go-occupy:latest
```

## 最佳实践

1. **标签管理**: 使用语义化版本标签
2. **安全扫描**: 定期扫描镜像漏洞
3. **镜像优化**: 定期清理未使用的镜像
4. **监控配置**: 根据实际需求调整资源限制
5. **备份策略**: 定期备份重要数据
6. **Docker Hub 管理**: 定期更新镜像，维护版本标签

## 相关文件

- `Dockerfile`: Docker 镜像构建文件
- `docker-build.sh`: Docker 构建脚本
- `docker-compose.yml`: Docker Compose 配置文件
- `.dockerignore`: Docker 构建忽略文件
- `Makefile`: 包含 Docker 相关命令 