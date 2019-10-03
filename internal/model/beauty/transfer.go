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
	"context"

	"github.com/go-pg/pg/orm"
	"go.opencensus.io/stats"

	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/model"
)

type Transfer struct {
	tableName struct{} `sql:"transactions"`

	ID            uint                `sql:",pk_id"`
	TxID          []byte              `sql:",notnull"`
	Amount        string              `sql:",notnull"`
	Fee           string              `sql:",notnull"`
	TransferDate  int64               `sql:",notnull"`
	PulseNum      insolar.PulseNumber `sql:",notnull"`
	Status        string              `sql:",notnull"`
	MemberFromRef []byte              `sql:",notnull"`
	MemberToRef   []byte              `sql:",notnull"`
	WalletFromRef []byte              `sql:",notnull"`
	WalletToRef   []byte              `sql:",notnull"`
	EthHash       string              `sql:",notnull"`
}

func (t *Transfer) Dump(ctx context.Context, tx orm.DB) error {
	res, err := tx.Model(t).OnConflict("DO NOTHING").Insert(t)
	if err != nil {
		return errors.Wrapf(err, "failed to insert transfer")
	}

	if res.RowsAffected() == 0 {
		stats.Record(ctx, model.ErrorsCount.M(1))
		logrus.Errorf("Failed to insert transfer: %v", t)
	}
	return nil
}
