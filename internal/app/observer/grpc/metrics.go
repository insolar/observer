package grpc

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/observer/observability"
)

var (
	isDeprecatedClient prometheus.Gauge
)

func NewDeprecatedClientMetric(obs *observability.Observability) {
	isDeprecatedClient = obs.Gauge(prometheus.GaugeOpts{
		Name: "observer_version_is_deprecated",
		Help: "Observer version is deprecated if equal 1.And need new version",
	})
}
