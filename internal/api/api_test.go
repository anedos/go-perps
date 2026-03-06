package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func newTestServer() *Server {
	return New(Config{
		Markets: []model.Market{
			{Symbol: "ETH-USD"},
			{Symbol: "BTC-USD"},
		},
	}, nil)
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
