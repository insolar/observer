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

package member

import (
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/beauty/member/wallet/account"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/panic"
	"github.com/insolar/observer/internal/replicator"
)

type BalanceUpdater struct {
	cache             []*beauty.BalanceUpdate
	technicalAccounts []*beauty.Member
}

func NewBalanceUpdater() *BalanceUpdater {
	return &BalanceUpdater{}
}

func (u *BalanceUpdater) Process(rec *record.Material) {
	defer panic.Log("member_balance_updater")

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return
	}
	if !account.IsAccountAmend(v.Amend) {
		return
	}
	u.processAccountAmend(rec.ID, rec)
}

func (u *BalanceUpdater) processAccountAmend(id insolar.ID, rec *record.Material) {
	amd := rec.Virtual.GetAmend()
	balance := account.AccountBalance(rec)
	if amd.PrevState.Pulse() == insolar.GenesisPulse.PulseNumber {
		randomRef := gen.Reference()
		u.technicalAccounts = append(u.technicalAccounts, &beauty.Member{
			MemberRef:    randomRef.String(),
			Balance:      balance,
			AccountState: id.String(),
			Status:       "INTERNAL",
		})
		return
	}
	u.cache = append(u.cache, &beauty.BalanceUpdate{
		ID:        id.String(),
		PrevState: amd.PrevState.String(),
		Balance:   balance,
	})
}

func (u *BalanceUpdater) Dump(tx orm.DB, pub replicator.OnDumpSuccess) error {
	log.Infof("dump member balances")

	for _, acc := range u.technicalAccounts {
		if err := acc.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump internal member")
		}
	}

	var deferred []*beauty.BalanceUpdate
	for _, upd := range u.cache {
		if err := upd.Dump(tx); err != nil {
			deferred = append(deferred, upd)
		}
	}

	for _, upd := range deferred {
		log.Infof("Wallet update %v", upd)
	}
	pub.Subscribe(func() {
		u.cache = deferred
		u.technicalAccounts = []*beauty.Member{}
	})
	return nil
}
