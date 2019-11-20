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
	"github.com/go-pg/pg"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/connectivity"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/observability"
)

func makeStorer(cfg *configuration.Configuration, obs *observability.Observability, conn *connectivity.Connectivity) func(*beauty) {
	log := obs.Log()
	db := conn.PG()

	metric := observability.MakeBeautyMetrics(obs, "stored")
	return func(b *beauty) {
		if b == nil {
			return
		}

		err := db.RunInTransaction(func(tx *pg.Tx) error {

			// plain objects

			pulses := postgres.NewPulseStorage(cfg, obs, tx)
			err := pulses.Insert(b.pulse)
			if err != nil {
				return err
			}

			records := postgres.NewRecordStorage(cfg, obs, tx)
			for _, rec := range b.records {
				err := records.Insert(rec)
				if err != nil {
					return err
				}
			}

			requests := postgres.NewRequestStorage(obs, tx)
			for _, req := range b.requests {
				err := requests.Insert(req)
				if err != nil {
					return err
				}
			}

			results := postgres.NewResultStorage(obs, tx)
			for _, res := range b.results {
				err := results.Insert(res)
				if err != nil {
					return err
				}
			}

			objects := postgres.NewObjectStorage(obs, tx)
			for _, act := range b.activates {
				err := objects.Insert(act)
				if err != nil {
					return err
				}
			}

			for _, amd := range b.amends {
				err := objects.Insert(amd)
				if err != nil {
					return err
				}
			}

			for _, deact := range b.deactivates {
				err := objects.Insert(deact)
				if err != nil {
					return err
				}
			}

			// new entities
			users := postgres.NewUserStorage(obs, tx)
			for _, user := range b.users {
				err := users.Insert(user)
				if err != nil {
					return err
				}
			}

			groups := postgres.NewGroupStorage(obs, tx)
			for _, group := range b.groups {
				err := groups.Insert(group)
				if err != nil {
					return err
				}
			}

			ug := postgres.NewUserGroupStorage(obs, tx)
			for _, group := range b.groups {
				err := ug.Insert(group)
				if err != nil {
					return err
				}
			}

			mgrs := postgres.NewMGRStorage(obs, tx)
			for _, mgr := range b.mgrs {
				err := mgrs.Insert(mgr)
				if err != nil {
					return err
				}
			}

			notifications := postgres.NewNotificationStorage(obs, tx)
			for _, ntf := range b.notifications {
				err := notifications.Insert(ntf)
				if err != nil {
					return err
				}
			}

			transactions := postgres.NewTransactionStorage(obs, tx)
			for _, tx := range b.transactions {
				err := transactions.Insert(tx)
				if err != nil {
					return err
				}
			}

			// updates
			for _, mgr := range b.mgrUpdates {
				err := mgrs.Update(mgr)
				if err != nil {
					return err
				}
			}

			for _, tx := range b.transactionsUpdate {
				err := transactions.Update(tx)
				if err != nil {
					return err
				}
			}

			for _, bln := range b.groupBalances {
				err := groups.UpdateBalance(bln)
				if err != nil {
					return err
				}
			}

			for _, group := range b.groupUpdates {
				err := groups.Update(group)
				if err != nil {
					return err
				}
			}

			for _, kyc := range b.kycs {
				err := users.Update(kyc)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Error(err)
			return
		}

		log.Info("items successfully stored")

		metric.Users.Add(float64(len(b.users)))
		metric.Groups.Add(float64(len(b.groups)))
		metric.Transactions.Add(float64(len(b.transactions)))
		metric.Notifications.Add(float64(len(b.notifications)))
		metric.MGRs.Add(float64(len(b.mgrs)))

		metric.UserUpdates.Add(float64(len(b.kycs)))
		metric.GroupUpdates.Add(float64(len(b.groupUpdates)))
		metric.BalanceUpdates.Add(float64(len(b.groupBalances)))
		metric.TransactionUpdates.Add(float64(len(b.transactionsUpdate)))
		metric.MGRUpdates.Add(float64(len(b.mgrUpdates)))
	}
}
