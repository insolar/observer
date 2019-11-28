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
		filtering.NewGroupUpdateFilter().Filter(b.groupUpdates, b.groups)
		filtering.NewMGRUpdateFilter().Filter(b.mgrUpdates, b.mgrs)
		filtering.NewSavingUpdateFilter().Filter(b.nsUpdates, b.savings)

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

		metric.Users.Add(float64(len(b.users)))
		metric.Groups.Add(float64(len(b.groups)))
		metric.Transactions.Add(float64(len(b.transactions)))
		metric.Notifications.Add(float64(len(b.notifications)))
		metric.MGRs.Add(float64(len(b.mgrs)))
		metric.Savings.Add(float64(len(b.savings)))

		metric.UserUpdates.Add(float64(len(b.kycs)))
		metric.GroupUpdates.Add(float64(len(b.groupUpdates)))
		metric.BalanceUpdates.Add(float64(len(b.groupBalances)))
		metric.TransactionUpdates.Add(float64(len(b.transactionsUpdate)))
		metric.MGRUpdates.Add(float64(len(b.mgrUpdates)))
		metric.SavingsUpdates.Add(float64(len(b.nsUpdates)))
		return b
	}
}
