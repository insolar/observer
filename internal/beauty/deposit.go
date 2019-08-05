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
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/deposit"
	depositProxy "github.com/insolar/insolar/logicrunner/builtin/proxy/deposit"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Deposit struct {
	tableName struct{} `sql:"deposits"`

	EthHash         string `sql:",pk"`
	DepositRef      string `sql:",notnull"`
	MemberRef       string `sql:",notnull"`
	TransferDate    int64  `sql:",notnull"`
	HoldReleaseDate int64  `sql:",notnull"`
	Amount          string `sql:",notnull"`
	Withdrawn       string `sql:",notnull"`
	DepositState    string `sql:",notnull"`
	Status          string `sql:",notnull"`
}

func (b *Beautifier) processDepositActivate(pn insolar.PulseNumber, id insolar.ID, act *record.Activate) {
	deposit := initialDepositState(act)
	b.deposits[id] = &Deposit{
		EthHash:         deposit.TxHash,
		DepositRef:      "",
		MemberRef:       "",
		TransferDate:    int64(deposit.PulseDepositCreate),
		HoldReleaseDate: int64(deposit.PulseDepositUnHold),
		Amount:          deposit.Amount,
		Withdrawn:       "0",
		DepositState:    id.String(),
		Status:          "MIGRATION",
	}
}

func initialDepositState(act *record.Activate) *deposit.Deposit {
	d := deposit.Deposit{}
	err := insolar.Deserialize(act.Memory, &d)
	if err != nil {
		log.Error(errors.New("failed to deserialize deposit contract state"))
	}
	return &d
}

func (b *Beautifier) processDepositAmend(id insolar.ID, amd *record.Amend) {
	deposit := depositState(amd)
	b.depositUpdates[id] = DepositUpdate{
		id:        id,
		amount:    deposit.Amount,
		withdrawn: "0",
		status:    "MIGRATION",
		prevState: amd.PrevState.String(),
	}
}

func depositState(amd *record.Amend) *deposit.Deposit {
	d := deposit.Deposit{}
	err := insolar.Deserialize(amd.Memory, &d)
	if err != nil {
		log.Error(errors.New("failed to deserialize deposit contract state"))
	}
	return &d
}

func isDepositActivate(act *record.Activate) bool {
	return act.Image.Equal(*depositProxy.PrototypeReference)
}

func isDepositAmend(amd *record.Amend) bool {
	return amd.Image.Equal(*depositProxy.PrototypeReference)
}

func storeDeposit(tx *pg.Tx, deposit *Deposit) error {
	_, err := tx.Model(deposit).OnConflict("(eth_hash) DO UPDATE").Insert()
	if err != nil {
		return err
	}
	return nil
}

func updateDeposit(tx *pg.Tx, id insolar.ID, amount, withdrawn, status, prevState string) error {
	res, err := tx.Model(&Deposit{}).
		Set("amount=?,wallet_state=?,withdrawn=?", amount, id.String(), withdrawn).
		Where("deposit_state=?", prevState).
		Update()
	if err != nil {
		return errors.Wrapf(err, "failed to update deposit state by amend record")

	}
	if res.RowsAffected() != 1 {
		return errors.Errorf("failed to update deposit state by amend record res=%v", res)
	}
	return nil
}
