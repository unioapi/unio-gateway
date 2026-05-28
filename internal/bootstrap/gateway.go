package bootstrap

import (
	"github.com/ThankCat/unio-api/internal/core/billing"
	"github.com/ThankCat/unio-api/internal/core/ledger"
	"github.com/ThankCat/unio-api/internal/core/requestlog"
	"github.com/ThankCat/unio-api/internal/platform/store/sqlc"
	"github.com/ThankCat/unio-api/internal/service/gateway"
)

// NewChatGateway 创建当前 server 进程使用的 chat gateway service。
func NewChatGateway(db gateway.ChatTxBeginner, queries *sqlc.Queries, router gateway.ChatRouter, registry gateway.AdapterRegistry) *gateway.ChatCompletionService {
	requestLogStore := requestlog.NewStore(queries)
	ledgerService := ledger.NewService(db, queries)
	chatSettlementService := gateway.NewChatSettlementService(
		db,
		queries,
		billing.Service{},
		ledgerService,
	)
	chatAuthorizationServer := gateway.NewChatAuthorizationService(
		queries,
		billing.Service{},
		ledgerService,
		registry,
	)

	return gateway.NewChatCompletionService(
		router,
		registry,
		nil,
		requestLogStore,
		chatSettlementService,
		chatAuthorizationServer,
	)
}
