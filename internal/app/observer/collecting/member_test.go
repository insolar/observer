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

	"github.com/gojuno/minimock"
	"github.com/insolar/insolar/application/api/requester"
	"github.com/insolar/insolar/application/builtin/contract/account"
	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/application/builtin/contract/wallet"
	proxyAccount "github.com/insolar/insolar/application/builtin/proxy/account"
	proxyMember "github.com/insolar/insolar/application/builtin/proxy/member"
	proxyWallet "github.com/insolar/insolar/application/builtin/proxy/wallet"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"
)

func makeAccountActivate(
	pulse insolar.PulseNumber,
	balance string,
	requestRef insolar.Reference,
) (*observer.Record, *record.Activate) {
	acc := &account.Account{
		Balance: balance,
	}
	memory, err := insolar.Serialize(acc)
	if err != nil {
		panic("failed to serialize arguments")
	}

	activateRecord := record.Activate{
		Request: requestRef,
		Memory:  memory,
		Image:   *proxyAccount.PrototypeReference,
	}

	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &activateRecord,
			},
		},
	}
	return (*observer.Record)(rec), &activateRecord
}

func makeMemberActivate(
	pulse insolar.PulseNumber,
	walletRef insolar.Reference,
	requestRef insolar.Reference,
) (*observer.Record, *record.Activate) {
	mbr := &member.Member{
		Wallet: walletRef,
	}
	memory, err := insolar.Serialize(mbr)
	if err != nil {
		panic("failed to serialize arguments")
	}

	activateRecord := record.Activate{
		Request: requestRef,
		Memory:  memory,
		Image:   *proxyMember.PrototypeReference,
	}

	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &activateRecord,
			},
		},
	}
	return (*observer.Record)(rec), &activateRecord
}

func makeWalletActivate(
	pulse insolar.PulseNumber,
	accountRef insolar.Reference,
	requestRef insolar.Reference,
) (*observer.Record, *record.Activate) {
	wlt := &wallet.Wallet{
		Accounts: map[string]string{"XNS": accountRef.String()},
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
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &activateRecord,
			},
		},
	}
	return (*observer.Record)(rec), &activateRecord
}

func makeNewAccountRequest(
	pulse insolar.PulseNumber,
	reason insolar.Reference,
) (*observer.Record, *record.IncomingRequest) {
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

	accountRequest := record.IncomingRequest{
		Method:    "New",
		Arguments: args,
		Prototype: proxyAccount.PrototypeReference,
		Reason:    reason,
	}

	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &accountRequest,
			},
		},
	}
	return (*observer.Record)(rec), &accountRequest
}

func makeNewWalletRequest(
	pulse insolar.PulseNumber,
	reason insolar.Reference,
) (*observer.Record, *record.IncomingRequest) {
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

	walletRequest := record.IncomingRequest{
		Method:    "New",
		Arguments: args,
		Prototype: proxyWallet.PrototypeReference,
		Reason:    reason,
	}

	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &walletRequest,
			},
		},
	}
	return (*observer.Record)(rec), &walletRequest
}

func makeNewMemberRequest(
	pulse insolar.PulseNumber,
	reason insolar.Reference,
) (*observer.Record, *record.IncomingRequest) {
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

	memberRequest := record.IncomingRequest{
		Method:    "New",
		Arguments: args,
		Prototype: proxyMember.PrototypeReference,
		Reason:    reason,
	}

	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &memberRequest,
			},
		},
	}
	return (*observer.Record)(rec), &memberRequest
}

func makeMemberCreateCall(pulse insolar.PulseNumber) (*observer.Record, *record.IncomingRequest) {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite:   "member.create",
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

	callRecord := record.IncomingRequest{
		Method:    "Call",
		Arguments: args,
	}

	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &callRecord,
			},
		},
	}
	return (*observer.Record)(rec), &callRecord
}

func makeMember() ([]*observer.Member, []*observer.Record) {
	pn := insolar.GenesisPulse.PulseNumber + 10
	balance := "42"
	memberRef := gen.IDWithPulse(pn)
	out := makeOutgoingRequest()
	call, _ := makeMemberCreateCall(pn)
	callRef := *insolar.NewReference(call.ID)
	newAccount, _ := makeNewAccountRequest(pn, callRef)
	newAccountRef := *insolar.NewReference(newAccount.ID)
	accountActivate, _ := makeAccountActivate(pn, balance, newAccountRef)
	records := []*observer.Record{
		out,
		makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		call,
		makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&member.CreateResponse{
			Reference: memberRef.String(),
		}, nil}}),
		newAccount,
		makeResultWith(newAccount.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		accountActivate,
	}
	member := &observer.Member{
		MemberRef:        memberRef,
		Balance:          balance,
		MigrationAddress: "",
		AccountState:     accountActivate.ID,
		Status:           "SUCCESS",
	}
	return []*observer.Member{member}, records
}

func TestMemberCollector_Collect(t *testing.T) {
	ctx := context.Background()

	table := []struct {
		name  string
		mocks func(t minimock.Tester) (
			stream []*observer.Record,
			fetcher store.RecordFetcher,
			builder tree.Builder,
			expectedResult []*observer.Member,
		)
	}{
		{
			name: "nil",
			mocks: func(t minimock.Tester) ([]*observer.Record, store.RecordFetcher, tree.Builder, []*observer.Member) {
				fetcher := store.NewRecordFetcherMock(t)
				builder := tree.NewBuilderMock(t)
				return []*observer.Record{nil}, fetcher, builder, []*observer.Member{}
			},
		},
		{
			name: "happy path",
			mocks: func(t minimock.Tester) ([]*observer.Record, store.RecordFetcher, tree.Builder, []*observer.Member) {
				fetcher := store.NewRecordFetcherMock(t)
				builder := tree.NewBuilderMock(t)

				// members, records := makeMember()

				pn := insolar.GenesisPulse.PulseNumber + 10
				balance := "42"
				memberRef := gen.IDWithPulse(pn)
				out := makeOutgoingRequest()
				call, callIncoming := makeMemberCreateCall(pn)
				callRef := *insolar.NewReference(call.ID)
				callResult := makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&member.CreateResponse{
					Reference: memberRef.String(),
				}, nil}})
				// Account
				newAccount, callNewAccount := makeNewAccountRequest(pn, callRef)
				newAccountRef := *insolar.NewReference(newAccount.ID)
				accountActivate, activateRequest := makeAccountActivate(pn, balance, newAccountRef)
				// Wallet
				newWallet, callNewWallet := makeNewWalletRequest(pn, callRef)
				newWalletRef := *insolar.NewReference(newWallet.ID)
				walletActivate, activateWallet := makeWalletActivate(pn, newAccountRef, newWalletRef)
				// Member
				newMember, callNewMember := makeNewMemberRequest(pn, callRef)
				newMemberRef := *insolar.NewReference(newMember.ID)
				memberActivate, activateMember := makeMemberActivate(pn, newWalletRef, newMemberRef)
				records := []*observer.Record{
					out,
					makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
					call,
					callResult,
					newAccount,
					makeResultWith(newAccount.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
					newWallet,
					makeResultWith(newWallet.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
					newMember,
					makeResultWith(newMember.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
					accountActivate,
					walletActivate,
					memberActivate,
				}
				member := &observer.Member{
					MemberRef:        memberRef,
					Balance:          balance,
					MigrationAddress: "",
					AccountState:     accountActivate.ID,
					Status:           "SUCCESS",
					AccountRef:       *newAccountRef.GetLocal(),
					WalletRef:        *newWalletRef.GetLocal(),
				}

				expectedContractStruct := tree.Structure{
					RequestID: insolar.ID{},
					Request:   *callIncoming,
					Outgoings: []tree.Outgoing{
						// Account
						{
							OutgoingRequestID: insolar.ID{},
							OutgoingRequest:   record.OutgoingRequest{},
							Structure: &tree.Structure{
								RequestID: insolar.ID{},
								Request:   *callNewAccount,
								Outgoings: nil,
								SideEffect: &tree.SideEffect{
									Activation:   activateRequest,
									Amend:        nil,
									Deactivation: nil,
								},
								Result: record.Result{},
							},
							Result: nil,
						},
						// Wallet
						{
							OutgoingRequestID: insolar.ID{},
							OutgoingRequest:   record.OutgoingRequest{},
							Structure: &tree.Structure{
								RequestID: insolar.ID{},
								Request:   *callNewWallet,
								Outgoings: nil,
								SideEffect: &tree.SideEffect{
									Activation:   activateWallet,
									Amend:        nil,
									Deactivation: nil,
								},
								Result: record.Result{},
							},
							Result: nil,
						},
						// Member
						{
							OutgoingRequestID: insolar.ID{},
							OutgoingRequest:   record.OutgoingRequest{},
							Structure: &tree.Structure{
								RequestID: insolar.ID{},
								Request:   *callNewMember,
								Outgoings: nil,
								SideEffect: &tree.SideEffect{
									Activation:   activateMember,
									Amend:        nil,
									Deactivation: nil,
								},
								Result: record.Result{},
							},
							Result: nil,
						},
					},
					SideEffect: nil,
					Result:     *callResult.Virtual.GetResult(),
				}

				expectedID := *callResult.Virtual.GetResult().Request.GetLocal()

				fetcher.RequestMock.Set(func(ctx context.Context, reqID insolar.ID) (m1 record.Material, err error) {
					if reqID != expectedID {
						return record.Material{}, store.ErrNotFound
					}
					return record.Material(*call), nil
				})

				builder.BuildMock.Expect(ctx, call.ID).Return(expectedContractStruct, nil)

				return records, fetcher, builder, []*observer.Member{member}
			},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mc := minimock.NewController(t)

			records, fetcher, builder, expected := test.mocks(mc)

			collector := NewMemberCollector(fetcher, builder)

			actual := make([]*observer.Member, 0)
			for _, rec := range records {
				mbr := collector.Collect(ctx, rec)
				if mbr != nil {
					actual = append(actual, mbr)
				}
			}

			require.Equal(t, expected, actual)
			mc.Finish()
		})

	}
}
