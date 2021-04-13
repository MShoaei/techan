package techan

//lint:file-ignore S1038 prefer Fprintln

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/sdcoffey/big"
)

// Analysis is an interface that describes a methodology for taking a TradingRecord as input,
// and giving back some float value that describes it's performance with respect to that methodology.
type Analysis interface {
	Analyze(*TradingRecord) float64
}

// TotalProfitAnalysis analyzes the trading record for total profit.
type TotalProfitAnalysis struct{}

// Analyze analyzes the trading record for total profit.
func (tps TotalProfitAnalysis) Analyze(record *TradingRecord) float64 {
	totalProfit := big.NewDecimal(0)
	for _, trade := range record.Trades {
		if trade.IsClosed() {

			costBasis := trade.CostBasis()
			exitValue := trade.ExitValue()

			if trade.IsLong() {
				totalProfit = totalProfit.Add(exitValue.Sub(costBasis))
			} else if trade.IsShort() {
				totalProfit = totalProfit.Sub(exitValue.Sub(costBasis))
			}

		}
	}

	return totalProfit.Float()
}

// PercentGainAnalysis analyzes the trading record for the percentage profit gained relative to start
type PercentGainAnalysis struct{}

// Analyze analyzes the trading record for the percentage profit gained relative to start
func (pga PercentGainAnalysis) Analyze(record *TradingRecord) float64 {
	if len(record.Trades) > 0 && record.Trades[0].IsClosed() {
		return (record.Trades[len(record.Trades)-1].ExitValue().Div(record.Trades[0].CostBasis())).Sub(big.NewDecimal(1)).Float()
	}

	return 0
}

// NumTradesAnalysis analyzes the trading record for the number of trades executed
type NumTradesAnalysis struct{}

// Analyze analyzes the trading record for the number of trades executed
func (nta NumTradesAnalysis) Analyze(record *TradingRecord) float64 {
	return float64(len(record.Trades))
}

// LogTradesAnalysis is a wrapper around an io.Writer, which logs every trade executed to that writer
type LogTradesAnalysis struct {
	io.Writer
}

// Analyze logs trades to provided io.Writer
func (lta LogTradesAnalysis) Analyze(record *TradingRecord) float64 {
	var profit big.Decimal
	logOrder := func(trade *Position) {
		if trade.IsShort() {
			fmt.Fprintln(lta.Writer, fmt.Sprintf("%s - enter with sell %s (%s @ $%s)", trade.EntranceOrder().ExecutionTime.UTC().Format(time.RFC822), trade.EntranceOrder().Security, trade.EntranceOrder().Amount, trade.EntranceOrder().Price))
			fmt.Fprintln(lta.Writer, fmt.Sprintf("%s - exit with buy %s (%s @ $%s)", trade.ExitOrder().ExecutionTime.UTC().Format(time.RFC822), trade.ExitOrder().Security, trade.ExitOrder().Amount, trade.ExitOrder().Price))
			profit = trade.ExitValue().Sub(trade.CostBasis()).Neg()
		} else {
			fmt.Fprintln(lta.Writer, fmt.Sprintf("%s - enter with buy %s (%s @ $%s)", trade.EntranceOrder().ExecutionTime.UTC().Format(time.RFC822), trade.EntranceOrder().Security, trade.EntranceOrder().Amount, trade.EntranceOrder().Price))
			fmt.Fprintln(lta.Writer, fmt.Sprintf("%s - exit with sell %s (%s @ $%s)", trade.ExitOrder().ExecutionTime.UTC().Format(time.RFC822), trade.ExitOrder().Security, trade.ExitOrder().Amount, trade.ExitOrder().Price))
			profit = trade.ExitValue().Sub(trade.CostBasis())
		}
		fmt.Fprintln(lta.Writer, fmt.Sprintf("Profit: $%s", profit))
	}

	for _, trade := range record.Trades {
		if trade.IsClosed() {
			logOrder(trade)
		}
	}
	return 0.0
}

// PeriodProfitAnalysis analyzes the trading record for the average profit based on the time period provided.
// i.e., if the trading record spans a year of trading, and PeriodProfitAnalysis wraps one month, Analyze will return
// the total profit for the whole time period divided by 12.
type PeriodProfitAnalysis struct {
	Period time.Duration
}

// Analyze returns the average profit for the trading record based on the given duration
func (ppa PeriodProfitAnalysis) Analyze(record *TradingRecord) float64 {
	var tp TotalProfitAnalysis
	totalProfit := tp.Analyze(record)

	periods := record.Trades[len(record.Trades)-1].ExitOrder().ExecutionTime.Sub(record.Trades[0].EntranceOrder().ExecutionTime) / ppa.Period
	return totalProfit / float64(periods)
}

// ProfitableTradesAnalysis analyzes the trading record for the number of profitable trades
type ProfitableTradesAnalysis struct{}

// Analyze returns the number of profitable trades in a trading record
func (pta ProfitableTradesAnalysis) Analyze(record *TradingRecord) float64 {
	var profitableTrades int
	for _, trade := range record.Trades {
		costBasis := trade.EntranceOrder().Amount.Mul(trade.EntranceOrder().Price)
		sellPrice := trade.ExitOrder().Amount.Mul(trade.ExitOrder().Price)

		if (trade.IsLong() && sellPrice.GT(costBasis)) || (trade.IsShort() && sellPrice.LT(costBasis)) {
			profitableTrades++
		}
	}

	return float64(profitableTrades)
}

// AverageProfitAnalysis returns the average profit for the trading record. Average profit is represented as the total
// profit divided by the number of trades executed.
type AverageProfitAnalysis struct{}

// Analyze returns the average profit of the trading record
func (apa AverageProfitAnalysis) Analyze(record *TradingRecord) float64 {
	var tp TotalProfitAnalysis
	totalProft := tp.Analyze(record)

	return totalProft / float64(len(record.Trades))
}

// BuyAndHoldAnalysis returns the profit based on a hypothetical where a purchase order was made on the first period available
// and held until the date on the last trade of the trading record. It's useful for comparing the performance of your strategy
// against a simple long position.
type BuyAndHoldAnalysis struct {
	TimeSeries    *TimeSeries
	StartingMoney float64
}

// Analyze returns the profit based on a simple buy and hold strategy
func (baha BuyAndHoldAnalysis) Analyze(record *TradingRecord) float64 {
	if len(record.Trades) == 0 {
		return 0
	}

	openOrder := Order{
		Side:   BUY,
		Amount: big.NewDecimal(baha.StartingMoney).Div(baha.TimeSeries.Candles[0].ClosePrice),
		Price:  baha.TimeSeries.Candles[0].ClosePrice,
	}

	closeOrder := Order{
		Side:   SELL,
		Amount: openOrder.Amount,
		Price:  baha.TimeSeries.Candles[len(baha.TimeSeries.Candles)-1].ClosePrice,
	}

	pos := NewPosition(openOrder)
	pos.Exit(closeOrder)

	return pos.ExitValue().Sub(pos.CostBasis()).Float()
}

// CommissionAnalysis calculates the total commission paid.
// Commission is the commission value in percent.
type CommissionAnalysis struct {
	Commission float64
}

// Analyze analyzes the trading record for the total commission cost.
func (ca CommissionAnalysis) Analyze(record *TradingRecord) float64 {
	total := big.ZERO
	for _, trade := range record.Trades {
		total = total.Add(trade.CostBasis().Mul(big.NewDecimal(ca.Commission * 0.01)))
		total = total.Add(trade.ExitValue().Mul(big.NewDecimal(ca.Commission * 0.01)))
	}
	return total.Float()
}

type OpenPLAnalysis struct {
	LastCandle *Candle
}

// Analyze calculates the profit if the current open position would be closed at current price.
func (o OpenPLAnalysis) Analyze(record *TradingRecord) float64 {
	if record.CurrentPosition().IsNew() {
		return 0
	}
	var profit big.Decimal
	trade := record.CurrentPosition()
	amount := trade.EntranceOrder().Amount
	if trade.IsShort() {
		profit = o.LastCandle.ClosePrice.Mul(amount).Sub(trade.CostBasis()).Neg()
	} else if trade.IsLong() {
		profit = o.LastCandle.ClosePrice.Mul(amount).Sub(trade.CostBasis())
	}
	return profit.Float()
}

func isProfitable(trade *Position) bool {
	return (trade.IsLong() && trade.ExitOrder().Price.GT(trade.EntranceOrder().Price)) || (trade.IsShort() && trade.ExitOrder().Price.LT(trade.EntranceOrder().Price))
}

type WinStreakAnalysis struct{}

func (w WinStreakAnalysis) Analyze(record *TradingRecord) float64 {
	max := 0
	currentStreak := 0
	for _, trade := range record.Trades {
		if !isProfitable(trade) {
			max = Max(max, currentStreak)
			currentStreak = 0
			continue
		}
		currentStreak++
	}
	return math.Max(float64(max), float64(currentStreak))
}

type LoseStreakAnalysis struct{}

func (l LoseStreakAnalysis) Analyze(record *TradingRecord) float64 {
	max := 0
	currentStreak := 0
	for _, trade := range record.Trades {
		if isProfitable(trade) {
			max = Max(max, currentStreak)
			currentStreak = 0
			continue
		}
		currentStreak++
	}
	return math.Max(float64(max), float64(currentStreak))
}

type MaxWinAnalysis struct{}

func (m MaxWinAnalysis) Analyze(record *TradingRecord) float64 {
	maxProfit := big.ZERO
	for _, trade := range record.Trades {
		if !isProfitable(trade) {
			continue
		}
		if trade.IsShort() {
			maxProfit = big.MaxSlice(maxProfit, trade.ExitValue().Sub(trade.CostBasis()).Neg())
		} else {
			maxProfit = big.MaxSlice(maxProfit, trade.ExitValue().Sub(trade.CostBasis()))
		}
	}
	return maxProfit.Float()
}

type MaxLossAnalysis struct{}

func (m MaxLossAnalysis) Analyze(record *TradingRecord) float64 {
	minProfit := big.ZERO
	for _, trade := range record.Trades {
		if isProfitable(trade) {
			continue
		}
		if trade.IsShort() {
			minProfit = big.MinSlice(minProfit, trade.ExitValue().Sub(trade.CostBasis()).Neg())
		} else {
			minProfit = big.MinSlice(minProfit, trade.ExitValue().Sub(trade.CostBasis()))
		}
	}
	return minProfit.Float()
}

type AverageWinAnalysis struct{}

func (a AverageWinAnalysis) Analyze(record *TradingRecord) float64 {
	win := big.ZERO
	count := len(record.Trades)
	for _, trade := range record.Trades {
		if !isProfitable(trade) {
			continue
		}
		count++
		if trade.IsShort() {
			win = win.Add(trade.ExitValue().Sub(trade.CostBasis()).Neg())
		} else {
			win = win.Add(trade.ExitValue().Sub(trade.CostBasis()))
		}
	}
	return win.Div(big.NewFromInt(count)).Float()
}

type AverageLossAnalysis struct{}

func (a AverageLossAnalysis) Analyze(record *TradingRecord) float64 {
	loss := big.ZERO
	count := len(record.Trades)
	for _, trade := range record.Trades {
		if isProfitable(trade) {
			continue
		}
		if trade.IsShort() {
			loss = loss.Add(trade.ExitValue().Sub(trade.CostBasis()).Neg())
		} else {
			loss = loss.Add(trade.ExitValue().Sub(trade.CostBasis()))
		}
	}
	return loss.Div(big.NewFromInt(count)).Float()
}
