# Go Perps

Go Perps aggregates perpetual futures order book data across exchanges and exposes
depth and slippage analytics through a REST API.

I've started this project as a way of learning Perps and improving my Go hands-on experience with logging,
databases and first-time usage of timescaledb.

## Architecture

I have split the project into 2 separate commands:

- `reader`: connects to exchange websockets, normalizes and sorts order books,
  computes metrics, and writes (throttled) snapshots to TimescaleDB.
- `api`: a read-only wrapper api that serves historical depth, slippage, stats, and metadata from TimescaleDB.

Both commands share common types, configuration and database access

## Layout

The intended package layout is:

```text
cmd/
  api/                    # REST API process
  reader/                 # websocket ingestion process
internal/
  api/                    # route handlers, request parsing
  config/                 # env configuration
  db/                     # SQL queries for read, writes, updates, connection pool, migrations
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

Exchange-specific websocket code stays inside exchange packages. The rest of
the system works with normalized order book types.

## Core Components

### Domain Model

The model is pretty small:

- `Exchange`: exchange identifiers.
- `Market`: manages the exchange-specific symbol
- `PriceLevel`: simple price and quantity struct using decimal arithmetic
- `OrderBook`: normalized snapshot with exchange, symbol, timestamp, bids, and asks
- `ProcessedOrderBook`: spread, depth, and per-size slippage derived from a snapshot

### Exchange Readers

Each exchange reader owns org-specific behavior:

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

Env variables:
- `GO_PERPS_DATABASE_URL`, the pg url to be used during runtime
- `GO_PERPS_TEST_DB`, runs integration tests against a local PG db
- `GO_PERPS_LOG_QUERIES`, log PG queries in tests or runtime mode
- `GO_PERPS_API_ADDR`, port number the API runs on
- `GO_PERPS_MARKETS`, a comma-separated list of market symbols, e.g., "ETH-USD,BTC-USD"
- `GO_PERPS_ENV`, "production" or "dev", influences how logs are marshalled and rendered 

- Expected commands:

```sh
go test ./...
GO_PERPS_TEST_DB=1 GO_PERPS_LOG_QUERIES=1  go test ./internal/storage -run '^TestWriterWritesToTimescaleDB$' -count=1 -v
go vet ./...
GO_PERPS_DATABASE_URL=postgresql://<username>:<password>@<hostname>:<port> go run ./cmd/api
O_PERPS_DATABASE_URL=postgresql://<username>:<password>@<hostname>:<port> go run ./cmd/reader
```
