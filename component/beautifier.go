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
	"github.com/insolar/insolar/insolar"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/collecting"
	"github.com/insolar/observer/observability"
)

func makeBeautifier(obs *observability.Observability) func(*raw) *beauty {
	log := obs.Log()
	metric := observability.MakeBeautyMetrics(obs, "collected")

	members := collecting.NewMemberCollector()
	transfers := collecting.NewTransferCollector(log)
	deposits := collecting.NewDepositCollector(log)
	addresses := collecting.NewMigrationAddressesCollector()

	balances := collecting.NewBalanceCollector(log)
	kycs := collecting.NewKYCCollector(log)
	groupUpdate := collecting.NewGroupUpdateCollector(log)
	updates := collecting.NewDepositUpdateCollector(log)
	wastings := collecting.NewWastingCollector()

	// SAIV

	users := collecting.NewUserCollector(log)
	groups := collecting.NewGroupCollector(log)

	return func(r *raw) *beauty {
		if r == nil {
			return nil
		}

		b := &beauty{
			pulse:        r.pulse,
			records:      r.batch,
			members:      make(map[insolar.ID]*observer.Member),
			deposits:     make(map[insolar.ID]*observer.Deposit),
			addresses:    make(map[string]*observer.MigrationAddress),
			balances:     make(map[insolar.ID]*observer.Balance),
			kycs:         make(map[insolar.Reference]*observer.UserKYC),
			groupUpdates: make(map[insolar.Reference]*observer.GroupUpdate),
			updates:      make(map[insolar.ID]*observer.DepositUpdate),
			wastings:     make(map[string]*observer.Wasting),
			users:        make(map[insolar.Reference]*observer.User),
			groups:       make(map[insolar.Reference]*observer.Group),
		}
		for _, rec := range r.batch {
			// entities

			member := members.Collect(rec)
			if member != nil {
				b.members[member.AccountState] = member
			}

			transfer := transfers.Collect(rec)
			if transfer != nil {
				b.transfers = append(b.transfers, transfer)
			}

			deposit := deposits.Collect(rec)
			if deposit != nil {
				b.deposits[deposit.DepositState] = deposit
			}

			for _, address := range addresses.Collect(rec) {
				b.addresses[address.Addr] = address
			}

			user := users.Collect(rec)
			if user != nil {
				b.users[user.UserRef] = user
			}

			group := groups.Collect(rec)
			if group != nil {
				b.groups[group.State] = group
			}

			// updates

			groupUpdate := groupUpdate.Collect(rec)
			if groupUpdate != nil {
				b.groupUpdates[groupUpdate.GroupState] = groupUpdate
			}

			kyc := kycs.Collect(rec)
			if kyc != nil {
				b.kycs[kyc.UserState] = kyc
			}

			balance := balances.Collect(rec)
			if balance != nil {
				b.balances[balance.AccountState] = balance
			}

			update := updates.Collect(rec)
			if update != nil {
				b.updates[update.ID] = update
			}

			wasting := wastings.Collect(rec)
			if wasting != nil {
				b.wastings[wasting.Addr] = wasting
			}
		}

		log := obs.Log()
		log.WithFields(logrus.Fields{
			"transfers": len(b.transfers),
			"members":   len(b.members),
			"deposits":  len(b.deposits),
			"addresses": len(b.addresses),
		}).Infof("collected entities")

		log.WithFields(logrus.Fields{
			"balances":                  len(b.balances),
			"deposit_updates":           len(b.updates),
			"migration_address_updates": len(b.wastings),
		}).Infof("collected updates")

		metric.Transfers.Add(float64(len(b.transfers)))
		metric.Members.Add(float64(len(b.members)))
		metric.Deposits.Add(float64(len(b.deposits)))
		metric.Addresses.Add(float64(len(b.addresses)))

		metric.Balances.Add(float64(len(b.balances)))
		metric.Updates.Add(float64(len(b.updates)))
		metric.Wastings.Add(float64(len(b.wastings)))

		return b
	}
}
