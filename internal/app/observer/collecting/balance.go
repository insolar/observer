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
	proxyAccount "github.com/insolar/insolar/application/builtin/proxy/account"
	"github.com/insolar/insolar/insolar/record"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
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
	if rec == nil {
		return nil
	}

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return nil
	}
	if !isAccountAmend(v.Amend) {
		return nil
	}
	amd := rec.Virtual.GetAmend()
	balance := accountBalance(rec)
	return &observer.Balance{
		PrevState:    amd.PrevState,
		AccountState: rec.ID,
		Balance:      balance,
	}
}

func isAccountAmend(amd *record.Amend) bool {
	return amd.Image.Equal(*proxyAccount.PrototypeReference)
}
