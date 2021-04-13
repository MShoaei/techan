package techan

import "github.com/sdcoffey/big"

type stopLossRule struct {
	Indicator
	tolerance big.Decimal
}

// NewStopLossRule returns a new rule that is satisfied when the given loss tolerance (a percentage) is met or exceeded.
// Loss tolerance should be a value between -1 and 1.
func NewStopLossRule(series *TimeSeries, tolerance float64) Rule {
	return stopLossRule{
		Indicator: NewClosePriceIndicator(series),
		tolerance: big.NewDecimal(1.0 + tolerance),
	}
}

func (slr stopLossRule) IsSatisfied(index int, record *TradingRecord) bool {
	if !record.CurrentPosition().IsOpen() {
		return false
	}

	openPrice := record.CurrentPosition().EntranceOrder().Price
	loss := slr.Indicator.Calculate(index).Div(openPrice)
	return loss.LTE(slr.tolerance)
}

type takeProfitRule struct {
	Indicator
	tolerance big.Decimal
}

func NewTakeProfitRule(series *TimeSeries, tolerance float64) Rule {
	return takeProfitRule{
		Indicator: NewClosePriceIndicator(series),
		tolerance: big.NewDecimal(1.0 + tolerance),
	}
}

func (tpr takeProfitRule) IsSatisfied(index int, record *TradingRecord) bool {
	if !record.CurrentPosition().IsOpen() {
		return false
	}

	amount := record.CurrentPosition().EntranceOrder().Amount

	openPrice := record.CurrentPosition().CostBasis()
	win := tpr.Indicator.Calculate(index).Mul(amount).Div(openPrice)
	return win.GTE(tpr.tolerance)
}
