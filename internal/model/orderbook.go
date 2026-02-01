package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type PriceLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

type OrderBook struct {
	Exchange  Exchange
	Symbol    string
	Timestamp time.Time
	Bids      []PriceLevel
	Asks      []PriceLevel
}

type SideMetric struct {
	Bid decimal.Decimal
	Ask decimal.Decimal
}

type SlippageLevel struct {
	Size     decimal.Decimal
	Slippage SideMetric
}

type ProcessedOrderBook struct {
	Exchange  Exchange
	Symbol    string
	Timestamp time.Time
	Spread    decimal.Decimal
	Depth     SideMetric
	Slippage  []SlippageLevel
}
