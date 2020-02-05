# Observer
Service that replicates smart-contract records data to DB,
collects various statistics and serves API.

# Depends on
Insolar heavy material executor node

PostgreSQL DB 11.4

# Installation and deployment

## Generate default configs

Run `make config`. Command generates two config files:
`observer.yaml` and `observerapi.yaml`. Places them into
`./.artifacts` directory.

## Build binaries

Run `make build`.

**WARNING:** Go modules are used, you may need `GO111MODULE=on` set.

## Above in one go

`make all`

## Replicator service

To run replicator you should provide config file `observer.yaml`
in the current working directory or in `.artifacts` directory.

Run `./bin/observer`

### Configuration

All options in `observer.yaml` config can be overridden with environment
variables using `OBSERVER` and `_` as delimiter, for example:
`OBSERVER_DB_URL=...`, `OBSERVER_REPLICATOR_LISTEN=...`

**WARNING:** overriding via ENV wouldn't work without config file with default.

### Config options

DB connection:
`OBSERVER_DB_URL=postgres://user:password@host/db_name?sslmode=disable`

Heavy node replication API:
`OBSERVER_REPLICATOR_ADDR=127.0.0.1:5678`

Log params:
```
OBSERVER_LOG_LEVEL=info
OBSERVER_LOG_FORMAT=text
OBSERVER_LOG_OUTPUTTYPE=stderr
OBSERVER_LOG_OUTPUTPARAMS=<some_text>
OBSERVER_LOG_BUFFER=0
```

### Metrics and health check

`OBSERVER_REPLICATOR_LISTEN=:8888`

Prometheus: `http://localhost:8888/metrics`

Health check: `http://localhost:8888/healthcheck`

Replicator dashboard: `./dashboard.json`

API dashboard: `./dashboard_api.json`

## API service

To run API you should provide config file `observerapi.yaml`
in the current working directory or in `.artifacts` directory.

Run `./bin/api`

### Configuration

All options in `observerapi.yaml` config can be overridden with environment
variables using `OBSERVERAPI` prefix and `_` as delimiter, for example:
`OBSERVERAPI_DB_URL=...`, `OBSERVERAPI_LISTEN=...`

**WARNING:** overriding via ENV wouldn't work without config file with default.

### Config options

API's endpoint:
`OBSERVERAPI_LISTEN=127.0.0.1:5678`
or
`OBSERVERAPI_LISTEN=:5678`

DB connection:
`OBSERVERAPI_DB_URL=postgres://user:password@host/db_name?sslmode=disable`

Max number of connections to DB:
`OBSERVERAPI_DB_POOLSIZE=20`

Log params:
```
OBSERVERAPI_LOG_LEVEL=info
OBSERVERAPI_LOG_FORMAT=text
OBSERVERAPI_LOG_OUTPUTTYPE=stderr
OBSERVERAPI_LOG_BUFFER=0
```

Coin statistic api:
You need to chose an origin of coin statistic.

```
PriceOrigin = const|binance|coin_market_cap
const - By default it's taken from values from config (somekind of hardcode)
binance - get data gathered by binance collector
coin_market_cap - get data gathered by cmc collector

## Stats collector
Command calculates, gathers and saves statistics, add to cron for 1/min execution.
Uses replicator's config (see above).

## Binance collector
Binance gathers info about the exchange rate of the INS/USD pair (could be calculated like a XNS/USD)

Collector should be run every **hour** (it mustn't be run more than 1 time per 5 minute), to run it call these commands:
```
    make build
    ./bin/binance-collector -symbol=XNS
```

 `-symbol` is used for providing symbol to the stats collector. 

Config is being taken from `.artifacts`. The observer config is needed, because `collector` uses `Log` and `Db` sections.

## Coin market cap collector
Binance gathers info about the exchange rate of the INS/USD pair (could be calculated like a XNS/USD)

Collector should be run every **hour** (it mustn't be run more than 1 time per 5 minute), to run it call these commands:
```
    make build
    ./bin/coin-market-cap-collector -cmc-token={CMC_API_TOKEN} -symbol={XNS|INS}
```

 `-cmc-token` is used for providing `Coin market cap` api token. That will be using for an every request.
  `-symbol` is used for providing symbol to the stats collector. 

Config is being taken from `.artifacts`. The observer config is needed, because `collector` uses `Log` and `Db` sections.

## Database initialization and upgrade
Run migrations (with go binary inside repository):
1. Run `make migrate`.

Run migrations (without go binary):
1. Run `make build` inside repository (requires go binary). 
2. Copy `bin/migrate` binary, `scripts/migrations` dir 
and config files (see section "Generate default configs") to the target environment.
3. Run migrate binary and provide migrations dir with `-dir` param. The binary will access the DB specified in the 
`observer.yaml` config.

If migrations are being run for the first time (on empty DB), provide `-init` param for migration command.

# Publishing notifications

To publish a message that all users will see in the UI during some period of
time insert it directly into `notifications` table in Pg:

```
INSERT INTO notifications(message, start, stop)
    VALUES('some message', NOW(), NOW() + interval '3 hours')
```

**WARNING:** only one notification is active, the one with older `start` value
wins.

# Development

## Installing required command line tools

Run `make install_deps`

**WARNING:** this step installs exact versions of tools to avoid constant
changes back and forth from different developers.

## Regenerate server API implementation from OpenAPI specification

Use oapi-codegen. Get it via:
```
go get github.com/deepmap/oapi-codegen/cmd/oapi-codegen
```

Generate combined `api-exported.yaml` file:
```
cd ../insolar-observer-api/api-exported.yaml
npm install
npm run export
```

Generate types and API from observer API:
```
oapi-codegen -package api -generate types,server ../insolar-observer-api/api-exported.yaml > internal/app/api/generated.go
```
