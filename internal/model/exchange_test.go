package model

import "testing"

func TestParseExchange(t *testing.T) {
	t.Parallel()

	exchange, err := ParseExchange("hyperliquid")
	if err != nil {
		t.Fatalf("ParseExchange returned error: %v", err)
	}

	if exchange != ExchangeHyperliquid {
		t.Fatalf("expected %q, got %q", ExchangeHyperliquid, exchange)
	}
}

func TestParseExchangeRejectsUnknownExchange(t *testing.T) {
	t.Parallel()

	if _, err := ParseExchange("unknown"); err == nil {
		t.Fatal("expected error for unknown exchange")
	}
}

func TestAllExchangesAreValid(t *testing.T) {
	t.Parallel()

	for _, exchange := range AllExchanges {
		if !exchange.Valid() {
			t.Fatalf("expected %q to be valid", exchange)
		}
	}
}
