package billing

import (
	"math/big"

	"github.com/ThankCat/unio-api/internal/platform/failure"
)

// Service 负责根据 usage 和 token 单价快照计算客户扣费与平台成本。
type Service struct{}

// CalculateCustomerCharge 根据 usage 和客户侧售价快照计算本次请求应扣金额。
func (s Service) CalculateCustomerCharge(usage Usage, price CustomerPriceSnapshot) (CustomerCharge, error) {
	if err := validateUsage(usage); err != nil {
		return CustomerCharge{}, err
	}

	rates, err := normalizeCustomerPriceRates(price)
	if err != nil {
		return CustomerCharge{}, err
	}

	amounts := calculateTokenAmountBreakdown(usage, rates)

	return CustomerCharge{
		Amount:         ratToNumeric(amounts.TotalAmount, amountDecimalScale),
		Currency:       rates.Currency,
		FormulaVersion: rates.FormulaVersion,
	}, nil
}

// CalculateProviderCost 根据 usage 和 provider/channel 成本价快照计算本次请求的平台成本分项。
func (s Service) CalculateProviderCost(usage Usage, cost ProviderCostSnapshot) (ProviderCost, error) {
	if err := validateUsage(usage); err != nil {
		return ProviderCost{}, err
	}

	rates, err := normalizeProviderCostRates(cost)
	if err != nil {
		return ProviderCost{}, err
	}

	amounts := calculateTokenAmountBreakdown(usage, rates)

	return ProviderCost{
		InputCostAmount:           ratToNumeric(amounts.InputAmount, amountDecimalScale),
		OutputCostAmount:          ratToNumeric(amounts.OutputAmount, amountDecimalScale),
		CachedInputCostAmount:     ratToNumeric(amounts.CachedInputAmount, amountDecimalScale),
		ReasoningOutputCostAmount: ratToNumeric(amounts.ReasoningOutputAmount, amountDecimalScale),
		TotalCostAmount:           ratToNumeric(amounts.TotalAmount, amountDecimalScale),
		Currency:                  rates.Currency,
		FormulaVersion:            rates.FormulaVersion,
	}, nil
}

// EstimateAuthorizationAmount 根据预估最大 token 用量计算调用上游前需要冻结的金额。
func (s Service) EstimateAuthorizationAmount(estimate AuthorizationEstimate, price CustomerPriceSnapshot) (CustomerCharge, error) {
	if estimate.PromptTokens < 0 || estimate.MaxCompletionTokens < 0 {
		return CustomerCharge{}, failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	rates, err := normalizeCustomerPriceRates(price)
	if err != nil {
		return CustomerCharge{}, err
	}

	maxCompletionRate := maxRat(rates.OutputRate, rates.ReasoningOutputRate)

	amount := new(big.Rat)
	amount.Add(amount, tokenCost(rates.InputRate, estimate.PromptTokens))
	amount.Add(amount, tokenCost(maxCompletionRate, estimate.MaxCompletionTokens))
	amount.Quo(amount, big.NewRat(1_000_000, 1))

	return CustomerCharge{
		Amount:         ratToNumeric(amount, amountDecimalScale),
		Currency:       rates.Currency,
		FormulaVersion: rates.FormulaVersion,
	}, nil
}

// validateUsage 校验 usage token 约束，保持和 usage_records 表约束一致。
func validateUsage(usage Usage) error {
	if usage.PromptTokens < 0 || usage.CompletionTokens < 0 || usage.TotalTokens < 0 || usage.CachedTokens < 0 || usage.ReasoningTokens < 0 {
		return failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
		return failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	if usage.CachedTokens > usage.PromptTokens {
		return failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	if usage.ReasoningTokens > usage.CompletionTokens {
		return failure.Wrap(
			failure.CodeBillingInvalidUsage,
			ErrInvalidUsage,
			failure.WithMessage(ErrInvalidUsage.Error()),
		)
	}

	return nil
}

// tokenAmountBreakdown 表示四类 token 分别计算出的金额。
type tokenAmountBreakdown struct {
	InputAmount           *big.Rat
	OutputAmount          *big.Rat
	CachedInputAmount     *big.Rat
	ReasoningOutputAmount *big.Rat
	TotalAmount           *big.Rat
}

// calculateTokenAmountBreakdown 按 token_v1 公式计算四类 token 的金额分项。
func calculateTokenAmountBreakdown(usage Usage, rates tokenRates) tokenAmountBreakdown {
	uncachedPrompt := usage.PromptTokens - usage.CachedTokens
	normalCompletion := usage.CompletionTokens - usage.ReasoningTokens

	inputAmount := tokenAmount(rates.InputRate, uncachedPrompt)
	cachedInputAmount := tokenAmount(rates.CachedInputRate, usage.CachedTokens)
	outputAmount := tokenAmount(rates.OutputRate, normalCompletion)
	reasoningOutputAmount := tokenAmount(rates.ReasoningOutputRate, usage.ReasoningTokens)

	// 调用方决定只使用总额，还是连同分项一起写入成本快照。
	totalAmount := new(big.Rat)
	totalAmount.Add(totalAmount, inputAmount)
	totalAmount.Add(totalAmount, cachedInputAmount)
	totalAmount.Add(totalAmount, outputAmount)
	totalAmount.Add(totalAmount, reasoningOutputAmount)

	return tokenAmountBreakdown{
		InputAmount:           inputAmount,
		OutputAmount:          outputAmount,
		CachedInputAmount:     cachedInputAmount,
		ReasoningOutputAmount: reasoningOutputAmount,
		TotalAmount:           totalAmount,
	}
}

// tokenAmount 计算某类 token 按 per_1m_tokens 计价后的金额。
func tokenAmount(unitPrice *big.Rat, tokens int64) *big.Rat {
	amount := tokenCost(unitPrice, tokens)
	return amount.Quo(amount, big.NewRat(1_000_000, 1))
}
