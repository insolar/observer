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

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

type TransferSchema struct {
	tableName struct{} `sql:"transactions"` //nolint: unused,structcheck

	ID            uint                `sql:",pk_id"`
	TxID          []byte              `sql:",unique"`
	Amount        string              `sql:",notnull"`
	Fee           string              `sql:",notnull"`
	TransferDate  int64               `sql:",notnull"`
	PulseNum      insolar.PulseNumber `sql:",notnull"`
	Status        string              `sql:",notnull"`
	MemberFromRef []byte              `sql:",notnull"`
	MemberToRef   []byte              `sql:",notnull"`
	EthHash       string              `sql:",notnull"`

	// TODO: hide them in `Details` field
	TransferRequestMember  []byte
	TransferRequestWallet  []byte
	TransferRequestAccount []byte
	AcceptRequestMember    []byte
	AcceptRequestWallet    []byte
	AcceptRequestAccount   []byte
	CalcFeeRequest         []byte
	FeeMemberRequest       []byte
	CostCenterRef          []byte
	FeeMemberRef           []byte
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

func (s *TransferStorage) Insert(model *observer.Transfer) error {
	if model == nil {
		s.log.Warnf("trying to insert nil transfer model")
		return nil
	}
	row := s.transferSchema(model)
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

func (s *TransferStorage) transferSchema(model *observer.Transfer) *TransferSchema {
	timestamp, err := model.TxID.Pulse().AsApproximateTime()
	if err != nil {
		s.log.Error(errors.Wrapf(err, "failed to calc approximate time from pulse"))
	}
	var (
		from = []byte{}
		to   = []byte{}
	)
	if model.From != nil {
		from = model.From.Bytes()
	}
	if model.To != nil {
		to = model.To.Bytes()
	}
	return &TransferSchema{
		TxID:          model.TxID.Bytes(),
		Amount:        model.Amount,
		Fee:           model.Fee,
		TransferDate:  timestamp.Unix(),
		PulseNum:      model.TxID.Pulse(),
		Status:        model.Status.String(),
		MemberFromRef: from,
		MemberToRef:   to,

		EthHash: model.EthHash,
	}
}
