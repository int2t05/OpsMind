// Package main 是 OpsMind 后端服务的入口。
//
// 负责初始化配置、数据库连接、路由注册和 HTTP 服务启动。
// 采用单体分层架构（Handler→Service→Repository），所有模块在同一进程内运行。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opsmind/internal/adapter"
	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/handler"
	"opsmind/internal/rag"
	"opsmind/internal/repository"
	"opsmind/internal/router"
	"opsmind/internal/service"
)

func main() {
	slog.Info("OpsMind 服务启动中...")

	// 1. 加载配置
	cfg, err := config.Load("")
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	// 生产模式下 JWT 密钥必须非空，否则拒绝启动。
	if cfg.JWT.Secret == "" {
		if cfg.Server.Mode == "release" {
			slog.Error("JWT 密钥为空，生产模式不允许启动，请设置环境变量 OPSMIND_JWT_SECRET")
			os.Exit(1)
		}
		slog.Warn("JWT 密钥为空，JWT 认证功能不可用（仅调试模式允许）")
	}

	// 2. 初始化数据库连接
	db, err := database.Init(cfg.Database)
	if err != nil {
		slog.Error("数据库连接失败", "error", err)
		os.Exit(1)
	}
	slog.Info("数据库连接成功")

	// 3. 自动迁移
	if err := database.AutoMigrate(db); err != nil {
		slog.Error("数据库迁移失败", "error", err)
		os.Exit(1)
	}
	slog.Info("数据库迁移完成")

	// 4. 初始化 Adapter 层（LLMClient / EmbeddingClient / VectorStore）
	llmTimeout := 60 * time.Second
	llmClient := adapter.NewOpenAIClient(cfg.LLM.BaseURL, cfg.LLM.APIKey, llmTimeout)

	// Embedding 优先使用独立 Base URL，空时回退到 LLM Base URL
	embedBaseURL := cfg.Embedding.BaseURL
	if embedBaseURL == "" {
		embedBaseURL = cfg.LLM.BaseURL
	}
	embeddingClient := adapter.NewOpenAIEmbeddingClient(embedBaseURL, cfg.LLM.APIKey, 30*time.Second)
	slog.Info("LLM/Embedding 客户端已初始化",
		"llm_base_url", cfg.LLM.BaseURL,
		"embedding_base_url", embedBaseURL,
		"llm_model", cfg.LLM.Model,
		"embedding_model", cfg.Embedding.Model)

	// pgvector 向量存储
	pgDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, cfg.Database.SSLMode)
	vectorStore, err := adapter.NewPgvectorStore(pgDSN)
	if err != nil {
		slog.Warn("pgvector 连接失败，向量检索功能不可用", "error", err)
	} else {
		slog.Info("pgvector VectorStore 已连接")
	}

	// 5. 初始化 Repository 层
	configRepo := repository.NewConfigRepo(db)
	userRepo := repository.NewUserRepo(db)
	roleRepo := repository.NewRoleRepo(db)
	ticketRepo := repository.NewTicketRepo(db)
	knowledgeRepo := repository.NewKnowledgeRepo(db)
	chatRepo := repository.NewChatRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	auditRepo := repository.NewAuditRepo(db)

	// 6. 初始化 Service 层
	authService := service.NewAuthService(userRepo, db)
	userService := service.NewUserService(userRepo, db)
	roleService := service.NewRoleService(roleRepo, userRepo, db)
	ticketService := service.NewTicketService(ticketRepo, db)
	messageService := service.NewMessageService(messageRepo)
	dashboardService := service.NewDashboardService(db)
	configService := service.NewConfigService(configRepo)

	// LLM 配置管理
	llmConfigRepo := repository.NewLlmConfigRepo(db)
	llmConfigSvc := service.NewLLMConfigService(llmConfigRepo)
	slog.Info("LLM 配置服务已初始化")

	// RAG 引擎组件
	embedder := rag.NewEmbedder(embeddingClient, 20)
	docParser := rag.NewDocParser()

	// KnowledgeService（CRUD + pgvector 管道 + 文档上传）
	knowledgeService := service.NewKnowledgeService(knowledgeRepo, nil, embedder, vectorStore, docParser, nil)
	slog.Info("KnowledgeService 已初始化")

	// ChatService（自建 Pipeline + LLMClient）
	chatService := service.NewChatService(knowledgeRepo, chatRepo, nil, llmClient, llmConfigSvc.GetManager())
	slog.Info("ChatService 已初始化")

	// AuditService
	auditService := service.NewAuditService(auditRepo, userRepo)

	// 7. 初始化 Handler 层
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	ticketHandler := handler.NewTicketHandler(ticketService, knowledgeService)
	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeService)
	chatHandler := handler.NewChatHandler(chatService, llmClient)
	messageHandler := handler.NewMessageHandler(messageService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	auditHandler := handler.NewAuditHandler(auditService)
	configHandler := handler.NewConfigHandler(configService)
	llmConfigHandler := handler.NewLLMConfigHandler(llmConfigSvc, llmClient)

	// 8. 初始化后台调度器
	scheduler := service.NewScheduler(ticketRepo)
	slog.Info("后台调度器已创建")

	// 9. 设置路由
	r := router.Setup(cfg, &router.Handlers{
		Auth:      authHandler,
		User:      userHandler,
		Role:      roleHandler,
		Ticket:    ticketHandler,
		Knowledge: knowledgeHandler,
		Chat:      chatHandler,
		Message:   messageHandler,
		Dashboard: dashboardHandler,
		Audit:     auditHandler,
		Config:    configHandler,
		LLMConfig: llmConfigHandler,
	})

	// 10. 创建 HTTP Server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // SSE 流式响应无固定超时
		// TODO: WriteTimeout=0 全局禁用写超时 — SSE 需要但影响所有非流式接口。
		// 应只对 SSE 路由单独设置超时，或使用 http.ResponseController 按请求控制。
		IdleTimeout:  60 * time.Second,
	}

	// 11. 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	// 12. 启动 HTTP 服务
	go func() {
		slog.Info("HTTP 服务已启动", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP 服务启动失败", "error", err)
			os.Exit(1)
		}
	}()

	// 13. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("收到退出信号，开始优雅关闭...", "signal", sig)

	scheduler.Stop()
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP 服务关闭失败", "error", err)
	} else {
		slog.Info("HTTP 服务已关闭")
	}

	slog.Info("OpsMind 服务已停止")
}
