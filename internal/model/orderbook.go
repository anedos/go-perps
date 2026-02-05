package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// PriceLevel is one price/quantity level on one side of an order book.
type PriceLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

// OrderBook is a normalized exchange order book snapshot.
type OrderBook struct {
	Exchange Exchange
	Symbol   string
	// Timestamp is the exchange or ingestion timestamp for this snapshot, TBD exact semantics
	Timestamp time.Time
	// Bids are sorted from best bid to worse bids.
	Bids []PriceLevel
	// Asks are sorted from best ask to worse asks.
	Asks []PriceLevel
}

// SideMetric stores one metric value for both bid and ask sides.
type SideMetric struct {
	Bid decimal.Decimal
	Ask decimal.Decimal
}

type SlippageLevel struct {
	Size     decimal.Decimal
	Slippage SideMetric
}

// ProcessedOrderBook contains exchange-neutral analytics derived from an OrderBook snapshot.
type ProcessedOrderBook struct {
	Exchange Exchange
	Symbol   string
	// Timestamp is copied from the source order book snapshot.
	Timestamp time.Time
	// Spread is best ask minus best bid.
	Spread decimal.Decimal
	// Depth is bid and ask quantity inside the configured mid-price band.
	Depth SideMetric
	// Slippage stores price impact at each configured notional size.
	Slippage []SlippageLevel
}
