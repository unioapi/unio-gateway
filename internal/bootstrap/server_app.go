package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ThankCat/unio-api/internal/config"
	"github.com/ThankCat/unio-api/internal/gateway"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/redis/go-redis/v9"
)

// ServerAppDB 定义 server app 构建时需要的数据库能力。
type ServerAppDB interface {
	sqlc.DBTX
	gateway.ChatTxBeginner
}

// ServerAppDeps 表示构建 server app 需要的进程级依赖。
type ServerAppDeps struct {
	Logger *slog.Logger
	Config config.Config
	DB     ServerAppDB
	Redis  redis.Cmdable
}

// ServerApp 表示当前 server 进程已经装配完成的 HTTP 应用。
type ServerApp struct {
	Handler http.Handler
}

// NewServerApp 装配当前 server 进程的业务应用。
func NewServerApp(ctx context.Context, deps ServerAppDeps) (*ServerApp, error) {
	queries := sqlc.New(deps.DB)

	chatRouter, err := NewChatRouter(queries)
	if err != nil {
		return nil, err
	}

	adapterRegistry, err := NewAdapterRegistry(http.DefaultClient)
	if err != nil {
		return nil, err
	}

	// TODO(阶段6/production): [GAP-6-003] 后台写入 provider.adapter 时仍缺少 registry 校验，可能把不可运行的 adapter key 写入业务数据；开放后台 provider 管理前；在阶段 9 provider CRUD 写入路径校验 adapter key 必须存在于 adapter registry。
	providerAdapterPreflight := NewProviderAdapterPreflight(queries, adapterRegistry)
	if err := providerAdapterPreflight.ValidateChatCapabilities(ctx); err != nil {
		return nil, err
	}

	chatCompletionService := NewChatGateway(
		deps.DB,
		queries,
		chatRouter,
		adapterRegistry,
	)

	handler := NewHTTPHandler(
		deps.Logger,
		queries,
		deps.Redis,
		deps.Config,
		chatCompletionService,
	)

	return &ServerApp{Handler: handler}, nil
}
