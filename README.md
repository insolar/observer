[<img src="https://github.com/insolar/doc-pics/raw/master/st/github-readme-banner.png">](http://insolar.io/?utm_source=Github)

# Insolar Observer node
Insolar Observer node (later "the Node") allows trusted agents such as crypto exchanges collect record data produced by Insolar smart contracts, organize it into a SQL database, and aggregate various statistical data.

Trusted agents can integrate the Node into their business applications or use the Node API to get the data at their discretion.

The Node communicates with Insolar Platform via gRPC and obtains data from a trusted Heavy Material Node run on Insolar Platform. 

Access to the Heavy Material Node is controlled by an Insolar authentication service and is limited to registered trusted agents.
This mechanism is designed to protect the Node users against inaccurate or corrupted data. 

Meanwhile, the Node users are responsible for the data they store locally on their Node instance and can regulate local access via an access control system of their choice.

To use Insolar Observer node, you need to:

1. Install prerequisites
2. Obtain an authorized access to Insolar Platform.
3. Build, deploy and monitor Insolar Observer node on the hardware of your choice.

# Install the prerequisites

1. Install and set up [PostgreSQL 11.4](https://www.postgresql.org/download/).

2. Install and set up [Go Tools 1.12](https://golang.org/doc/install).

3. Install [Docker Desktop](https://www.docker.com/products/docker-desktop).

# Obtain an authorized access to Insolar Platform

The Node users need to obtain an authorized access, otherwise they are not able to address the trusted Heavy Material Node on Insolar Platform or to collect the data. 

To obtain it:
1. [Contact Insolar Team](https://insolar.io/contact) to register as a trusted agent.
2. After the registration, the Team will send you your login along with a unique link to set your password. The link doesn't have a common Web UI and should be addressed via a CLI tool such as Curl.
3. Set your password using the link. Consider this command as the reference example: 
   ```
   curl -d '{"login":"your_login", "password":"password_of_your_choice"}' -H "Content-Type: application/json" -X POST https://<api-url>/auth/set-password?code=XXXXXXXXXXXXXXXXX
   ```
   The correct expected result is to see no errors returned by Curl.
4. After setting your password, put your login and password into the `/.artifacts/observer.yaml` configuration file (see **Build binaries**).
   Working with Insolar Platform, your Node instance uses your credentials from `observer.yaml` to obtain an access token to successfully communicate with the Platform.

# Build, deploy and monitor

Choose an appropriate mode and proceed with the instructions.

## Using Docker Compose

### Build, deploy

1. Make sure you've obtained an authorized access to Insolar Platform and set your `login` and `password` correctly. Check them in `./.artifacts/observer.yaml` under the `auth` parameter section.
2. Clone the Observer and change to its directory: `git clone git@github.com:insolar/observer.git && cd observer`.
3. Generate the needed configuration files: `make config-node`.
2. Run `docker-compose up -d migrate` to create a database and the necessary tables in it. You may want to wait for 10-15 seconds to ensure the database has been set correctly.
3. Run `docker-compose up -d` to fire up the services.
4. Check the services are up and running, and the migration service has done its job correctly: `docker-compose ps`. You should see the API (`observer_api_1`), database service (`observer_postgres_1`),Observer service (`observer_replicator_1`) in the `Up` status, and the migration service (`observer_migrate_1`) in the `Exit 0` status.
5. Run a safe check on your credentials: `docker-compose logs -f replicator`. 
   1. If you see `ERR failed to get gRPC stream from exporter.Export method: rpc error: code = Unauthenticated desc = transport: can't get access_token:...` it means your credentials have been set wrong or you haven't set them. If so, check #1 in `Build using Docker Compose` or check #4 in `Install the prerequisites and get access` and then said #1.
   2. Otherwise, if you see blazing-fast amending log lines starting with `DBG...`, your Node is fine and reading data from a trusted Heavy Material Node. 

## Using raw binary files

### Build
1. Clone the Observer and change to its directory: `git clone git@github.com:insolar/observer.git && cd observer`.

2. Build binaries automatically using the instructions from the Makefile: `make all-node`.

    This command generates:
    * Three necessary configuration files (`migrate.yaml`, `observer.yaml`, `observerapi.yaml`) and places them into the hidden `./.artifacts` directory.
    * Thee binaries (`migrate`, `observer`, `api`) and places them into `./.bin/migrate`, `./.bin/observer`, `./bin/api` respectively.

    **Warning:** The Observer uses Go modules. You may need to set the [Go modules environment variable](https://golang.org/cmd/go/#hdr-Module_support) to `on`: `GO111MODULE=on`.

### Deploy

1. Initialize your PostgreSQL database.

2. Configure and deploy the Node.

3. Configure and deploy the Node API.

4. Deploy the monitoring system.

#### Initialize your PostgreSQL database

Initialize your PostgreSQL database (generated go binaries required): `make migrate-init`.

**Tip**: `migrate-init` is only for the initial database setting-up. Later if needed, you should use `./bin/migrate --dir=scripts/migrations --init --config=.artifacts/migrate.yaml` for updating the database structure.

#### Configure and deploy the Node

1. To configure, edit the configuration parameters in `./.artifacts/observer.yaml`:

   * Database connection in the `db` section:
   `url: postgres://user:password@host/db_name?sslmode=disable`.

   * Insolar network address and user credentials to access it in the `auth` section:
   `url: https://<api-url>/auth/token`
   `login: "<your_login>"`
   `password: "<your_password"`.

   * Log parameters in the `log` section:
   ```
     level: debug
     format: text
     outputtype: stderr
     outputparams:
     buffer: 0
   ```
   For the full list of parameters and their description, check [Configuration parameters](https://github.com/insolar/observer/wiki/Configuration-parameters).
   
   **Tip:** You can override all parameters in `observer.yaml` via environment variables that start with `OBSERVER` and use `_` as a separator. For example, `OBSERVER_DB_URL=...` or `OBSERVER_REPLICATOR_LISTEN=...`.

   **Warning:** Overriding via ENV variables works only with the configuration file in place. The configuration file must have the default number of parameters.
   
2. Make sure the `observer.yaml` configuration file is in the `./.artifacts` directory.

3. Run the Node: ```./bin/observer --config .artifacts/observer.yaml```.

   Wait for a while for it to sync with the trusted HMN. 
   
   **Tip:** initial synching can take up to 20 hours as Insolar Platform has a lot of data.

#### Configure and deploy the Observer API

1. To configure, edit the configuration parameters in `./.artifacts/observerapi.yaml`:

   * API endpoint:
   `listen: 127.0.0.1:5678 or listen: :5678`.

   * Database connection in the `db` section:
   `url: postgres://user:password@host/db_name?sslmode=disable`.

   * Maximum number of connections to the database: 
   `poolsize: 20`.

   * Log params in the `log` section:
   ```
    level: debug
    format: json
    outputtype: stderr
    outputparams: ""
    buffer: 0
   ```
   For the full list of parameters and their description, check [Configuration parameters](https://github.com/insolar/observer/wiki/Configuration-parameters).
   
   **Tip**: You can override all parameters in `observerapi.yaml` via environment variables that start with `OBSERVERAPI` and use `_` as a separator. For example, `OBSERVERAPI_DB_URL=...` or `OBSERVERAPI_LISTEN=...`.

   **Warning**: overriding via ENV variables works only with the configuration file in place with the default number of parameters.

3. Run the Node API: ```./bin/api --config .artifacts/observerapi.yaml```.

#### Deploy the monitoring system

1. Make sure you've installed [Docker compose](https://docs.docker.com/compose/install/ "Install Compose ").

2. Choose to deploy the built-in or a customized monitoring system as described below.

##### Built-in monitoring system

To deploy the built-in monitoring system, execute the script: 
```./scripts/monitor/monitor.sh```

`monitor.sh` starts Grafana and Prometheus configured for the Node at:

* Grafana: `http://localhost:3000` with the default login and password: `admin` and `pass` respectively. Grafana starts with a preset dashboard. To navigate to the dashboard, use the left menu: `Dashboards > Manage > Observer`.
 
* Prometheus: `http://localhost:9090/graph`

* Observer node metrics: `http://localhost:8888`
 
* Observer node health check service: `http://localhost:8888/healthcheck`

##### Сustomized monitoring system

To deploy a customized monitoring system:

1. Deploy [Grafana](https://grafana.com/docs/grafana/latest/installation/ "Install Grafana ") and [Prometheus](https://prometheus.io/docs/prometheus/latest/installation/ "Install Prometheus ").

   You can get the config for Prometheus [here](https://github.com/insolar/observer/blob/master/scripts/monitor/prometheus/prometheus.yaml).

2. Import [this Grafana dashboard](https://github.com/insolar/observer/blob/master/scripts/monitor/grafana/dashboards/observer.json) into Grafana or create your own. 
 
   If you need to, [read how to import a dashboard]( https://grafana.com/docs/grafana/latest/reference/export_import/).

## Contribute!

Feel free to submit issues, fork the repository and send pull requests! 

To make the process smooth for both reviewers and contributors, familiarize yourself with the following guidelines:

1. [Open source contributor guide](https://github.com/freeCodeCamp/how-to-contribute-to-open-source).
2. [Style guide: Effective Go](https://golang.org/doc/effective_go.html).
3. [List of shorthands for Go code review comments](https://github.com/golang/go/wiki/CodeReviewComments).

When submitting an issue, **include a complete test function** that reproduces it.

Thank you for your intention to contribute to the Insolar Observer project. As a company developing open-source code, we highly appreciate external contributions to our project.

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
