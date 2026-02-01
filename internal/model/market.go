package model

type Market struct {
	Symbol          string
	ExchangeSymbols map[Exchange]string
}

func (m Market) SymbolFor(exchange Exchange) string {
	if symbol, ok := m.ExchangeSymbols[exchange]; ok {
		return symbol
	}

	return m.Symbol
}
