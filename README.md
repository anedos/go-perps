# Go Perps

Go Perps aggregates perpetual futures order book data across exchanges and exposes
depth and slippage analytics through a REST API.

The project is intentionally small and explicit. It is meant to be a practical
Go codebase for learning market data ingestion, exchange websocket protocols,
TimescaleDB storage, concurrency, testing, and REST API design.

## Architecture

Go Perps is split into two long-running commands:

- `reader`: connects to exchange websocket feeds, normalizes order books,
  computes derived metrics, and writes throttled snapshots to TimescaleDB.
- `api`: serves historical depth, slippage, stats, and metadata from TimescaleDB.

Both commands share common domain types, configuration, database access, and
calculation logic.

```text
Exchange WebSockets
        |
        v
 exchange adapters
        |
        v
 normalized order books
        |
        v
 processor: spread, depth, slippage
        |
        v
 throttled writer
        |
        v
 TimescaleDB <---- REST API
```

## Project Layout

The intended package layout is:

```text
cmd/
  api/                    # REST API process
  reader/                 # websocket ingestion process
internal/
  api/                    # route handlers, request parsing, response DTOs
  config/                 # environment and typed runtime configuration
  db/                     # SQL queries, connection pool, migrations
  exchange/               # exchange reader interface and implementations
    extended/
    hyperliquid/
    lighter/
  ingest/                 # reader orchestration, fan-in, reconnect handling
  model/                  # Exchange, Market, OrderBook, ProcessedOrderBook
  processor/              # spread, depth, and slippage calculations
  storage/                # snapshot/slippage writer and write throttling
migrations/               # TimescaleDB schema migrations
```

Protocol-specific websocket code stays inside exchange packages. The rest of
the system works with normalized order book types.

## Core Components

### Domain Model

The core model is intentionally compact:

- `Exchange`: supported exchange identifiers.
- `Market`: configured market symbol and exchange-specific symbol mapping.
- `PriceLevel`: price and quantity using decimal arithmetic.
- `OrderBook`: normalized snapshot with exchange, symbol, timestamp, bids, and asks.
- `ProcessedOrderBook`: spread, depth, and per-size slippage derived from a snapshot.

Prices, quantities, and notionals should use decimal arithmetic. Floating point
types are acceptable for rough telemetry, but not for order book calculations.

### Exchange Readers

Each exchange reader owns protocol-specific behavior:

- websocket URL and subscription messages
- symbol formatting
- snapshot and delta parsing
- local order book state
- reconnect and resubscribe behavior

Readers expose a shared interface that emits normalized `OrderBook` values.
Downstream code should not know exchange-specific message formats.

Initial target exchanges:

- Extended
- Hyperliquid
- Lighter

Hyperliquid's public websocket only returns the best 20 bids and asks. Full
depth requires a non-validating node.

### Ingestion

The reader command coordinates all exchange readers:

- starts one goroutine per reader or per market, depending on exchange protocol
- fans all reader outputs into a bounded channel
- forwards normalized order books into the processor
- propagates shutdown through `context.Context`
- reports reader and processing errors without crashing the whole process unless
  the error is unrecoverable

### Processor

The processor is stateless per snapshot. Each incoming `OrderBook` produces one
`ProcessedOrderBook` with:

- spread: best ask minus best bid
- depth: bid and ask quantity inside the configured mid-price band
- slippage: estimated price impact for configured notional sizes

This package should be heavily unit tested because it is exchange-independent
and contains most of the business logic.

### Storage

TimescaleDB is the persistence layer:

- snapshots hypertable keyed by time, symbol, and exchange
- slippage hypertable keyed by time, symbol, exchange, and size
- compression after 7 days
- retention after 90 days

The writer throttles writes per `(symbol, exchange)` so high-frequency websocket
updates do not overload the database. The initial target is one write every
5-15 seconds per key.

SQL should stay explicit. The query layer should use generated or typed helpers
where they reduce risk without hiding the SQL.

### REST API

The API command reads from TimescaleDB and exposes:

- `GET /ping`
- `GET /info/exchanges`
- `GET /info/markets`
- `GET /stats/{symbol}`
- `GET /chart/slippage/{symbol}`
- `GET /chart/depth/{symbol}`

Query parameters:

- `period`: required for chart and stats endpoints, for example `5m`, `1h`,
  `1D`, `1W`, `1M`, `3M`
- `exchange`: optional, defaults to all exchanges

## Configuration

Configuration is loaded from environment variables and optional local `.env`
files during development.

Initial configuration groups:

- database connection settings
- API bind address
- enabled exchanges
- enabled markets
- slippage notional levels
- writer throttle interval
- websocket reconnect policy

Configuration should be parsed once at startup into typed structs and passed to
the components that need it.

## Development

Expected local services:

- Go 1.24 or newer
- PostgreSQL with TimescaleDB

Expected commands:

```sh
go test ./...
GO_PERPS_TEST_DB=1 GO_PERPS_LOG_QUERIES=1  go test ./internal/storage -run '^TestWriterWritesToTimescaleDB$' -count=1 -v
go vet ./...
go run ./cmd/api
go run ./cmd/reader
```
