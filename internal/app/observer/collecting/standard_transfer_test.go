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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"

	proxyAccount "github.com/insolar/insolar/application/builtin/proxy/account"
	proxyCostCenter "github.com/insolar/insolar/application/builtin/proxy/costcenter"
	proxyWallet "github.com/insolar/insolar/application/builtin/proxy/wallet"
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

func makeCalcFeeCall(pn insolar.PulseNumber, reason insolar.Reference) *observer.Record {
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
					Method:    "CalcFee",
					Arguments: args,
					Prototype: proxyCostCenter.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeGetFeeMemberCall(pn insolar.PulseNumber, reason insolar.Reference) *observer.Record {
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
					Method:    "GetFeeMember",
					Arguments: args,
					Prototype: proxyCostCenter.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeStandardTransferCall(amount, from, to string, pulse insolar.PulseNumber) *observer.Record {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite: StandardTransferMethod,
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

func makeWalletTransfer(pulse insolar.PulseNumber) *observer.Record {
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "Transfer",
					Prototype: proxyWallet.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeAccountTransfer(pulse insolar.PulseNumber) *observer.Record {
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "Transfer",
					Prototype: proxyAccount.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func TestStandardTransferCollector_Collect(t *testing.T) {
	log := logrus.New()
	fetcher := store.NewRecordFetcherMock(t)
	builder := tree.NewBuilderMock(t)
	collector := NewStandardTransferCollector(log, fetcher, builder)
	ctx := context.Background()

	pn := insolar.GenesisPulse.PulseNumber
	amount := "42"
	fee := "7"
	from := gen.IDWithPulse(pn)
	to := gen.IDWithPulse(pn)
	out := makeOutgoingRequest()
	call := makeStandardTransferCall(amount, from.String(), to.String(), pn)
	walletTransfer := makeWalletTransfer(pn)
	accountTransfer := makeAccountTransfer(pn)
	calcFee := makeCalcFeeCall(pn, *insolar.NewReference(accountTransfer.ID))
	getFeeMember := makeGetFeeMemberCall(pn, *insolar.NewReference(accountTransfer.ID))
	feeMember := gen.ReferenceWithPulse(pn)
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

	builder.BuildMock.Set(func(_ context.Context, reqID insolar.ID) (s1 tree.Structure, err error) {
		switch reqID {
		case out.ID:
			return tree.Structure{}, nil
		case call.ID:
			return tree.Structure{Outgoings: []tree.Outgoing{
				{
					Structure: &tree.Structure{
						RequestID: walletTransfer.ID,
						Request:   *walletTransfer.Virtual.GetIncomingRequest(),
						Outgoings: []tree.Outgoing{
							{
								Structure: &tree.Structure{
									RequestID: accountTransfer.ID,
									Request:   *accountTransfer.Virtual.GetIncomingRequest(),
									Outgoings: []tree.Outgoing{
										{
											Structure: &tree.Structure{
												RequestID: calcFee.ID,
												Request:   *calcFee.Virtual.GetIncomingRequest(),
											},
										},
										{
											Structure: &tree.Structure{
												RequestID: getFeeMember.ID,
												Request:   *getFeeMember.Virtual.GetIncomingRequest(),
												Result: *makeResultWith(getFeeMember.ID,
													&foundation.Result{Returns: []interface{}{feeMember.Bytes(), nil}}).
													Virtual.
													GetResult(),
											},
										},
									},
								},
							},
						},
					},
				},
			}}, nil
		default:
			panic("unexpected call")
		}
	})
	expected := []*observer.Transfer{
		{
			TxID:          call.ID,
			From:          &from,
			To:            &to,
			Amount:        amount,
			Fee:           fee,
			Status:        observer.Success,
			Kind:          observer.Standard,
			Direction:     observer.APICall,
			DetachRequest: &accountTransfer.ID,
		},
		{
			TxID:          call.ID,
			From:          &from,
			To:            feeMember.GetLocal(),
			Amount:        fee,
			Fee:           "0",
			Status:        observer.Success,
			Kind:          observer.Standard,
			Direction:     observer.APICall,
			DetachRequest: &accountTransfer.ID,
		},
	}

	var actual []*observer.Transfer
	for _, r := range records {
		transfers := collector.Collect(ctx, r)
		actual = append(actual, transfers...)
	}

	require.Len(t, actual, 2)
	require.Equal(t, expected, actual)
}
