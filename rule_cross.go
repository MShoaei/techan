package techan

// NewCrossUpIndicatorRule returns a new rule that is satisfied when the lower indicator has crossed above the upper
// indicator.
func NewCrossUpIndicatorRule(upper, lower Indicator) Rule {
	return crossRule{
		upper: upper,
		lower: lower,
		cmp:   1,
	}
}

// NewCrossDownIndicatorRule returns a new rule that is satisfied when the upper indicator has crossed below the lower
// indicator.
func NewCrossDownIndicatorRule(upper, lower Indicator) Rule {
	return crossRule{
		upper: lower,
		lower: upper,
		cmp:   -1,
	}
}

type crossRule struct {
	upper Indicator
	lower Indicator
	cmp   int
}

func (cr crossRule) IsSatisfied(index int, _ *TradingRecord) bool {
	if index == 0 {
		return false
	}

	if cmp := cr.lower.Calculate(index).Cmp(cr.upper.Calculate(index)); cmp == cr.cmp {
		for i := index - 1; i >= 0; i-- {
			if cmp = cr.lower.Calculate(i).Cmp(cr.upper.Calculate(i)); cmp == cr.cmp {
				return false
			}
			if cmp = cr.lower.Calculate(i).Cmp(cr.upper.Calculate(i)); cmp == 0 {
				continue
			}
			if cmp = cr.lower.Calculate(i).Cmp(cr.upper.Calculate(i)); cmp == -cr.cmp {
				return true
			}
		}
	}

	return false
}
