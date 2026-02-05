package logging

import (
	"context"

	"go.uber.org/zap"
)

// keep contextKey private and avoid collisions if another package sets logger in context
type contextKey struct{}

// Config controls logger construction, todo: move to env files
type Config struct {
	// Development enables human-readable development logging.
	Development bool
	// Service is added to every log entry when set.
	Service string
}

// New creates a Zap logger from config.
func New(config Config) (*zap.Logger, error) {
	var (
		logger *zap.Logger
		err    error
	)

	if config.Development {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		return nil, err
	}

	if config.Service == "" {
		return logger, nil
	}

	return logger.With(zap.String("service", config.Service)), nil
}

// MustNew creates a Zap logger and panics if construction fails.
func MustNew(config Config) *zap.Logger {
	logger, err := New(config)
	if err != nil {
		panic(err)
	}

	return logger
}

// WithLogger stores logger in ctx.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext returns the logger stored in ctx or a no-op logger when none is present.
func FromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(contextKey{}).(*zap.Logger)
	if ok && logger != nil {
		return logger
	}

	return zap.NewNop()
}
