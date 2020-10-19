module github.com/insolar/observer

go 1.12

require (
	github.com/deepmap/oapi-codegen v1.3.0
	github.com/dgraph-io/badger v1.6.0 // indirect
	github.com/globocom/echo-prometheus v0.1.2
	github.com/go-pg/migrations v6.7.3+incompatible
	github.com/go-pg/pg v8.0.6+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/gojuno/minimock/v3 v3.0.5
	github.com/golang/groupcache v0.0.0-20191002201903-404acd9df4cc // indirect
	github.com/google/uuid v1.1.1
	github.com/hashicorp/golang-lru v0.5.3
	github.com/insolar/insconfig v0.0.0-20200227134411-011eca6dc866
	github.com/insolar/insolar v1.9.0
	github.com/insolar/mainnet v1.12.1
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/labstack/echo/v4 v4.1.11
	github.com/mitchellh/mapstructure v1.1.2
	github.com/ory/dockertest/v3 v3.5.2
	github.com/pelletier/go-toml v1.5.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/ugorji/go v1.1.12 // indirect
	go.opencensus.io v0.22.1 // indirect
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d // indirect
	gonum.org/v1/gonum v0.6.0 // indirect
	google.golang.org/appengine v1.4.0 // indirect
	google.golang.org/genproto v0.0.0-20191009194640-548a555dbc03 // indirect
	google.golang.org/grpc v1.21.0
	gopkg.in/yaml.v2 v2.3.0
	mellium.im/sasl v0.2.1 // indirect
)

replace github.com/insolar/observer => ./

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
