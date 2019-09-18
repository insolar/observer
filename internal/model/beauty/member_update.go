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
	"github.com/pkg/errors"
)

type BalanceUpdate struct {
	ID        []byte
	PrevState []byte
	Balance   string
}

func (u *BalanceUpdate) Dump(tx orm.DB) error {
	res, err := tx.Model(&Member{}).
		Where("account_state=?", u.PrevState).
		Set("balance=?,account_state=?", u.Balance, u.ID).
		Update()
	if err != nil {
		return errors.Wrapf(err, "failed to update member balance")
	}
	if res.RowsAffected() != 1 {
		return errors.Errorf("failed to update member balance rows_affected=%d", res.RowsAffected())
	}
	return nil
}
