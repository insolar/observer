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
  migration: 5.0.0-beauty
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
