package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/anedos/go-perps/internal/db"
	"github.com/anedos/go-perps/internal/exchange"
	"github.com/anedos/go-perps/internal/exchange/hyperliquid"
	"github.com/anedos/go-perps/internal/ingest"
	"github.com/anedos/go-perps/internal/logging"
	"github.com/anedos/go-perps/internal/model"
	"github.com/anedos/go-perps/internal/processor"
	"github.com/anedos/go-perps/internal/storage"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func main() {
	logger := logging.MustNew(logging.Config{
		Development: isDevelopment(),
		Service:     "reader",
	})
	defer logger.Sync() //nolint:errcheck

	if err := run(logger); err != nil {
		logger.Fatal("reader exited", zap.Error(err))
	}
}

// XXX pass a context through the call stack instead of the logger
func run(logger *zap.Logger) error {
	databaseURL := os.Getenv("GO_PERPS_DATABASE_URL")
	if databaseURL == "" {
		return errors.New("GO_PERPS_DATABASE_URL is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, db.Config{
		URL:         databaseURL,
		MaxConns:    5,
		QueryLogger: logger,
	})
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	runner, err := ingest.New(ingest.Config{
		Markets:   configuredMarkets(),
		Readers:   configuredReaders(),
		Processor: processor.New(configuredSlippageLevels()),
		Writer:    storage.NewWriter(pool, storage.NewWriteThrottler(configuredThrottleInterval())),
		Logger:    logger,
	})
	if err != nil {
		return err
	}

	logger.Info("starting reader")
	return runner.Run(ctx)
}

func configuredReaders() []exchange.Reader {
	return []exchange.Reader{
		hyperliquid.New(),
	}
}

func configuredMarkets() []model.Market {
	value := os.Getenv("GO_PERPS_MARKETS")
	if value == "" {
		return []model.Market{
			{
				Symbol: "ETH-USD",
				ExchangeSymbols: map[model.Exchange]string{
					model.ExchangeHyperliquid: "ETH",
				},
			},
			{
				Symbol: "BTC-USD",
				ExchangeSymbols: map[model.Exchange]string{
					model.ExchangeHyperliquid: "BTC",
				},
			},
		}
	}

	parts := strings.Split(value, ",")
	markets := make([]model.Market, 0, len(parts))
	for _, part := range parts {
		symbol := strings.TrimSpace(part)
		if symbol == "" {
			continue
		}
		markets = append(markets, model.Market{Symbol: symbol})
	}

	return markets
}

func configuredSlippageLevels() []decimal.Decimal {
	value := os.Getenv("GO_PERPS_SLIPPAGE_LEVELS")
	if value == "" {
		return []decimal.Decimal{
			decimal.NewFromInt(100),
			decimal.NewFromInt(250),
			decimal.NewFromInt(500),
		}
	}

	parts := strings.Split(value, ",")
	levels := make([]decimal.Decimal, 0, len(parts))
	for _, part := range parts {
		level := strings.TrimSpace(part)
		if level == "" {
			continue
		}
		levels = append(levels, decimal.RequireFromString(level))
	}

	return levels
}

func configuredThrottleInterval() time.Duration {
	value := os.Getenv("GO_PERPS_WRITER_THROTTLE")
	if value == "" {
		return 5 * time.Second
	}

	interval, err := time.ParseDuration(value)
	if err != nil {
		return 5 * time.Second
	}

	return interval
}

func isDevelopment() bool {
	return os.Getenv("GO_PERPS_ENV") != "production"
}
