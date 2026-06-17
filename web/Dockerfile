# OpsMind Web — Vue 3 前端多阶段构建
#
# 阶段 1: Node 编译 Vue 项目
# 阶段 2: nginx 运行时，反向代理 /api 到 opsmind-server
#
# 构建方式：
#   docker build -t opsmind-web -f web/Dockerfile web/
# 或在项目根目录通过 docker compose 一键构建：
#   docker compose build opsmind-web

# ===== 阶段 1: 编译 =====
FROM node:18-alpine AS builder

WORKDIR /build

# 利用 Docker 层缓存：先复制依赖文件
COPY package.json package-lock.json* ./
# 使用淘宝镜像加速（国内环境），fallback 到官方源
RUN npm config set registry https://registry.npmmirror.com && \
    npm ci --silent || npm install --silent

# 复制源码并构建
COPY . .
RUN npm run build

# ===== 阶段 2: nginx 运行时 =====
FROM nginx:1.25-alpine

# 复制自定义 nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 从构建阶段复制静态产物
COPY --from=builder /build/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
