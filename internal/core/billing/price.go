package billing

import (
	"math/big"

	"github.com/ThankCat/unio-api/internal/platform/failure"
	"github.com/jackc/pgx/v5/pgtype"
)

// tokenRateSnapshot 表示待校验的一组 token 单价快照。
// 它是 billing 内部的中性结构，可由客户售价或 provider 成本价转换而来。
type tokenRateSnapshot struct {
	Currency            string
	PricingUnit         string
	InputRate           pgtype.Numeric
	OutputRate          pgtype.Numeric
	CachedInputRate     pgtype.Numeric
	ReasoningOutputRate pgtype.Numeric
	FormulaVersion      string
}

// tokenRates 表示已校验并转成有理数的 token 单价。
type tokenRates struct {
	Currency            string
	FormulaVersion      string
	InputRate           *big.Rat
	OutputRate          *big.Rat
	CachedInputRate     *big.Rat
	ReasoningOutputRate *big.Rat
}

// normalizeCustomerPriceRates 校验客户侧售价快照，并转换为可计算的 token 单价。
func normalizeCustomerPriceRates(price CustomerPriceSnapshot) (tokenRates, error) {
	return normalizeTokenRates(tokenRateSnapshot{
		Currency:            price.Currency,
		PricingUnit:         price.PricingUnit,
		InputRate:           price.InputPrice,
		OutputRate:          price.OutputPrice,
		CachedInputRate:     price.CachedInputPrice,
		ReasoningOutputRate: price.ReasoningOutputPrice,
		FormulaVersion:      price.FormulaVersion,
	})
}

// normalizeProviderCostRates 校验 provider/channel 成本价快照，并转换为可计算的 token 单价。
func normalizeProviderCostRates(cost ProviderCostSnapshot) (tokenRates, error) {
	return normalizeTokenRates(tokenRateSnapshot{
		Currency:            cost.Currency,
		PricingUnit:         cost.PricingUnit,
		InputRate:           cost.InputCost,
		OutputRate:          cost.OutputCost,
		CachedInputRate:     cost.CachedInputCost,
		ReasoningOutputRate: cost.ReasoningOutputCost,
		FormulaVersion:      cost.FormulaVersion,
	})
}

// normalizeTokenRates 执行客户售价和 provider 成本价共用的基础单价校验。
func normalizeTokenRates(snapshot tokenRateSnapshot) (tokenRates, error) {
	if snapshot.PricingUnit != PricingUnitPer1MTokens {
		return tokenRates{}, failure.Wrap(
			failure.CodeBillingUnsupportedPricingUnit,
			ErrUnsupportedPricingUnit,
			failure.WithMessage(ErrUnsupportedPricingUnit.Error()),
		)
	}

	if snapshot.Currency == "" {
		return tokenRates{}, failure.Wrap(
			failure.CodeBillingInvalidPrice,
			ErrInvalidRate,
			failure.WithMessage(ErrInvalidRate.Error()),
		)
	}

	formulaVersion := snapshot.FormulaVersion
	if formulaVersion == "" {
		formulaVersion = FormulaVersionV1
	}
	if formulaVersion != FormulaVersionV1 {
		return tokenRates{}, failure.Wrap(
			failure.CodeBillingUnsupportedFormula,
			ErrUnsupportedFormula,
			failure.WithMessage(ErrUnsupportedFormula.Error()),
		)
	}

	inputRate, err := requiredNonNegativeNumeric(snapshot.InputRate)
	if err != nil {
		return tokenRates{}, err
	}

	outputRate, err := requiredNonNegativeNumeric(snapshot.OutputRate)
	if err != nil {
		return tokenRates{}, err
	}

	cachedInputRate := inputRate
	if snapshot.CachedInputRate.Valid {
		cachedInputRate, err = requiredNonNegativeNumeric(snapshot.CachedInputRate)
		if err != nil {
			return tokenRates{}, err
		}
	}

	reasoningOutputRate := outputRate
	if snapshot.ReasoningOutputRate.Valid {
		reasoningOutputRate, err = requiredNonNegativeNumeric(snapshot.ReasoningOutputRate)
		if err != nil {
			return tokenRates{}, err
		}
	}

	return tokenRates{
		Currency:            snapshot.Currency,
		FormulaVersion:      formulaVersion,
		InputRate:           inputRate,
		OutputRate:          outputRate,
		CachedInputRate:     cachedInputRate,
		ReasoningOutputRate: reasoningOutputRate,
	}, nil
}
