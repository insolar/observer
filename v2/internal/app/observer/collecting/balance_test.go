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
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/account"
	proxyAccount "github.com/insolar/insolar/logicrunner/builtin/proxy/account"
	"github.com/insolar/insolar/pulse"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/internal/app/observer"
)

func makeAccountAmend(pn insolar.PulseNumber, balance string, prev insolar.ID) *observer.Record {
	acc := &account.Account{
		Balance: balance,
	}
	memory, err := insolar.Serialize(acc)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_Amend{
				Amend: &record.Amend{
					Request:   *insolar.NewReference(gen.IDWithPulse(pn)),
					Memory:    memory,
					Image:     *proxyAccount.PrototypeReference,
					PrevState: prev,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeBalanceUpdate() (*observer.Balance, *observer.Record) {
	pn := insolar.PulseNumber(pulse.OfNow())
	balance := "43"
	prev := gen.IDWithPulse(pn)
	rec := makeAccountAmend(pn, balance, prev)
	update := &observer.Balance{
		PrevState:    prev,
		AccountState: rec.ID,
		Balance:      balance,
	}
	return update, rec
}

func TestBalanceCollector_Collect(t *testing.T) {
	log := logrus.New()
	collector := NewBalanceCollector(log)

	t.Run("nil", func(t *testing.T) {
		require.Nil(t, collector.Collect(nil))
	})

	t.Run("non_amend", func(t *testing.T) {
		empty := insolar.ID{}
		require.Nil(t, collector.Collect(makeResultWith(empty, nil)))
	})

	t.Run("non_account_amend", func(t *testing.T) {
		pn := pulse.OfNow()
		empty := insolar.ID{}
		require.Nil(t, collector.Collect(makeDepositAmend(pn, pn, "", "", empty)))
	})

	t.Run("amend_genesis", func(t *testing.T) {
		genesisRecordID := gen.IDWithPulse(insolar.GenesisPulse.PulseNumber)
		require.Nil(t, collector.Collect(makeAccountAmend(pulse.OfNow(), "", genesisRecordID)))
	})

	t.Run("ordinary", func(t *testing.T) {
		update, rec := makeBalanceUpdate()
		actual := collector.Collect(rec)

		require.Equal(t, update, actual)
	})
}
