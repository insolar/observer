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

package postgres

import (
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

type DepositSchema struct {
	tableName struct{} `sql:"deposits"` //nolint: unused,structcheck

	EthHash         string `sql:",pk"`
	MemberRef       []byte `sql:",pk"`
	DepositRef      []byte `sql:",notnull"`
	TransferDate    int64  `sql:",notnull"`
	HoldReleaseDate int64  `sql:",notnull"`
	Amount          string `sql:",notnull"`
	Balance         string `sql:",notnull"`
	DepositState    []byte `sql:",notnull"`
	DepositNumber   int64  `sql:",notnull"`
	Vesting         int64
	VestingStep     int64
}

type DepositStorage struct {
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewDepositStorage(obs *observability.Observability, db orm.DB) *DepositStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_deposit_storage_error_counter",
		Help: "",
	})
	return &DepositStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *DepositStorage) Insert(model *observer.Deposit) error {
	if model == nil {
		s.log.Warnf("trying to insert nil deposit model")
		return nil
	}
	row := depositSchema(model)

	return s.insertDeposit(row)
}

func (s *DepositStorage) insertDeposit(deposit *DepositSchema) error {
	res, err := s.db.Query(deposit, `
		insert into deposits (
			eth_hash,
			deposit_ref,
			member_ref,
			transfer_date,
			hold_release_date,
			amount,
			balance,
			deposit_state,
			vesting,
			vesting_step,
			deposit_number
		) values (
			?,
			?,
			?,
			?,
			?,
			?,
			?,
			?,
			?,
			?,
			(select
				(case
					when (select max(deposit_number) from deposits where member_ref=?) isnull
						then 1
					else
						(select (max(deposit_number) + 1) from deposits where member_ref=?)
					end
				)
			)
		)`,
		deposit.EthHash,
		deposit.DepositRef,
		deposit.MemberRef,
		deposit.TransferDate,
		deposit.HoldReleaseDate,
		deposit.Amount,
		deposit.Balance,
		deposit.DepositState,
		deposit.Vesting,
		deposit.VestingStep,
		deposit.MemberRef,
		deposit.MemberRef,
	)

	if err != nil {
		return errors.Wrapf(err, "failed to insert deposit %v", deposit)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("deposit_row", deposit).Errorf("failed to insert deposit")
		return errors.New("failed to insert, affected is 0")
	}

	return nil
}

func (s *DepositStorage) Update(model *observer.DepositUpdate) error {
	if model == nil {
		s.log.Warnf("trying to apply nil deposit update model")
		return nil
	}

	res, err := s.db.Model(&DepositSchema{}).
		Where("deposit_state=?", model.PrevState.Bytes()).
		Set("amount=?,deposit_state=?,balance=?,hold_release_date=?", model.Amount, model.ID.Bytes(), model.Balance, model.HoldReleaseDate).
		Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update deposit upd=%v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("upd", model).Errorf("failed to update deposit")
		// TODO: uncomment it. It's just a temporary change. Genesis deposits are conflicted because of the same eth hash
		// return errors.New("failed to update, affected is 0")
	}
	return nil
}

func depositSchema(model *observer.Deposit) *DepositSchema {
	return &DepositSchema{
		EthHash:         model.EthHash,
		DepositRef:      model.Ref.Bytes(),
		MemberRef:       model.Member.Bytes(),
		TransferDate:    model.Timestamp,
		HoldReleaseDate: model.HoldReleaseDate,
		Amount:          model.Amount,
		Balance:         model.Balance,
		DepositState:    model.DepositState.Bytes(),
		Vesting:         model.Vesting,
		VestingStep:     model.VestingStep,
	}
}
