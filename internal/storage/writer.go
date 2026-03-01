package storage

import (
	"context"

	"github.com/anedos/go-perps/internal/db"
	"github.com/anedos/go-perps/internal/model"
)

// Writer persists processed order books to storage with write throttling.
type Writer struct {
	execer    db.Execer
	throttler *WriteThrottler
}

// NewWriter creates a storage writer.
func NewWriter(execer db.Execer, throttler *WriteThrottler) *Writer {
	return &Writer{
		execer:    execer,
		throttler: throttler,
	}
}

// Write persists orderBook when the writer is not throttled. It returns true
// when a write was performed.
func (w *Writer) Write(ctx context.Context, orderBook model.ProcessedOrderBook) (bool, error) {
	if !w.throttler.ShouldWrite(orderBook) {
		return false, nil
	}

	if err := db.InsertSnapshot(ctx, w.execer, db.SnapshotRowFrom(orderBook)); err != nil {
		return false, err
	}

	if err := db.InsertSlippage(ctx, w.execer, db.SlippageRowsFrom(orderBook)); err != nil {
		return false, err
	}

	return true, nil
}
