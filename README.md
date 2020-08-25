[<img src="https://github.com/insolar/doc-pics/raw/master/st/github-readme-banner.png">](http://insolar.io/?utm_source=Github)

# Insolar Observer node
Insolar Observer node (later "the Node") allows trusted agents such as crypto exchanges to collect record data produced by Insolar smart contracts, organize it into a SQL database, and aggregate various statistical data.

Trusted agents can integrate the Node into their business applications or use the Node API to get data at their discretion.

The Node communicates with Insolar MainNet via gRPC and obtains data from a Heavy Material Node run on Insolar MainNet. 

Access to this Heavy Material Node is controlled by the Insolar authentication service and is limited to registered trusted agents.
This mechanism is designed to protect the Node users against inaccurate or corrupted data. 

Meanwhile, the Node users are responsible for the data they store locally on their Node instance and can regulate local access via an access control system of their choice.

To use the Node, you need to:

1. Install the prerequisites.
2. Obtain an authorized access to Insolar MainNet.
3. Build, deploy and monitor Insolar Observer node on the hardware of your choice.

There are no strict hardware or network recommendations for the Node, and users can choose the hardware and network connection at their discretion.

The tests showed the following configuration gives satisfying results:

* Quad core CPU 3.5GHz
* 4GB DDR3 RAM
* SATA SSD
* 10Mbps Internet connection with low latency

## Install the prerequisites

Install and set up [PostgreSQL 11.4](https://www.postgresql.org/download/) and [Go Tools 1.12](https://golang.org/doc/install).

## <a name="obtain-access">Obtain an authorized access</a> to Insolar MainNet

The Node users need to obtain an authorized access, otherwise their Node instance is not able to access the Heavy Material Node on Insolar MainNet or to collect data. 

To obtain it:
1. [Contact Insolar Team](mailto:support@insolar.io) to register as a trusted agent.
2. After the registration, the Team will send you your login along with a unique link to set your password. 

   Customary with the modern security policies, the link is active for a limited time and should be used within the next 24h after its generation.
   
   The link doesn't have a common Web UI and should be addressed via an HTTP client. For example, curl.
3. Set your password using the link. Consider this command as the reference example: 
   ```
   curl --request POST \
   --url '<unique_link_given_by_Insolar_Team>' \
   --header 'content-type: application/json' \
   --data '{
   "login": "<your_login>",
   "password": "<your_password>"
   } '
   ```
   The correct expected result is to see no errors returned by Curl.
4. After setting your password, put your login and password into the `/.artifacts/observer.yaml` configuration file (see [Build binaries](#build-binaries)).

   Working with Insolar MainNet, your Node instance uses your credentials from `observer.yaml` to obtain an access token to successfully communicate with the MainNet.

## Build

1. Clone the Node and go to its directory: 
   ```
   git clone git@github.com:insolar/observer.git && cd observer
   ```
   
   **Tip:** syntax of all subsequent commands and directories implies that you run or see them from the Node root directory.

2. <a name="build-binaries">Build binaries</a> using the instructions from the Makefile: 
   ```
   make all-node
   ```

   This command generates:
   * Three necessary configuration files (`migrate.yaml`, `observer.yaml`, `observerapi.yaml`) and places them into the `./.artifacts` directory.
   * Thee binaries (`migrate`, `observer`, `api`) and places them into the `./bin` directory.

   **Warning:** The Node uses Go modules. You may need to set the [Go modules environment variable](https://golang.org/cmd/go/#hdr-Module_support) to `on`: 
   ```
   GO111MODULE=on
   ```

## Deploy
To deploy, do the following:

1. Initialize your PostgreSQL database.

2. Configure and deploy the Node.

3. Configure and deploy the Node API.

4. Deploy the monitoring system.

### Initialize your PostgreSQL instance

Initialize the necessary database and tables in your PostgreSQL instance: 
```
make migrate-init
```

**Tip**: `migrate-init` is only for the initial database setting-up. For updating the database structure later on, run:
 ```
./bin/migrate --dir=scripts/migrations --init --config=.artifacts/migrate.yaml
``` 

### Configure and deploy the Node

1. Configure your access credentials in the `auth` subsection `./.artifacts/observer.yaml` (see [Obtain an authorized access to Insolar MainNet](#obtain-access)):
 
     ```
     replicator:
        auth:
           login: ""
           password: ""
     ```

     For the full list of parameters and their description, check [Configuration parameters](https://github.com/insolar/observer/wiki/Configuration-parameters).
   
     **Tip:** You can override all parameters in `observer.yaml` via environment variables that start with `OBSERVER` and use `_` as a separator. For example, `OBSERVER_DB_URL=...` or `OBSERVER_REPLICATOR_LISTEN=...`.

     **Warning:** Overriding via ENV variables works only with the configuration file in place. The configuration file must have the default number of parameters.


2. Run the Node: 
   ```
   ./bin/observer --config .artifacts/observer.yaml
   ```

   Wait for a while for it to sync with the trusted HMN. 
   
   **Tip:** initial synching can take various time depending on your hardware and network properties. An approximate time is 18 hours on a quad core CPU 3.5GHz, SATA SSD, 4GB DDR3 RAM on a 10Mbps connection with averagely 50ms network latency, and default Node configuration parameters.

### Configure and deploy the Node API

1. Configure the API endpoint parameter in `./.artifacts/observerapi.yaml`. 

   For example:

   ```
   listen: 127.0.0.1:5678
   ```

   For the full list of parameters and their description, check [Configuration parameters](https://github.com/insolar/observer/wiki/Configuration-parameters).
   
   **Tip**: You can override all parameters in `observerapi.yaml` via environment variables that start with `OBSERVERAPI` and use `_` as a separator. For example, `OBSERVERAPI_DB_URL=...` or `OBSERVERAPI_LISTEN=...`.

   **Warning**: Overriding via ENV variables works only with the configuration file in place. The configuration file must have the default number of parameters.

2. Run the Node API: 
   ```
   ./bin/api --config .artifacts/observerapi.yaml
   ```
   
3. Run a healthcheck on the Node and Node API: in a web browser of your choice, address the local `pulse_number` endpoint to get the current pulse; then refresh the page to make sure the pulse has changed to the next one.
   ```
   http://127.0.0.1:8080/api/pulse/number
   ```
   **Tip:** Read [the Node API description](https://apidocs.insolar.io/observer-node/v1) for the complete set of available API requests. 
   
### Deploy the monitoring system

1. Install and deploy [Grafana](https://grafana.com/docs/grafana/latest/installation/ "Install Grafana ") and [Prometheus](https://prometheus.io/docs/prometheus/latest/installation/ "Install Prometheus ").

   You can get the config for Prometheus [here](https://github.com/insolar/observer/blob/master/scripts/monitor/prometheus/prometheus.yaml).  

2. Import [this Grafana dashboard](https://github.com/insolar/observer/blob/master/scripts/monitor/grafana/dashboards/observer.json) into Grafana or create your own. 
 
   If necessary, [read how to import a dashboard]( https://grafana.com/docs/grafana/latest/reference/export_import/).
   
## Future Node updates

Upon an upcoming Insolar MainNet update, the Node of the current version may suspend synching. To resume synching, update your Node instance.

Each time your Node instance connects to the trusted Heavy Material Node, the instance sends its client type (`client_type`), communication protocol version (`heavy_version`), and smart contract version (`contract_version`) to this Heavy Material Node.

If the protocol and/or smart contract versions are older than those accepted by the Heavy Material Node, the Heavy Material Node sends back the following message: `version of the observer is outdated. Please upgrade this client`.

Upon receiving this message, your Node instance logs it and stops in 12s. Attempts to reconnect result in a stop.

Your Node instance also changes the `is_deprecated_client` metrics value to `1`, which causes the following message on the Grafana board used for the Node monitoring: `Your Node is out of date. Please update your Node.`

To update your Node instance:

1. Stop your Node instance if it didn't stop automatically. 
2. Clone and build the Node as described in [**Build** section](https://github.com/insolar/observer/blob/master/README.md#build). Alternatively, you can pull to the existing cloned repo and then build the Node.
3. Deploy the Node and Node API as described in the [**Configure and deploy the Node**](https://github.com/insolar/observer/blob/master/README.md#configure-and-deploy-the-node) and [**Configure and deploy the Node API**](https://github.com/insolar/observer/blob/master/README.md#configure-and-deploy-the-node-api) sections respectively.  

After the update has been completed, run your Node instance.
   
## Additional details

[Visit the Node wiki](https://github.com/insolar/observer/wiki) for additional details on database description, configuration parameters description and metrics description.

## Contribute!

Feel free to submit issues, fork the repository and send pull requests! 

To make the process smooth for both reviewers and contributors, familiarize yourself with the following guidelines:

1. [Open source contributor guide](https://github.com/freeCodeCamp/how-to-contribute-to-open-source).
2. [Style guide: Effective Go](https://golang.org/doc/effective_go.html).
3. [List of shorthands for Go code review comments](https://github.com/golang/go/wiki/CodeReviewComments).

When submitting an issue, **include a complete test function** that reproduces it.

Thank you for your intention to contribute to the Insolar Observer node project. As a company developing open-source code, we highly appreciate external contributions to our project.

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
