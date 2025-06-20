# 多阶段构建 Dockerfile
# 第一阶段：构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的系统依赖
RUN apk add --no-cache git ca-certificates tzdata && \
    go env -w GOPROXY=https://goproxy.cn

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-occupy main.go

# 第二阶段：运行阶段
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/go-occupy .

# 设置健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ps aux | grep go-occupy || exit 1

# 设置入口点
ENTRYPOINT ["./go-occupy"]

# 默认命令
CMD ["--help"] 