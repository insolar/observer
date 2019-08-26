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
	"github.com/insolar/insolar/pulse"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/panic"
	"github.com/insolar/observer/internal/replication"
)

type DepositKeeper struct {
	cache []*beauty.DepositUpdate
}

func NewKeeper() *DepositKeeper {
	return &DepositKeeper{}
}

func (k *DepositKeeper) Process(rec *record.Material) {
	defer panic.Log("deposit_keeper")

	if isDepositAmend(rec) {
		amd := rec.Virtual.GetAmend()
		d := depositState(amd)
		releaseTimestamp := int64(0)
		if holdReleadDate, err := pulse.Number(d.PulseDepositUnHold).AsApproximateTime(); err == nil {
			releaseTimestamp = holdReleadDate.Unix()
		}
		k.cache = append(k.cache, &beauty.DepositUpdate{
			ID:              rec.ID.String(),
			HoldReleaseDate: releaseTimestamp,
			Amount:          d.Amount,
			Balance:         d.Balance,
			PrevState:       amd.PrevState.String(),
		})
	}
}

func (k *DepositKeeper) Dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	log.Infof("dump deposit updates")

	deferred := []*beauty.DepositUpdate{}
	for _, upd := range k.cache {
		if err := upd.Dump(tx); err != nil {
			deferred = append(deferred, upd)
		}
	}

	for _, upd := range deferred {
		log.Infof("Deposit update %v", upd)
	}

	pub.Subscribe(func() {
		k.cache = deferred
	})
	return nil
}

func isDepositAmend(rec *record.Material) bool {
	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return false
	}

	return v.Amend.Image.Equal(*depositProxy.PrototypeReference)
}

func depositState(amd *record.Amend) *deposit.Deposit {
	d := deposit.Deposit{}
	err := insolar.Deserialize(amd.Memory, &d)
	if err != nil {
		log.Error(errors.New("failed to deserialize deposit contract state"))
	}
	return &d
}
