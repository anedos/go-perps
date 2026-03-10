// Package api contains the read-only HTTP API
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/anedos/go-perps/internal/db"
	"github.com/anedos/go-perps/internal/model"
	"go.uber.org/zap"
)

// Config contains static API metadata used by the read-only endpoints (todo move to env file?)
type Config struct {
	// Markets lists normalized markets supported by this process.
	Markets []model.Market
	// Now returns the current time for period calculations.
	Now func() time.Time
}

// Server owns HTTP routes and shared API dependencies
type Server struct {
	config Config
	logger *zap.Logger
	mux    *http.ServeMux
	store  Store
}

// New creates an API server with all routes registered.
func New(config Config, logger *zap.Logger, store Store) *Server {
	//TODO cleanup this conditions and be stricter on what New expects
	if logger == nil {
		logger = zap.NewNop()
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if store == nil {
		store = emptyStore{}
	}

	server := &Server{
		config: config,
		logger: logger,
		mux:    http.NewServeMux(),
		store:  store,
	}
	server.routes()

	return server
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	start := time.Now()
	s.mux.ServeHTTP(writer, request)
	s.logger.Info("http request",
		zap.String("method", request.Method),
		zap.String("path", request.URL.Path),
		zap.Duration("duration", time.Since(start)),
	)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /ping", s.ping)
	s.mux.HandleFunc("GET /info/exchanges", s.exchanges)
	s.mux.HandleFunc("GET /info/markets", s.markets)
	s.mux.HandleFunc("GET /stats/{symbol}", s.stats)
	s.mux.HandleFunc("GET /chart/depth/{symbol}", s.depthChart)
	s.mux.HandleFunc("GET /chart/slippage/{symbol}", s.slippageChart)
}

func (s *Server) ping(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (s *Server) exchanges(writer http.ResponseWriter, _ *http.Request) {
	exchanges := make([]string, 0, len(model.AllExchanges))
	for _, exchange := range model.AllExchanges {
		exchanges = append(exchanges, exchange.String())
	}

	writeJSON(writer, http.StatusOK, exchanges)
}

func (s *Server) markets(writer http.ResponseWriter, _ *http.Request) {
	markets := make([]string, 0, len(s.config.Markets))
	for _, market := range s.config.Markets {
		markets = append(markets, market.Symbol)
	}

	writeJSON(writer, http.StatusOK, markets)
}

func (s *Server) stats(writer http.ResponseWriter, request *http.Request) {
	params, response, ok := s.metricsParams(writer, request)
	if !ok {
		return
	}

	rows, err := s.store.Stats(request.Context(), params)
	if err != nil {
		s.writeError(writer, "query stats", err)
		return
	}

	response.Data = rows
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) depthChart(writer http.ResponseWriter, request *http.Request) {
	params, response, ok := s.metricsParams(writer, request)
	if !ok {
		return
	}

	rows, err := s.store.DepthChart(request.Context(), params)
	if err != nil {
		s.writeError(writer, "query depth chart", err)
		return
	}

	response.Data = rows
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) slippageChart(writer http.ResponseWriter, request *http.Request) {
	params, response, ok := s.metricsParams(writer, request)
	if !ok {
		return
	}

	rows, err := s.store.SlippageChart(request.Context(), params)
	if err != nil {
		s.writeError(writer, "query slippage chart", err)
		return
	}

	response.Data = rows
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) metricsParams(writer http.ResponseWriter, request *http.Request) (db.MetricsParams, metricsResponse, bool) {
	symbol := request.PathValue("symbol")
	period := request.URL.Query().Get("period")
	exchange := request.URL.Query().Get("exchange")

	since, err := parsePeriod(period, s.config.Now())
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return db.MetricsParams{}, metricsResponse{}, false
	}
	if exchange != "" {
		parsed, err := model.ParseExchange(exchange)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return db.MetricsParams{}, metricsResponse{}, false
		}
		exchange = parsed.String()
	}

	return db.MetricsParams{
			Symbol:   symbol,
			Since:    since,
			Exchange: exchange,
		}, metricsResponse{
			Symbol:   symbol,
			Exchange: exchange,
			Period:   period,
			Since:    since,
		}, true
}

func (s *Server) writeError(writer http.ResponseWriter, message string, err error) {
	if errors.Is(err, context.Canceled) {
		writeJSON(writer, http.StatusRequestTimeout, map[string]string{"error": message})
		return
	}

	s.logger.Error(message, zap.Error(err))
	writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	if err := json.NewEncoder(writer).Encode(value); err != nil {
		http.Error(writer, "encode response", http.StatusInternalServerError)
	}
}
