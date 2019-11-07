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
	"fmt"
	"strings"
	"time"

	"github.com/go-pg/pg/orm"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
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
					if rec == nil {
						continue
					}
					obsRec := observer.Record(rec.Record)
					err := records.Insert(&obsRec)
					if err != nil {
						return err
					}
				}

				requests := postgres.NewRequestStorage(obs, tx)
				for _, req := range b.requests {
					if req == nil {
						continue
					}
					err := requests.Insert(req)
					if err != nil {
						return err
					}
				}

				results := postgres.NewResultStorage(obs, tx)
				for _, res := range b.results {
					if res == nil {
						continue
					}
					err := results.Insert(res)
					if err != nil {
						return err
					}
				}

				objects := postgres.NewObjectStorage(obs, tx)
				for _, act := range b.activates {
					if act == nil {
						continue
					}
					err := objects.Insert(act)
					if err != nil {
						return err
					}
				}

				for _, amd := range b.amends {
					if amd == nil {
						continue
					}
					err := objects.Insert(amd)
					if err != nil {
						return err
					}
				}

				for _, deact := range b.deactivates {
					if deact == nil {
						continue
					}
					err := objects.Insert(deact)
					if err != nil {
						return err
					}
				}

				// new entities

				members := postgres.NewMemberStorage(obs, tx)
				for _, member := range b.members {
					if member == nil {
						continue
					}
					err := members.Insert(member)
					if err != nil {
						return err
					}
				}

				err = StoreTxRegister(tx, b.txRegister)
				if err != nil {
					return err
				}
				err = StoreTxResult(tx, b.txResult)
				if err != nil {
					return err
				}
				err = StoreTxSagaResult(tx, b.txSagaResult)
				if err != nil {
					return err
				}

				deposits := postgres.NewDepositStorage(obs, tx)
				for _, deposit := range b.deposits {
					if deposit == nil {
						continue
					}
					err := deposits.Insert(deposit)
					if err != nil {
						return err
					}
				}

				addresses := postgres.NewMigrationAddressStorage(cfg, obs, tx)
				for _, address := range b.addresses {
					if address == nil {
						continue
					}
					err := addresses.Insert(address)
					if err != nil {
						return err
					}
				}

				// updates

				for _, balance := range b.balances {
					if balance == nil {
						continue
					}
					err := members.Update(balance)
					if err != nil {
						return err
					}
				}

				for _, update := range b.depositUpdates {
					if update == nil {
						continue
					}
					err := deposits.Update(update)
					if err != nil {
						return err
					}
				}

				for _, wasting := range b.wastings {
					if wasting == nil {
						continue
					}
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
					byMonth = s.stat.LastMonthTransfers + len(b.txSagaResult)
				} else {
					byMonth = len(b.txSagaResult)
				}
				statistics := postgres.NewStatisticStorage(cfg, obs, tx)
				stat = &observer.Statistic{
					Pulse:              b.pulse.Number,
					Transfers:          len(b.txSagaResult),
					TotalTransfers:     s.stat.TotalTransfers + len(b.txSagaResult),
					TotalMembers:       s.stat.TotalMembers + len(b.members),
					Nodes:              nodes,
					MaxTransfers:       math.Max(s.stat.MaxTransfers, len(b.txSagaResult)),
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

		metric.Transfers.Add(float64(len(b.txSagaResult)))
		metric.Members.Add(float64(len(b.members)))
		metric.Deposits.Add(float64(len(b.deposits)))
		metric.Addresses.Add(float64(len(b.addresses)))

		metric.Balances.Add(float64(len(b.balances)))
		metric.Updates.Add(float64(len(b.depositUpdates)))
		metric.Wastings.Add(float64(len(b.wastings)))

		return stat
	}
}

type Execer interface {
	Exec(query interface{}, params ...interface{}) (pg.Result, error)
}

func StoreTxRegister(tx Execer, transactions []observer.TxRegister) error {
	if len(transactions) == 0 {
		return nil
	}

	columns := []string{
		"tx_id",
		"status_registered",
		"type",
		"pulse_record",
		"member_from_ref",
		"member_to_ref",
		"deposit_to_ref",
		"deposit_from_ref",
		"amount",
	}
	var values []interface{}
	for _, t := range transactions {
		values = append(
			values,
			t.TransactionID,
			true,
			t.Type,
			pg.Array([2]int64{t.PulseNumber, t.RecordNumber}),
			t.MemberFromReference,
			t.MemberToReference,
			t.DepositToReference,
			t.DepositFromReference,
			t.Amount,
		)
	}
	_, err := tx.Exec(
		fmt.Sprintf( // nolint: gosec
			`
				INSERT INTO simple_transactions (%s) VALUES %s
				ON CONFLICT (tx_id) DO UPDATE SET 
					status_registered = EXCLUDED.status_registered,
					type = EXCLUDED.type,
					pulse_record = EXCLUDED.pulse_record,
					member_from_ref = EXCLUDED.member_from_ref,
					member_to_ref = EXCLUDED.member_to_ref,
					deposit_to_ref = EXCLUDED.deposit_to_ref,
					deposit_from_ref = EXCLUDED.deposit_from_ref,
					amount = EXCLUDED.amount
			`,
			strings.Join(columns, ","),
			valuesTemplate(len(columns), len(transactions)),
		),
		values...,
	)
	return err
}

func StoreTxResult(tx Execer, transactions []observer.TxResult) error {
	if len(transactions) == 0 {
		return nil
	}

	columns := []string{
		"tx_id",
		"status_sent",
		"fee",
	}
	var values []interface{}
	for _, t := range transactions {
		values = append(
			values,
			t.TransactionID,
			true,
			t.Fee,
		)
	}
	_, err := tx.Exec(
		fmt.Sprintf( // nolint: gosec
			`
				INSERT INTO simple_transactions (%s) VALUES %s
				ON CONFLICT (tx_id) DO UPDATE SET
					status_sent = EXCLUDED.status_sent,
					fee = EXCLUDED.fee
			`,
			strings.Join(columns, ","),
			valuesTemplate(len(columns), len(transactions)),
		),
		values...,
	)
	return err
}

func StoreTxSagaResult(tx Execer, transactions []observer.TxSagaResult) error {
	if len(transactions) == 0 {
		return nil
	}

	columns := []string{
		"tx_id",
		"status_finished",
		"finish_success",
		"finish_pulse_record",
	}
	var values []interface{}
	for _, t := range transactions {
		values = append(
			values,
			t.TransactionID,
			true,
			t.FinishSuccess,
			pg.Array([2]int64{t.FinishPulseNumber, t.FinishRecordNumber}),
		)
	}
	_, err := tx.Exec(
		fmt.Sprintf( // nolint: gosec
			`
				INSERT INTO simple_transactions (%s) VALUES %s
				ON CONFLICT (tx_id) DO UPDATE SET 
					status_finished = EXCLUDED.status_finished,
					finish_success = EXCLUDED.finish_success,
					finish_pulse_record = EXCLUDED.finish_pulse_record
			`,
			strings.Join(columns, ","),
			valuesTemplate(len(columns), len(transactions)),
		),
		values...,
	)
	return err
}

type Querier interface {
	QueryOne(model, query interface{}, params ...interface{}) (pg.Result, error)
	QueryOneContext(c context.Context, model, query interface{}, params ...interface{}) (pg.Result, error)
	QueryContext(c context.Context, model, query interface{}, params ...interface{}) (pg.Result, error)
}

var (
	ErrTxNotFound        = errors.New("tx not found")
	ErrReferenceNotFound = errors.New("Reference not found")
)

func GetMemberBalance(ctx context.Context, db Querier, reference []byte) (*models.Member, error) {
	return getMember(ctx, db, reference, []string{"balance"})
}

func GetMember(ctx context.Context, db Querier, reference []byte) (*models.Member, error) {
	return getMember(ctx, db, reference, models.Member{}.Fields())
}

func getMember(ctx context.Context, db Querier, reference []byte, fields []string) (*models.Member, error) {
	member := &models.Member{}
	_, err := db.QueryOneContext(ctx, member,
		fmt.Sprintf( // nolint: gosec
			`select %s from members where member_ref = ?0`, strings.Join(fields, ",")),
		reference)
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, ErrReferenceNotFound
		}
		return nil, errors.Wrap(err, "failed to fetch member")
	}
	return member, nil
}

func GetDeposits(ctx context.Context, db Querier, memberReference []byte) (*[]models.Deposit, error) {
	deposits := &[]models.Deposit{}
	_, err := db.QueryContext(ctx, deposits,
		fmt.Sprintf( // nolint: gosec
			`select %s from deposits where member_ref = ?0`, strings.Join(models.Deposit{}.Fields(), ",")),
		memberReference)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch deposit")
	}
	return deposits, nil
}

func GetTx(ctx context.Context, db Querier, txID []byte) (*models.Transaction, error) {
	tx := &models.Transaction{}
	_, err := db.QueryOneContext(ctx, tx,
		fmt.Sprintf( // nolint: gosec
			`select %s from simple_transactions where tx_id = ?0`, strings.Join(tx.Fields(), ",")),
		txID)
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, ErrTxNotFound
		}
		return nil, errors.Wrap(err, "failed to fetch tx")
	}
	return tx, nil
}

func FilterByStatus(query *orm.Query, status string) (*orm.Query, error) {
	switch status {
	case "registered":
		query = query.Where("status_registered = true")
	case "sent":
		query = query.Where("status_registered = true and status_sent = true")
	case "received":
		query = query.Where("status_registered = true and status_finished = true and finish_success = true")
	case "failed":
		query = query.Where("status_registered = true and status_finished = true and finish_success = false")
	default:
		return query, errors.New("Query parameter 'status' should be 'registered', 'sent', 'received' or 'failed'.") // nolint
	}
	return query, nil
}

func FilterByType(query *orm.Query, t string) (*orm.Query, error) {
	if t != "transfer" && t != "migration" && t != "release" {
		return query, errors.New("Query parameter 'type' should be 'transfer', 'migration' or 'release'.") // nolint
	}
	query = query.Where("type = ?", t)
	return query, nil
}

func FilterByMemberReferenceAndDirection(query *orm.Query, ref *insolar.Reference, d *string) (*orm.Query, error) {
	direction := "all"
	if d != nil {
		direction = *d
	}
	switch direction {
	case "incoming":
		query = query.Where("member_to_ref = ?", ref.Bytes())
	case "outgoing":
		query = query.Where("member_from_ref = ?", ref.Bytes())
	case "all":
		query = query.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("member_from_ref = ?", ref.Bytes()).
				WhereOr("member_to_ref = ?", ref.Bytes())
			return q, nil
		})
	default:
		return query, errors.New("Query parameter 'direction' should be 'incoming', 'outgoing' or 'all'.") // nolint
	}
	return query, nil
}

func FilterByValue(query *orm.Query, value string) (*orm.Query, error) {
	pulseNumber, err := insolar.NewPulseNumberFromStr(value)
	if err == nil {
		query = query.Where("pulse_record[1] = ?", pulseNumber)
	} else {
		ref, err := insolar.NewReferenceFromString(value)
		if err != nil {
			return query, errors.New("Query parameter 'value' should be txID, fromMemberReference, toMemberReference or pulseNumber.") // nolint
		}
		query = query.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("tx_id = ?", ref.Bytes()).
				WhereOr("member_from_ref = ?", ref.Bytes()).
				WhereOr("member_to_ref = ?", ref.Bytes())
			return q, nil
		})
	}

	return query, nil
}

func indexTypeToColumnName(indexType models.TxIndexType) string {
	var result string
	switch indexType {
	case models.TxIndexTypeFinishPulseRecord:
		result = "finish_pulse_record"
	default: // models.TxIndexTypePulseRecord
		result = "pulse_record"
	}
	return result
}

func OrderByIndex(query *orm.Query, ord *string, pulse int64, record int64, whereCondition bool, indexType models.TxIndexType) (*orm.Query, error) {
	order := "reverse"
	if ord != nil {
		order = *ord
	}

	columnName := indexTypeToColumnName(indexType)
	switch order {
	case "reverse":
		if whereCondition {
			query = query.Where(columnName+" < array[?0,?1]::bigint[]", pulse, record)
		}
		query = query.Order(columnName + " DESC")
	case "chronological":
		if whereCondition {
			query = query.Where(columnName+" > array[?,?]::bigint[]", pulse, record)
		}
		query = query.Order(columnName + " ASC")
	default:
		return query, errors.New("Query parameter 'order' should be 'reverse' or 'chronological'.") // nolint
	}
	return query, nil
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
