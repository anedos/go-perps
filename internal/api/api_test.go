package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anedos/go-perps/internal/db"
	"github.com/anedos/go-perps/internal/model"
)

func TestPing(t *testing.T) {
	t.Parallel()

	response := request(t, newTestServer(), "/ping")

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body map[string]string
	decodeJSON(t, response, &body)

	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestInfoExchanges(t *testing.T) {
	t.Parallel()

	response := request(t, newTestServer(), "/info/exchanges")

	var body []string
	decodeJSON(t, response, &body)

	assertStringSlice(t, body, []string{"extended", "hyperliquid", "lighter"})
}

func TestInfoMarkets(t *testing.T) {
	t.Parallel()

	response := request(t, newTestServer(), "/info/markets")

	var body []string
	decodeJSON(t, response, &body)

	assertStringSlice(t, body, []string{"ETH-USD", "BTC-USD"})
}

func TestStats(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	response := request(t, server, "/stats/ETH-USD?period=1h&exchange=hyperliquid")

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body metricsResponse
	decodeJSON(t, response, &body)

	if body.Symbol != "ETH-USD" || body.Exchange != "hyperliquid" || body.Period != "1h" {
		t.Fatalf("unexpected response metadata: %+v", body)
	}
}

func TestDepthChart(t *testing.T) {
	t.Parallel()

	response := request(t, newTestServer(), "/chart/depth/ETH-USD?period=1D")

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestSlippageChart(t *testing.T) {
	t.Parallel()

	response := request(t, newTestServer(), "/chart/slippage/ETH-USD?period=1W")

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestMetricsRoutesRequirePeriod(t *testing.T) {
	t.Parallel()

	response := request(t, newTestServer(), "/stats/ETH-USD")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestMetricsRoutesValidateExchange(t *testing.T) {
	t.Parallel()

	response := request(t, newTestServer(), "/stats/ETH-USD?period=1h&exchange=unknown")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func newTestServer() *Server {
	return New(Config{
		Markets: []model.Market{
			{Symbol: "ETH-USD"},
			{Symbol: "BTC-USD"},
		},
		Now: func() time.Time {
			return time.Unix(1_704_241_860, 0).UTC()
		},
	}, nil, fakeStore{})
}

func request(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	return response
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d values, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected value %d to be %q, got %q", i, want[i], got[i])
		}
	}
}

type fakeStore struct{}

func (s fakeStore) Stats(context.Context, db.MetricsParams) ([]db.StatsRow, error) {
	return []db.StatsRow{
		{
			Exchange:    "hyperliquid",
			SampleCount: 1,
			AvgSpread:   "2",
			AvgDepthBid: "3",
			AvgDepthAsk: "4",
		},
	}, nil
}

func (s fakeStore) DepthChart(context.Context, db.MetricsParams) ([]db.DepthPoint, error) {
	return []db.DepthPoint{
		{
			Time:     time.Unix(1_704_241_860, 0).UTC(),
			Exchange: "hyperliquid",
			Spread:   "2",
			DepthBid: "3",
			DepthAsk: "4",
		},
	}, nil
}

func (s fakeStore) SlippageChart(context.Context, db.MetricsParams) ([]db.SlippagePoint, error) {
	return []db.SlippagePoint{
		{
			Time:        time.Unix(1_704_241_860, 0).UTC(),
			Exchange:    "hyperliquid",
			Size:        "100",
			SlippageBid: "0.01",
			SlippageAsk: "0.02",
		},
	}, nil
}
