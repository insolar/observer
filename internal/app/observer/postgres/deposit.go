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
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/observability"
)

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

func (s *DepositStorage) insertDeposit(deposit *models.Deposit) error {
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
			status
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
			?
		)`,
		deposit.EtheriumHash,
		deposit.Reference,
		deposit.MemberReference,
		deposit.TransferDate,
		deposit.HoldReleaseDate,
		deposit.Amount,
		deposit.Balance,
		deposit.State,
		deposit.Vesting,
		deposit.VestingStep,
		models.Created,
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

	deposit := new(models.Deposit)
	err := s.db.Model(deposit).Where("deposit_state=?", model.PrevState.Bytes()).Select()
	if err != nil {
		return errors.Wrapf(err, "failed to find deposit for update upd=%#v", model)
	}

	status := models.Created
	if model.IsConfirmed {
		status = models.Confirmed
	}

	res, err := s.db.Model(&models.Deposit{}).
		Where("deposit_ref=?", deposit.Reference).
		Set(`amount=?,deposit_state=?,balance=?,hold_release_date=?,
deposit_number = (select
				(case
					when (select max(deposit_number) from deposits where member_ref=?) isnull
						then 1
					else
						(select (max(deposit_number) + 1) from deposits where member_ref=?)
					end
				)
			), status=?`, model.Amount, model.ID.Bytes(), model.Balance, model.HoldReleaseDate, deposit.MemberReference,
			deposit.MemberReference, status).
		Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update deposit upd=%v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("upd", model).WithField("TxHash", model.TxHash).Errorf("failed to update deposit")
		return errors.New("failed to update, affected is 0")
	}
	return nil
}

func depositSchema(model *observer.Deposit) *models.Deposit {
	return &models.Deposit{
		EtheriumHash:    model.EthHash,
		Reference:       model.Ref.Bytes(),
		MemberReference: model.Member.Bytes(),
		TransferDate:    model.Timestamp,
		HoldReleaseDate: model.HoldReleaseDate,
		Amount:          model.Amount,
		Balance:         model.Balance,
		State:           model.DepositState.Bytes(),
		Vesting:         model.Vesting,
		VestingStep:     model.VestingStep,
	}
}
