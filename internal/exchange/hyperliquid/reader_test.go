package hyperliquid

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/anedos/go-perps/internal/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectSubscribesToMarkets(t *testing.T) {
	t.Parallel()

	conn := &fakeConn{}
	reader := newWithDialer(fakeDialer{conn: conn})

	err := reader.Connect(context.Background(), []model.Market{
		{
			Symbol: "ETH-USD",
			ExchangeSymbols: map[model.Exchange]string{
				model.ExchangeHyperliquid: "ETH",
			},
		},
		{Symbol: "BTC"},
	})
	require.NoError(t, err)
	defer reader.Close()

	require.Len(t, conn.writes, 2)

	assertSubscription(t, conn.writes[0], "ETH")
	assertSubscription(t, conn.writes[1], "BTC")
}

func TestParseMessageNormalizesOrderBook(t *testing.T) {
	t.Parallel()

	orderBook, ok, err := ParseMessage([]byte(`{
		"channel": "l2Book",
		"data": {
			"coin": "ETH",
			"time": 1704241860000,
			"levels": [
				[
					{"px": "99", "sz": "1"},
					{"px": "98", "sz": "2"}
				],
				[
					{"px": "101", "sz": "3"}
				]
			]
		}
	}`), map[string]string{"ETH": "ETH-USD"})
	require.NoError(t, err)
	require.True(t, ok)

	assert.Equal(t, model.ExchangeHyperliquid, orderBook.Exchange)
	assert.Equal(t, "ETH-USD", orderBook.Symbol)
	assert.True(t, orderBook.Timestamp.Equal(time.Unix(1_704_241_860, 0).UTC()))
	assertDecimal(t, orderBook.Bids[0].Price, "99")
	assertDecimal(t, orderBook.Bids[0].Quantity, "1")
	assertDecimal(t, orderBook.Asks[0].Price, "101")
	assertDecimal(t, orderBook.Asks[0].Quantity, "3")
}

func TestParseMessageIgnoresOtherChannels(t *testing.T) {
	t.Parallel()

	_, ok, err := ParseMessage([]byte(`{"channel":"subscriptionResponse","data":{}}`), nil)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestParseMessageReturnsExchangeErrors(t *testing.T) {
	t.Parallel()

	_, _, err := ParseMessage([]byte(`{"channel":"error","data":"bad subscription"}`), nil)
	require.Error(t, err)
}

func TestReadLoopEmitsOrderBooks(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := &fakeConn{
		readMessages: [][]byte{
			[]byte(`{
				"channel": "l2Book",
				"data": {
					"coin": "ETH",
					"time": 1704241860000,
					"levels": [[{"px": "99", "sz": "1"}], [{"px": "101", "sz": "1"}]]
				}
			}`),
		},
		readErr: errors.New("done"),
	}
	reader := newWithDialer(fakeDialer{conn: conn})

	err := reader.Connect(ctx, []model.Market{
		{
			Symbol: "ETH-USD",
			ExchangeSymbols: map[model.Exchange]string{
				model.ExchangeHyperliquid: "ETH",
			},
		},
	})
	require.NoError(t, err)
	defer reader.Close()

	select {
	case orderBook := <-reader.Stream():
		assert.Equal(t, "ETH-USD", orderBook.Symbol)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for order book")
	}
}

func assertSubscription(t *testing.T, value any, coin string) {
	t.Helper()

	message, ok := value.(map[string]any)
	require.Truef(t, ok, "expected map subscription, got %T", value)
	assert.Equal(t, "subscribe", message["method"])

	subscription, ok := message["subscription"].(map[string]any)
	require.Truef(t, ok, "expected subscription map, got %T", message["subscription"])
	assert.Equal(t, orderBookChannel, subscription["type"])
	assert.Equal(t, coin, subscription["coin"])
}

func assertDecimal(t *testing.T, got decimal.Decimal, want string) {
	t.Helper()

	expected := decimal.RequireFromString(want)
	assert.Truef(t, got.Equal(expected), "expected %s, got %s", expected, got)
}

type fakeDialer struct {
	conn websocketConn
	err  error
}

func (d fakeDialer) DialContext(context.Context, string, http.Header) (websocketConn, *http.Response, error) {
	if d.err != nil {
		return nil, nil, d.err
	}

	return d.conn, nil, nil
}

type fakeConn struct {
	writes       []any
	readMessages [][]byte
	readErr      error
	closed       bool
}

func (c *fakeConn) ReadMessage() (int, []byte, error) {
	if len(c.readMessages) == 0 {
		if c.readErr != nil {
			return 0, nil, c.readErr
		}

		return 0, nil, errors.New("no read message")
	}

	msg := c.readMessages[0]
	c.readMessages = c.readMessages[1:]

	return 1, msg, nil
}

func (c *fakeConn) WriteJSON(value any) error {
	c.writes = append(c.writes, value)
	return nil
}

func (c *fakeConn) Close() error {
	c.closed = true
	return nil
}
