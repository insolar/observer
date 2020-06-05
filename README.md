[<img src="https://github.com/insolar/doc-pics/raw/master/st/github-readme-banner.png">](http://insolar.io/?utm_source=Github)

# Insolar Observer
Insolar Obserber is a service that collects record data produced by Insolar smart contracts, organizes it into a SQL database and aggregates various statistical data.

Insolar Observer allows trusted agents such as crypto exchanges read data from Insolar Platform via gRPC. Trusted agents can integrate the Observer into their business applications or use the Observer API to get the data at their discretion.

The Observer obtains data from a trusted Heavy Material Node run within Insolar Platform. This way Observer users are insured against inacurate or corrupted data. 

Observer users are responsible for the data they store on their Observer instance and can regulate access via an access  control system of their choice.

# Build

To build the Observer, first:

* Install and set [PostgreSQL 11.4](https://www.postgresql.org/download/) (for example, via [PostresApp](https://postgresapp.com/) on macOS)

* Install and set [Go Tools 1.12](https://golang.org/doc/install)

* Get an authorized access to Insolar Platform.

## Get authorized access to Insolar Platform

Observer users need to obtain an address of a trusted Heavy Material Node run on Insolar Platform or [Insolar MainNet] to collect data. 

How to obtain it:
1. [Contact Insolar Team](https://insolar.io/contact) to register as a trusted agent.
2. After the registration, the Team will send you your login and a unique link to set your password.
4. After setting your password, put your login and password into the `observer.yaml` configuration file (see **Build binaries**).
3. Working with Insolar Platform, you use your credentials in `observer.yaml` to obtain an access token to address the Platform.

## Clone and change to Insolar Observer

Clone the Observer and change to its directory: `git clone git@github.com:insolar/observer.git && cd observer`.

## Build binaries

Build binaries automatically using the instructions from the Makefile: `make all-public`.

This command:
* Generates three configuration files (`migrate.yaml`, `observer.yaml`, `observerapi.yaml`) and places them into the hidden `./.artifacts` directory.
* Generates thee binaries (`migrate`, `observer`, `api`) and places them into `./.bin/migrate/*.go`, `./.bin/observer/*.go`, `./bin/api/*.go` respectively.

**WARNING:** The Observer uses Go modules. You may need to set the [Go modules environment variable](https://golang.org/cmd/go/#hdr-Module_support) on: `GO111MODULE=on`.

## Deploy

To successfully deploy the Observer, you need to:
1. Initialize your SQL database
2. Deploy the Observer
3. Deploy the Observer API
4. Deploy and start moniroting services

Read about every step below in detail.

### Initialize your SQL database

Initialize your SQL database (generated go binaries required): `migrate-init`.

Tip: `migrate-init` is only for the initial migration. Later, you should use `./bin/migrate --dir=scripts/migrations --init --config=.artifacts/migrate.yaml` for updating your SQL database.

### Deploy the Observer

### Configure the Observer

Configure your Observer instance via configuration parameters in `observer.yaml`.

**Tip:** You can override all parameters in `observer.yaml` via environment variables that start with `OBSERVER` and use `_` as a separator. For example: `OBSERVER_DB_URL=...`, `OBSERVER_REPLICATOR_LISTEN=...`

**WARNING:** Overriding via ENV variables works only with the configuration file in place with the default number of parameters.

#### Configuration parameters

Database connection:
`OBSERVER_DB_URL=postgres://user:password@host/db_name?sslmode=disable`

Heavy Material Node replication API:
`OBSERVER_REPLICATOR_ADDR=<ip_address>:5678`

Log parameters:
```
OBSERVER_LOG_LEVEL=info
OBSERVER_LOG_FORMAT=text
OBSERVER_LOG_OUTPUTTYPE=stderr
OBSERVER_LOG_OUTPUTPARAMS=<some_text>
OBSERVER_LOG_BUFFER=0
```
#### Run the Observer

Run the Observer and wait for a while for it to sync with the trusted HMN: `./bin/observer --config .artifacts/observer.yaml`

Tip: To run the Observer, you need to use the `observer.yaml` configuration file. The file should be in the `.artifacts` directory.

### Deploy the Observer API

### Configure the Observer API 
All options in observerapi.yaml config can be overridden with environment variables using OBSERVERAPI prefix and _ as delimiter, for example: OBSERVERAPI_DB_URL=..., OBSERVERAPI_LISTEN=...

**WARNING**: overriding via ENV variables works only with the configuration file in place with the default number of parameters.

#### Configuration parameters

API endpoint:
`OBSERVERAPI_LISTEN=127.0.0.1:5678 or OBSERVERAPI_LISTEN=:5678`

Database connection:
`OBSERVERAPI_DB_URL=postgres://user:password@host/db_name?sslmode=disable`

Maximum number of connections to the database: 
`OBSERVERAPI_DB_POOLSIZE=20`

Log params:
```
OBSERVERAPI_LOG_LEVEL=info
OBSERVERAPI_LOG_FORMAT=text
OBSERVERAPI_LOG_OUTPUTTYPE=stderr
OBSERVERAPI_LOG_BUFFER=0
```

#### Run the Observer API

Run the Observer API: `./bin/api`.

Note: To run the Observer, you need to use the `observerapi.yaml` configuration file. The file should be in the `.artifacts` directory.

These endpoints are restricted from Observer users:
* /api/fee/{amount}

* /api/notification

* /api/stats/supply/total


### Deploy and start moniroting services

Before launching the monitoring script, install and set [Docker compose](https://docs.docker.com/compose/install/ "Install Compose ").

### Deploy the built-in monitoring system

Deploy the built-in system: `./scripts/monitor/monitor.sh`

`monitor.sh` starts Grafana and Prometheus configured by the Observer at:

* Grafana: `http://localhost:3000` with the default login and password: `login=admin` and `password=pass`
 
* Prometheus: `http://localhost:9090/graph`

* Observer мetrics: `http://localhost:8888` by default
 
* Observer health check service: `http://localhost:8888/healthcheck`

### Deploy a customized monitoring system

You can install, customize and deploy the monitoring system yourself. 

To do this:
1. Deploy [Grafana](https://grafana.com/docs/grafana/latest/installation/ "Install Grafana ") and [Prometheus](https://prometheus.io/docs/prometheus/latest/installation/ "Install Prometheus ").

   You can get the config for Prometheus [here](https://github.com/insolar/observer/blob/master/scripts/monitor/prometheus/prometheus.yaml).

2. Import [this Grafana dashboard](https://github.com/insolar/observer/blob/master/scripts/monitor/grafana/dashboards/observer.json) into Grafana. 
 
   If you need to, [read how to import a dashboard]( https://grafana.com/docs/grafana/latest/reference/export_import/).

## Contribute!

Feel free to submit issues, fork the repository and send pull requests! 

To make the process smooth for both reviewers and contributors, familiarize yourself with the following guidelines:

1. [Open source contributor guide](https://github.com/freeCodeCamp/how-to-contribute-to-open-source).
2. [Style guide: Effective Go](https://golang.org/doc/effective_go.html).
3. [List of shorthands for Go code review comments](https://github.com/golang/go/wiki/CodeReviewComments).

When submitting an issue, **include a complete test function** that reproduces it.

Thank you for your intention to contribute to the Insolar Mainnet project. As a company developing open-source code, we highly appreciate external contributions to our project.

Here is a helping hand if you decide to contribute.

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
