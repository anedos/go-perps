package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestZapQueryTracerLogsQuery(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.InfoLevel)
	tracer := newZapQueryTracer(zap.New(core))

	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{
		SQL:  "SELECT $1::int",
		Args: []any{1},
	})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{
		CommandTag: pgconn.NewCommandTag("SELECT 1"),
	})

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	if entry.Message != "pg query" {
		t.Fatalf("expected pg query message, got %q", entry.Message)
	}
	if sql := entry.ContextMap()["sql"]; sql != "SELECT $1::int" {
		t.Fatalf("expected SQL field, got %v", sql)
	}
	if _, ok := entry.ContextMap()["args"]; !ok {
		t.Fatal("expected args field")
	}
}
