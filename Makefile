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

.PHONY: dev build up up-ai down down-v test test-integration migrate seed db-reset db-init db-drop clean help model-download

# 默认目标
help:
	@echo "OpsMind 构建和开发命令"
	@echo ""
	@echo "  make dev             本地开发启动（仅启动依赖服务）"
	@echo "  make build           构建全部 Docker 镜像"
	@echo "  make up              一键启动全部服务"
	@echo "  make up-ai           启动含 llama.cpp 的完整 AI 环境"
	@echo "  make down            停止全部服务"
	@echo "  make down-v          停止并清除数据卷"
	@echo "  make test            运行全部测试"
	@echo "  make test-integration 运行集成测试"
	@echo "  make seed            加载演示数据"
	@echo "  make db-reset        清空所有数据（保留表结构）"
	@echo "  make db-init         清空并重新加载演示数据"
	@echo "  make db-drop         仅清空数据"
	@echo "  make model-download  下载 llama.cpp 对话模型和 Embedding 模型（GGUF 格式）"
	@echo "  make clean           清理构建产物和运行时数据"

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

# 一键启动全部服务（4 个必须服务，不含 llama.cpp）
up:
	docker compose up -d --build
	@echo ""
	@echo "============================================"
	@echo "  OpsMind 服务已启动"
	@echo "============================================"
	@echo "  前端:     http://localhost:5173"
	@echo "  后端 API: http://localhost:8080"
	@echo ""
	@echo "  如需启动 llama.cpp: make up-ai"
	@echo "============================================"

# 启动含 llama.cpp 的完整 AI 环境（需要先下载模型: make model-download）
up-ai:
	docker compose --profile ai-local up -d --build
	@echo ""
	@echo "llama.cpp 已启动。如果使用 OpenAI/DeepSeek 等云 API，"
	@echo "只需要 make up 并在 .env 中配置 LLM_BASE_URL 即可。"

# 停止全部服务
down:
	docker compose down

# 停止并清除数据卷
down-v:
	docker compose down -v

# ===== 测试 =====

# 运行全部测试（非集成）
test:
	cd server && go test ./tests/pkg/... ./tests/middleware/... ./tests/router/... ./tests/config/... ./tests/adapter/... -v

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
	docker compose exec -T postgres psql -U opsmind -d opsmind < server/migrations/001_init.sql
	@echo "演示数据加载完成"

# 清空所有数据（保留表结构）
db-reset:
	docker compose exec -T postgres psql -U opsmind -d opsmind < scripts/reset-db.sql
	@echo "数据库已清空"

# 清空并重新加载演示数据（一键初始化数据库）
db-init:
	@bash scripts/seed-db.sh --reset

# 仅清空数据
db-drop:
	@bash scripts/seed-db.sh --drop

# ===== 模型下载 =====

# 下载 llama.cpp GGUF 模型文件（对话模型 + Embedding 模型）
#
# llama.cpp 需要 .gguf 格式的模型文件，而非原始 HuggingFace 模型。
# 最便捷的下载方式是通过 huggingface-cli（需 Python 3.8+）。
#
# 没有 huggingface-cli？手动从 HuggingFace 下载 .gguf 文件到 ./models/ 即可。
model-download:
	@echo "=== 下载 llama.cpp GGUF 模型文件 ==="
	@echo ""
	@mkdir -p models
	@if command -v huggingface-cli >/dev/null 2>&1; then \
		echo "使用 huggingface-cli 下载..."; \
		echo ""; \
		echo "下载对话模型 Qwen3-4B-Instruct Q4_K_M (~2.5 GB)..."; \
		huggingface-cli download bartowski/Qwen3-4B-Instruct-2507-GGUF \
			--include "*Q4_K_M*" --local-dir ./models/ || \
			echo "下载失败，请手动下载 .gguf 文件放到 ./models/"; \
		echo ""; \
		echo "下载 Embedding 模型 BGE-M3 Q4_K_M (~1.5 GB)..."; \
		huggingface-cli download ChristianAzinn/bge-m3-gguf \
			--include "*Q4_K_M*" --local-dir ./models/ || \
			echo "下载失败，请手动下载 .gguf 文件放到 ./models/"; \
	else \
		echo "未安装 huggingface-cli。"; \
		echo ""; \
		echo "安装方式："; \
		echo "  pip install huggingface_hub"; \
		echo ""; \
		echo "然后重新运行: make model-download"; \
	fi
	@echo ""
	@echo "模型文件位于 ./models/ 目录"
	@echo "现在可以运行: make up-ai"

# ===== 清理 =====

# 清理构建产物和运行时数据
clean:
	docker compose down -v
	rm -rf server/bin/
	rm -rf server/*.exe
	rm -rf web/dist/
	rm -rf web/node_modules/.vite/
	@echo "清理完成"
