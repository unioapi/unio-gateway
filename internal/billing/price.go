package billing

import (
	"math/big"

	"github.com/ThankCat/unio-api/internal/failure"
)

// tokenPrices 表示已校验并转成有理数的 token 价格。
type tokenPrices struct {
	Currency             string
	FormulaVersion       string
	InputPrice           *big.Rat
	OutputPrice          *big.Rat
	CachedInputPrice     *big.Rat
	ReasoningOutputPrice *big.Rat
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
