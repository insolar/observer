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

package replicator

import (
	"github.com/insolar/insolar/instrumentation/insmetrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	replicator = insmetrics.MustTagKey("replicator")
)

var (
	lastSyncPulse = stats.Int64(
		"last_sync_pulse",
		"last pulse-number that was replicated from HME",
		stats.UnitDimensionless,
	)
	processingTime = stats.Float64(
		"processing_duration_seconds",
		"time that needs to replicate and beautify data",
		stats.UnitMilliseconds,
	)
	pullRecordsTime = stats.Float64(
		"pull_records_time_milliseconds",
		"time that needs to pull current batch of records",
		stats.UnitMilliseconds,
	)
)

func init() {
	err := view.Register(
		&view.View{
			Name:        lastSyncPulse.Name(),
			Description: lastSyncPulse.Description(),
			Measure:     lastSyncPulse,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{replicator},
		},
		&view.View{
			Name:        processingTime.Name(),
			Description: processingTime.Description(),
			Measure:     processingTime,
			Aggregation: view.Distribution(1, 1000, 2000, 5000, 10000),
			TagKeys:     []tag.Key{replicator},
		},
		&view.View{
			Name:        pullRecordsTime.Name(),
			Description: pullRecordsTime.Description(),
			Measure:     pullRecordsTime,
			Aggregation: view.Distribution(1, 1000, 2000, 5000, 10000),
			TagKeys:     []tag.Key{replicator},
		},
	)
	if err != nil {
		panic(err)
	}
}
