package db

import (
	"testing"
	"time"

	"github.com/anedos/go-perps/internal/model"
	"github.com/shopspring/decimal"
)

func TestSnapshotRowFromProcessedOrderBook(t *testing.T) {
	t.Parallel()

	orderBook := processedOrderBook()

	row := SnapshotRowFrom(orderBook)

	if row.Symbol != "ETH-USD" {
		t.Fatalf("expected symbol ETH-USD, got %s", row.Symbol)
	}
	if row.Exchange != "hyperliquid" {
		t.Fatalf("expected exchange hyperliquid, got %s", row.Exchange)
	}
	if row.Spread != "2" {
		t.Fatalf("expected spread 2, got %s", row.Spread)
	}
	if row.DepthBid != "3" || row.DepthAsk != "4" {
		t.Fatalf("expected depth 3/4, got %s/%s", row.DepthBid, row.DepthAsk)
	}
}

func TestSlippageRowsFromProcessedOrderBook(t *testing.T) {
	t.Parallel()

	rows := SlippageRowsFrom(processedOrderBook())

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Size != "100" {
		t.Fatalf("expected size 100, got %s", rows[0].Size)
	}
	if rows[0].SlippageBid != "0.01" || rows[0].SlippageAsk != "0.02" {
		t.Fatalf("expected slippage 0.01/0.02, got %s/%s", rows[0].SlippageBid, rows[0].SlippageAsk)
	}
}

func processedOrderBook() model.ProcessedOrderBook {
	return model.ProcessedOrderBook{
		Exchange:  model.ExchangeHyperliquid,
		Symbol:    "ETH-USD",
		Timestamp: time.Unix(1_704_241_860, 0).UTC(),
		Spread:    decimal.NewFromInt(2),
		Depth: model.SideMetric{
			Bid: decimal.NewFromInt(3),
			Ask: decimal.NewFromInt(4),
		},
		Slippage: []model.SlippageLevel{
			{
				Size: decimal.NewFromInt(100),
				Slippage: model.SideMetric{
					Bid: decimal.RequireFromString("0.01"),
					Ask: decimal.RequireFromString("0.02"),
				},
			},
		},
	}
}
