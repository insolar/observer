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
	"fmt"
	"strings"

	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
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

func (s *DepositStorage) Insert(model observer.Deposit) error {
	row := depositSchema(model)

	log := s.log.WithField("deposit", model)

	var (
		fields = []string{"eth_hash", "deposit_ref", "member_ref", "transfer_date", "hold_release_date", "amount",
			"balance", "deposit_state", "vesting", "vesting_step", "status"}
		values = []interface{}{
			row.EtheriumHash,
			row.Reference,
			row.MemberReference,
			row.TransferDate,
			row.HoldReleaseDate,
			row.Amount,
			row.Balance,
			row.State,
			row.Vesting,
			row.VestingStep,
		}
	)

	if model.IsConfirmed {
		fields = append(fields, `deposit_number = (select
				(case
					when (select max(deposit_number) from deposits where member_ref=?) isnull
						then 1
					else
						(select (max(deposit_number) + 1) from deposits where member_ref=?)
					end
				)
			)`)
		values = append(values, models.DepositStatusConfirmed, row.MemberReference, row.MemberReference)
	} else {
		values = append(values, models.DepositStatusCreated)
	}

	res, err := s.db.Query(model, fmt.Sprintf( // nolint: gosec
		`insert into deposits (
			%s
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
		)`, strings.Join(fields, ",")),
		values...,
	)

	if err != nil {
		return errors.Wrapf(err, "failed to insert deposit %v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		log.WithField("deposit_row", model).Errorf("failed to insert deposit")
		return errors.New("failed to insert, affected is 0")
	}

	log.Debug("insert")

	return nil
}

func (s *DepositStorage) Update(model observer.DepositUpdate) error {
	deposit := new(models.Deposit)
	err := s.db.Model(deposit).Where("deposit_state=?", model.PrevState.Bytes()).Select()
	if err != nil {
		return errors.Wrapf(err, "failed to find deposit for update upd=%s", model.PrevState.String())
	}

	log := s.log.WithField("deposit", insolar.NewReferenceFromBytes(deposit.Reference).String()).WithField("upd", model)

	query := s.db.Model(&models.Deposit{}).
		Where("deposit_ref=?", deposit.Reference).
		Set(`amount=?,deposit_state=?,balance=?,hold_release_date=?`, model.Amount, model.ID.Bytes(), model.Balance, model.HoldReleaseDate)

	if model.IsConfirmed {
		if deposit.InnerStatus == models.DepositStatusCreated {
			query.Set("status=?", models.DepositStatusConfirmed)
		}
		if deposit.DepositNumber == nil {
			query.Set(`deposit_number = (select
				(case
					when (select max(deposit_number) from deposits where member_ref=?) isnull
						then 1
					else
						(select (max(deposit_number) + 1) from deposits where member_ref=?)
					end
				)
			)`, deposit.MemberReference, deposit.MemberReference)
		}
	}

	res, err := query.Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update deposit upd=%v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		log.Errorf("failed to update deposit")
		return errors.New("failed to update, affected is 0")
	}

	log.Debug("updated")

	return nil
}

func (s *DepositStorage) GetDeposit(ref []byte) (*models.Deposit, error) {
	deposit := new(models.Deposit)
	err := s.db.Model(deposit).Where("deposit_ref=?", ref).Select()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find deposit with ref = %q", ref)
	}
	return deposit, nil
}

func depositSchema(model observer.Deposit) models.Deposit {
	return models.Deposit{
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
