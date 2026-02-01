package model

import "testing"

func TestMarketSymbolForExchange(t *testing.T) {
	t.Parallel()

	market := Market{
		Symbol: "ETH-USD",
		ExchangeSymbols: map[Exchange]string{
			ExchangeHyperliquid: "ETH",
		},
	}

	if got := market.SymbolFor(ExchangeHyperliquid); got != "ETH" {
		t.Fatalf("expected exchange symbol %q, got %q", "ETH", got)
	}

	if got := market.SymbolFor(ExchangeLighter); got != "ETH-USD" {
		t.Fatalf("expected default symbol %q, got %q", "ETH-USD", got)
	}
}
