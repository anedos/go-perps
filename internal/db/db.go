// Package db owns PostgreSQL setup, migrations and SQL commands
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgx-contrib/pgxtrace"
	"go.uber.org/zap"
)

// Config describes PostgreSQL connection settings, TODO, load from env
type Config struct {
	// the PostgreSQL connection string
	URL string
	// MaxConns caps the number of connections in the pool
	MaxConns int32
	// QueryLogger enables PGX query
	QueryLogger *zap.Logger
}

// Connect creates and verifies a PG connection pool and sets pgx tracing
func Connect(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(config.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	if config.MaxConns > 0 {
		poolConfig.MaxConns = config.MaxConns
	}

	if config.QueryLogger != nil {
		poolConfig.ConnConfig.Tracer = pgxtrace.CompositeQueryTracer{
			newZapQueryTracer(config.QueryLogger),
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
