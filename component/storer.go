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
	"fmt"
	"strings"
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/pkg/cycle"
	"github.com/insolar/observer/internal/pkg/math"
	"github.com/insolar/observer/observability"
)

func makeStorer(
	cfg *configuration.Configuration,
	obs *observability.Observability,
	conn PGer,
) func(*beauty, *state) *observer.Statistic {
	log := obs.Log()
	db := conn.PG()

	metric := observability.MakeBeautyMetrics(obs, "stored")
	platformNodes := obs.Gauge(prometheus.GaugeOpts{
		Name: "observer_platform_nodes",
	})
	return func(b *beauty, s *state) *observer.Statistic {
		if b == nil {
			return nil
		}

		var stat *observer.Statistic

		cycle.UntilError(func() error {
			err := db.RunInTransaction(func(tx *pg.Tx) error {

				// plain records

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

				members := postgres.NewMemberStorage(obs, tx)
				for _, member := range b.members {
					err := members.Insert(member)
					if err != nil {
						return err
					}
				}

				transfers := postgres.NewTransferStorage(obs, tx)
				for _, transfer := range b.transfers {
					err := transfers.Insert(transfer)
					if err != nil {
						return err
					}
				}

				// Uncomment when migrations are ready.
				// err = StoreTxRegister(tx, b.txRegister)
				// if err != nil {
				// 	return err
				// }
				// err = StoreTxResult(tx, b.txResult)
				// if err != nil {
				// 	return err
				// }
				// err = StoreTxSagaResult(tx, b.txSagaResult)
				// if err != nil {
				// 	return err
				// }

				deposits := postgres.NewDepositStorage(obs, tx)
				for _, deposit := range b.deposits {
					err := deposits.Insert(deposit)
					if err != nil {
						return err
					}
				}

				addresses := postgres.NewMigrationAddressStorage(cfg, obs, tx)
				for _, address := range b.addresses {
					err := addresses.Insert(address)
					if err != nil {
						return err
					}
				}

				// updates

				for _, balance := range b.balances {
					err := members.Update(balance)
					if err != nil {
						return err
					}
				}

				for _, update := range b.depositUpdates {
					err := deposits.Update(update)
					if err != nil {
						return err
					}
				}

				for _, wasting := range b.wastings {
					err := addresses.Update(wasting)
					if err != nil {
						return err
					}
				}

				// statistic
				if b.pulse == nil {
					return nil
				}

				nodes := len(b.pulse.Nodes)
				byMonth := 0
				if month(s.stat.Pulse) == month(b.pulse.Number) {
					byMonth = s.stat.LastMonthTransfers + len(b.transfers)
				} else {
					byMonth = len(b.transfers)
				}
				statistics := postgres.NewStatisticStorage(cfg, obs, tx)
				stat = &observer.Statistic{
					Pulse:              b.pulse.Number,
					Transfers:          len(b.transfers),
					TotalTransfers:     s.stat.TotalTransfers + len(b.transfers),
					TotalMembers:       s.stat.TotalMembers + len(b.members),
					Nodes:              nodes,
					MaxTransfers:       math.Max(s.stat.MaxTransfers, len(b.transfers)),
					LastMonthTransfers: byMonth,
				}
				err = statistics.Insert(stat)
				if err != nil {
					return err
				}

				platformNodes.Set(float64(nodes))
				return nil
			})
			if err != nil {
				log.Error(err)
			}
			return err
		}, cfg.DB.AttemptInterval, cfg.DB.Attempts)

		log.Info("items successfully stored")

		// restore metrics
		if s.ms.totalMigrationAddresses > 0 || s.ms.totalWasting > 0 {
			metric.Addresses.Add(float64(s.ms.totalMigrationAddresses))
			metric.Wastings.Add(float64(s.ms.totalWasting))
			s.ms.Reset()
		}

		metric.Transfers.Add(float64(len(b.transfers)))
		metric.Members.Add(float64(len(b.members)))
		metric.Deposits.Add(float64(len(b.deposits)))
		metric.Addresses.Add(float64(len(b.addresses)))

		metric.Balances.Add(float64(len(b.balances)))
		metric.Updates.Add(float64(len(b.depositUpdates)))
		metric.Wastings.Add(float64(len(b.wastings)))

		return stat
	}
}

func StoreTxRegister(tx *pg.Tx, transactions []observer.TxRegister) error {
	if len(transactions) == 0 {
		return nil
	}

	columns := []string{
		"tx_id",
		"status_registered",
		"pulse_number",
		"member_from_ref",
		"member_to_ref",
		"migration_to_ref",
		"vesting_from_ref",
		"amount",
		"fee",
	}
	var values []interface{}
	for _, t := range transactions {
		values = append(
			values,
			t.TransactionID,
			true,
			t.PulseNumber,
			t.MemberFromReference,
			t.MemberToReference,
			t.MigrationsToReference,
			t.VestingFromReference,
			t.Amount,
			t.Fee,
		)
	}
	_, err := tx.Exec(
		fmt.Sprintf( // nolint: gosec
			`
				INSERT INTO simple_transactions (%s) VALUES %s
				ON CONFLICT (tx_id) DO UPDATE SET 
					status_registered = EXCLUDED.status_registered,
					pulse_number = EXCLUDED.pulse_number,
					member_from_ref = EXCLUDED.member_from_ref,
					member_to_ref = EXCLUDED.member_to_ref,
					migration_to_ref = EXCLUDED.migration_to_ref,
					vesting_from_ref = EXCLUDED.vesting_from_ref,
					amount = EXCLUDED.amount,
					fee = EXCLUDED.fee
			`,
			strings.Join(columns, ","),
			valuesTemplate(len(columns), len(transactions)),
		),
		values...,
	)
	return err
}

func StoreTxResult(tx *pg.Tx, transactions []observer.TxResult) error {
	if len(transactions) == 0 {
		return nil
	}

	columns := []string{
		"tx_id",
		"status_sent",
	}
	var values []interface{}
	for _, t := range transactions {
		values = append(
			values,
			t.TransactionID,
			true,
		)
	}
	_, err := tx.Exec(
		fmt.Sprintf( // nolint: gosec
			`
				INSERT INTO simple_transactions (%s) VALUES %s
				ON CONFLICT (tx_id) DO UPDATE SET
					status_sent = EXCLUDED.status_sent
			`,
			strings.Join(columns, ","),
			valuesTemplate(len(columns), len(transactions)),
		),
		values...,
	)
	return err
}

func StoreTxSagaResult(tx *pg.Tx, transactions []observer.TxSagaResult) error {
	if len(transactions) == 0 {
		return nil
	}

	columns := []string{
		"tx_id",
		"status_finished",
		"finish_success",
		"finish_pulse_number",
		"finish_record_number",
	}
	var values []interface{}
	for _, t := range transactions {
		values = append(
			values,
			t.TransactionID,
			true,
			t.FinishSuccess,
			t.FinishPulseNumber,
			t.FinishRecordNumber,
		)
	}
	_, err := tx.Exec(
		fmt.Sprintf( // nolint: gosec
			`
				INSERT INTO simple_transactions (%s) VALUES %s
				ON CONFLICT (tx_id) DO UPDATE SET 
					status_finished = EXCLUDED.status_finished,
					finish_success = EXCLUDED.finish_success,
					finish_pulse_number = EXCLUDED.finish_pulse_number,
					finish_record_number = EXCLUDED.finish_record_number
			`,
			strings.Join(columns, ","),
			valuesTemplate(len(columns), len(transactions)),
		),
		values...,
	)
	return err
}

func valuesTemplate(columns, rows int) string {
	b := strings.Builder{}
	for r := 0; r < rows; r++ {
		b.WriteString("(")
		for c := 0; c < columns; c++ {
			b.WriteString("?")
			if c < columns-1 {
				b.WriteString(",")
			}
		}
		b.WriteString(")")
		if r < rows-1 {
			b.WriteString(",")
		}
	}
	return b.String()
}

func month(pn insolar.PulseNumber) int64 {
	t, err := pn.AsApproximateTime()
	if err != nil {
		return 0
	}
	rounded := time.Date(t.Year(), t.Month(), 0, 0, 0, 0, 0, t.Location())
	month := rounded.Unix()
	return month
}
