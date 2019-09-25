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
	"fmt"
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func Make() *Observability {
	return &Observability{
		log:     logrus.New(),
		metrics: prometheus.NewRegistry(),
	}
}

type Observability struct {
	log     *logrus.Logger
	metrics *prometheus.Registry
}

func (o *Observability) Log() *logrus.Logger {
	return o.log
}

func (o *Observability) Metrics() *prometheus.Registry {
	return o.metrics
}

func MakeBeautyMetrics(obs *Observability, action string) *beautyMetrics {
	metrics := obs.Metrics()
	counters := &beautyMetrics{}
	v := reflect.ValueOf(counters).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := strings.ToLower(t.Field(i).Name)
		name := fmt.Sprintf("observer_%s_%s_total", field, action)
		help := fmt.Sprintf("Number of %s successfully %s in DB.", field, action)
		collector := prometheus.NewCounter(prometheus.CounterOpts{
			Name: name,
			Help: help,
		})
		metrics.MustRegister(collector)
		v.Field(i).Set(reflect.ValueOf(collector))
	}
	return counters
}

type beautyMetrics struct {
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
