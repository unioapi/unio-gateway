package billing

import (
	"math/big"

	"github.com/ThankCat/unio-api/internal/failure"
)

// Service 负责根据 usage 和 price snapshot 计算请求应扣金额。
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

// EstimateAuthorizationAmount 根据预估最大 token 用量计算调用上游前需要冻结的金额。
func (s Service) EstimateAuthorizationAmount(estimate AuthorizationEstimate, price PriceSnapshot) (Settlement, error) {
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
