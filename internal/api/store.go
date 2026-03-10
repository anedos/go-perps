package api

import (
	"context"
	"time"

	"github.com/anedos/go-perps/internal/db"
)

// Store provides historical metrics from the 3 exchanges
type Store interface {
	Stats(ctx context.Context, params db.MetricsParams) ([]db.StatsRow, error)
	DepthChart(ctx context.Context, params db.MetricsParams) ([]db.DepthPoint, error)
	SlippageChart(ctx context.Context, params db.MetricsParams) ([]db.SlippagePoint, error)
}

// DBStore mirrors the Writer struct and is used only for queries
type DBStore struct {
	queryer db.Queryer
}

// NewDBStore creates a Store backed by PostgreSQL queries.
func NewDBStore(queryer db.Queryer) *DBStore {
	return &DBStore{queryer: queryer}
}

// Stats returns aggregate snapshot metrics
func (s *DBStore) Stats(ctx context.Context, params db.MetricsParams) ([]db.StatsRow, error) {
	return db.SelectStats(ctx, s.queryer, params)
}

// DepthChart returns historical depth points
func (s *DBStore) DepthChart(ctx context.Context, params db.MetricsParams) ([]db.DepthPoint, error) {
	return db.SelectDepthChart(ctx, s.queryer, params)
}

// SlippageChart returns historical slippage points
func (s *DBStore) SlippageChart(ctx context.Context, params db.MetricsParams) ([]db.SlippagePoint, error) {
	return db.SelectSlippageChart(ctx, s.queryer, params)
}

type emptyStore struct{}

func (s emptyStore) Stats(context.Context, db.MetricsParams) ([]db.StatsRow, error) {
	return nil, nil
}

func (s emptyStore) DepthChart(context.Context, db.MetricsParams) ([]db.DepthPoint, error) {
	return nil, nil
}

func (s emptyStore) SlippageChart(context.Context, db.MetricsParams) ([]db.SlippagePoint, error) {
	return nil, nil
}

type metricsResponse struct {
	Symbol   string    `json:"symbol"`
	Exchange string    `json:"exchange,omitempty"`
	Period   string    `json:"period"`
	Since    time.Time `json:"since"`
	Data     any       `json:"data"`
}
