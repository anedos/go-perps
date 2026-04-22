package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anedos/go-perps/internal/exchange"
	"github.com/anedos/go-perps/internal/model"
	"github.com/anedos/go-perps/internal/processor"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunnerProcessesAndWritesOrderBooks(t *testing.T) {
	t.Parallel()

	reader := newFakeReader()
	writer := &fakeWriter{}
	runner, err := New(Config{
		Markets:   []model.Market{{Symbol: "ETH-USD"}},
		Readers:   []exchange.Reader{reader},
		Processor: processor.New([]decimal.Decimal{decimal.NewFromInt(100)}),
		Writer:    writer,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	reader.stream <- orderBook("ETH-USD")

	require.Eventually(t, func() bool {
		return len(writer.writes) == 1
	}, time.Second, 10*time.Millisecond)

	cancel()
	require.NoError(t, <-done)
	assert.Equal(t, "ETH-USD", writer.writes[0].Symbol)
	assert.True(t, reader.closed)
}

func TestRunnerReturnsWriterErrors(t *testing.T) {
	t.Parallel()

	reader := newFakeReader()
	writeErr := errors.New("write failed")
	runner, err := New(Config{
		Markets:   []model.Market{{Symbol: "ETH-USD"}},
		Readers:   []exchange.Reader{reader},
		Processor: processor.New(nil),
		Writer:    &fakeWriter{err: writeErr},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	reader.stream <- orderBook("ETH-USD")

	require.ErrorIs(t, <-done, writeErr)
	assert.True(t, reader.closed)
}

func TestRunnerValidation(t *testing.T) {
	t.Parallel()

	_, err := New(Config{})
	require.Error(t, err)
}

type fakeExchangeReader struct {
	stream    chan model.OrderBook
	errs      chan error
	connected bool
	closed    bool
}

func newFakeReader() *fakeExchangeReader {
	return &fakeExchangeReader{
		stream: make(chan model.OrderBook, 1),
		errs:   make(chan error, 1),
	}
}

func (r *fakeExchangeReader) Connect(context.Context, []model.Market) error {
	r.connected = true
	return nil
}

func (r *fakeExchangeReader) Stream() <-chan model.OrderBook {
	return r.stream
}

func (r *fakeExchangeReader) Errors() <-chan error {
	return r.errs
}

func (r *fakeExchangeReader) Close() error {
	r.closed = true
	return nil
}

type fakeWriter struct {
	writes []model.ProcessedOrderBook
	err    error
}

func (w *fakeWriter) Write(_ context.Context, orderBook model.ProcessedOrderBook) (bool, error) {
	if w.err != nil {
		return false, w.err
	}

	w.writes = append(w.writes, orderBook)
	return true, nil
}

func orderBook(symbol string) model.OrderBook {
	return model.OrderBook{
		Exchange:  model.ExchangeHyperliquid,
		Symbol:    symbol,
		Timestamp: time.Unix(1_704_241_860, 0).UTC(),
		Bids: []model.PriceLevel{
			{
				Price:    decimal.NewFromInt(99),
				Quantity: decimal.NewFromInt(1),
			},
		},
		Asks: []model.PriceLevel{
			{
				Price:    decimal.NewFromInt(101),
				Quantity: decimal.NewFromInt(1),
			},
		},
	}
}
