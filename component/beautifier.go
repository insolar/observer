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
	"context"

	gopg "github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/collecting"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/store/pg"
	"github.com/insolar/observer/internal/app/observer/tree"
	"github.com/insolar/observer/observability"
)

type PGer interface {
	PG() *gopg.DB
}

func makeBeautifier(
	cfg *configuration.Configuration,
	obs *observability.Observability,
	conn PGer,
) func(context.Context, *raw) *beauty {
	log := obs.Log()
	metric := observability.MakeBeautyMetrics(obs, "collected")

	cachedStore, err := store.NewCacheRecordStore(pg.NewPgStore(conn.PG()), cfg.Replicator.CacheSize)
	if err != nil {
		panic(errors.Wrap(err, "failed to init cached record store"))
	}
	treeBuilder := tree.NewBuilder(cachedStore)

	members := collecting.NewMemberCollector(log, cachedStore, treeBuilder)
	txRegisters := collecting.NewTxRegisterCollector(log)
	txResults := collecting.NewTxResultCollector(log, cachedStore)
	txSagaResults := collecting.NewTxSagaResultCollector(log, cachedStore)
	deposits := collecting.NewDepositCollector(log, cachedStore)
	addresses := collecting.NewMigrationAddressesCollector(log, cachedStore)

	balances := collecting.NewBalanceCollector(log)
	depositUpdates := collecting.NewDepositUpdateCollector(log)
	wastings := collecting.NewWastingCollector(cachedStore)

	return func(ctx context.Context, r *raw) *beauty {
		if r == nil {
			return nil
		}

		b := &beauty{
			pulse:          r.pulse,
			members:        make(map[insolar.ID]*observer.Member),
			deposits:       make(map[insolar.ID]*observer.Deposit),
			addresses:      make(map[string]*observer.MigrationAddress),
			balances:       make(map[insolar.ID]*observer.Balance),
			depositUpdates: make(map[insolar.ID]*observer.DepositUpdate),
			wastings:       make(map[string]*observer.Wasting),
		}

		for _, rec := range r.batch {
			switch rec.Record.Virtual.Union.(type) {
			case *record.Virtual_IncomingRequest, *record.Virtual_OutgoingRequest:
				err = cachedStore.SetRequest(ctx, rec.Record)
			case *record.Virtual_Activate, *record.Virtual_Amend, *record.Virtual_Deactivate:
				err = cachedStore.SetSideEffect(ctx, rec.Record)
			case *record.Virtual_Result:
				err = cachedStore.SetResult(ctx, rec.Record)
			}
			if err != nil {
				panic(errors.Wrap(err, "failed to insert record to storage"))
			}
		}

		for _, rec := range r.batch {
			// entities
			obsRecord := observer.Record(rec.Record)

			members := members.Collect(ctx, &obsRecord)
			for _, member := range members {
				b.members[member.AccountState] = member
			}
			reg := txRegisters.Collect(ctx, *rec)
			if reg != nil {
				b.txRegister = append(b.txRegister, *reg)
			}
			res := txResults.Collect(ctx, *rec)
			if res != nil {
				b.txResult = append(b.txResult, *res)
			}
			sagRes := txSagaResults.Collect(ctx, *rec)
			if sagRes != nil {
				b.txSagaResult = append(b.txSagaResult, *sagRes)
			}

			deposits := deposits.Collect(ctx, &obsRecord)
			for _, deposit := range deposits {
				b.deposits[deposit.DepositState] = deposit
			}

			for _, address := range addresses.Collect(ctx, &obsRecord) {
				b.addresses[address.Addr] = address
			}

			// updates

			balance := balances.Collect(&obsRecord)
			if balance != nil {
				b.balances[balance.AccountState] = balance
			}

			update := depositUpdates.Collect(&obsRecord)
			if update != nil {
				b.depositUpdates[update.ID] = update
			}

			wasting := wastings.Collect(ctx, &obsRecord)
			if wasting != nil {
				b.wastings[wasting.Addr] = wasting
			}
		}

		log := obs.Log()
		log.WithFields(logrus.Fields{
			"tx_registrations": len(b.txRegister),
			"tx_results":       len(b.txResult),
			"tx_saga_results":  len(b.txSagaResult),
			"members":          len(b.members),
			"deposits":         len(b.deposits),
			"addresses":        len(b.addresses),
		}).Infof("collected entities")

		log.WithFields(logrus.Fields{
			"balances":                  len(b.balances),
			"deposit_updates":           len(b.depositUpdates),
			"migration_address_updates": len(b.wastings),
		}).Infof("collected depositUpdates")

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
