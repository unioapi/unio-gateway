package billing

import (
	"errors"
	"math/big"

	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	// PricingUnitPer1MTokens 表示每 100 万 token 的计价单位。
	PricingUnitPer1MTokens = "per_1m_tokens"

	// FormulaVersionV1 表示当前 token 计费公式版本。
	FormulaVersionV1 = "token_v1"

	// amountDecimalScale 表示金额写入 NUMERIC(20,10) 前保留的小数位数。
	amountDecimalScale = 10
)

var (
	// ErrInvalidUsage 表示 usage token 数量不满足计费约束。
	ErrInvalidUsage = errors.New("billing: invalid usage")
	// ErrInvalidPrice 表示 price snapshot 缺少必需价格或价格无效。
	ErrInvalidPrice = errors.New("billing: invalid price")
	// ErrUnsupportedPricingUnit 表示当前 billing service 不支持该计价单位。
	ErrUnsupportedPricingUnit = errors.New("billing: unsupported pricing unit")
	// ErrUnsupportedFormula 表示当前 billing service 不支持该价格计算公式。
	ErrUnsupportedFormula = errors.New("billing: unsupported formula")
)

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

type tokenPrices struct {
	Currency             string
	FormulaVersion       string
	InputPrice           *big.Rat
	OutputPrice          *big.Rat
	CachedInputPrice     *big.Rat
	ReasoningOutputPrice *big.Rat
}

// AuthorizationEstimate 表示调用上游前用于预授权的保守 token 估算。
type AuthorizationEstimate struct {
	PromptTokens        int64
	MaxCompletionTokens int64
}

// Service 负责根据 usage 和 price snapshot 计算请求应扣金额。
// TODO(教学/refactor): 下节课先拆分 billing service 文件；保持行为不变，把 DTO/价格规范化、预授权估算、真实结算和 NUMERIC helper 分到独立文件。
type Service struct{}

// Calculate 根据 usage 和 price snapshot 计算本次请求应扣金额。
func (s Service) Calculate(usage Usage, price PriceSnapshot) (Settlement, error) {
	if usage.PromptTokens < 0 || usage.CompletionTokens < 0 || usage.TotalTokens < 0 || usage.CachedTokens < 0 || usage.ReasoningTokens < 0 {
		return Settlement{}, failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
		return Settlement{}, failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	if usage.CachedTokens > usage.PromptTokens {
		return Settlement{}, failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	if usage.ReasoningTokens > usage.CompletionTokens {
		return Settlement{}, failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	prices, err := normalizeTokenPrices(price)
	if err != nil {
		return Settlement{}, err
	}

	uncachedPrompt := usage.PromptTokens - usage.CachedTokens
	normalCompletion := usage.CompletionTokens - usage.ReasoningTokens

	// 公式版本 token_v1：四类 token 分别按快照价格计费，再统一除以 100 万。
	amount := new(big.Rat)
	amount.Add(amount, tokenCost(prices.InputPrice, uncachedPrompt))
	amount.Add(amount, tokenCost(prices.CachedInputPrice, usage.CachedTokens))
	amount.Add(amount, tokenCost(prices.OutputPrice, normalCompletion))
	amount.Add(amount, tokenCost(prices.ReasoningOutputPrice, usage.ReasoningTokens))
	amount.Quo(amount, big.NewRat(1_000_000, 1))

	return Settlement{
		Amount:         ratToNumeric(amount, amountDecimalScale),
		Currency:       prices.Currency,
		FormulaVersion: prices.FormulaVersion,
	}, nil
}

// EstimateAuthorization 根据预估最大 token 用量计算需要冻结的金额。
func (s Service) EstimateAuthorization(estimate AuthorizationEstimate, price PriceSnapshot) (Settlement, error) {
	if estimate.PromptTokens < 0 || estimate.MaxCompletionTokens < 0 {
		return Settlement{}, failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	prices, err := normalizeTokenPrices(price)
	if err != nil {
		return Settlement{}, err
	}

	maxCompletionPrice := maxRat(prices.OutputPrice, prices.ReasoningOutputPrice)

	amount := new(big.Rat)
	amount.Add(amount, tokenCost(prices.InputPrice, estimate.PromptTokens))
	amount.Add(amount, tokenCost(maxCompletionPrice, estimate.MaxCompletionTokens))
	amount.Quo(amount, big.NewRat(1_000_000, 1))

	return Settlement{
		Amount:         ratToNumeric(amount, amountDecimalScale),
		Currency:       prices.Currency,
		FormulaVersion: prices.FormulaVersion,
	}, nil
}

// requiredNonNegativeNumeric 将必填 NUMERIC 价格转换成非负有理数。
func requiredNonNegativeNumeric(value pgtype.Numeric) (*big.Rat, error) {
	rat, err := numericToRat(value)
	if err != nil {
		return nil, err
	}

	if rat.Sign() < 0 {
		return nil, failure.Wrap(
			failure.CodeBillingInvalidPrice,
			ErrInvalidPrice,
			failure.WithMessage(ErrInvalidPrice.Error()),
		)
	}

	return rat, nil
}

// numericToRat 将 pgtype.Numeric 转成 big.Rat，避免 float64 精度损失。
func numericToRat(value pgtype.Numeric) (*big.Rat, error) {
	if !value.Valid || value.NaN || value.InfinityModifier != pgtype.Finite {
		return nil, failure.Wrap(
			failure.CodeBillingInvalidPrice,
			ErrInvalidPrice,
			failure.WithMessage(ErrInvalidPrice.Error()),
		)
	}
	if value.Int == nil {
		return new(big.Rat), nil
	}

	rat := new(big.Rat).SetInt(new(big.Int).Set(value.Int))
	if value.Exp > 0 {
		rat.Mul(rat, new(big.Rat).SetInt(pow10(value.Exp)))
	}
	if value.Exp < 0 {
		rat.Quo(rat, new(big.Rat).SetInt(pow10(-value.Exp)))
	}

	return rat, nil
}

// tokenCost 计算某类 token 在除以 100 万之前的原始金额。
func tokenCost(unitPrice *big.Rat, tokens int64) *big.Rat {
	return new(big.Rat).Mul(unitPrice, big.NewRat(tokens, 1))
}

// ratToNumeric 将金额四舍五入到固定小数位，匹配 NUMERIC(20,10)。
func ratToNumeric(value *big.Rat, scale int32) pgtype.Numeric {
	multiplier := pow10(scale)
	scaled := new(big.Rat).Mul(value, new(big.Rat).SetInt(multiplier))

	return pgtype.Numeric{
		Int:   roundHalfUp(scaled),
		Exp:   -scale,
		Valid: true,
	}
}

// roundHalfUp 对非负有理数执行四舍五入。
func roundHalfUp(value *big.Rat) *big.Int {
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(value.Num(), value.Denom(), remainder)

	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(value.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}

	return quotient
}

// pow10 返回 10 的 exp 次方。
func pow10(exp int32) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
}

func normalizeTokenPrices(price PriceSnapshot) (tokenPrices, error) {
	if price.PricingUnit != PricingUnitPer1MTokens {
		return tokenPrices{}, failure.Wrap(
			failure.CodeBillingUnsupportedPricingUnit,
			ErrUnsupportedPricingUnit,
			failure.WithMessage(ErrUnsupportedPricingUnit.Error()),
		)
	}

	if price.Currency == "" {
		return tokenPrices{}, failure.Wrap(
			failure.CodeBillingInvalidPrice,
			ErrInvalidPrice,
			failure.WithMessage(ErrInvalidPrice.Error()),
		)
	}

	formulaVersion := price.FormulaVersion
	if formulaVersion == "" {
		formulaVersion = FormulaVersionV1
	}
	if formulaVersion != FormulaVersionV1 {
		return tokenPrices{}, failure.Wrap(
			failure.CodeBillingUnsupportedFormula,
			ErrUnsupportedFormula,
			failure.WithMessage(ErrUnsupportedFormula.Error()),
		)
	}

	inputPrice, err := requiredNonNegativeNumeric(price.InputPrice)
	if err != nil {
		return tokenPrices{}, err
	}

	outputPrice, err := requiredNonNegativeNumeric(price.OutputPrice)
	if err != nil {
		return tokenPrices{}, err
	}

	cachedInputPrice := inputPrice
	if price.CachedInputPrice.Valid {
		cachedInputPrice, err = requiredNonNegativeNumeric(price.CachedInputPrice)
		if err != nil {
			return tokenPrices{}, err
		}
	}

	reasoningOutputPrice := outputPrice
	if price.ReasoningOutputPrice.Valid {
		reasoningOutputPrice, err = requiredNonNegativeNumeric(price.ReasoningOutputPrice)
		if err != nil {
			return tokenPrices{}, err
		}
	}

	return tokenPrices{
		Currency:             price.Currency,
		FormulaVersion:       formulaVersion,
		InputPrice:           inputPrice,
		OutputPrice:          outputPrice,
		CachedInputPrice:     cachedInputPrice,
		ReasoningOutputPrice: reasoningOutputPrice,
	}, nil
}

// maxRat 返回两个非 nil 有理数中的较大值。
func maxRat(left *big.Rat, right *big.Rat) *big.Rat {
	if left.Cmp(right) >= 0 {
		return new(big.Rat).Set(left)
	}

	return new(big.Rat).Set(right)
}
