package processor

import (
	"github.com/anedos/go-perps/internal/model"
	"github.com/sanity-io/litter"
	"github.com/shopspring/decimal"
)

// midPriceDepthBand is used to calculate the price range which is then used for the liquidity depth calculations
const midPriceDepthBand = 0.5

// Processor converts normalized order book snapshots into derived metrics
type Processor struct {
	slippageLevels []decimal.Decimal
}

func New(slippageLevels []decimal.Decimal) *Processor {
	levels := make([]decimal.Decimal, len(slippageLevels))
	copy(levels, slippageLevels)

	return &Processor{
		slippageLevels: levels,
	}
}

// Process calculates spread, depth, and slippage for one order book snapshot
func (p *Processor) Process(orderBook model.OrderBook) model.ProcessedOrderBook {
	litter.Dump(orderBook)

	return model.ProcessedOrderBook{
		Exchange:  orderBook.Exchange,
		Symbol:    orderBook.Symbol,
		Timestamp: orderBook.Timestamp,
		Spread:    Spread(orderBook),
		Depth:     Depth(orderBook),
		Slippage:  SlippageLevels(orderBook, p.slippageLevels),
	}
}

// Spread returns best ask minus best bid. It returns zero when either side is empty.
func Spread(orderBook model.OrderBook) decimal.Decimal {
	if len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		return decimal.Zero
	}

	return orderBook.Asks[0].Price.Sub(orderBook.Bids[0].Price)
}

// MidPrice returns the midpoint between best bid and best ask. It returns zero
// when either side is empty.
func MidPrice(orderBook model.OrderBook) decimal.Decimal {
	if len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		return decimal.Zero
	}

	return orderBook.Bids[0].Price.Add(orderBook.Asks[0].Price).Div(decimal.NewFromInt(2))
}

// Depth sums bid and ask quantities inside the configured mid-price band
func Depth(orderBook model.OrderBook) model.SideMetric {
	midPrice := MidPrice(orderBook)
	if midPrice.IsZero() {
		return model.SideMetric{
			Bid: decimal.Zero,
			Ask: decimal.Zero,
		}
	}

	minBidPrice := midPrice.Mul(decimal.NewFromFloat(1 - midPriceDepthBand))
	maxAskPrice := midPrice.Mul(decimal.NewFromFloat(1 + midPriceDepthBand))

	bidDepth := decimal.Zero
	for _, bid := range orderBook.Bids {
		if bid.Price.LessThan(minBidPrice) {
			break
		}
		bidDepth = bidDepth.Add(bid.Quantity)
	}

	askDepth := decimal.Zero
	for _, ask := range orderBook.Asks {
		if ask.Price.GreaterThan(maxAskPrice) {
			break
		}
		askDepth = askDepth.Add(ask.Quantity)
	}

	return model.SideMetric{
		Bid: bidDepth,
		Ask: askDepth,
	}
}

// SlippageLevels calculates slippage for each configured quote-notional size
func SlippageLevels(orderBook model.OrderBook, levels []decimal.Decimal) []model.SlippageLevel {
	midPrice := MidPrice(orderBook)
	result := make([]model.SlippageLevel, 0, len(levels))

	for _, size := range levels {
		buyPrice := ExecutionPrice(orderBook.Asks, size)
		sellPrice := ExecutionPrice(orderBook.Bids, size)

		result = append(result, model.SlippageLevel{
			Size: size,
			Slippage: model.SideMetric{
				Bid: Slippage(buyPrice, midPrice),
				Ask: Slippage(sellPrice, midPrice),
			},
		})
	}

	return result
}

// ExecutionPrice returns the price level where notionalSize is fully filled. It
// returns zero when the book side cannot fill the requested size.
func ExecutionPrice(levels []model.PriceLevel, notionalSize decimal.Decimal) decimal.Decimal {
	filled := decimal.Zero

	for _, level := range levels {
		filled = filled.Add(level.Quantity.Mul(level.Price))
		if filled.GreaterThanOrEqual(notionalSize) {
			return level.Price
		}
	}

	return decimal.Zero
}

// Slippage returns absolute price impact versus midPrice. It returns zero when
// either input is zero
func Slippage(executionPrice, midPrice decimal.Decimal) decimal.Decimal {
	if executionPrice.IsZero() || midPrice.IsZero() {
		return decimal.Zero
	}

	return executionPrice.Sub(midPrice).Div(midPrice).Abs()
}
