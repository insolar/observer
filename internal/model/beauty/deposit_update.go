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

type DepositUpdate struct {
	ID              []byte
	HoldReleaseDate int64
	Amount          string
	Balance         string
	PrevState       []byte
}

func (u *DepositUpdate) Dump(tx orm.DB) error {
	res, err := tx.Model(&Deposit{}).
		Where("deposit_state=?", u.PrevState).
		Set("amount=?,deposit_state=?,balance=?,hold_release_date=?", u.Amount, u.ID, u.Balance, u.HoldReleaseDate).
		Update()
	if err != nil {
		return errors.Wrapf(err, "failed to update deposit state")

	}
	if res.RowsAffected() != 1 {
		return errors.Errorf("failed to update deposit state rows_affected=%d", res.RowsAffected())
	}
	return nil
}
