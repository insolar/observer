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
)

func makeAccountActivate(
	pulse insolar.PulseNumber,
	balance string,
	requestRef insolar.Reference,
) *observer.Record {
	acc := &account.Account{
		Balance: balance,
	}
	memory, err := insolar.Serialize(acc)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: requestRef,
					Memory:  memory,
					Image:   *proxyAccount.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeMemberActivate(
	pulse insolar.PulseNumber,
	walletRef insolar.Reference,
	requestRef insolar.Reference,
) *observer.Record {
	mbr := &member.Member{
		Wallet: walletRef,
	}
	memory, err := insolar.Serialize(mbr)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: requestRef,
					Memory:  memory,
					Image:   *proxyMember.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeWalletActivate(
	pulse insolar.PulseNumber,
	accountRef insolar.Reference,
	requestRef insolar.Reference,
) *observer.Record {
	wlt := &wallet.Wallet{
		Accounts: map[string]string{"XNS": accountRef.String()},
	}
	memory, err := insolar.Serialize(wlt)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: requestRef,
					Memory:  memory,
					Image:   *proxyWallet.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeNewAccountRequest(pulse insolar.PulseNumber, reason insolar.Reference) *observer.Record {
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
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "New",
					Arguments: args,
					Prototype: proxyAccount.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeMemberCreateCall(pulse insolar.PulseNumber) *observer.Record {
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

func makeMember() ([]*observer.Member, []*observer.Record) {
	pn := insolar.GenesisPulse.PulseNumber + 10
	balance := "42"
	memberRef := gen.IDWithPulse(pn)
	out := makeOutgoingRequest()
	call := makeMemberCreateCall(pn)
	callRef := *insolar.NewReference(call.ID)
	newAccount := makeNewAccountRequest(pn, callRef)
	newAccountRef := *insolar.NewReference(newAccount.ID)
	accountActivate := makeAccountActivate(pn, balance, newAccountRef)
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
	//
	// fetcher := store.NewRecordFetcherMock(t)
	// builder := tree.NewBuilderMock(t)
	//
	// pn := insolar.GenesisPulse.PulseNumber + 10
	// balance := "42"
	// memberRef := gen.IDWithPulse(pn)
	// // out := makeOutgouingRequest()
	//
	// // Call member.create.
	// call := makeMemberCreateCall(pn)
	// callRef := *insolar.NewReference(call.ID)
	//
	// // Result for member.create.
	// memberCreateResult := makeResultWith(
	// 	call.ID,
	// 	&foundation.Result{
	// 		Returns: []interface{}{&member.CreateResponse{Reference: memberRef.String()}, nil},
	// 	},
	// )

	// // ===== ACCOUNT =====
	// // Call account.new.
	// newAccount := makeNewAccountRequest(pn, callRef)
	// newAccountRef := *insolar.NewReference(newAccount.ID)
	//
	// // // Result for account.new.
	// // accountCreateResult := makeResultWith(newAccount.ID, &foundation.Result{Returns: []interface{}{nil, nil}})
	//
	// // SideEffect from account.new - activate record.
	// accountActivate := makeAccountActivate(pn, balance, newAccountRef)
	//
	// // ===== WALLET =====
	// // Call wallet.new.
	// newWallet := makeNewAccountRequest(pn, callRef)
	// newWalletRef := *insolar.NewReference(newWallet.ID)
	//
	// // Result for wallet.new.
	// walletCreateResult := makeResultWith(newWallet.ID, &foundation.Result{Returns: []interface{}{nil, nil}})
	//
	// // SideEffect from wallet.new - activate record.
	// walletActivate := makeWalletActivate(pn, newAccountRef, newWalletRef)
	//
	// // ===== MEMBER ====
	// // Call member.new.
	// newMember := makeNewAccountRequest(pn, callRef)
	// newMemberRef := *insolar.NewReference(newMember.ID)
	//
	// // Result for member.new.
	// memberCreateResult := makeResultWith(newMember.ID, &foundation.Result{Returns: []interface{}{nil, nil}})
	//
	// // SideEffect from member.new - activate record.
	// memberActivate := makeMemberActivate(pn, newWalletRef, newMemberRef)
	//
	// contractMemberCreate := &tree.Structure{
	// 	RequestID:  call.ID,
	// 	Request:    *record.Unwrap(&call.Virtual).(*record.IncomingRequest),
	// 	Outgoings:  nil,
	// 	SideEffect: nil,
	// 	Result:     record.Result{},
	// }
	//
	// records := []*observer.Record{
	// 	// out,
	// 	// makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
	// 	call,
	// 	makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&member.CreateResponse{
	// 		Reference: memberRef.String(),
	// 	}, nil}}),
	// 	newAccount,
	// 	makeResultWith(newAccount.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
	// 	accountActivate,
	// }
	// member := &observer.Member{
	// 	MemberRef:        memberRef,
	// 	Balance:          balance,
	// 	MigrationAddress: "",
	// 	AccountState:     accountActivate.ID,
	// 	Status:           "SUCCESS",
	// }

	collector := NewMemberCollector(nil, nil) // FIXME: change nil to normal values
	// collector := NewMemberCollector(fetcher, builder) // FIXME: change nil to normal values

	expected, records := makeMember()
	var actual []*observer.Member
	for _, r := range records {
		member := collector.Collect(ctx, r)
		if member != nil {
			actual = append(actual, member)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}
