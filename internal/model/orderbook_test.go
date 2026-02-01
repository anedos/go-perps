package model

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestOrderBookCarriesNormalizedDepth(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_704_241_860, 0).UTC()
	orderBook := OrderBook{
		Exchange:  ExchangeHyperliquid,
		Symbol:    "ETH-USD",
		Timestamp: now,
		Bids: []PriceLevel{
			{
				Price:    decimal.NewFromInt(99),
				Quantity: decimal.NewFromInt(2),
			},
		},
		Asks: []PriceLevel{
			{
				Price:    decimal.NewFromInt(101),
				Quantity: decimal.NewFromInt(3),
			},
		},
	}

	if orderBook.Timestamp != now {
		t.Fatalf("expected timestamp %s, got %s", now, orderBook.Timestamp)
	}

	if !orderBook.Bids[0].Price.Equal(decimal.NewFromInt(99)) {
		t.Fatalf("expected bid price 99, got %s", orderBook.Bids[0].Price)
	}
}

func TestProcessedOrderBookCarriesDerivedMetrics(t *testing.T) {
	t.Parallel()

	processed := ProcessedOrderBook{
		Exchange: ExchangeHyperliquid,
		Symbol:   "ETH-USD",
		Spread:   decimal.NewFromInt(2),
		Depth: SideMetric{
			Bid: decimal.NewFromInt(5),
			Ask: decimal.NewFromInt(6),
		},
		Slippage: []SlippageLevel{
			{
				Size: decimal.NewFromInt(100),
				Slippage: SideMetric{
					Bid: decimal.RequireFromString("0.01"),
					Ask: decimal.RequireFromString("0.02"),
				},
			},
		},
	}

	if !processed.Spread.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("expected spread 2, got %s", processed.Spread)
	}

	if len(processed.Slippage) != 1 {
		t.Fatalf("expected 1 slippage level, got %d", len(processed.Slippage))
	}
}
