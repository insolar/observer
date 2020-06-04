[<img src="https://github.com/insolar/doc-pics/raw/master/st/github-readme-banner.png">](http://insolar.io/?utm_source=Github)

# Insolar Observer
Insolar Obserber is a service that replicates record data produced by Insolar smart contracts, organizes it into a SQL database and aggregates various statistical data.

The Observer allows trusted agents such as crypto exchanges read data from Insolar Platform via the public Observer API. Trusted agents can integrate the Observer into their business applications or use the Observer API to get the data at their discretion.

The Observer obtains data from a trusted Heavy Material Executor run within Insolar Platform. This way Observer users are insured against inacurate or corrupted data. 

Observer users are responsible for the data they store on their Observer instance and can regulate access via an access  control system of their choice.


# Build

## Prerequisites

* Address of a trusted Heavy Material Executor run on Insolar Platform or [Insolar MainNet](https://github.com/insolar/mainnet) deployed locally (for testing)

* [PostgreSQL 11.4](https://www.postgresql.org/download/) (for example, via [PostresApp](https://postgresapp.com/) on macOS)

* [Go Tools 1.14](https://golang.org/doc/install)

## Clone and change to Insolar Observer

Clone the Observer and change to its directory: `git clone git@github.com:insolar/observer.git && cd observer`.

## Build binaries

Build binaries automatically using the instructions from the Makefile: `make all-public`.

This command:
* Generates three configuration files (`migrate.yaml`, `observer.yaml`, `observerapi.yaml`) and places them into the hidden `./.artifacts` directory.
* Generates thee binaries (`migrate`, `observer`, `api`) and places them into `./cmd/migrate/*.go`, `cmd/observer/*.go`, `./cmd/api/*.go` respectively.

**WARNING:** The Observer uses Go modules. You may need to set the [Go modules environment variable](https://golang.org/cmd/go/#hdr-Module_support) on: `GO111MODULE=on`.

### Configure

All parameters in `observer.yaml` can be overridden via environment variables that start with `OBSERVER` and use `_` as a separator. For example: `OBSERVER_DB_URL=...`, `OBSERVER_REPLICATOR_LISTEN=...`

**WARNING:** overriding via ENV variables doesn't work without a configuration file in place.

### Configuration parameters

Database connection:
`OBSERVER_DB_URL=postgres://user:password@host/db_name?sslmode=disable`

Heavy Material Node replication API:
`OBSERVER_REPLICATOR_ADDR=127.0.0.1:5678`

Log parameters:
```
OBSERVER_LOG_LEVEL=info
OBSERVER_LOG_FORMAT=text
OBSERVER_LOG_OUTPUTTYPE=stderr
OBSERVER_LOG_OUTPUTPARAMS=<some_text>
OBSERVER_LOG_BUFFER=0
```

### Deploy

TBD

## Metrics and health check

## Prerequisites

Before launching the monitoring script, make sure you have installed and set [Docker compose(https://docs.docker.com/compose/install/ "Install Compose ").

### Deploy the built-in monitoring system

Deploy the built-in system: `./scripts/monitor/monitor.sh`

This script starts Grafana and Prometheus configured by the Observer at:

* Grafana: `http://localhost:3000` with default login and password: `login=admin` and `password=pass`
 
* Prometheus: `http://localhost:9090/graph`

* Observer мetrics: `http://localhost:8888` by default
 
* Observer health check service: `http://localhost:8888/healthcheck`

### Deploy a customized monitoring system

You can install, customize and deploy the monitoring system yourself. 

To do this:
1. Deploy  [Grafana](https://grafana.com/docs/grafana/latest/installation/ "Install Grafana ") and  [Prometheus](https://prometheus.io/docs/prometheus/latest/installation/ "Install Prometheus ").

   You can get the config for Prometheus [here](https://github.com/insolar/observer/blob/master/scripts/monitor/prometheus/prometheus.yaml).

2. Import [this Grafana dashboard](https://github.com/insolar/observer/blob/master/scripts/monitor/grafana/dashboards/observer.json) into Grafana. 
 
   If you need to, [read how to import a dashboard]( https://grafana.com/docs/grafana/latest/reference/export_import/).


## API service

To run API you should provide config file `observerapi.yaml`
in the current working directory or in `.artifacts` directory.

Run `./bin/api -- config .artifacts/observerapi.yaml`

### Configure the API

All parameters in `observer.yaml` can be overridden via environment variables that start with `OBSERVER` and use `_` as a separator. For example: `OBSERVER_DB_URL=...`, `OBSERVER_REPLICATOR_LISTEN=...`

**WARNING:** overriding via ENV variables doesn't work without a configuration file in place.

### Configuration parameters

API endpoint:
`OBSERVERAPI_LISTEN=127.0.0.1:5678`
or
`OBSERVERAPI_LISTEN=:5678`

Database connection:
`OBSERVERAPI_DB_URL=postgres://user:password@host/db_name?sslmode=disable`

Max number of connections to the database:
`OBSERVERAPI_DB_POOLSIZE=20`

Log parameters:
```
OBSERVERAPI_LOG_LEVEL=info
OBSERVERAPI_LOG_FORMAT=text
OBSERVERAPI_LOG_OUTPUTTYPE=stderr
OBSERVERAPI_LOG_BUFFER=0
```

Coin statistic API:
You need to choose an origin of coin statistic.

```
PriceOrigin = const|binance|coin_market_cap
const - By default it's taken from values from config (somekind of hardcode)
binance - get data gathered by binance collector
coin_market_cap - get data gathered by cmc collector
```

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


## Contribute!

Feel free to submit issues, fork the repository and send pull requests! 

To make the process smooth for both reviewers and contributors, familiarize yourself with the following guidelines:

1. [Open source contributor guide](https://github.com/freeCodeCamp/how-to-contribute-to-open-source).
2. [Style guide: Effective Go](https://golang.org/doc/effective_go.html).
3. [List of shorthands for Go code review comments](https://github.com/golang/go/wiki/CodeReviewComments).

When submitting an issue, **include a complete test function** that reproduces it.

Thank you for your intention to contribute to the Insolar Mainnet project. As a company developing open-source code, we highly appreciate external contributions to our project.

Here is a helping hand if you decide to contribute.

# Installing required command line tools

Install required command line tools: `make install_deps`

**WARNING:** this option installs the unified versions of tools to avoid constant preference changes from different developers.

## Regenerate server API implementation from the OpenAPI specification

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

## Contacts

If you have any additional questions, join our [developers chat on Telegram](https://t.me/InsolarTech).

Our social media:

[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-facebook.png" width="36" height="36">](https://facebook.com/insolario)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-twitter.png" width="36" height="36">](https://twitter.com/insolario)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-medium.png" width="36" height="36">](https://medium.com/insolar)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-youtube.png" width="36" height="36">](https://youtube.com/insolar)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-reddit.png" width="36" height="36">](https://www.reddit.com/r/insolar/)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-linkedin.png" width="36" height="36">](https://www.linkedin.com/company/insolario/)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-instagram.png" width="36" height="36">](https://instagram.com/insolario)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-telegram.png" width="36" height="36">](https://t.me/InsolarAnnouncements) 

## License

This project is licensed under the terms of the [Insolar License 1.0](LICENSE.md).
