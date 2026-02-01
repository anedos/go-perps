package model

import "fmt"

type Exchange string

const (
	ExchangeExtended    Exchange = "extended"
	ExchangeHyperliquid Exchange = "hyperliquid"
	ExchangeLighter     Exchange = "lighter"
)

var AllExchanges = []Exchange{
	ExchangeExtended,
	ExchangeHyperliquid,
	ExchangeLighter,
}

func ParseExchange(value string) (Exchange, error) {
	exchange := Exchange(value)

	if exchange.Valid() {
		return exchange, nil
	}

	return "", fmt.Errorf("unknown exchange %q", value)
}

func (e Exchange) String() string {
	return string(e)
}

func (e Exchange) Valid() bool {
	switch e {
	case ExchangeExtended, ExchangeHyperliquid, ExchangeLighter:
		return true
	default:
		return false
	}
}
