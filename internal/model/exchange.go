package model

import "fmt"

// Exchange identifies a perpetual futures exchange supported by the system.
type Exchange string

const (
	// ExchangeExtended identifies Extended.
	ExchangeExtended Exchange = "extended"
	// ExchangeHyperliquid identifies Hyperliquid.
	ExchangeHyperliquid Exchange = "hyperliquid"
	// ExchangeLighter identifies Lighter.
	ExchangeLighter Exchange = "lighter"
)

// AllExchanges lists every exchange known to the domain model.
var AllExchanges = []Exchange{
	ExchangeExtended,
	ExchangeHyperliquid,
	ExchangeLighter,
}

// ParseExchange converts a string exchange identifier into an Exchange.
func ParseExchange(value string) (Exchange, error) {
	exchange := Exchange(value)

	if exchange.Valid() {
		return exchange, nil
	}

	return "", fmt.Errorf("unknown exchange %q", value)
}

// String returns the wire-format exchange identifier.
func (e Exchange) String() string {
	return string(e)
}

// Valid reports whether e is one of the supported exchanges.
func (e Exchange) Valid() bool {
	switch e {
	case ExchangeExtended, ExchangeHyperliquid, ExchangeLighter:
		return true
	default:
		return false
	}
}
