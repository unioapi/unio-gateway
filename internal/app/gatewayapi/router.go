package gatewayapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ThankCat/unio-api/internal/app/gatewayapi/middleware"
	"github.com/ThankCat/unio-api/internal/platform/httpmw"
	"github.com/ThankCat/unio-api/internal/platform/httpx"
)

// RouterDeps 保存构建 HTTP router 所需的外部依赖。
type RouterDeps struct {
	Logger                 *slog.Logger
	APIKeyAuthenticator    middleware.APIKeyAuthenticator
	ChatCompletionService  ChatCompletionService
	RateLimiter            middleware.RateLimiter
	RateLimitLimit         int64
	RateLimitWindow        time.Duration
	ModelCatalogService    ModelCatalogService
	RateLimitFailurePolicy string
}

// NewRouter 创建 API server 使用的 HTTP handler。
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(httpmw.RequestID)
	r.Use(httpmw.Logger(deps.Logger))
	r.Use(httpmw.Recoverer(deps.Logger))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.WriteError(
			w,
			http.StatusNotFound,
			"not_found",
			"route not found",
		)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.WriteError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(deps.APIKeyAuthenticator))
		r.Use(middleware.RateLimit(deps.RateLimiter, middleware.RateLimitOptions{
			Limit:         deps.RateLimitLimit,
			Window:        deps.RateLimitWindow,
			FailurePolicy: middleware.RateLimitFailurePolicy(deps.RateLimitFailurePolicy),
			Logger:        deps.Logger,
		}))

		modelsHandler := &modelsHandler{
			service: deps.ModelCatalogService,
		}
		r.Get("/models", modelsHandler.handleModels)

		chatHandler := &chatCompletionsHandler{
			service: deps.ChatCompletionService,
		}
		r.Method(http.MethodPost, "/chat/completions", chatHandler)
	})

	return r
}
