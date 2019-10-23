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

	"github.com/insolar/insolar/application"
	"github.com/insolar/insolar/application/api/requester"
	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/application/genesisrefs"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	proxyMigrationAdmin "github.com/insolar/insolar/application/builtin/proxy/migrationadmin"
	proxyDaemon "github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"
)

func makeMigrationTransferCall(amount, from, ethHash string, pulse insolar.PulseNumber) *observer.Record {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite: MigrationTransferMethod,
			CallParams: TransferCallParams{
				Amount:    amount,
				EthTxHash: ethHash,
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

func makeDaemonDepositMigrationCall(pulse insolar.PulseNumber) *observer.Record {
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "DepositMigrationCall",
					Prototype: proxyDaemon.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeGetMemberByMigrationAddress(pulse insolar.PulseNumber) *observer.Record {
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "GetMemberByMigrationAddress",
					Prototype: proxyMigrationAdmin.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeDepositConfirm(pulse insolar.PulseNumber) *observer.Record {
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "Confirm",
					Prototype: proxyDeposit.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeTransferToDeposit(pulse insolar.PulseNumber) *observer.Record {
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "TransferToDeposit",
					Prototype: proxyDeposit.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func TestMigrationTransferCollector_Collect(t *testing.T) {
	log := logrus.New()
	fetcher := store.NewRecordFetcherMock(t)
	builder := tree.NewBuilderMock(t)
	collector := NewMigrationTransferCollector(log, fetcher, builder)
	ctx := context.Background()

	pn := insolar.GenesisPulse.PulseNumber
	amount := "42"
	fee := "0"
	from := gen.IDWithPulse(pn)
	to := gen.ReferenceWithPulse(pn)
	ethHash := "0x1234567890"
	out := makeOutgoingRequest()
	call := makeMigrationTransferCall(amount, from.String(), ethHash, pn)
	daemonMigrationCall := makeDaemonDepositMigrationCall(pn)
	getMemberByAddress := makeGetMemberByMigrationAddress(pn)
	depositConfirm := makeDepositConfirm(pn)
	transferToDeposit := makeTransferToDeposit(pn)

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
			objRef, err := insolar.NewObjectReferenceFromString(to.String())
			require.NoError(t, err)
			return tree.Structure{Outgoings: []tree.Outgoing{
				{
					Structure: &tree.Structure{
						RequestID: daemonMigrationCall.ID,
						Request:   *daemonMigrationCall.Virtual.GetIncomingRequest(),
						Outgoings: []tree.Outgoing{
							{
								Structure: &tree.Structure{
									RequestID: getMemberByAddress.ID,
									Request:   *getMemberByAddress.Virtual.GetIncomingRequest(),
									Result: *makeResultWith(getMemberByAddress.ID, &foundation.Result{
										Error:   nil,
										Returns: []interface{}{objRef, nil},
									}).Virtual.GetResult(),
								},
							},
							{
								Structure: &tree.Structure{
									RequestID: depositConfirm.ID,
									Request:   *depositConfirm.Virtual.GetIncomingRequest(),
									Outgoings: []tree.Outgoing{
										{
											Structure: &tree.Structure{
												RequestID: transferToDeposit.ID,
												Request:   *transferToDeposit.Virtual.GetIncomingRequest(),
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

	memberFrom := genesisrefs.GenesisRef(application.GenesisNameMigrationAdminMember)
	expected := []*observer.Transfer{
		{
			TxID:          call.ID,
			From:          memberFrom.GetLocal(),
			To:            to.GetLocal(),
			EthHash:       ethHash,
			Amount:        amount,
			Fee:           fee,
			Status:        observer.Success,
			Kind:          observer.Migration,
			Direction:     observer.APICall,
			DetachRequest: &transferToDeposit.ID,
		},
	}

	var actual []*observer.Transfer
	for _, r := range records {
		transfer := collector.Collect(ctx, r)
		if transfer != nil {
			actual = append(actual, transfer)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}
