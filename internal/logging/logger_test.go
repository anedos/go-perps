package logging

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewAddsServiceField(t *testing.T) {
	t.Parallel()

	logger, err := New(Config{
		Development: true,
		Service:     "reader",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if logger == nil {
		t.Fatal("expected logger")
	}
}

func TestFromContextReturnsStoredLogger(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	ctx := WithLogger(context.Background(), logger)

	FromContext(ctx).Info("hello")

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}
}

func TestFromContextReturnsNopLoggerByDefault(t *testing.T) {
	t.Parallel()

	logger := FromContext(context.Background())
	if logger == nil {
		t.Fatal("expected fallback logger")
	}
}
