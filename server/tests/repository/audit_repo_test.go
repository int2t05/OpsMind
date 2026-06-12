//go:build integration

// Package repository_test 验证 AuditRepo 数据访问层。
//
// 测试覆盖 PLAN.md Task33 定义的审计日志查询和写入功能：
// Create / List（按操作人/操作类型筛选/分页）。
package repository_test

import (
	"testing"
	"time"

	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/model"
	"opsmind/internal/repository"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// =============================================================================
// 测试基础设施
// =============================================================================

var auditRepoDB *gorm.DB

func init() {
	cfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, User: "opsmind", Password: "opsmind_dev",
		DBName: "opsmind_test", SSLMode: "disable",
	}
	db, err := database.Init(cfg)
	if err != nil {
		panic(err)
	}
	auditRepoDB = db
}

func setupAuditRepoTest(t *testing.T) *repository.AuditRepo {
	t.Helper()

	// 创建需要的表
	auditRepoDB.Exec(`CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY, username VARCHAR(64) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL, real_name VARCHAR(64) NOT NULL,
		phone VARCHAR(11) NOT NULL, email VARCHAR(128),
		status SMALLINT NOT NULL DEFAULT 1, first_login BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	auditRepoDB.Exec(`CREATE TABLE IF NOT EXISTS audit_logs (
		id BIGSERIAL PRIMARY KEY, operator_id BIGINT NOT NULL,
		action VARCHAR(64) NOT NULL, target_type VARCHAR(32),
		target_id BIGINT, detail JSONB, ip_address VARCHAR(45),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	// 清理旧数据
	auditRepoDB.Exec("DELETE FROM audit_logs")
	auditRepoDB.Exec("DELETE FROM users")

	// 创建测试用户（供 JOIN 查询操作人姓名）
	auditRepoDB.Exec(
		`INSERT INTO users (id, username, password_hash, real_name, phone, status, first_login)
		 VALUES (1, 'admin', '$2a$10$test', '系统管理员', '13800000001', 1, false)`)
	auditRepoDB.Exec(
		`INSERT INTO users (id, username, password_hash, real_name, phone, status, first_login)
		 VALUES (2, 'operator', '$2a$10$test', '运维人员', '13800000002', 1, false)`)

	return repository.NewAuditRepo(auditRepoDB)
}

func seedAuditLogs(t *testing.T, repo *repository.AuditRepo) {
	t.Helper()

	logs := []model.AuditLog{
		{
			OperatorID: 1, Action: "user.create", TargetType: "user", TargetID: 10,
			Detail:    datatypes.JSON(`{"username":"test1"}`),
			IPAddress: "192.168.1.1", CreatedAt: time.Now(),
		},
		{
			OperatorID: 1, Action: "user.update", TargetType: "user", TargetID: 11,
			Detail:    datatypes.JSON(`{"username":"test2","field":"real_name"}`),
			IPAddress: "192.168.1.1", CreatedAt: time.Now(),
		},
		{
			OperatorID: 2, Action: "ticket.create", TargetType: "ticket", TargetID: 100,
			Detail:    datatypes.JSON(`{"title":"服务器宕机"}`),
			IPAddress: "10.0.0.1", CreatedAt: time.Now(),
		},
		{
			OperatorID: 1, Action: "knowledge.publish", TargetType: "knowledge", TargetID: 50,
			Detail:    datatypes.JSON(`{"kb_id":1,"article_id":5}`),
			IPAddress: "192.168.1.1", CreatedAt: time.Now().Add(-time.Hour),
		},
		{
			OperatorID: 2, Action: "user.freeze", TargetType: "user", TargetID: 12,
			Detail:    datatypes.JSON(`{"username":"test3","reason":"安全原因"}`),
			IPAddress: "10.0.0.2", CreatedAt: time.Now().Add(-2 * time.Hour),
		},
	}

	for _, log := range logs {
		if err := repo.Create(&log); err != nil {
			t.Fatalf("创建测试审计日志失败: %v", err)
		}
	}
}

// =============================================================================
// Create
// =============================================================================

// TestAuditRepo_Create 验证审计日志写入后可通过 ID 查询。
func TestAuditRepo_Create(t *testing.T) {
	repo := setupAuditRepoTest(t)

	log := &model.AuditLog{
		OperatorID: 1,
		Action:     "user.login",
		TargetType: "user",
		TargetID:   1,
		Detail:     datatypes.JSON(`{"ip":"127.0.0.1"}`),
		IPAddress:  "127.0.0.1",
	}

	err := repo.Create(log)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if log.ID == 0 {
		t.Error("期望 ID 被填充, got 0")
	}
}

// =============================================================================
// List
// =============================================================================

// TestAuditRepo_List_All 验证不带筛选条件的全量分页查询。
func TestAuditRepo_List_All(t *testing.T) {
	repo := setupAuditRepoTest(t)
	seedAuditLogs(t, repo)

	logs, total, err := repo.List(0, "", 1, 10)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if total != 5 {
		t.Errorf("期望 total=5, got %d", total)
	}
	if len(logs) != 5 {
		t.Errorf("期望 5 条记录, got %d", len(logs))
	}
}

// TestAuditRepo_List_ByOperator 验证按操作人 ID 筛选。
func TestAuditRepo_List_ByOperator(t *testing.T) {
	repo := setupAuditRepoTest(t)
	seedAuditLogs(t, repo)

	logs, total, err := repo.List(1, "", 1, 10)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if total != 3 {
		t.Errorf("操作人 1: 期望 total=3, got %d", total)
	}
	if len(logs) != 3 {
		t.Errorf("操作人 1: 期望 3 条记录, got %d", len(logs))
	}
	// 验证排序为时间倒序
	if len(logs) >= 2 && logs[0].CreatedAt.Before(logs[1].CreatedAt) {
		t.Error("期望按时间倒序排列")
	}
}

// TestAuditRepo_List_ByAction 验证按操作类型筛选。
func TestAuditRepo_List_ByAction(t *testing.T) {
	repo := setupAuditRepoTest(t)
	seedAuditLogs(t, repo)

	logs, total, err := repo.List(0, "user.create", 1, 10)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if total != 1 {
		t.Errorf("action=user.create: 期望 total=1, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("action=user.create: 期望 1 条记录, got %d", len(logs))
	}
}

// TestAuditRepo_List_ByOperatorAndAction 验证操作人和操作类型组合筛选。
func TestAuditRepo_List_ByOperatorAndAction(t *testing.T) {
	repo := setupAuditRepoTest(t)
	seedAuditLogs(t, repo)

	logs, total, err := repo.List(2, "ticket.create", 1, 10)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if total != 1 {
		t.Errorf("期望 total=1, got %d", total)
	}
	_ = logs // 仅验证计数，不检查具体内容
}

// TestAuditRepo_List_Pagination 验证分页功能。
func TestAuditRepo_List_Pagination(t *testing.T) {
	repo := setupAuditRepoTest(t)
	seedAuditLogs(t, repo)

	// 第一页：2 条
	logs, total, err := repo.List(0, "", 1, 2)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if total != 5 {
		t.Errorf("期望 total=5, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("第1页: 期望 2 条, got %d", len(logs))
	}

	// 第二页：2 条
	logs, total, err = repo.List(0, "", 2, 2)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if total != 5 {
		t.Errorf("期望 total=5, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("第2页: 期望 2 条, got %d", len(logs))
	}

	// 第三页：1 条
	logs, total, err = repo.List(0, "", 3, 2)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("第3页: 期望 1 条, got %d", len(logs))
	}
}

// TestAuditRepo_List_Empty 验证无数据时不报错。
func TestAuditRepo_List_Empty(t *testing.T) {
	repo := setupAuditRepoTest(t)
	// 不插入任何数据

	logs, total, err := repo.List(0, "", 1, 10)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if total != 0 {
		t.Errorf("期望 total=0, got %d", total)
	}
	_ = logs // 空列表验证，仅检查长度
	if len(logs) != 0 {
		t.Errorf("期望空列表, got %d 条", len(logs))
	}
}
