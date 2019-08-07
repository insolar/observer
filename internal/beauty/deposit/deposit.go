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

package deposit

import (
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/deposit"
	depositProxy "github.com/insolar/insolar/logicrunner/builtin/proxy/deposit"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/replication"

	log "github.com/sirupsen/logrus"
)

type Composer struct {
	cache []*beauty.Deposit
}

func NewComposer() *Composer {
	return &Composer{}
}

func (c *Composer) Process(rec *record.Material) {
	v, ok := rec.Virtual.Union.(*record.Virtual_Activate)
	if !ok {
		return
	}

	if isDepositActivate(v.Activate) {
		c.processDepositActivate(rec)
	}
}

func (c *Composer) processDepositActivate(rec *record.Material) {
	id := rec.ID
	act := rec.Virtual.GetActivate()
	deposit := initialDepositState(act)
	c.cache = append(c.cache, &beauty.Deposit{
		EthHash:         deposit.TxHash,
		DepositRef:      "",
		MemberRef:       "",
		TransferDate:    int64(deposit.PulseDepositCreate),
		HoldReleaseDate: int64(deposit.PulseDepositUnHold),
		Amount:          deposit.Amount,
		Withdrawn:       "0",
		DepositState:    id.String(),
		Status:          "MIGRATION",
	})
}

func (c *Composer) Dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	for _, dep := range c.cache {
		if err := dep.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump deposits")
		}
	}

	pub.Subscribe(func() {
		c.cache = []*beauty.Deposit{}
	})
	return nil
}

func initialDepositState(act *record.Activate) *deposit.Deposit {
	d := deposit.Deposit{}
	err := insolar.Deserialize(act.Memory, &d)
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
