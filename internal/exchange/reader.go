// Package exchange defines common interfaces for exchange readers
package exchange

import (
	"context"

	"github.com/anedos/go-perps/internal/model"
)

// Reader streams canonicalises/normalises order books per exchange
type Reader interface {
	Connect(ctx context.Context, markets []model.Market) error
	Stream() <-chan model.OrderBook
	Errors() <-chan error
	Close() error
	// todo once we have additional Reader implementations, we should add a Subscribe signature
}
