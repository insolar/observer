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

Log level:
`OBSERVER_LOGLEVEL=info`

### Metrics and health check

`OBSERVER_REPLICATOR_LISTEN=:8888`

Prometheus: `http://localhost:8888/metrics`

Health check: `http://localhost:8888/healthcheck`

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

Log level:
`OBSERVERAPI_LOGLEVEL=info`

## Statistics collector

Command calculates, gathers and saves statistics, add to cron for 1/min execution.
Uses replicator's config (see above).
```
./bin/stats-collector
```

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

## Stats collector

`./bin/stats-collector` takes optional param -time, allows to calculate supply stats like it's a
specified date/time. Example `-time="2006-01-02 15:04:05"`
