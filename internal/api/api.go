// Package api contains the read-only HTTP API
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/anedos/go-perps/internal/model"
	"go.uber.org/zap"
)

// Config contains static API metadata used by informational endpoints.
type Config struct {
	// Markets lists normalized markets supported by this process.
	Markets []model.Market
}

// Server owns HTTP routes and shared API dependencies.
type Server struct {
	config Config
	logger *zap.Logger
	mux    *http.ServeMux
}

// New creates an API server with all routes registered.
func New(config Config, logger *zap.Logger) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}

	server := &Server{
		config: config,
		logger: logger,
		mux:    http.NewServeMux(),
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

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	if err := json.NewEncoder(writer).Encode(value); err != nil {
		http.Error(writer, "encode response", http.StatusInternalServerError)
	}
}
