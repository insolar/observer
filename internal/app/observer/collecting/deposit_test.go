// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"context"
	"errors"
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/mainnet/application/appfoundation"
	"github.com/insolar/mainnet/application/builtin/contract/deposit"
	"github.com/insolar/mainnet/application/builtin/contract/pkshard"
	"github.com/insolar/mainnet/application/builtin/contract/wallet"
	proxyDeposit "github.com/insolar/mainnet/application/builtin/proxy/deposit"
	proxyPKShard "github.com/insolar/mainnet/application/builtin/proxy/pkshard"
	proxyWallet "github.com/insolar/mainnet/application/builtin/proxy/wallet"
	"github.com/insolar/mainnet/application/genesisrefs"
	"github.com/insolar/observer/internal/models"
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
		VestingType:        appfoundation.Vesting2,
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
		VestingType:     models.DepositTypeDefaultFund,
	}}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}

func TestDepositCollector_CollectDeposit(t *testing.T) {
	log := inslogger.FromContext(inslogger.TestContext(t))
	fetcher := store.NewRecordFetcherMock(t)
	collector := NewDepositCollector(log, fetcher)
	ctx := context.Background()

	pn := insolar.GenesisPulse.PulseNumber + 1
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
		VestingType:        appfoundation.DefaultVesting,
	}

	depositRef := gen.ReferenceWithPulse(pn)

	rec := makeDepositResult(pn)
	incRequest := makeNewDepositIncRequest(pn)
	activationRecord := makeDepositActivate(pn, dep, depositRef)

	fetcher.SideEffectMock.Set(func(ctx context.Context, reqID insolar.ID) (m1 record.Material, err error) {
		return *(*record.Material)(activationRecord), nil
	})
	fetcher.RequestMock.Set(func(ctx context.Context, reqID insolar.ID) (m1 record.Material, err error) {
		return *(*record.Material)(incRequest), nil
	})
	fetcher.ResultMock.Set(func(ctx context.Context, reqID insolar.ID) (m1 record.Material, err error) {
		return *(*record.Material)(rec), nil
	})
	fetcher.CalledRequestsMock.Set(func(ctx context.Context, reqID insolar.ID) (ma1 []record.Material, err error) {
		return []record.Material{}, nil
	})

	actual := collector.Collect(ctx, rec)

	expected := []observer.Deposit{{
		EthHash:         txHash,
		Ref:             depositRef,
		Timestamp:       1546300801,
		Balance:         balance,
		Amount:          amount,
		DepositState:    activationRecord.ID,
		Vesting:         vPeriod,
		VestingStep:     vStep,
		VestingType:     models.DepositTypeNonLinear,
		HoldReleaseDate: 1546300921,
		IsConfirmed:     false,
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

func makeNewDepositIncRequest(pn insolar.PulseNumber) *observer.Record {
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "New",
					Prototype: proxyDeposit.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeDepositResult(pn insolar.PulseNumber) *observer.Record {
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_Result{
				Result: &record.Result{},
			},
		},
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
