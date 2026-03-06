package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/anedos/go-perps/internal/api"
	"github.com/anedos/go-perps/internal/logging"
	"github.com/anedos/go-perps/internal/model"
	"go.uber.org/zap"
)

func main() {
	logger := logging.MustNew(logging.Config{
		Development: isDevelopment(),
		Service:     "api",
	})
	defer logger.Sync() //nolint:errcheck

	if err := run(logger); err != nil {
		logger.Fatal("api exited", zap.Error(err))
	}
}

func run(logger *zap.Logger) error {
	server := &http.Server{
		Addr:         apiAddress(),
		Handler:      api.New(api.Config{Markets: configuredMarkets()}, logger),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown api server", zap.Error(err))
		}
	}()

	logger.Info("starting api server", zap.String("addr", server.Addr))
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func apiAddress() string {
	if value := os.Getenv("GO_PERPS_API_ADDR"); value != "" {
		return value
	}

	return ":8080"
}

func configuredMarkets() []model.Market {
	value := os.Getenv("GO_PERPS_MARKETS")
	if value == "" {
		return []model.Market{
			{Symbol: "ETH-USD"},
			{Symbol: "BTC-USD"},
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

func isDevelopment() bool {
	return os.Getenv("GO_PERPS_ENV") != "production"
}
