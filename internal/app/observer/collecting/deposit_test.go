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
	"encoding/json"
	"testing"

	"github.com/insolar/insolar/application/api/requester"
	"github.com/insolar/insolar/application/builtin/contract/deposit"
	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	"github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	proxyDaemon "github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	"github.com/insolar/insolar/application/genesisrefs"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

func makeDepositMigrationCall(pn insolar.PulseNumber) *observer.Record {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite:   "deposit.migration",
			CallParams: nil,
		},
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		panic("failed to marshal request")
	}
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{requestBody, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "Call",
					Arguments: args,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeMigrationDaemonCall(pn insolar.PulseNumber, reason insolar.Reference) *observer.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "DepositMigrationCall",
					Arguments: args,
					Prototype: proxyDaemon.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeNewDepositRequest(pn insolar.PulseNumber, reason insolar.Reference) *observer.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "New",
					Arguments: args,
					Prototype: proxyDeposit.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeDepositActivate(pn insolar.PulseNumber, balance, amount, txHash string, requestRef insolar.Reference) *observer.Record {
	dep := &deposit.Deposit{
		Balance:            balance,
		Amount:             amount,
		TxHash:             txHash,
		PulseDepositUnHold: pn + 3,
	}
	memory, err := insolar.Serialize(dep)
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
					Image:   *proxyDeposit.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeDeposit() ([]*observer.Deposit, []*observer.Record) {
	pn := insolar.GenesisPulse.PulseNumber + 100
	amount := "42"
	balance := "0"
	txHash := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
	memberRef := gen.IDWithPulse(pn)
	out := makeOutgoingRequest()
	call := makeDepositMigrationCall(pn)
	callRef := *insolar.NewReference(call.ID)
	daemonCall := makeMigrationDaemonCall(pn, callRef)
	daemonCallRef := *insolar.NewReference(daemonCall.ID)
	newDeposit := makeNewDepositRequest(pn, daemonCallRef)
	depositRef := *insolar.NewRecordReference(newDeposit.ID)
	depositActivate := makeDepositActivate(pn, balance, amount, txHash, depositRef)
	records := []*observer.Record{
		out,
		makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		call,
		makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&migrationdaemon.DepositMigrationResult{
			Reference: memberRef.String(),
		}, nil}}),
		daemonCall,
		makeResultWith(daemonCall.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		newDeposit,
		makeResultWith(newDeposit.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		depositActivate,
	}
	timestamp, err := pn.AsApproximateTime()
	if err != nil {
		panic("invalid pulse")
	}
	deposit := &observer.Deposit{
		EthHash:      txHash,
		Ref:          *depositRef.GetLocal(),
		Member:       memberRef,
		Timestamp:    timestamp.Unix(),
		Balance:      balance,
		Amount:       amount,
		DepositState: depositActivate.ID,
	}
	return []*observer.Deposit{deposit}, records
}

func TestDepositCollector_Collect(t *testing.T) {
	log := logrus.New()
	fetcher := store.NewRecordFetcherMock(t)
	collector := NewDepositCollector(log, fetcher)
	ctx := context.Background()

	expected, records := makeDeposit()
	var actual []*observer.Deposit
	for _, r := range records {
		deposit := collector.Collect(ctx, r)
		if deposit != nil {
			actual = append(actual, deposit)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}

func TestDepositCollector_CollectGenesisDeposit(t *testing.T) {
	log := logrus.New()
	fetcher := store.NewRecordFetcherMock(t)
	collector := NewDepositCollector(log, fetcher)

	pn := insolar.GenesisPulse.PulseNumber
	amount := "42"
	balance := "0"
	txHash := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
	depositActivate := makeDepositActivate(pn, balance, amount, txHash, gen.ReferenceWithPulse(pn))
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
	}}

	ctx := context.Background()

	var actual []*observer.Deposit
	for _, r := range records {
		deposit := collector.Collect(ctx, r)
		if deposit != nil {
			actual = append(actual, deposit)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}
