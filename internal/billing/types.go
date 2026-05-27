package billing

import "github.com/jackc/pgx/v5/pgtype"

// Usage 表示一次请求最终用于计费的 token 用量。
type Usage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CachedTokens     int64
	ReasoningTokens  int64
}

// CustomerPriceSnapshot 表示一次请求结算时冻结下来的客户侧售价副本。
type CustomerPriceSnapshot struct {
	Currency             string
	PricingUnit          string
	InputPrice           pgtype.Numeric
	OutputPrice          pgtype.Numeric
	CachedInputPrice     pgtype.Numeric
	ReasoningOutputPrice pgtype.Numeric
	FormulaVersion       string
}

// ProviderCostSnapshot 表示一次请求结算时冻结下来的 provider/channel 成本价副本。
type ProviderCostSnapshot struct {
	Currency            string
	PricingUnit         string
	InputCost           pgtype.Numeric
	OutputCost          pgtype.Numeric
	CachedInputCost     pgtype.Numeric
	ReasoningOutputCost pgtype.Numeric
	FormulaVersion      string
}

// ProviderCost 表示 billing service 计算出来的平台上游成本结果。
type ProviderCost struct {
	InputCostAmount           pgtype.Numeric
	OutputCostAmount          pgtype.Numeric
	CachedInputCostAmount     pgtype.Numeric
	ReasoningOutputCostAmount pgtype.Numeric
	TotalCostAmount           pgtype.Numeric
	Currency                  string
	FormulaVersion            string
}

// CustomerCharge 表示 billing service 计算出来的客户侧扣费结果。
type CustomerCharge struct {
	Amount         pgtype.Numeric
	Currency       string
	FormulaVersion string
}

// AuthorizationEstimate 表示调用上游前用于冻结金额计算的保守 token 估算。
type AuthorizationEstimate struct {
	PromptTokens        int64
	MaxCompletionTokens int64
}
