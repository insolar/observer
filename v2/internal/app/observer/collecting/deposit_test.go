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
	"encoding/json"
	"testing"

	"github.com/insolar/insolar/api/requester"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/deposit"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	proxyDeposit "github.com/insolar/insolar/logicrunner/builtin/proxy/deposit"
	"github.com/insolar/insolar/logicrunner/builtin/proxy/migrationdaemon"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/internal/app/observer"
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
	pn := insolar.GenesisPulse.PulseNumber
	amount := "42"
	balance := "0"
	txHash := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
	memberRef := *insolar.NewReference(gen.IDWithPulse(pn))
	out := makeOutgouingRequest()
	call := makeDepositMigrationCall(pn)
	callRef := *insolar.NewReference(call.ID)
	newDeposit := makeNewDepositRequest(pn, callRef)
	depositRef := *insolar.NewReference(newDeposit.ID)
	depositActivate := makeDepositActivate(pn, balance, amount, txHash, depositRef)
	records := []*observer.Record{
		out,
		makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		call,
		makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&migrationdaemon.DepositMigrationResult{
			Reference: memberRef.String(),
		}, nil}}),
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
		Ref:          depositRef,
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
	collector := NewDepositCollector(log)

	expected, records := makeDeposit()
	var actual []*observer.Deposit
	for _, r := range records {
		deposit := collector.Collect(r)
		if deposit != nil {
			actual = append(actual, deposit)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}
