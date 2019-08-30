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

package beauty

import (
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
)

type Transfer struct {
	tableName struct{} `sql:"transactions"`

	ID            uint                `sql:",pk_id"`
	TxID          string              `sql:",notnull"`
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

func (m *Transfer) Dump(tx orm.DB) error {
	if err := tx.Insert(m); err != nil {
		return errors.Wrapf(err, "failed to insert transfer")
	}
	return nil
}
