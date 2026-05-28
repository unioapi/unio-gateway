package bootstrap

import (
	"log/slog"
	"net/http"

	"github.com/ThankCat/unio-api/internal/app/gatewayapi"
	"github.com/ThankCat/unio-api/internal/core/auth"
	"github.com/ThankCat/unio-api/internal/core/modelcatalog"
	"github.com/ThankCat/unio-api/internal/platform/config"
	"github.com/ThankCat/unio-api/internal/platform/ratelimit"
	"github.com/ThankCat/unio-api/internal/platform/store/sqlc"
	"github.com/redis/go-redis/v9"
)

// NewHTTPHandler 创建当前 server 进程使用的 HTTP handler。
func NewHTTPHandler(
	logger *slog.Logger,
	queries *sqlc.Queries,
	redisClient redis.Cmdable,
	cfg config.Config,
	chatCompletionService gatewayapi.ChatCompletionService,
) http.Handler {
	apiKeyAuthenticator := auth.NewAPIKeyAuthenticator(queries)
	modelCatalogService := modelcatalog.NewService(queries)

	rateLimitStore := ratelimit.NewRedisStore(redisClient, cfg.Redis.KeyNamespace)
	rateLimiter := ratelimit.NewLimiter(rateLimitStore)

	return gatewayapi.NewRouter(gatewayapi.RouterDeps{
		Logger:              logger,
		APIKeyAuthenticator: apiKeyAuthenticator,
		RateLimiter:         rateLimiter,

		RateLimitLimit:         cfg.RateLimit.DefaultLimit,
		RateLimitWindow:        cfg.RateLimit.DefaultWindow,
		RateLimitFailurePolicy: cfg.RateLimit.FailurePolicy,

		ChatCompletionService: chatCompletionService,
		ModelCatalogService:   modelCatalogService,
	})
}
