CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS snapshots (
    time TIMESTAMPTZ NOT NULL,
    symbol TEXT NOT NULL,
    exchange TEXT NOT NULL,
    spread NUMERIC NOT NULL,
    depth_bid NUMERIC NOT NULL,
    depth_ask NUMERIC NOT NULL,
    PRIMARY KEY (time, symbol, exchange)
);

SELECT create_hypertable('snapshots', 'time', if_not_exists => TRUE);

ALTER TABLE snapshots SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'symbol,exchange'
);

SELECT add_compression_policy('snapshots', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('snapshots', INTERVAL '90 days', if_not_exists => TRUE);

CREATE TABLE IF NOT EXISTS slippage (
    time TIMESTAMPTZ NOT NULL,
    symbol TEXT NOT NULL,
    exchange TEXT NOT NULL,
    size NUMERIC NOT NULL,
    slippage_bid NUMERIC NOT NULL,
    slippage_ask NUMERIC NOT NULL,
    PRIMARY KEY (time, symbol, exchange, size)
);

SELECT create_hypertable('slippage', 'time', if_not_exists => TRUE);

ALTER TABLE slippage SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'symbol,exchange,size'
);

SELECT add_compression_policy('slippage', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('slippage', INTERVAL '90 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS snapshots_symbol_exchange_time_idx
    ON snapshots (symbol, exchange, time DESC);

CREATE INDEX IF NOT EXISTS slippage_symbol_exchange_size_time_idx
    ON slippage (symbol, exchange, size, time DESC);
