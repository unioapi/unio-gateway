package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/adapter/openai"
	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/config"
	"github.com/ThankCat/unio-api/internal/credential"
	"github.com/ThankCat/unio-api/internal/gateway"
	"github.com/ThankCat/unio-api/internal/httpapi"
	"github.com/ThankCat/unio-api/internal/modelcatalog"
	"github.com/ThankCat/unio-api/internal/ratelimit"
	"github.com/ThankCat/unio-api/internal/redis"
	"github.com/ThankCat/unio-api/internal/requestlog"
	"github.com/ThankCat/unio-api/internal/routing"
	"github.com/ThankCat/unio-api/internal/store"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
)

func main() {
	bootstrapLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		bootstrapLogger.Error("load config failed", "error", err)

		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Log.Level}))

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startupCancel()

	// TODO(阶段1/production): 将启动超时、HTTP server timeout 和 shutdown timeout 纳入 config，并配合 readiness/metrics 暴露运行状态。
	// TODO(阶段2/production): 启动前接入 migration runner（迁移执行器）或 schema 版本检查，避免服务连接到未迁移数据库。
	// DB 启动期先检查数据库可用，避免服务带病启动。
	pgPool, err := store.OpenPostgres(startupCtx, cfg.DB.URL)
	if err != nil {
		logger.Error("open postgres failed", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()
	logger.Info("postgres connected")

	// Redis
	redisClient, err := redis.OpenRedis(startupCtx, cfg.Redis)
	if err != nil {
		logger.Error("open redis failed", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("redis connected", "addr", cfg.Redis.Addr, "db", cfg.Redis.DB)

	// TODO(阶段6/production): main 函数直接装配 credential、routing、adapter registry 和 gateway 会让启动逻辑膨胀；阶段 6 收口或进入后台管理装配前；抽出 server bootstrap/app wiring 组件，保持 main 只负责配置、生命周期和退出信号。
	queries := sqlc.New(pgPool)
	apiKeyAuthenticator := auth.NewAPIKeyAuthenticator(queries)
	modelCatalogService := modelcatalog.NewService(queries)
	requestLogStore := requestlog.NewStore(queries)
	credentialResolver := credential.NewStaticResolver(map[string]string{})

	chatRouter := routing.NewRouter(queries, credentialResolver, 30*time.Second)
	openaiAdapter := openai.NewAdapter(http.DefaultClient)
	// TODO(阶段6/production): provider.adapter 缺少启动/后台写入校验会导致运行时 registry miss；开放后台管理或启用真实 channel 前；在 provider 写入和启动 preflight 中校验 adapter key 必须存在于 adapter registry。
	adapterRegistry, err := adapter.NewRegistry(adapter.Registration{
		Key:        "openai",
		Chat:       openaiAdapter,
		StreamChat: openaiAdapter,
	})
	if err != nil {
		logger.Error("openai adapter failed", "error", err)
		os.Exit(1)
	}

	chatCompletionService := gateway.NewChatCompletionService(chatRouter, adapterRegistry, nil, requestLogStore)

	rateLimitStore := ratelimit.NewRedisStore(redisClient)
	rateLimiter := ratelimit.NewLimiter(rateLimitStore)

	handler := httpapi.NewRouter(httpapi.RouterDeps{
		Logger:              logger,
		APIKeyAuthenticator: apiKeyAuthenticator,
		RateLimiter:         rateLimiter,

		// TODO(阶段3/production): 将默认 rate limit（限流）阈值和窗口迁入 config；项目级、模型级和 channel 级策略后续来自数据库。
		RateLimitLimit:  60,
		RateLimitWindow: time.Minute,

		ChatCompletionService: chatCompletionService,
		ModelCatalogService:   modelCatalogService,
	})

	server := &http.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: handler,

		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		logger.Info("server starting", "addr", cfg.HTTP.Addr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}

		close(errCh)
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownCh)

	select {
	case err, ok := <-errCh:
		if ok && err != nil {
			// 服务启动失败时走这里
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdownCh:
		// 收到 Ctrl+C / SIGTERM 时走这里
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	// 给服务最多 10 秒时间处理完正在进行的请求，然后再退出。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown 会停止接收新请求，并等待已有请求在 ctx 超时前完成。
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
