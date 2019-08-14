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
  addr: :8080
replicator:
  addr: 127.0.0.1:5678
  maxtransportmsg: 1073741824
  requestdelay: 10s
  batchsize: 1000
  transactionretrydelay: 3s
db:
  url: postgres://postgres@localhost/postgres?sslmode=disable
  createtables: true
```

#### Run and build
To run observer node you should provide config file: `observer.yaml` (like described above) at working directory.

To build and run:

`make all && ./bin/observer`

#### Metrics
Prometheus url: ${api.addr}/metrics

Healthcheck url: ${api.addr}/healthcheck

#### Alerts
Currently not provided.
