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

// PriceSnapshot 表示一次请求结算时冻结下来的价格副本。
type PriceSnapshot struct {
	Currency             string
	PricingUnit          string
	InputPrice           pgtype.Numeric
	OutputPrice          pgtype.Numeric
	CachedInputPrice     pgtype.Numeric
	ReasoningOutputPrice pgtype.Numeric
	FormulaVersion       string
}

// Settlement 表示 billing service 计算出来的结算结果。
type Settlement struct {
	Amount         pgtype.Numeric
	Currency       string
	FormulaVersion string
}

// AuthorizationEstimate 表示调用上游前用于冻结金额计算的保守 token 估算。
type AuthorizationEstimate struct {
	PromptTokens        int64
	MaxCompletionTokens int64
}
