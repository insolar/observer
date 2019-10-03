//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package metrics

import (
	"context"
	"net/http"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"

	"github.com/insolar/observer/internal/configuration"
)

func NewMetrics() *metrics {
	return &metrics{}
}

type metrics struct {
	Configurator configuration.Configurator `inject:""`
	cfg          configuration.Metrics

	server *http.Server
}

func (m *metrics) Init(ctx context.Context) error {
	if m.Configurator != nil {
		m.cfg = m.Configurator.Actual().Metrics
	} else {
		m.cfg = configuration.Default().Metrics
	}

	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: m.cfg.Namespace,
	})
	if err != nil {
		logrus.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	}

	view.SetReportingPeriod(m.cfg.ReportingPeriod)

	mux := http.NewServeMux()
	mux.Handle("/metrics", pe)

	m.server = &http.Server{
		Addr:    m.cfg.ListenAddress,
		Handler: mux,
	}

	return nil
}

func (m *metrics) Start(ctx context.Context) error {
	// Now finally run the Prometheus exporter as a scrape endpoint.
	go func() {
		if err := m.server.ListenAndServe(); err != nil {
			logrus.Fatalf("Failed to run Prometheus scrape endpoint: %v", err)
		}
	}()

	return nil
}

func (m *metrics) Stop(ctx context.Context) error {
	if m.server == nil {
		return nil
	}

	const timeOut = 3
	logrus.Info("Shutting down metrics server")
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeOut)*time.Second)
	defer cancel()

	err := m.server.Shutdown(ctxWithTimeout)
	return errors.Wrap(err, "Can't gracefully stop metrics server")
}
