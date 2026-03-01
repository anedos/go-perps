// Package storage owns persistence of processed order book metrics.
package storage

import (
	"time"

	"github.com/anedos/go-perps/internal/model"
)

type writeKey struct {
	symbol   string
	exchange model.Exchange
}

// WriteThrottler limits writes per symbol and exchange.
type WriteThrottler struct {
	interval       time.Duration
	now            func() time.Time
	lastWriteByKey map[writeKey]time.Time
}

// NewWriteThrottler creates a throttler using time.Now.
func NewWriteThrottler(interval time.Duration) *WriteThrottler {
	return newWriteThrottler(interval, time.Now)
}

func newWriteThrottler(interval time.Duration, now func() time.Time) *WriteThrottler {
	return &WriteThrottler{
		interval:       interval,
		now:            now,
		lastWriteByKey: make(map[writeKey]time.Time),
	}
}

// ShouldWrite reports whether orderBook should be written now. It records the
// write time when it returns true.
func (t *WriteThrottler) ShouldWrite(orderBook model.ProcessedOrderBook) bool {
	key := writeKey{
		symbol:   orderBook.Symbol,
		exchange: orderBook.Exchange,
	}
	now := t.now()

	lastWrite, ok := t.lastWriteByKey[key]
	if ok && now.Sub(lastWrite) < t.interval {
		return false
	}

	t.lastWriteByKey[key] = now
	return true
}
