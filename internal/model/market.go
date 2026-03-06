package model

// Market describes a normalized trading market and optional exchange-specific symbol mappings.
type Market struct {
	// Symbol is the normalized market identifier
	Symbol string
	// ExchangeSymbols maps exchanges to the symbol expected by that exchange
	ExchangeSymbols map[Exchange]string
}

// SymbolFor returns the exchange-specific symbol when configured, otherwise it
// falls back to the normalized market symbol.
func (m Market) SymbolFor(exchange Exchange) string {
	if symbol, ok := m.ExchangeSymbols[exchange]; ok {
		return symbol
	}

	return m.Symbol
}
