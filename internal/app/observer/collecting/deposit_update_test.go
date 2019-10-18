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

	"github.com/insolar/insolar/application/builtin/contract/deposit"
	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
)

func makeDepositAmend(pn, unholdPulse insolar.PulseNumber, balance, amount string, prev insolar.ID) *observer.Record {
	acc := &deposit.Deposit{
		Balance:            balance,
		Amount:             amount,
		PulseDepositUnHold: unholdPulse,
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
					Image:     *proxyDeposit.PrototypeReference,
					PrevState: prev,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeDepositUpdate() (*observer.DepositUpdate, *observer.Record) {
	pn := insolar.GenesisPulse.PulseNumber
	unholdPulse := pn + 3
	amount := "4"
	balance := "3"
	prev := gen.IDWithPulse(pn)
	rec := makeDepositAmend(pn, unholdPulse, balance, amount, prev)
	timestamp, err := unholdPulse.AsApproximateTime()
	if err != nil {
		panic("invalid pulse")
	}
	upd := &observer.DepositUpdate{
		ID:              rec.ID,
		HoldReleaseDate: timestamp.Unix(),
		Amount:          amount,
		Balance:         balance,
		PrevState:       prev,
	}
	return upd, rec
}

func TestDepositUpdateCollector_Collect(t *testing.T) {
	log := logrus.New()
	collector := NewDepositUpdateCollector(log)

	t.Run("nil", func(t *testing.T) {
		require.Nil(t, collector.Collect(nil))
	})

	t.Run("ordinary", func(t *testing.T) {
		upd, rec := makeDepositUpdate()
		actual := collector.Collect(rec)
		require.Equal(t, upd, actual)
	})
}
