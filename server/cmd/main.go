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
	// TODO(cmd/main): 将 main 中的初始化流程拆成 wireApp()/runServer() 两层。
	// 现在 main 同时负责配置、DB、Adapter、Service、Handler、Scheduler 和 HTTP 生命周期，
	// 后续新增依赖时会继续膨胀，不利于集成测试单独复用应用装配逻辑。
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
	// TODO(cmd/main): 数据库连接池参数目前写死在 database.Init 内部。
	// 应将 MaxOpenConns/MaxIdleConns/ConnMaxLifetime 放入配置，避免生产环境只能改代码调参。
	db, err := database.Init(cfg.Database)
	if err != nil {
		slog.Error("数据库连接失败", "error", err)
		os.Exit(1)
	}
	slog.Info("数据库连接成功")

	// 3. 自动迁移
	// TODO(cmd/main): 生产环境应改为显式迁移命令或启动前迁移任务。
	// AutoMigrate 适合开发环境，但在生产库上自动变更表结构缺少审批、回滚和审计。
	if err := database.AutoMigrate(db); err != nil {
		slog.Error("数据库迁移失败", "error", err)
		os.Exit(1)
	}
	slog.Info("数据库迁移完成")

	// 4. 初始化 Adapter 层（LLMClient / EmbeddingClient / VectorStore）
	// TODO(cmd/main): LLM/Embedding 超时时间应来自配置，并区分 query rewrite、rerank、最终生成等场景。
	// 当前 60s/30s 写死会让短请求和长生成共享同一超时策略。
	llmTimeout := 60 * time.Second
	llmClient := adapter.NewOpenAIClient(cfg.LLM.BaseURL, cfg.LLM.APIKey, llmTimeout)

	// Embedding 优先使用独立 Base URL 和 API Key，空时回退到 LLM 配置
	embedBaseURL := cfg.Embedding.BaseURL
	if embedBaseURL == "" {
		embedBaseURL = cfg.LLM.BaseURL
	}
	embedAPIKey := cfg.Embedding.APIKey
	if embedAPIKey == "" {
		embedAPIKey = cfg.LLM.APIKey
	}
	embeddingClient := adapter.NewOpenAIEmbeddingClient(embedBaseURL, embedAPIKey, 30*time.Second)
	slog.Info("LLM/Embedding 客户端已初始化",
		"llm_base_url", cfg.LLM.BaseURL,
		"embedding_base_url", embedBaseURL,
		"llm_model", cfg.LLM.Model,
		"embedding_model", cfg.Embedding.Model)

	// pgvector 向量存储
	// TODO(cmd/main): VectorStore 初始化失败后仅 warn，但后续 KnowledgeService/ChatService 仍可启动。
	// 应提供健康状态并让依赖向量核心路径的接口返回明确 20002，而不是在 nil store 处退化为未知错误。
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
	dashboardRepo := repository.NewDashboardRepo(db)

	// 6. 初始化 Service 层
	txManager := service.NewGormTxManager(db)
	authService := service.NewAuthService(userRepo, db)
	userService := service.NewUserService(userRepo, db)
	roleService := service.NewRoleService(roleRepo, userRepo, db)
	ticketService := service.NewTicketService(ticketRepo, txManager)
	messageService := service.NewMessageService(messageRepo)
	dashboardService := service.NewDashboardService(dashboardRepo)
	configService := service.NewConfigService(configRepo)

	// LLM 配置管理
	// TODO(cmd/main): 启动时应把 config.yaml/env 的 LLM 配置同步为默认 LLMConfig 或作为 fallback 注入 Manager。
	// 当前 ChatService 可能从 LLMConfigManager 读取到 nil，然后使用 model="default"，与 cfg.LLM.Model 不一致。
	llmConfigRepo := repository.NewLlmConfigRepo(db)
	llmConfigSvc := service.NewLLMConfigService(llmConfigRepo)
	slog.Info("LLM 配置服务已初始化")

	// RAG 引擎组件
	// TODO(cmd/main): Chunker、BM25Retriever、Pipeline、Processor 目前没有完整装配。
	// ChatService 注入 nil pipeline 会让问答绕过 RAG，KnowledgeService 注入 nil chunker 会让发布路径不可用。
	embedder := rag.NewEmbedder(embeddingClient, 20)
	docParser := rag.NewDocParser()

	// KnowledgeService（CRUD + pgvector 管道 + 文档上传）
	knowledgeService := service.NewKnowledgeService(knowledgeRepo, nil, embedder, vectorStore, docParser, nil)
	slog.Info("KnowledgeService 已初始化")

	// ChatService（自建 Pipeline + LLMClient）
	chatService := service.NewChatService(knowledgeRepo, chatRepo, nil, llmClient, llmConfigSvc.GetManager(), cfg.AI.DefaultTopK)
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
	scheduler := service.NewScheduler(ticketService)
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
	// TODO(cmd/main): ReadTimeout/WriteTimeout/IdleTimeout 应配置化，并为 SSE 单独提供更长写超时策略。
	// 全局 WriteTimeout=60s 对慢速 LLM 生成仍可能过短。
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
			WriteTimeout: 60 * time.Second, // SSE 路由内部通过 SetWriteDeadline 续期
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
