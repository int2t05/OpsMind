// Package main 是 OpsMind 后端服务的入口。
//
// 负责初始化配置、数据库连接、路由注册和 HTTP 服务启动。
// MVP 阶段采用单体分层架构，所有模块在同一进程内运行。
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

	// 3. 自动迁移（开发/测试阶段自动建表，生产环境建议用独立迁移脚本）
	if err := database.AutoMigrate(db); err != nil {
		slog.Error("数据库迁移失败", "error", err)
		os.Exit(1)
	}
	slog.Info("数据库迁移完成")

	// 4. 初始化 Adapter 层（外部服务适配器）
	ragClient := adapter.NewAnythingLLMClient(adapter.AnythingLLMConfig{
		BaseURL:        cfg.AnythingLLM.BaseURL,
		APIKey:         cfg.AnythingLLM.APIKey,
		TimeoutSeconds: cfg.AnythingLLM.TimeoutSeconds,
	})
	slog.Info("RagClient 已初始化", "base_url", cfg.AnythingLLM.BaseURL)

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
	roleService := service.NewRoleService(roleRepo, db)
	ticketService := service.NewTicketService(ticketRepo)
	knowledgeService := service.NewKnowledgeService(knowledgeRepo, ragClient)
	chatService := service.NewChatService(knowledgeRepo, chatRepo, ragClient)
	messageService := service.NewMessageService(messageRepo)
	dashboardService := service.NewDashboardService(db)
	configService := service.NewConfigService(configRepo)

	// 7. 初始化 Handler 层
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	ticketHandler := handler.NewTicketHandler(ticketService)
	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeService)
	chatHandler := handler.NewChatHandler(chatService)
	messageHandler := handler.NewMessageHandler(messageService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	auditHandler := handler.NewAuditHandler(auditRepo, db)
	configHandler := handler.NewConfigHandler(configService)

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
	})

	// 10. 创建 HTTP Server（支持优雅关闭）
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 11. 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	// 12. 启动 HTTP 服务（goroutine）
	go func() {
		slog.Info("HTTP 服务已启动", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP 服务启动失败", "error", err)
			os.Exit(1)
		}
	}()

	// 13. 等待退出信号（SIGINT / SIGTERM）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("收到退出信号，开始优雅关闭...", "signal", sig)

	// 14. 优雅关闭
	// 先停止调度器
	scheduler.Stop()
	cancel()

	// 再关闭 HTTP 服务（最多等待 10 秒）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP 服务关闭失败", "error", err)
	} else {
		slog.Info("HTTP 服务已关闭")
	}

	slog.Info("OpsMind 服务已停止")
}
