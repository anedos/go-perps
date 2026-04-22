// Package ingest coordinates exchange readers, processing, and storage writes
package ingest

import (
	"context"
	"fmt"
	"sync"

	"github.com/anedos/go-perps/internal/exchange"
	"github.com/anedos/go-perps/internal/model"
	"github.com/anedos/go-perps/internal/processor"
	"go.uber.org/zap"
)

// Writer persists processed order book snapshots
type Writer interface {
	Write(ctx context.Context, orderBook model.ProcessedOrderBook) (bool, error)
}

// Config contains dependencies for the ingestion runner.
type Config struct {
	Markets   []model.Market
	Readers   []exchange.Reader
	Processor *processor.Processor
	Writer    Writer
	Logger    *zap.Logger
}

// Runner fans exchange reader output into the processor and storage writer.
type Runner struct {
	config Config
	logger *zap.Logger
}

// New creates an ingestion runner.
func New(config Config) (*Runner, error) {
	if len(config.Markets) == 0 {
		return nil, fmt.Errorf("markets are required")
	}
	if len(config.Readers) == 0 {
		return nil, fmt.Errorf("readers are required")
	}
	if config.Processor == nil {
		return nil, fmt.Errorf("processor is required")
	}
	if config.Writer == nil {
		return nil, fmt.Errorf("writer is required")
	}
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}

	return &Runner{
		config: config,
		logger: config.Logger,
	}, nil
}

// Run starts readers and processes order books until ctx is canceled or a
// storage write fails.
func (r *Runner) Run(ctx context.Context) error {
	orderBooks := make(chan model.OrderBook, len(r.config.Readers))
	readerErrors := make(chan error, len(r.config.Readers))

	var wg sync.WaitGroup
	for _, reader := range r.config.Readers {
		if err := reader.Connect(ctx, r.config.Markets); err != nil {
			return fmt.Errorf("connect reader: %w", err)
		}
		defer reader.Close()

		wg.Add(1)
		go fanInReader(ctx, &wg, reader, orderBooks, readerErrors)
	}

	go func() {
		wg.Wait()
		close(orderBooks)
		close(readerErrors)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-readerErrors:
			if !ok {
				readerErrors = nil
				continue
			}
			r.logger.Warn("reader error", zap.Error(err))
		case orderBook, ok := <-orderBooks:
			if !ok {
				orderBooks = nil
				if readerErrors == nil {
					return nil
				}
				continue
			}

			processed := r.config.Processor.Process(orderBook)
			wrote, err := r.config.Writer.Write(ctx, processed)
			if err != nil {
				return fmt.Errorf("write processed order book: %w", err)
			}

			r.logger.Debug("processed order book",
				zap.String("exchange", processed.Exchange.String()),
				zap.String("symbol", processed.Symbol),
				zap.Bool("wrote", wrote),
			)
		}
	}
}

func fanInReader(
	ctx context.Context,
	wg *sync.WaitGroup,
	reader exchange.Reader,
	orderBooks chan<- model.OrderBook,
	readerErrors chan<- error,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case orderBook, ok := <-reader.Stream():
			if !ok {
				return
			}
			select {
			case <-ctx.Done():
				return
			case orderBooks <- orderBook:
			}
		case err, ok := <-reader.Errors():
			if !ok {
				return
			}
			select {
			case <-ctx.Done():
				return
			case readerErrors <- err:
			}
		}
	}
}
