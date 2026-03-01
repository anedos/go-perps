package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type queryTraceContextKey struct{}

type queryTraceContext struct {
	sql   string
	args  []any
	start time.Time
}

type zapQueryTracer struct {
	logger *zap.Logger
}

func newZapQueryTracer(logger *zap.Logger) *zapQueryTracer {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &zapQueryTracer{
		logger: logger,
	}
}

func (t *zapQueryTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	return context.WithValue(ctx, queryTraceContextKey{}, queryTraceContext{
		sql:   data.SQL,
		args:  data.Args,
		start: time.Now(),
	})
}

func (t *zapQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	trace, _ := ctx.Value(queryTraceContextKey{}).(queryTraceContext)

	fields := []zap.Field{
		zap.String("sql", trace.sql),
		zap.Duration("duration", time.Since(trace.start)),
		zap.String("command_tag", data.CommandTag.String()),
	}
	// let's log the args also
	fields = append(fields, zap.Any("args", trace.args))

	if data.Err != nil {
		fields = append(fields, zap.Error(data.Err))
		t.logger.Warn("pg query failed", fields...)
		return
	}

	t.logger.Info("pg query", fields...)
}
