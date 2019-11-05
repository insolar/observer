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
	"context"
	"testing"

	"github.com/insolar/insolar/application/builtin/contract/deposit"
	proxyDaemon "github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	"github.com/insolar/insolar/application/genesisrefs"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

func makeDepositActivate(pn insolar.PulseNumber, dep deposit.Deposit, requestRef insolar.Reference) *observer.Record {
	memory, err := insolar.Serialize(&dep)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: requestRef,
					Memory:  memory,
					Image:   *proxyDaemon.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func TestDepositCollector_CollectGenesisDeposit(t *testing.T) {
	t.Skip("should create pkshard with members with wallets with deposits")
	log := logrus.New()
	fetcher := store.NewRecordFetcherMock(t)
	collector := NewDepositCollector(log, fetcher)

	pn := insolar.GenesisPulse.PulseNumber
	amount := "42"
	balance := "0"
	txHash := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
	vPeriod := int64(3 * 24 * 60 * 60)
	vStep := int64(24 * 60 * 60)
	dep := deposit.Deposit{
		Balance:            balance,
		Amount:             amount,
		TxHash:             txHash,
		PulseDepositUnHold: pn + 3,
		Vesting:            vPeriod,
		VestingStep:        vStep,
	}

	depositActivate := makeDepositActivate(pn, dep, gen.ReferenceWithPulse(pn))
	records := []*observer.Record{
		depositActivate,
	}
	timestamp, err := pn.AsApproximateTime()
	if err != nil {
		panic("invalid pulse")
	}
	expected := []*observer.Deposit{{
		EthHash:      txHash,
		Ref:          *genesisrefs.ContractMigrationDeposit.GetLocal(),
		Member:       *genesisrefs.ContractMigrationAdminMember.GetLocal(),
		Timestamp:    timestamp.Unix(),
		Balance:      balance,
		Amount:       amount,
		DepositState: depositActivate.ID,
		Vesting:      vPeriod,
		VestingStep:  vStep,
	}}

	ctx := context.Background()

	var actual []*observer.Deposit
	for _, r := range records {
		deposit := collector.Collect(ctx, r)
		if deposit != nil {
			actual = append(actual, deposit...)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}
