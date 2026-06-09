//go:build integration

// Package handler_test 验证 TicketHandler HTTP 接口。
//
// 测试覆盖 PLAN.md Task24 定义的后台管理和门户端申告端点。
package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/dto/request"
	"opsmind/internal/handler"
	"opsmind/internal/middleware"
	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// handlerTestEnv 封装 handler 测试环境。
type handlerTestEnv struct {
	r    *gin.Engine
	db   *gorm.DB
	repo *repository.TicketRepo
}

func setupTicketHandlerTest(t *testing.T) *handlerTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbCfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, User: "opsmind", Password: "opsmind123",
		DBName: "opsmind_test", SSLMode: "disable",
	}
	db, err := database.Init(dbCfg)
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}

	// 建表
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY, username VARCHAR(64) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL, real_name VARCHAR(64) NOT NULL,
		phone VARCHAR(11) NOT NULL, email VARCHAR(128),
		status SMALLINT NOT NULL DEFAULT 1, first_login BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS tickets (
		id BIGSERIAL PRIMARY KEY, ticket_no VARCHAR(32) NOT NULL UNIQUE,
		user_id BIGINT NOT NULL, title VARCHAR(255) NOT NULL, description TEXT NOT NULL,
		urgency SMALLINT NOT NULL, impact_scope SMALLINT DEFAULT 1,
		affected_systems JSONB, contact_phone VARCHAR(11) NOT NULL, contact_email VARCHAR(128),
		status SMALLINT NOT NULL DEFAULT 1, supplement_count SMALLINT NOT NULL DEFAULT 0,
		chat_context JSONB, source SMALLINT NOT NULL DEFAULT 1,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS ticket_records (
		id BIGSERIAL PRIMARY KEY, ticket_id BIGINT NOT NULL, operator_id BIGINT NOT NULL,
		action VARCHAR(32) NOT NULL, content TEXT, detail JSONB,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	ticketRepo := repository.NewTicketRepo(db)
	ticketSvc := service.NewTicketService(ticketRepo)
	ticketH := handler.NewTicketHandler(ticketSvc)

	r := gin.New()
	r.Use(middleware.RequestID())
	// 模拟认证中间件：注入测试用户（user_id=1）
	r.Use(func(c *gin.Context) {
		c.Set("currentUser", map[string]interface{}{
			"user_id":  float64(1),
			"username": "admin",
			"roles":    []interface{}{"admin"},
		})
		c.Next()
	})

	admin := r.Group("/api/v1/admin")
	{
		admin.GET("/tickets", ticketH.ListAll)
		admin.GET("/tickets/:id", ticketH.GetDetail)
		admin.PATCH("/tickets/:id/status", ticketH.UpdateStatus)
		admin.POST("/tickets/:id/records", ticketH.AddRecord)
	}

	portal := r.Group("/api/v1/portal")
	{
		portal.POST("/tickets", ticketH.CreateTicket)
		portal.GET("/tickets", ticketH.ListByUser)
		portal.GET("/tickets/:id", ticketH.GetDetail)
		portal.PATCH("/tickets/:id/supplement", ticketH.SupplementTicket)
	}

	return &handlerTestEnv{r: r, db: db, repo: ticketRepo}
}

// createHandlerUser 在测试 DB 中创建用户并返回。
func createHandlerUser(t *testing.T, db *gorm.DB, username string) *model.User {
	t.Helper()
	now := time.Now()
	u := &model.User{
		Username:     username,
		PasswordHash: "$2a$10$hash",
		RealName:     "测试用户",
		Phone:        "13800000001",
		Status:       1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return u
}

// createHandlerTicket 在测试 DB 中创建申告，返回创建后的 ID。
func createHandlerTicket(t *testing.T, db *gorm.DB, ticket *model.Ticket) int64 {
	t.Helper()
	if err := db.Create(ticket).Error; err != nil {
		t.Fatalf("创建测试申告失败: %v", err)
	}
	return ticket.ID
}

// =============================================================================
// Portal: POST /api/v1/portal/tickets
// =============================================================================

func TestTicketHandler_CreateTicket(t *testing.T) {
	env := setupTicketHandlerTest(t)
	defer cleanupHandlerTables(t, env.db)

	body := request.CreateTicketRequest{
		Title:        "网络连接异常",
		Description:  "办公区网络频繁断开",
		Urgency:      2,
		ImpactScope:  1,
		ContactPhone: "13800000001",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/portal/tickets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("期望 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Errorf("期望 code=0, got %v", resp)
	}
}

func TestTicketHandler_CreateTicket_InvalidParam(t *testing.T) {
	env := setupTicketHandlerTest(t)
	defer cleanupHandlerTables(t, env.db)

	// 空标题
	body := request.CreateTicketRequest{
		Title: "", Description: "描述", Urgency: 1, ContactPhone: "13800000001",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/portal/tickets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if code, ok := resp["code"].(float64); !ok || code != 10003 {
		t.Errorf("期望 code=10003(参数校验失败), got %v", resp)
	}
}

// =============================================================================
// Admin: GET /api/v1/admin/tickets
// =============================================================================

func TestTicketHandler_ListAll(t *testing.T) {
	env := setupTicketHandlerTest(t)
	defer cleanupHandlerTables(t, env.db)

	createHandlerTicket(t, env.db, &model.Ticket{
		TicketNo: "TK-20260609-H001", UserID: 1, Title: "测试申告1",
		Description: "描述", Urgency: 1, ContactPhone: "x", Status: 1, Source: 1,
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/tickets?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("期望 200, got %d", w.Code)
	}
}

// =============================================================================
// Admin: PATCH /api/v1/admin/tickets/:id/status
// =============================================================================

func TestTicketHandler_UpdateStatus(t *testing.T) {
	env := setupTicketHandlerTest(t)
	defer cleanupHandlerTables(t, env.db)

	id := createHandlerTicket(t, env.db, &model.Ticket{
		TicketNo: "TK-20260609-H002", UserID: 1, Title: "测试申告",
		Description: "描述", Urgency: 1, ContactPhone: "x", Status: 1, Source: 1,
	})

	body := request.UpdateTicketStatusRequest{
		Action: "start",
		Result: "开始处理",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/admin/tickets/%d/status", id), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("期望 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

// =============================================================================
// Admin: GET /api/v1/admin/tickets/:id (GetDetail)
// =============================================================================

func TestTicketHandler_GetDetail(t *testing.T) {
	env := setupTicketHandlerTest(t)
	defer cleanupHandlerTables(t, env.db)

	createHandlerUser(t, env.db, "htest_detail")
	id := createHandlerTicket(t, env.db, &model.Ticket{
		TicketNo: "TK-20260609-H003", UserID: 1, Title: "详情测试",
		Description: "详细描述", Urgency: 2, ContactPhone: "13800000001", Status: 1, Source: 1,
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/tickets/%d", id), nil)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("期望 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Errorf("期望 code=0, got %v", resp)
	}
}

// =============================================================================
// Helper
// =============================================================================

func cleanupHandlerTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	db.Exec("DELETE FROM ticket_records")
	db.Exec("DELETE FROM tickets")
	db.Exec("DELETE FROM users WHERE username LIKE 'htest_%'")
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
}
