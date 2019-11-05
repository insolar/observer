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
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer/filtering"
	"github.com/insolar/observer/observability"
)

func makeFilter(obs *observability.Observability) func(*beauty) *beauty {
	log := obs.Log()
	metric := observability.MakeBeautyMetrics(obs, "filtered")
	return func(b *beauty) *beauty {
		if b == nil {
			return nil
		}

		filtering.NewBalanceFilter().Filter(b.balances, b.members)
		filtering.NewDepositUpdateFilter().Filter(b.depositUpdates, b.deposits)
		filtering.NewWastingFilter().Filter(b.wastings, b.addresses)

		b.requests, b.results, b.activates, b.amends, b.deactivates = filtering.NewSeparatorFilter().
			Filter(b.records)

		log.Info("items successfully filtered")
		log.WithFields(logrus.Fields{
			"requests":   len(b.requests),
			"results":    len(b.results),
			"activates":  len(b.activates),
			"amends":     len(b.amends),
			"deactivate": len(b.deactivates),
		}).Infof("separated records")

		metric.Transfers.Add(float64(len(b.txSagaResult)))
		metric.Members.Add(float64(len(b.members)))
		metric.Deposits.Add(float64(len(b.deposits)))
		metric.Addresses.Add(float64(len(b.addresses)))

		metric.Balances.Add(float64(len(b.balances)))
		metric.Updates.Add(float64(len(b.depositUpdates)))
		metric.Wastings.Add(float64(len(b.wastings)))
		return b
	}
}
