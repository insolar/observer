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
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type TransactionSchema struct {
	tableName             struct{} `sql:"transactions"`
	Ref                   []byte   `sql:",pk"`
	ExternalTransactionId string
	Timestamp             int64
	Direction             string
	GroupRef              []byte
	UserRef               []byte
	Amount                string
	UID                   string
	Status                string
}

type TransactionStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewTransactionStorage(obs *observability.Observability, db orm.DB) *TransactionStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_transaction_storage_error_counter",
		Help: "",
	})
	return &TransactionStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *TransactionStorage) Insert(model *observer.Transaction) error {
	if model == nil {
		s.log.Warnf("trying to insert nil transaction model")
		return nil
	}
	row := transactionSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert(row)

	if err != nil {
		return errors.Wrapf(err, "failed to insert transaction %v, %v", row, err.Error())
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("group_row", row).
			Errorf("failed to insert group")
	}
	return nil
}

func transactionSchema(model *observer.Transaction) *TransactionSchema {
	return &TransactionSchema{
		Ref:                   model.Reference.Bytes(),
		ExternalTransactionId: model.ExtTxId,
		Timestamp:             model.Timestamp,
		Direction:             model.TxDirection,
		GroupRef:              model.GroupRef.Bytes(),
		UserRef:               model.MemberRef.Bytes(),
		Amount:                model.Amount,
		UID:                   model.UID,
		Status:                model.Status,
	}
}
