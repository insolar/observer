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
	"sync/atomic"

	"github.com/insolar/insolar/insolar"
	"github.com/prometheus/client_golang/prometheus"
)

type PulseCollector struct {
	desc *prometheus.Desc

	value uint32
}

type PulseOps prometheus.Opts

func NewPulseCollector(opts PulseOps) *PulseCollector {
	c := &PulseCollector{desc: prometheus.NewDesc(
		prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		opts.Help,
		nil,
		opts.ConstLabels,
	)}
	prometheus.MustRegister(c)
	return c
}

func (c *PulseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *PulseCollector) Collect(ch chan<- prometheus.Metric) {
	val := atomic.LoadUint32(&c.value)
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.UntypedValue, float64(val))
}

func (c *PulseCollector) Set(pn insolar.PulseNumber) {
	atomic.StoreUint32(&c.value, uint32(pn))
}
