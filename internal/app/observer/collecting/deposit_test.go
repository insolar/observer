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
	"errors"
	"testing"

	"github.com/insolar/insolar/application/builtin/contract/deposit"
	"github.com/insolar/insolar/application/builtin/contract/pkshard"
	"github.com/insolar/insolar/application/builtin/contract/wallet"
	proxyPKShard "github.com/insolar/insolar/application/builtin/proxy/pkshard"
	proxyWallet "github.com/insolar/insolar/application/builtin/proxy/wallet"
	"github.com/insolar/insolar/application/genesisrefs"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

func TestDepositCollector_CollectGenesisDeposit(t *testing.T) {
	log := inslogger.FromContext(inslogger.TestContext(t))
	fetcher := store.NewRecordFetcherMock(t)
	collector := NewDepositCollector(log, fetcher)
	ctx := context.Background()
	cache := make(map[insolar.ID]*observer.Record)

	pn := insolar.GenesisPulse.PulseNumber
	amount := "42"
	balance := "0"
	txHash := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
	lockup := int64(120)
	vPeriod := int64(3 * 24 * 60 * 60)
	vStep := int64(24 * 60 * 60)
	dep := deposit.Deposit{
		Balance:            balance,
		Amount:             amount,
		TxHash:             txHash,
		PulseDepositUnHold: pn + insolar.PulseNumber(lockup),
		Vesting:            vPeriod,
		VestingStep:        vStep,
		Lockup:             lockup,
	}

	depositRef := genesisrefs.ContractMigrationDeposit
	memberRef := genesisrefs.ContractMigrationAdminMember
	walletRef1 := gen.ReferenceWithPulse(pn)
	accountRef1 := gen.ReferenceWithPulse(pn)

	pks := pkshard.PKShard{
		Map: foundation.StableMap{
			"pk1": memberRef.String(),
		},
	}
	pkshardActivate := makePKShardActivate(pn, pks, gen.ReferenceWithPulse(pn))

	cache[*depositRef.GetLocal()] = makeDepositActivate(pn, dep, depositRef)
	cache[*walletRef1.GetLocal()], _ = makeGenesisWalletActivate(pn, accountRef1, walletRef1, depositRef)
	cache[*memberRef.GetLocal()], _ = makeMemberActivate(pn, walletRef1, memberRef, "test_public_key")

	fetcher.SideEffectMock.Set(func(ctx context.Context, reqID insolar.ID) (m1 record.Material, err error) {
		if rec, ok := cache[reqID]; ok == true {
			return *(*record.Material)(rec), nil
		}
		return record.Material{}, errors.New("record not found in cache")
	})

	actual := collector.Collect(ctx, pkshardActivate)

	expected := []observer.Deposit{{
		EthHash:         txHash,
		Ref:             genesisrefs.ContractMigrationDeposit,
		Member:          genesisrefs.ContractMigrationAdminMember,
		Timestamp:       1546300800,
		Balance:         balance,
		Amount:          amount,
		DepositState:    cache[*depositRef.GetLocal()].ID,
		Vesting:         vPeriod,
		VestingStep:     vStep,
		HoldReleaseDate: 1546300920,
		IsConfirmed:     true,
	}}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}

func makeDepositActivate(pn insolar.PulseNumber, dep deposit.Deposit, requestRef insolar.Reference) *observer.Record {
	memory, err := insolar.Serialize(&dep)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Wrap(&record.Activate{
			Request: requestRef,
			Memory:  memory,
			Image:   proxyPKShard.GetPrototype(),
		}),
	}
	return (*observer.Record)(rec)
}

func makeGenesisWalletActivate(
	pulse insolar.PulseNumber,
	accountRef insolar.Reference,
	requestRef insolar.Reference,
	depositRef insolar.Reference,
) (*observer.Record, *record.Activate) {
	wlt := &wallet.Wallet{
		Accounts: foundation.StableMap{"XNS": accountRef.String()},
		Deposits: foundation.StableMap{"dep1": depositRef.String()},
	}
	memory, err := insolar.Serialize(wlt)
	if err != nil {
		panic("failed to serialize arguments")
	}

	activateRecord := record.Activate{
		Request: requestRef,
		Memory:  memory,
		Image:   *proxyWallet.PrototypeReference,
	}

	rec := &record.Material{
		ID:      gen.IDWithPulse(pulse),
		Virtual: record.Wrap(&activateRecord),
	}
	return (*observer.Record)(rec), &activateRecord
}

func makePKShardActivate(pn insolar.PulseNumber, pks pkshard.PKShard, requestRef insolar.Reference) *observer.Record {
	memory, err := insolar.Serialize(&pks)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Wrap(&record.Activate{
			Request: requestRef,
			Memory:  memory,
			Image:   proxyPKShard.GetPrototype(),
		}),
	}
	return (*observer.Record)(rec)
}
