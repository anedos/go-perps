package storage

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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type queryExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func TestWriterWritesToTimescaleDB(t *testing.T) {
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

	assertTimescaleDBInstalled(t, ctx, pool)
	assertHypertablesExist(t, ctx, pool)

	symbol := fmt.Sprintf("TEST-%d-%d", time.Now().UnixNano(), os.Getpid())
	orderBook := processedOrderBookForIntegrationTest(symbol)
	writer := NewWriter(pool, NewWriteThrottler(0))

	wrote, err := writer.Write(ctx, orderBook)
	if err != nil {
		t.Fatalf("write order book: %v", err)
	}
	if !wrote {
		t.Fatal("expected writer to persist order book")
	}

	assertRowCount(t, ctx, pool, "snapshots", symbol, 1)
	assertRowCount(t, ctx, pool, "slippage", symbol, 2)

	cleanupRows(t, ctx, pool, symbol)
}

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

func assertTimescaleDBInstalled(t *testing.T, ctx context.Context, execer queryExecer) {
	t.Helper()

	var version string
	err := execer.QueryRow(ctx, `
		SELECT extversion
		FROM pg_extension
		WHERE extname = 'timescaledb'
	`).Scan(&version)
	if err != nil {
		t.Fatalf("query timescaledb extension: %v", err)
	}
	if version == "" {
		t.Fatal("expected TimescaleDB extension version")
	}
}

func assertHypertablesExist(t *testing.T, ctx context.Context, execer queryExecer) {
	t.Helper()

	rows, err := execer.Query(ctx, `
		SELECT hypertable_name
		FROM timescaledb_information.hypertables
		WHERE hypertable_name IN ('snapshots', 'slippage')
	`)
	if err != nil {
		t.Fatalf("query hypertables: %v", err)
	}
	defer rows.Close()

	found := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan hypertable: %v", err)
		}
		found[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate hypertables: %v", err)
	}

	for _, name := range []string{"snapshots", "slippage"} {
		if !found[name] {
			t.Fatalf("expected hypertable %q", name)
		}
	}
}

func assertRowCount(t *testing.T, ctx context.Context, execer queryExecer, table, symbol string, want int) {
	t.Helper()

	var count int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE symbol = $1", table)
	if err := execer.QueryRow(ctx, query, symbol).Scan(&count); err != nil {
		t.Fatalf("count %s rows: %v", table, err)
	}
	if count != want {
		t.Fatalf("expected %d %s rows, got %d", want, table, count)
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

func processedOrderBookForIntegrationTest(symbol string) model.ProcessedOrderBook {
	return model.ProcessedOrderBook{
		Exchange:  model.ExchangeHyperliquid,
		Symbol:    symbol,
		Timestamp: time.Now().UTC(),
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
