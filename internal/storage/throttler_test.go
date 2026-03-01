package storage

import (
	"testing"
	"time"

	"github.com/anedos/go-perps/internal/model"
	"github.com/shopspring/decimal"
)

func TestWriteThrottlerThrottlesSameSymbolAndExchange(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_704_241_860, 0).UTC()
	throttler := newWriteThrottler(5*time.Second, func() time.Time {
		return now
	})

	orderBook := processedOrderBook("ETH-USD", model.ExchangeHyperliquid)

	if !throttler.ShouldWrite(orderBook) {
		t.Fatal("expected first write to be allowed")
	}
	if throttler.ShouldWrite(orderBook) {
		t.Fatal("expected second write to be throttled")
	}
}

func TestWriteThrottlerAllowsDifferentSymbolOrExchange(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_704_241_860, 0).UTC()
	throttler := newWriteThrottler(5*time.Second, func() time.Time {
		return now
	})

	if !throttler.ShouldWrite(processedOrderBook("ETH-USD", model.ExchangeHyperliquid)) {
		t.Fatal("expected first symbol to be allowed")
	}
	if !throttler.ShouldWrite(processedOrderBook("BTC-USD", model.ExchangeHyperliquid)) {
		t.Fatal("expected different symbol to be allowed")
	}
	if !throttler.ShouldWrite(processedOrderBook("ETH-USD", model.ExchangeLighter)) {
		t.Fatal("expected different exchange to be allowed")
	}
}

func TestWriteThrottlerAllowsWriteAfterInterval(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_704_241_860, 0).UTC()
	throttler := newWriteThrottler(5*time.Second, func() time.Time {
		return now
	})
	orderBook := processedOrderBook("ETH-USD", model.ExchangeHyperliquid)

	if !throttler.ShouldWrite(orderBook) {
		t.Fatal("expected first write to be allowed")
	}

	now = now.Add(5 * time.Second)

	if !throttler.ShouldWrite(orderBook) {
		t.Fatal("expected write after interval to be allowed")
	}
}

func processedOrderBook(symbol string, exchange model.Exchange) model.ProcessedOrderBook {
	return model.ProcessedOrderBook{
		Exchange:  exchange,
		Symbol:    symbol,
		Timestamp: time.Unix(1_704_241_860, 0).UTC(),
		Spread:    decimal.NewFromInt(2),
		Depth: model.SideMetric{
			Bid: decimal.NewFromInt(3),
			Ask: decimal.NewFromInt(4),
		},
	}
}
