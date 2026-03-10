package db

import (
	"context"
	"time"

	"github.com/anedos/go-perps/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Execer is the subset of pgxpool.Pool used by write queries, might be a premature abstraction but helps with testing
// right now
type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Queryer is the subset of pgxpool.Pool used by read queries, same as above
type Queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// SnapshotRow is one row for the snapshots HT table
type SnapshotRow struct {
	Time     time.Time
	Symbol   string
	Exchange string
	Spread   string
	DepthBid string
	DepthAsk string
}

// SnapshotRowFrom converts a processed order book into a database row
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

// SlippageRow is one row for the slippage HT table
type SlippageRow struct {
	Time        time.Time
	Symbol      string
	Exchange    string
	Size        string
	SlippageBid string
	SlippageAsk string
}

// SlippageRowsFrom converts processed slippage levels into database rows
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

// InsertSnapshot upserts one processed order book snapshot
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

// InsertSlippage upserts processed slippage rows
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

// MetricsParams filters historical metric queries
type MetricsParams struct {
	Symbol   string
	Since    time.Time
	Exchange string
}

// StatsRow is an aggregate metric row grouped by exchange
type StatsRow struct {
	Exchange    string `json:"exchange"`
	SampleCount int64  `json:"sample_count"`
	AvgSpread   string `json:"avg_spread"`
	AvgDepthBid string `json:"avg_depth_bid"`
	AvgDepthAsk string `json:"avg_depth_ask"`
}

// DepthPoint is one historical depth chart point
type DepthPoint struct {
	Time     time.Time `json:"time"`
	Exchange string    `json:"exchange"`
	Spread   string    `json:"spread"`
	DepthBid string    `json:"depth_bid"`
	DepthAsk string    `json:"depth_ask"`
}

// SlippagePoint is one historical slippage chart point
type SlippagePoint struct {
	Time        time.Time `json:"time"`
	Exchange    string    `json:"exchange"`
	Size        string    `json:"size"`
	SlippageBid string    `json:"slippage_bid"`
	SlippageAsk string    `json:"slippage_ask"`
}

// SelectStats returns aggregate snapshot metrics for a symbol and period
func SelectStats(ctx context.Context, queryer Queryer, params MetricsParams) ([]StatsRow, error) {
	rows, err := queryer.Query(ctx, `
		SELECT
			exchange,
			count(*)::bigint AS sample_count,
			coalesce(avg(spread), 0)::text AS avg_spread,
			coalesce(avg(depth_bid), 0)::text AS avg_depth_bid,
			coalesce(avg(depth_ask), 0)::text AS avg_depth_ask
		FROM snapshots
		WHERE symbol = $1
			AND time >= $2
			AND ($3 = '' OR exchange = $3)
		GROUP BY exchange
		ORDER BY exchange
	`, params.Symbol, params.Since, params.Exchange)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]StatsRow, 0)
	for rows.Next() {
		var row StatsRow
		if err := rows.Scan(&row.Exchange, &row.SampleCount, &row.AvgSpread, &row.AvgDepthBid, &row.AvgDepthAsk); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

// SelectDepthChart returns historical depth points for a symbol and period.
func SelectDepthChart(ctx context.Context, queryer Queryer, params MetricsParams) ([]DepthPoint, error) {
	rows, err := queryer.Query(ctx, `
		SELECT
			time,
			exchange,
			spread::text,
			depth_bid::text,
			depth_ask::text
		FROM snapshots
		WHERE symbol = $1
			AND time >= $2
			AND ($3 = '' OR exchange = $3)
		ORDER BY time ASC, exchange ASC
	`, params.Symbol, params.Since, params.Exchange)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]DepthPoint, 0)
	for rows.Next() {
		var point DepthPoint
		if err := rows.Scan(&point.Time, &point.Exchange, &point.Spread, &point.DepthBid, &point.DepthAsk); err != nil {
			return nil, err
		}
		result = append(result, point)
	}

	return result, rows.Err()
}

// SelectSlippageChart returns historical slippage points for a symbol and period.
func SelectSlippageChart(ctx context.Context, queryer Queryer, params MetricsParams) ([]SlippagePoint, error) {
	rows, err := queryer.Query(ctx, `
		SELECT
			time,
			exchange,
			size::text,
			slippage_bid::text,
			slippage_ask::text
		FROM slippage
		WHERE symbol = $1
			AND time >= $2
			AND ($3 = '' OR exchange = $3)
		ORDER BY time ASC, exchange ASC, size ASC
	`, params.Symbol, params.Since, params.Exchange)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SlippagePoint, 0)
	for rows.Next() {
		var point SlippagePoint
		if err := rows.Scan(&point.Time, &point.Exchange, &point.Size, &point.SlippageBid, &point.SlippageAsk); err != nil {
			return nil, err
		}
		result = append(result, point)
	}

	return result, rows.Err()
}
