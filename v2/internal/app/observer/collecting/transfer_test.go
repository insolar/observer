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
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/insolar/pulse"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/internal/app/observer"
)

func makeOutgouingRequest() *observer.Record {
	rec := &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeResultWith(requestID insolar.ID, result *foundation.Result) *observer.Record {
	payload, err := insolar.Serialize(result)
	if err != nil {
		panic("failed to serialize result")
	}
	ref := insolar.NewReference(requestID)
	rec := &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_Result{
				Result: &record.Result{
					Request: *ref,
					Payload: payload,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeTransferCall(amount, from, to string, pulse insolar.PulseNumber) *observer.Record {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite: "member.transfer",
			CallParams: transferCallParams{
				Amount:            amount,
				ToMemberReference: to,
			},
			Reference: from,
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
		ID: gen.IDWithPulse(pulse),
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

func makeTransfer() ([]*observer.DepositTransfer, []*observer.Record) {
	pn := insolar.GenesisPulse.PulseNumber
	amount := "42"
	fee := "7"
	from := gen.IDWithPulse(pn)
	to := gen.IDWithPulse(pn)
	out := makeOutgouingRequest()
	call := makeTransferCall(amount, from.String(), to.String(), pn)
	records := []*observer.Record{
		out,
		makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		call,
		makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&member.TransferResponse{Fee: fee}, nil}}),
	}

	timestamp, err := pulse.Number(pn).AsApproximateTime()
	if err != nil {
		panic("failed to calc timestamp by pulse")
	}
	transfer := &observer.DepositTransfer{
		Transfer: observer.Transfer{
			TxID:      call.ID,
			From:      from,
			To:        to,
			Amount:    amount,
			Fee:       fee,
			Pulse:     pn,
			Timestamp: timestamp.Unix(),
		},
	}
	return []*observer.DepositTransfer{transfer}, records
}

func TestTransferCollector_Collect(t *testing.T) {
	log := logrus.New()
	collector := NewTransferCollector(log)

	expected, records := makeTransfer()
	var actual []*observer.DepositTransfer
	for _, r := range records {
		transfer := collector.Collect(r)
		if transfer != nil {
			actual = append(actual, transfer)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}
