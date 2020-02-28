// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gojuno/minimock/v3"
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
	"github.com/insolar/insolar/instrumentation/inslogger"
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
		ID:      gen.IDWithPulse(pulse),
		Virtual: record.Wrap(&activateRecord),
	}
	return (*observer.Record)(rec), &activateRecord
}

func makeMemberActivate(
	pulse insolar.PulseNumber,
	walletRef insolar.Reference,
	requestRef insolar.Reference,
	publicKey string,
) (*observer.Record, *record.Activate) {
	mbr := &member.Member{
		Wallet:    walletRef,
		PublicKey: publicKey,
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
		ID:      gen.IDWithPulse(pulse),
		Virtual: record.Wrap(&activateRecord),
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
		ID:      gen.IDWithPulse(pulse),
		Virtual: record.Wrap(&activateRecord),
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
		ID:      gen.IDWithPulse(pulse),
		Virtual: record.Wrap(&accountRequest),
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
		ID:      gen.IDWithPulse(pulse),
		Virtual: record.Wrap(&walletRequest),
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
		ID:      gen.IDWithPulse(pulse),
		Virtual: record.Wrap(&memberRequest),
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
		ID:      gen.IDWithPulse(pulse),
		Virtual: record.Wrap(&callRecord),
	}
	return (*observer.Record)(rec), &callRecord
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

				pn := insolar.GenesisPulse.PulseNumber + 10
				balance := "42"
				memberRef := gen.ReferenceWithPulse(pn)
				// Useless records that shouldn't used by collector.
				uselessOut := makeOutgoingRequest()
				uselessResult := makeResultWith(uselessOut.ID, &foundation.Result{Returns: []interface{}{nil, nil}})

				// === ROOT ===
				// Root member.create call (and it's satellite).
				call, callIncoming := makeMemberCreateCall(pn)
				callRef := *insolar.NewReference(call.ID)
				callResult := makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&member.CreateResponse{
					Reference: memberRef.String(),
				}, nil}})

				// === ACCOUNT ===
				// Account.new call (and it's satellite).
				newAccount, newAccountIncoming := makeNewAccountRequest(pn, callRef)
				newAccountRef := *insolar.NewReference(newAccount.ID)
				accountActivate, accountActivateSideEffect := makeAccountActivate(pn, balance, newAccountRef)

				// === WALLET ===
				// Wallet.new call (and it's satellite).
				newWallet, callNewWallet := makeNewWalletRequest(pn, callRef)
				newWalletRef := *insolar.NewReference(newWallet.ID)
				walletActivate, walletActiveteSideEffect := makeWalletActivate(pn, newAccountRef, newWalletRef)

				// === MEMBER ===
				// Member.new call (and it's satellite).
				newMember, callNewMember := makeNewMemberRequest(pn, callRef)
				newMemberRef := *insolar.NewReference(newMember.ID)
				publicKey := "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEwDcgWZ1SbG+nbiXZkmYUZEfk2nkk\n1PEmEWoj4g6DLEkdaQVorOkqlloEz1zXclQaAE1S8i3F7OFNrNxLkm34ow==\n-----END PUBLIC KEY-----\n"

				memberActivate, memberActivateSideEffect := makeMemberActivate(
					pn,
					newWalletRef,
					newMemberRef,
					publicKey,
				)

				records := []*observer.Record{
					// Useless.
					uselessOut,
					uselessResult,
					// Call root and children.
					call,
					callResult,
					newAccount,
					makeResultWith(newAccount.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
					newWallet,
					makeResultWith(newWallet.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
					newMember,
					makeResultWith(newMember.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
					// SideEffects.
					accountActivate,
					walletActivate,
					memberActivate,
				}

				expectedMember := &observer.Member{
					MemberRef:        memberRef,
					Balance:          balance,
					MigrationAddress: "",
					AccountState:     accountActivate.ID,
					Status:           "SUCCESS",
					AccountRef:       newAccountRef,
					WalletRef:        newWalletRef,
					PublicKey:        "A8A3IFmdUmxvp24l2ZJmFGRH5Np5JNTxJhFqI-IOgyxJ",
				}

				expectedContractStruct := tree.Structure{
					Request: *callIncoming,
					Outgoings: []tree.Outgoing{
						// Account
						{
							Structure: &tree.Structure{
								RequestID: accountActivate.ID,
								Request:   *newAccountIncoming,
								SideEffect: &tree.SideEffect{
									ID:         accountActivate.ID,
									Activation: accountActivateSideEffect,
								},
							},
						},
						// Wallet
						{
							Structure: &tree.Structure{
								Request: *callNewWallet,
								SideEffect: &tree.SideEffect{
									Activation: walletActiveteSideEffect,
								},
							},
						},
						// Member
						{
							OutgoingRequest: record.OutgoingRequest{},
							Structure: &tree.Structure{
								Request: *callNewMember,
								SideEffect: &tree.SideEffect{
									Activation: memberActivateSideEffect,
								},
							},
						},
					},
					Result: *callResult.Virtual.GetResult(),
				}

				fetcher.RequestMock.Set(func(ctx context.Context, reqID insolar.ID) (m1 record.Material, err error) {
					switch reqID {
					// Just a stub.
					case uselessOut.ID:
						return record.Material(*uselessOut), nil
					case newAccount.ID:
						return record.Material(*newAccount), nil
					case newWallet.ID:
						return record.Material(*newWallet), nil
					case newMember.ID:
						return record.Material(*newMember), nil
					// Right ID
					case call.ID:
						return record.Material(*call), nil
					default:
						panic("unexpected ID")
					}
				})

				builder.BuildMock.Expect(ctx, call.ID).Return(expectedContractStruct, nil)

				return records, fetcher, builder, []*observer.Member{expectedMember}
			},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mc := minimock.NewController(t)
			log := inslogger.FromContext(inslogger.TestContext(t))

			records, fetcher, builder, expected := test.mocks(mc)

			collector := NewMemberCollector(log, fetcher, builder)

			actual := make([]*observer.Member, 0)
			for _, rec := range records {
				mbrs := collector.Collect(ctx, rec)
				for _, mbr := range mbrs {
					actual = append(actual, mbr)
				}
			}

			require.Equal(t, expected, actual)
			mc.Finish()
		})
	}
}
