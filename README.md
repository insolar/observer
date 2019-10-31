# Observer
Service that replicates smart-contract records data to DB 
and constructs some of new structures: user accounts, token transfers, deposit migrations, migration addresses.

#### Networking
Opens ports: ${api.addr}

#### Depends on
Insolar HME node: ${replicator.addr}

PostgreSQL DB 11.4: ${db.url}

#### Example configuration
```
api:
  addr: :0
replicator:
  addr: 127.0.0.1:5678
  maxtransportmsg: 1073741824
  attempts: 2147483647
  attemptinterval: 10s
  batchsize: 1000
  transactionretrydelay: 3s
db:
  url: postgres://postgres@localhost/postgres?sslmode=disable
  attempts: 2147483647
  attemptinterval: 3s
  createtables: false
```

#### Run and build
To run observer node you should provide config file: `observer.yaml` (like described above) at working directory.

You can generate it from 
`configuration/configuration.go`
like this 

`make env`

To build and run:

`make all && ./bin/observer`

#### Metrics
Prometheus url: ${api.addr}/metrics

Healthcheck url: ${api.addr}/healthcheck

#### Alerts
Currently not provided.


#### Development
Install deps (dep, minimock) running

`make install_deps`

## API Observer
API for observer service. We use chi as router.

To generate API implementation from open-api spec, use oapi-codegen. Get it via:
```
go get github.com/deepmap/oapi-codegen/cmd/oapi-codegen
``` 
Generate types and API from observer API:
```
oapi-codegen -package api -generate types,server ../insolar-observer-api/api-exported.yaml > internal/app/api/generated.go
```
