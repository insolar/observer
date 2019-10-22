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
	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

func makeOutgoingRequest() *observer.Record {
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
			CallSite: TransferMethod,
			CallParams: TransferCallParams{
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

func TestTransferCollector_Collect(t *testing.T) {
	fetcher := store.NewRecordFetcherMock(t)
	collector := NewTransferCollector(fetcher)
	ctx := context.Background()

	pn := insolar.GenesisPulse.PulseNumber
	amount := "42"
	fee := "7"
	from := gen.IDWithPulse(pn)
	to := gen.IDWithPulse(pn)
	out := makeOutgoingRequest()
	call := makeTransferCall(amount, from.String(), to.String(), pn)
	records := []*observer.Record{
		makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&member.TransferResponse{Fee: fee}, nil}}),
	}

	fetcher.RequestMock.Set(func(_ context.Context, reqID insolar.ID) (m1 record.Material, err error) {
		switch reqID {
		case out.ID:
			return record.Material(*out), nil
		case call.ID:
			return record.Material(*call), nil
		default:
			panic("unexpected call")
		}
	})

	timestamp, err := pn.AsApproximateTime()
	if err != nil {
		panic("failed to calc timestamp by pulse")
	}
	expected := []*observer.ExtendedTransfer{
		{
			DepositTransfer: observer.DepositTransfer{
				Transfer: observer.Transfer{
					TxID:      call.ID,
					From:      from,
					To:        to,
					Amount:    amount,
					Fee:       fee,
					Pulse:     pn,
					Timestamp: timestamp.Unix(),
				},
			},
		},
	}

	var actual []*observer.ExtendedTransfer
	for _, r := range records {
		transfer := collector.Collect(ctx, r)
		if transfer != nil {
			actual = append(actual, transfer)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}
