package bootstrap

import (
	"github.com/ThankCat/unio-api/internal/billing"
	"github.com/ThankCat/unio-api/internal/gateway"
	"github.com/ThankCat/unio-api/internal/ledger"
	"github.com/ThankCat/unio-api/internal/requestlog"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
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
