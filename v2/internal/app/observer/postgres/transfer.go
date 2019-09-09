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

	"github.com/insolar/observer/v2/internal/app/observer"
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
	db orm.DB
}

func NewTransferStorage(db orm.DB) *TransferStorage {
	return &TransferStorage{db: db}
}

func (s *TransferStorage) Insert(model *observer.DepositTransfer) error {
	row := transferSchema(model)
	if row != nil {
		return s.db.Insert(row)
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
