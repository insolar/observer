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
	DepositRef      []byte `sql:",notnull"`
	MemberRef       []byte `sql:",notnull"`
	TransferDate    int64  `sql:",notnull"`
	HoldReleaseDate int64  `sql:",notnull"`
	Amount          string `sql:",notnull"`
	Balance         string `sql:",notnull"`
	DepositState    []byte `sql:",notnull"`
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
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert()

	if err != nil {
		return errors.Wrapf(err, "failed to insert deposit %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("deposit_row", row).Errorf("failed to insert deposit")
		// TODO: uncomment it. It's just a temporary change. Genesis deposits are conflicted because of the same eth hash
		// return errors.New("failed to insert, affected is 0")
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
	}
}
