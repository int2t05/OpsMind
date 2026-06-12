#!/bin/bash
# OpsMind 数据库种子数据加载脚本
#
# 功能：清空所有数据并加载预设演示数据（角色/用户/知识库/申告/消息）
# 依赖：docker compose 服务已启动
#
# 使用方式：
#   bash scripts/seed-db.sh          加载种子数据
#   bash scripts/seed-db.sh --reset  先清空再加载（初始化）
#   bash scripts/seed-db.sh --drop   仅清空所有数据

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SEED_SQL="$PROJECT_ROOT/server/migrations/001_init.sql"
RESET_SQL="$SCRIPT_DIR/reset-db.sql"

cd "$PROJECT_ROOT"

# 检查 postgres 服务是否运行
if ! docker compose ps postgres 2>/dev/null | grep -q "Up"; then
  echo "错误：PostgreSQL 容器未运行，请先执行 docker compose up -d postgres"
  exit 1
fi

PSQL_CMD="docker compose exec -T postgres psql -U opsmind -d opsmind"

case "${1:-}" in
  --drop)
    echo "=== 清空所有数据 ==="
    $PSQL_CMD < "$RESET_SQL"
    echo "数据库已清空。"
    ;;
  --reset)
    echo "=== 清空已有数据 ==="
    $PSQL_CMD < "$RESET_SQL"
    echo ""
    echo "=== 加载种子数据 ==="
    $PSQL_CMD < "$SEED_SQL"
    echo ""
    echo "初始化完成！以下账号可用于登录："
    echo "  admin       / Admin@123    (系统管理员)"
    echo "  operator1   / Operator@123 (运维人员)"
    echo "  operator2   / Operator@123 (运维人员)"
    echo "  knowledge   / Knowledge@123(知识库管理员)"
    echo "  reporter1   / Reporter@123 (报障人)"
    echo "  reporter2   / Reporter@123 (报障人)"
    ;;
  *)
    echo "=== 加载种子数据（保留已有数据）==="
    $PSQL_CMD < "$SEED_SQL"
    echo ""
    echo "种子数据加载完成。"
    echo ""
    echo "如果存在主键冲突，请使用 --reset 先清空再加载："
    echo "  bash scripts/seed-db.sh --reset"
    ;;
esac
