# OpsMind Makefile — 构建和开发命令
#
# 使用方式：
#   make dev          本地开发启动
#   make build        构建全部 Docker 镜像
#   make up           启动全部服务
#   make down         停止全部服务
#   make test         运行全部测试
#   make migrate      运行数据库迁移
#   make seed         加载演示数据

.PHONY: dev build up down test migrate seed clean help

# 默认目标
help:
	@echo "OpsMind 构建和开发命令"
	@echo ""
	@echo "  make dev          本地开发启动（仅启动依赖服务）"
	@echo "  make build        构建全部 Docker 镜像"
	@echo "  make up           启动全部服务"
	@echo "  make down         停止全部服务"
	@echo "  make test         运行全部测试"
	@echo "  make migrate      运行数据库自动迁移"
	@echo "  make seed         加载演示数据"
	@echo "  make clean        清理构建产物和运行时数据"

# ===== 本地开发 =====

# 启动本地开发所需的依赖服务（PostgreSQL + MinIO）
dev:
	docker compose up -d postgres minio
	@echo "依赖服务已启动"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  MinIO API:  localhost:9000"
	@echo "  MinIO Web:  http://localhost:9001"
	@echo ""
	@echo "接下来手动启动："
	@echo "  cd server && go run ./cmd/main.go"
	@echo "  cd web && npm run dev"

# ===== Docker 构建和启动 =====

# 构建全部镜像
build:
	docker compose build

# 一键启动全部服务（不含 vLLM）
up:
	docker compose up -d --build

# 一键启动含 vLLM 的完整环境
up-ai:
	docker compose --profile ai-local up -d --build

# 停止全部服务
down:
	docker compose down

# 停止并清除数据卷
down-v:
	docker compose down -v

# ===== 测试 =====

# 运行全部测试（非集成）
test:
	cd server && go test ./tests/pkg/... ./tests/middleware/... ./tests/router/... -v

# 运行集成测试（需要 PostgreSQL opsmind_test 库）
test-integration:
	cd server && go test ./tests/... -tags=integration -v

# ===== 数据库 =====

# 数据库自动迁移（启动后端后自动执行，也可手动触发）
migrate:
	cd server && go run ./cmd/main.go &
	sleep 5
	@echo "数据库迁移已通过 AutoMigrate 完成"
	kill %1 2>/dev/null || true

# 加载演示数据（需要先启动 PostgreSQL）
seed:
	docker compose exec -T postgres psql -U opsmind -d opsmind < server/migrations/seed.sql
	@echo "演示数据加载完成"

# ===== 清理 =====

# 清理构建产物和运行时数据
clean:
	docker compose down -v
	rm -rf server/bin/
	rm -rf server/*.exe
	rm -rf web/dist/
	rm -rf web/node_modules/.vite/
	@echo "清理完成"
