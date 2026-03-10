package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/anedos/go-perps/internal/db"
	"github.com/anedos/go-perps/internal/model"
	"github.com/anedos/go-perps/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type queryExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func TestDBStore_Stats(t *testing.T) {
	t.Parallel()

	if os.Getenv("GO_PERPS_TEST_DB") == "" {
		t.Skip("set GO_PERPS_TEST_DB=1 to run the PostgreSQL/TimescaleDB integration test")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, db.Config{
		URL:         testDatabaseURL(),
		MaxConns:    5,
		QueryLogger: testQueryLogger(t),
	})
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.MigrateDir(ctx, pool, filepath.Join(repoRoot(t), "migrations")); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	symbol := fmt.Sprintf("TEST-%d-%d", time.Now().UnixNano(), os.Getpid())
	ts := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	orderBook := processedOrderBookForIntegrationTest(symbol, ts)

	writer := storage.NewWriter(pool, storage.NewWriteThrottler(0))

	wrote, err := writer.Write(ctx, orderBook)
	if err != nil {
		t.Fatalf("write order book: %v", err)
	}
	if !wrote {
		t.Fatal("expected writer to persist order book")
	}

	dbstore := NewDBStore(pool)

	since := ts.Add(-10 * time.Minute)
	stats, err := dbstore.Stats(ctx, db.MetricsParams{Symbol: symbol, Since: since, Exchange: string(model.ExchangeHyperliquid)})
	assert.Equal(t, len(stats), 1, "unexpected number of stats")
	assert.EqualExportedValues(t, db.StatsRow{
		Exchange:    string(model.ExchangeHyperliquid),
		SampleCount: int64(1),
		AvgSpread:   "2.0000000000000000",
		AvgDepthBid: "3.0000000000000000",
		AvgDepthAsk: "4.0000000000000000",
	},
		stats[0],
		"unexpected order book")

	cleanupRows(t, ctx, pool, symbol)
}

// dups from the other integration test, will unify them later
func testQueryLogger(t *testing.T) *zap.Logger {
	t.Helper()

	if os.Getenv("GO_PERPS_LOG_QUERIES") != "1" {
		return nil
	}

	return zaptest.NewLogger(t)
}

func testDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	if user := os.Getenv("USER"); user != "" {
		return "postgres:///" + user
	}

	return "postgres:///postgres"
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func processedOrderBookForIntegrationTest(symbol string, now time.Time) model.ProcessedOrderBook {
	return model.ProcessedOrderBook{
		Exchange:  model.ExchangeHyperliquid,
		Symbol:    symbol,
		Timestamp: now,
		Spread:    decimal.NewFromInt(2),
		Depth: model.SideMetric{
			Bid: decimal.NewFromInt(3),
			Ask: decimal.NewFromInt(4),
		},
		Slippage: []model.SlippageLevel{
			{
				Size: decimal.NewFromInt(100),
				Slippage: model.SideMetric{
					Bid: decimal.RequireFromString("0.01"),
					Ask: decimal.RequireFromString("0.02"),
				},
			},
			{
				Size: decimal.NewFromInt(250),
				Slippage: model.SideMetric{
					Bid: decimal.RequireFromString("0.02"),
					Ask: decimal.RequireFromString("0.03"),
				},
			},
		},
	}
}

func cleanupRows(t *testing.T, ctx context.Context, execer queryExecer, symbol string) {
	t.Helper()

	if _, err := execer.Exec(ctx, "DELETE FROM slippage WHERE symbol = $1", symbol); err != nil {
		t.Fatalf("cleanup slippage rows: %v", err)
	}
	if _, err := execer.Exec(ctx, "DELETE FROM snapshots WHERE symbol = $1", symbol); err != nil {
		t.Fatalf("cleanup snapshot rows: %v", err)
	}
}
