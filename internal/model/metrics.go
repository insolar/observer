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

package model

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	ErrorsCount = stats.Int64(
		"errors_count",
		"total count of errors that were happened at observer",
		stats.UnitDimensionless,
	)
)

func init() {
	err := view.Register(
		&view.View{
			Name:        ErrorsCount.Name(),
			Description: ErrorsCount.Description(),
			Measure:     ErrorsCount,
			Aggregation: view.Count(),
		},
	)
	if err != nil {
		panic(err)
	}
}
