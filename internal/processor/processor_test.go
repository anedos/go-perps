package processor

import (
	"testing"
	"time"

	"github.com/anedos/go-perps/internal/model"
	"github.com/sanity-io/litter"
	"github.com/shopspring/decimal"
)

func TestProcessorCalculatesSpreadDepthAndSlippage(t *testing.T) {
	// boilerplate, test can run in parallel with other tests marked with t.Parallel
	t.Parallel()

	p := New([]decimal.Decimal{dec("100"), dec("250")})
	orderBook := newOrderBook(
		[]model.PriceLevel{
			priceLevel("99", "1"),
			priceLevel("98", "2"),
			priceLevel("49", "100"),
		},
		[]model.PriceLevel{
			priceLevel("101", "1"),
			priceLevel("102", "2"),
			priceLevel("151", "100"),
		},
	)

	processed := p.Process(orderBook)
	litter.Dump(processed)

	assertDecimal(t, processed.Spread, "2")
	assertDecimal(t, processed.Depth.Bid, "3")
	assertDecimal(t, processed.Depth.Ask, "3")

	if len(processed.Slippage) != 2 {
		t.Fatalf("expected 2 slippage levels, got %d", len(processed.Slippage))
	}

	assertDecimal(t, processed.Slippage[0].Size, "100")
	assertDecimal(t, processed.Slippage[0].Slippage.Bid, "0.01")
	assertDecimal(t, processed.Slippage[0].Slippage.Ask, "0.02")
	assertDecimal(t, processed.Slippage[1].Size, "250")
	assertDecimal(t, processed.Slippage[1].Slippage.Bid, "0.02")
	assertDecimal(t, processed.Slippage[1].Slippage.Ask, "0.02")
}

func TestProcessorReturnsZeroMetricsForEmptyOrderBook(t *testing.T) {
	t.Parallel()

	p := New([]decimal.Decimal{dec("100")})
	processed := p.Process(newOrderBook(nil, nil))

	assertDecimal(t, processed.Spread, "0")
	assertDecimal(t, processed.Depth.Bid, "0")
	assertDecimal(t, processed.Depth.Ask, "0")

	if len(processed.Slippage) != 1 {
		t.Fatalf("expected 1 slippage level, got %d", len(processed.Slippage))
	}

	assertDecimal(t, processed.Slippage[0].Size, "100")
	assertDecimal(t, processed.Slippage[0].Slippage.Bid, "0")
	assertDecimal(t, processed.Slippage[0].Slippage.Ask, "0")
}

func TestProcessorReturnsZeroSlippageWhenBookCannotFillSize(t *testing.T) {
	t.Parallel()

	p := New([]decimal.Decimal{dec("1000000")})
	orderBook := newOrderBook(
		[]model.PriceLevel{priceLevel("99", "1")},
		[]model.PriceLevel{priceLevel("101", "1")},
	)

	processed := p.Process(orderBook)

	assertDecimal(t, processed.Slippage[0].Slippage.Bid, "0")
	assertDecimal(t, processed.Slippage[0].Slippage.Ask, "0")
}

func TestNewCopiesSlippageLevels(t *testing.T) {
	t.Parallel()

	levels := []decimal.Decimal{dec("100")}
	p := New(levels)
	levels[0] = dec("250")

	processed := p.Process(newOrderBook(
		[]model.PriceLevel{priceLevel("99", "2")},
		[]model.PriceLevel{priceLevel("101", "2")},
	))

	assertDecimal(t, processed.Slippage[0].Size, "100")
}

func newOrderBook(bids, asks []model.PriceLevel) model.OrderBook {
	return model.OrderBook{
		Exchange:  model.ExchangeHyperliquid,
		Symbol:    "ETH-USD",
		Timestamp: time.Unix(1_704_241_860, 0).UTC(),
		Bids:      bids,
		Asks:      asks,
	}
}

func priceLevel(price, quantity string) model.PriceLevel {
	return model.PriceLevel{
		Price:    dec(price),
		Quantity: dec(quantity),
	}
}

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func assertDecimal(t *testing.T, got decimal.Decimal, want string) {
	t.Helper()

	expected := dec(want)
	if !got.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}
