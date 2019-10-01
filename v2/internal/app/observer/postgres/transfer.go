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
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/v2/internal/app/observer"
	"github.com/insolar/observer/v2/observability"
)

type TransferSchema struct {
	tableName struct{} `sql:"transactions"`

	ID            uint                `sql:",pk_id"`
	TxID          string              `sql:",unique"`
	Amount        string              `sql:",notnull"`
	Fee           string              `sql:",notnull"`
	TransferDate  int64               `sql:",notnull"`
	PulseNum      insolar.PulseNumber `sql:",notnull"`
	Status        string              `sql:",notnull"`
	MemberFromRef string              `sql:",notnull"`
	MemberToRef   string              `sql:",notnull"`
	WalletFromRef string              `sql:",notnull"`
	WalletToRef   string              `sql:",notnull"`
	EthHash       string              `sql:",notnull"`
}

type TransferStorage struct {
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewTransferStorage(obs *observability.Observability, db orm.DB) *TransferStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_transfer_storage_error_counter",
		Help: "",
	})
	return &TransferStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *TransferStorage) Insert(model *observer.DepositTransfer) error {
	if model == nil {
		s.log.Warnf("trying to insert nil pulse model")
		return nil
	}
	row := transferSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert()

	if err != nil {
		return errors.Wrapf(err, "failed to insert transfer %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("transfer_row", row).Errorf("failed to insert transfer")
	}
	return nil
}

func transferSchema(model *observer.DepositTransfer) *TransferSchema {
	return &TransferSchema{
		TxID:          model.TxID.String(),
		Amount:        model.Amount,
		Fee:           model.Fee,
		TransferDate:  model.Timestamp,
		PulseNum:      model.Pulse,
		Status:        "SUCCESS",
		MemberFromRef: model.From.String(),
		MemberToRef:   model.To.String(),

		EthHash: model.EthHash,
	}
}
