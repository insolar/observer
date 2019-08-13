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
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
)

type MigrationAddress struct {
	tableName struct{} `sql:"migration_addresses"`

	Addr      string `sql:",pk"`
	Timestamp int64  `sql:",notnull"`
	Wasted    bool
}

func (a *MigrationAddress) Dump(tx *pg.Tx) error {
	if _, err := tx.Model(a).
		Where("addr=?", a.Addr).
		OnConflict("DO NOTHING").
		SelectOrInsert(); err != nil {
		return errors.Wrapf(err, "failed to insert migration address")
	}
	return nil
}
