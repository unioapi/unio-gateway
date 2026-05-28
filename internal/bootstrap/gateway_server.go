package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ThankCat/unio-api/internal/platform/config"
	"github.com/ThankCat/unio-api/internal/platform/store/sqlc"
	"github.com/ThankCat/unio-api/internal/service/gateway"
	"github.com/redis/go-redis/v9"
)

// GatewayServerAppDB 定义 gateway server app 构建时需要的数据库能力。
type GatewayServerAppDB interface {
	sqlc.DBTX
	gateway.ChatTxBeginner
}

// GatewayServerAppDeps 表示构建 gateway server app 需要的进程级依赖。
type GatewayServerAppDeps struct {
	Logger *slog.Logger
	Config config.Config
	DB     GatewayServerAppDB
	Redis  redis.Cmdable
}

// GatewayServerApp 表示当前 gateway-server 进程已经装配完成的 HTTP 应用。
type GatewayServerApp struct {
	Handler http.Handler
}

// NewGatewayServerApp 装配当前 gateway-server 进程的业务应用。
func NewGatewayServerApp(ctx context.Context, deps GatewayServerAppDeps) (*GatewayServerApp, error) {
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

	return &GatewayServerApp{Handler: handler}, nil
}
