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
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/collecting"
	"github.com/insolar/observer/observability"
	"github.com/sirupsen/logrus"
)

func makeBeautifier(obs *observability.Observability) func(*raw) *beauty {
	log := obs.Log()
	metric := observability.MakeBeautyMetrics(obs, "collected")

	mgrsUpdate := collecting.NewMGRUpdateCollector(log)
	groupUpdate := collecting.NewGroupUpdateCollector(log)
	nsUpdate := collecting.NewSavingsUpdateCollector(log)
	transactionsUpdate := collecting.NewTransactionUpdateCollector(log)
	kycs := collecting.NewKYCCollector(log)
	users := collecting.NewUserCollector(log)
	groups := collecting.NewGroupCollector(log)
	mgrs := collecting.NewMGRCollector(log)
	savings := collecting.NewSavingCollector(log)
	notifications := collecting.NewNotificationCollector(log)
	transactions := collecting.NewTransactionCollector(log)
	groupBalances := collecting.NewBalanceUpdateCollector(log)

	return func(r *raw) *beauty {
		if r == nil {
			return nil
		}

		b := &beauty{
			pulse:              r.pulse,
			records:            r.batch,
			kycs:               make(map[insolar.ID]*observer.UserKYC),
			groupUpdates:       make(map[insolar.ID]*observer.GroupUpdate),
			users:              make(map[insolar.Reference]*observer.User),
			groups:             make(map[insolar.ID]*observer.Group),
			mgrs:               make(map[insolar.ID]*observer.MGR),
			savings:            make(map[insolar.ID]*observer.NormalSaving),
			mgrUpdates:         make(map[insolar.ID]*observer.MGRUpdate),
			notifications:      make(map[insolar.Reference]*observer.Notification),
			transactions:       []*observer.Transaction{},
			transactionsUpdate: []*observer.TransactionUpdate{},
			groupBalances:      []*observer.BalanceUpdate{},
		}
		for _, rec := range r.batch {
			// entities
			user := users.Collect(rec)
			if user != nil {
				b.users[user.UserRef] = user
			}

			group := groups.Collect(rec)
			if group != nil {
				b.groups[group.State] = group
			}

			mgr := mgrs.Collect(rec)
			if mgr != nil {
				b.mgrs[mgr.State] = mgr
			}

			notification := notifications.Collect(rec)
			if notification != nil {
				b.notifications[notification.Ref] = notification
			}

			tx := transactions.Collect(rec)
			if tx != nil {
				b.transactions = append(b.transactions, tx)
			}

			saving := savings.Collect(rec)
			if saving != nil {
				b.savings[saving.State] = saving
			}

			// updates

			groupUpdate := groupUpdate.Collect(rec)
			if groupUpdate != nil {
				b.groupUpdates[groupUpdate.GroupState] = groupUpdate
			}

			mgrUpdate := mgrsUpdate.Collect(rec)
			if mgrUpdate != nil {
				b.mgrUpdates[mgrUpdate.MGRState] = mgrUpdate
			}

			nsUpdate := nsUpdate.Collect(rec)
			if nsUpdate != nil {
				b.groupUpdates[nsUpdate.SavingState] = groupUpdate
			}

			balanceUpdate := groupBalances.Collect(rec)
			if balanceUpdate != nil {
				b.groupBalances = append(b.groupBalances, balanceUpdate)
			}

			transactionUpdate := transactionsUpdate.Collect(rec)
			if transactionUpdate != nil {
				b.transactionsUpdate = append(b.transactionsUpdate, transactionUpdate)
			}

			kyc := kycs.Collect(rec)
			if kyc != nil {
				b.kycs[kyc.UserState] = kyc
			}
		}

		log := obs.Log()
		log.WithFields(logrus.Fields{
			"users":         len(b.users),
			"groups":        len(b.groups),
			"transactions":  len(b.transactions),
			"notifications": len(b.notifications),
			"MGR":           len(b.mgrs),
			"Savings":       len(b.savings),
		}).Infof("collected entities")

		log.WithFields(logrus.Fields{
			"user_update":        len(b.kycs),
			"group_update":       len(b.groupUpdates),
			"balance_update":     len(b.groupBalances),
			"MGR_update":         len(b.mgrUpdates),
			"transaction_update": len(b.transactionsUpdate),
		}).Infof("collected updates")

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
