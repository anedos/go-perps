package db

import (
	"context"
	"time"

	"github.com/anedos/go-perps/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
)

// Execer is the subset of pgxpool.Pool used by write queries, might be a premature abstraction but helps with testing
// right now
type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// SnapshotRow is one row for the snapshots hypertable.
type SnapshotRow struct {
	Time     time.Time
	Symbol   string
	Exchange string
	Spread   string
	DepthBid string
	DepthAsk string
}

// SnapshotRowFrom converts a processed order book into a database row.
func SnapshotRowFrom(orderBook model.ProcessedOrderBook) SnapshotRow {
	return SnapshotRow{
		Time:     orderBook.Timestamp,
		Symbol:   orderBook.Symbol,
		Exchange: orderBook.Exchange.String(),
		Spread:   orderBook.Spread.String(),
		DepthBid: orderBook.Depth.Bid.String(),
		DepthAsk: orderBook.Depth.Ask.String(),
	}
}

// SlippageRow is one row for the slippage hypertable.
type SlippageRow struct {
	Time        time.Time
	Symbol      string
	Exchange    string
	Size        string
	SlippageBid string
	SlippageAsk string
}

// SlippageRowsFrom converts processed slippage levels into database rows.
func SlippageRowsFrom(orderBook model.ProcessedOrderBook) []SlippageRow {
	rows := make([]SlippageRow, 0, len(orderBook.Slippage))
	for _, level := range orderBook.Slippage {
		rows = append(rows, SlippageRow{
			Time:        orderBook.Timestamp,
			Symbol:      orderBook.Symbol,
			Exchange:    orderBook.Exchange.String(),
			Size:        level.Size.String(),
			SlippageBid: level.Slippage.Bid.String(),
			SlippageAsk: level.Slippage.Ask.String(),
		})
	}

	return rows
}

// InsertSnapshot upserts one processed order book snapshot.
func InsertSnapshot(ctx context.Context, execer Execer, row SnapshotRow) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO snapshots (
			time,
			symbol,
			exchange,
			spread,
			depth_bid,
			depth_ask
		)
		VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6::numeric)
		ON CONFLICT (time, symbol, exchange) DO UPDATE
		SET
			spread = EXCLUDED.spread,
			depth_bid = EXCLUDED.depth_bid,
			depth_ask = EXCLUDED.depth_ask
	`, row.Time, row.Symbol, row.Exchange, row.Spread, row.DepthBid, row.DepthAsk)

	return err
}

// InsertSlippage upserts processed slippage rows.
func InsertSlippage(ctx context.Context, execer Execer, rows []SlippageRow) error {
	for _, row := range rows {
		_, err := execer.Exec(ctx, `
			INSERT INTO slippage (
				time,
				symbol,
				exchange,
				size,
				slippage_bid,
				slippage_ask
			)
			VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6::numeric)
			ON CONFLICT (time, symbol, exchange, size) DO UPDATE
			SET
				slippage_bid = EXCLUDED.slippage_bid,
				slippage_ask = EXCLUDED.slippage_ask
		`, row.Time, row.Symbol, row.Exchange, row.Size, row.SlippageBid, row.SlippageAsk)
		if err != nil {
			return err
		}
	}

	return nil
}
