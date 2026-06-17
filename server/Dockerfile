# OpsMind Server — Go 后端多阶段构建
#
# 阶段 1: 编译 Go 二进制
# 阶段 2: Python slim 运行时（Go 二进制 + Python cross-encoder 推理）
#
# 构建方式：
#   docker build -t opsmind-server -f server/Dockerfile server/
# 或在项目根目录通过 docker compose 一键构建：
#   docker compose build opsmind-server

# ===== 阶段 1: 编译 Go =====
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# 利用 Docker 层缓存：先复制依赖文件
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /build/opsmind-server ./cmd/main.go

# ===== 阶段 2: 运行时（Python 3.11 slim + Go 二进制）=====
# 选择 python slim 而非 alpine：PyTorch 对 Debian/glibc 开箱即用，
# 避免 Alpine musl 需要特殊 wheel 的问题。
FROM python:3.11-slim-bookworm

# Go 运行时依赖
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Shanghai

WORKDIR /app

# 安装 Python 依赖（sentence-transformers + PyTorch CPU 版）
# --no-cache-dir 减少镜像体积
RUN pip install --no-cache-dir \
    torch --index-url https://download.pytorch.org/whl/cpu \
    sentence-transformers

# 预下载 rerank 模型到镜像（避免容器启动时网络依赖）
# 使用 huggingface_hub 直接下载，比 sentence-transformers 更可控
RUN python -c "\
from sentence_transformers import CrossEncoder; \
CrossEncoder('BAAI/bge-reranker-base', device='cpu')"

# 从构建阶段复制 Go 二进制和配置
COPY --from=builder /build/opsmind-server .
COPY --from=builder /build/internal/config/config.yaml ./internal/config/config.yaml

# 复制 Python 推理脚本
COPY rerank_server.py .

# 暴露 8080 端口
EXPOSE 8080

# 运行服务
ENTRYPOINT ["./opsmind-server"]
