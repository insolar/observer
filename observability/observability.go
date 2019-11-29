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

package observability

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/prometheus/client_golang/prometheus"
)

func Make(ctx context.Context) *Observability {
	return &Observability{
		log:      inslogger.FromContext(ctx),
		metrics:  prometheus.NewRegistry(),
		counters: make(map[string]prometheus.Counter),
		gauges:   make(map[string]prometheus.Gauge),
	}
}

type Observability struct {
	log      insolar.Logger
	metrics  *prometheus.Registry
	counters map[string]prometheus.Counter
	gauges   map[string]prometheus.Gauge
}

func (o *Observability) Log() insolar.Logger {
	return o.log
}

func (o *Observability) Metrics() *prometheus.Registry {
	return o.metrics
}

func (o *Observability) Counter(opts prometheus.CounterOpts) prometheus.Counter {
	c, ok := o.counters[opts.Name]
	if ok {
		return c
	}
	c = prometheus.NewCounter(opts)
	err := o.metrics.Register(c)
	if err != nil {
		o.log.WithField("metric_collector", opts.Name).
			Errorf("failed to register metric")
		return c
	}
	o.counters[opts.Name] = c
	return c
}

func (o *Observability) Gauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	g, ok := o.gauges[opts.Name]
	if ok {
		return g
	}
	g = prometheus.NewGauge(opts)
	err := o.metrics.Register(g)
	if err != nil {
		o.log.WithField("metric_collector", opts.Name).
			Errorf("failed to register metric")
		return g
	}
	o.gauges[opts.Name] = g
	return g
}

func MakeBeautyMetrics(obs *Observability, action string) *BeautyMetrics {
	counters := &BeautyMetrics{}
	v := reflect.ValueOf(counters).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := strings.ToLower(t.Field(i).Name)
		name := fmt.Sprintf("observer_%s_%s_total", field, action)
		help := fmt.Sprintf("Number of %s successfully %s in DB.", field, action)
		opts := prometheus.CounterOpts{
			Name: name,
			Help: help,
		}
		collector := obs.Counter(opts)
		v.Field(i).Set(reflect.ValueOf(collector))
	}
	return counters
}

type BeautyMetrics struct {
	Pulses      prometheus.Counter
	Records     prometheus.Counter
	Requests    prometheus.Counter
	Results     prometheus.Counter
	Activates   prometheus.Counter
	Amends      prometheus.Counter
	Deactivates prometheus.Counter

	Transfers prometheus.Counter
	Members   prometheus.Counter
	Balances  prometheus.Counter
	Deposits  prometheus.Counter
	Updates   prometheus.Counter
	Addresses prometheus.Counter
	Wastings  prometheus.Counter
}

type CommonObserverMetrics struct {
	PulseProcessingTime prometheus.Gauge
}

func MakeCommonMetrics(obs *Observability) *CommonObserverMetrics {
	m := CommonObserverMetrics{
		PulseProcessingTime: obs.Gauge(prometheus.GaugeOpts{
			Name: "observer_pulse_processing_time",
			Help: "Seconds spent on processing data from pulse",
		}),
	}

	return &m
}
