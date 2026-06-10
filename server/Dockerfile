# OpsMind Server — Go 后端多阶段构建
#
# 阶段 1: 编译 Go 二进制
# 阶段 2: 最小 Alpine 运行时
#
# 构建方式：
#   docker build -t opsmind-server -f server/Dockerfile server/
# 或在项目根目录通过 docker compose 一键构建：
#   docker compose build opsmind-server

# ===== 阶段 1: 编译 =====
FROM golang:1.26-alpine AS builder

# 安装 git（Go module 下载需要）和 ca-certificates
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# 利用 Docker 层缓存：先复制依赖文件
COPY go.mod go.sum ./
# 设置 Go module 代理（国内环境加速）
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /build/opsmind-server ./cmd/main.go

# ===== 阶段 2: 运行时 =====
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从构建阶段复制二进制和配置文件
COPY --from=builder /build/opsmind-server .
COPY --from=builder /build/internal/config/config.yaml ./internal/config/config.yaml

# 暴露 8080 端口
EXPOSE 8080

# 运行服务
ENTRYPOINT ["./opsmind-server"]
