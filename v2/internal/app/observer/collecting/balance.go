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

package collecting

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	proxyAccount "github.com/insolar/insolar/logicrunner/builtin/proxy/account"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/v2/internal/app/observer"
	"github.com/insolar/observer/v2/internal/pkg/panic"
)

type BalanceCollector struct {
	log *logrus.Logger
}

func NewBalanceCollector(log *logrus.Logger) *BalanceCollector {
	return &BalanceCollector{
		log: log,
	}
}

func (c *BalanceCollector) Collect(rec *observer.Record) *observer.Balance {
	defer panic.Log("balance_update_collector")

	if rec == nil {
		return nil
	}
	log := c.log

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		log.Infof("is not amend")
		return nil
	}
	if !isAccountAmend(v.Amend) {
		log.Infof("is not account amend")
		return nil
	}
	amd := rec.Virtual.GetAmend()
	balance := accountBalance(rec)
	if amd.PrevState.Pulse() == insolar.GenesisPulse.PulseNumber { // internal account
		return nil
	}
	// if amd.PrevState.Pulse() == insolar.GenesisPulse.PulseNumber {
	// 	randomRef := gen.Reference()
	// 	u.technicalAccounts = append(u.technicalAccounts, &beauty.Member{
	// 		MemberRef:    randomRef.String(),
	// 		Balance:      balance,
	// 		AccountState: rec.ID.String(),
	// 		Status:       "INTERNAL",
	// 	})
	// 	return
	// }
	return &observer.Balance{
		PrevState:    amd.PrevState,
		AccountState: rec.ID,
		Balance:      balance,
	}
}

func isAccountAmend(amd *record.Amend) bool {
	return amd.Image.Equal(*proxyAccount.PrototypeReference)
}
