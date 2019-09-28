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

package component

import (
	"github.com/insolar/observer/v2/internal/app/observer/collecting"
	"github.com/insolar/observer/v2/observability"
)

func makeBeautifier(obs *observability.Observability) func(*raw) *beauty {
	log := obs.Log()
	metric := observability.MakeBeautyMetrics(obs, "collected")

	transfers := collecting.NewTransferCollector(log)
	addresses := collecting.NewMigrationAddressesCollector()
	users := collecting.NewUserCollector(log)

	return func(d *raw) *beauty {
		if d == nil {
			return nil
		}

		b := &beauty{}
		for _, rec := range d.batch {
			transfer := transfers.Collect(rec)
			if transfer != nil {
				b.transfers = append(b.transfers, transfer)
			}

			user := users.Collect(rec)
			if user != nil {
				b.users = append(b.users, user)
			}

			b.addresses = append(b.addresses, addresses.Collect(rec)...)
		}

		metric.Transfers.Add(float64(len(b.transfers)))

		return b
	}
}
