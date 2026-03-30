// Package hyperliquid implements the Hyperliquid websocket order book reader
package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/anedos/go-perps/internal/model"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

const (
	orderBookChannel = "l2Book"
	websocketURL     = "wss://api.hyperliquid.xyz/ws"
)

// define a partial websocketConn interface for better testability, we can now have easier isolation in our unit tests
// see reader_test.go
type websocketConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteJSON(v any) error
	Close() error
}

type websocketDialer interface {
	DialContext(ctx context.Context, url string, requestHeader http.Header) (websocketConn, *http.Response, error)
}

type gorillaDialer struct {
	dialer *websocket.Dialer
}

func (d gorillaDialer) DialContext(
	ctx context.Context,
	url string,
	requestHeader http.Header,
) (websocketConn, *http.Response, error) {
	return d.dialer.DialContext(ctx, url, requestHeader)
}

// Reader connects to Hyperliquid's public websocket and emits normalized order
// book snapshots.
type Reader struct {
	dialer websocketDialer

	mu           sync.RWMutex
	conn         websocketConn
	symbolByCoin map[string]string

	outputCh chan model.OrderBook
	errorCh  chan error
}

// New creates a Hyperliquid reader using the default websocket dialer.
func New() *Reader {
	return newWithDialer(gorillaDialer{dialer: websocket.DefaultDialer})
}

func newWithDialer(dialer websocketDialer) *Reader {
	return &Reader{
		dialer:       dialer,
		symbolByCoin: make(map[string]string),
		outputCh:     make(chan model.OrderBook, 100),
		errorCh:      make(chan error, 100),
	}
}

// Connect opens the websocket, subscribes to markets, and starts the read loop.
func (r *Reader) Connect(ctx context.Context, markets []model.Market) error {
	conn, _, err := r.dialer.DialContext(ctx, websocketURL, nil)
	if err != nil {
		return fmt.Errorf("connect hyperliquid websocket: %w", err)
	}

	r.mu.Lock()
	r.conn = conn
	r.symbolByCoin = make(map[string]string, len(markets))
	r.mu.Unlock()

	for _, market := range markets {
		coin := market.SymbolFor(model.ExchangeHyperliquid)
		if err := r.subscribe(coin); err != nil {
			_ = conn.Close()
			return fmt.Errorf("subscribe hyperliquid %s: %w", coin, err)
		}

		r.mu.Lock()
		r.symbolByCoin[coin] = market.Symbol
		r.mu.Unlock()
	}

	go r.readLoop(ctx)
	go func() {
		<-ctx.Done()
		_ = r.Close()
	}()

	return nil
}

// Stream returns normalized order books emitted by the reader.
func (r *Reader) Stream() <-chan model.OrderBook {
	return r.outputCh
}

// Errors returns non-fatal reader errors.
func (r *Reader) Errors() <-chan error {
	return r.errorCh
}

// Close closes the underlying websocket connection.
func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn == nil {
		return nil
	}

	err := r.conn.Close()
	r.conn = nil

	return err
}

func (r *Reader) subscribe(coin string) error {
	r.mu.RLock()
	conn := r.conn
	r.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("hyperliquid websocket is not connected")
	}

	return conn.WriteJSON(map[string]any{
		"method": "subscribe",
		"subscription": map[string]any{
			"type": orderBookChannel,
			"coin": coin,
		},
	})
}

func (r *Reader) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		r.mu.RLock()
		conn := r.conn
		symbolByCoin := copySymbolMap(r.symbolByCoin)
		r.mu.RUnlock()

		if conn == nil {
			return
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			r.sendError(fmt.Errorf("read hyperliquid message: %w", err))
			return
		}

		orderBook, ok, err := ParseMessage(msg, symbolByCoin)
		if err != nil {
			r.sendError(err)
			continue
		}
		if !ok {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case r.outputCh <- orderBook:
		}
	}
}

func (r *Reader) sendError(err error) {
	select {
	case r.errorCh <- err:
	default:
	}
}

func copySymbolMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}

	return output
}

// captures
//
//	interface WsLevel {
//	 px: string; // price
//	 sz: string; // size
//	 n: number; // number of orders
//	}
type priceLevel struct {
	Price    string `json:"px"`
	Quantity string `json:"sz"`
}

type orderBookPayload struct {
	Coin      string          `json:"coin"`
	Timestamp int64           `json:"time"`
	Levels    [2][]priceLevel `json:"levels"`
}

type message struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

// ParseMessage converts a Hyperliquid websocket message into a normalized order
// book. It returns ok=false for messages outside the l2Book channel.
// Used https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/websocket/subscriptions#data-type-definitions
// as reference.
func ParseMessage(payload []byte, symbolByCoin map[string]string) (model.OrderBook, bool, error) {
	var msg message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return model.OrderBook{}, false, fmt.Errorf("parse hyperliquid message: %w", err)
	}

	if msg.Channel == "error" {
		return model.OrderBook{}, false, fmt.Errorf("hyperliquid error: %s", string(msg.Data))
	}
	if msg.Channel != orderBookChannel {
		return model.OrderBook{}, false, nil
	}

	var book orderBookPayload
	if err := json.Unmarshal(msg.Data, &book); err != nil {
		return model.OrderBook{}, false, fmt.Errorf("parse hyperliquid order book: %w", err)
	}

	symbol := symbolByCoin[book.Coin]
	if symbol == "" {
		symbol = book.Coin
	}

	return model.OrderBook{
		Exchange:  model.ExchangeHyperliquid,
		Symbol:    symbol,
		Timestamp: time.UnixMilli(book.Timestamp),
		Bids:      parsePriceLevels(book.Levels[0]),
		Asks:      parsePriceLevels(book.Levels[1]),
	}, true, nil
}

func parsePriceLevels(levels []priceLevel) []model.PriceLevel {
	result := make([]model.PriceLevel, 0, len(levels))
	for _, level := range levels {
		result = append(result, model.PriceLevel{
			Price:    decimal.RequireFromString(level.Price),
			Quantity: decimal.RequireFromString(level.Quantity),
		})
	}

	return result
}
