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
	"runtime/debug"

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
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"
)

type MemberCollector struct {
	fetcher store.RecordFetcher
	builder tree.Builder

	// collector *BoundCollector
}

func NewMemberCollector(
	fetcher store.RecordFetcher,
	builder tree.Builder,
) *MemberCollector {
	// collector := NewBoundCollector(isMemberCreateRequest, successResult, isNewAccount, isAccountActivate)
	return &MemberCollector{
		fetcher: fetcher,
		builder: builder,
		// collector: collector,
	}
}

func (c *MemberCollector) Collect(ctx context.Context, rec *observer.Record) *observer.Member {
	if rec == nil {
		return nil
	}

	// TODO: check - Are we really need int?
	// act := observer.CastToActivate(rec)
	// if act.IsActivate() {
	// 	if act.Virtual.GetActivate().Image.Equal(*proxyAccount.PrototypeReference) {
	// 		if act.ID.Pulse() == insolar.GenesisPulse.PulseNumber {
	// 			balance := accountBalance(rec)
	// 			return &observer.Member{
	// 				MemberRef:    gen.ID(),
	// 				Balance:      balance,
	// 				AccountState: rec.ID,
	// 				Status:       "INTERNAL",
	// 				// TODO: Some fields
	// 			}
	// 		}
	// 	}
	// }

	result := observer.CastToResult(rec)
	if !result.IsResult() {
		return nil
	}

	if !result.IsSuccess() {
		// TODO
		return badMember()
	}

	requestID := *result.Virtual.GetResult().Request.GetLocal()

	// Fetch root request.
	originRequest, err := c.fetcher.Request(ctx, requestID)
	if err != nil {
		// TODO
		panic("SOMETHING WENT WRONG: can't fetch request for result")
		return badMember()
	}

	// Build contract tree.
	memberContractTree, err := c.builder.Build(ctx, originRequest.ID)
	if err != nil {
		panic("SOMETHING WENT WRONG: can't build tree")
		return badMember()
	}

	callRequest := memberContractTree.Request
	children := memberContractTree.Outgoings
	// sideEffect := memberContractTree.SideEffect // TODO: check is empty (because member.create doesn't has side effect)
	contractResult := memberContractTree.Result // TODO: check equality with result above (maybe)

	request := observer.CastToRequest(callRequest)
	if request == nil {
		// TODO
		panic("SOMETHING WENT WRONG: can't cast request")
		return badMember()
	}

	if !isMemberCreateRequest(request) {
		// TODO
		panic("SOMETHING WENT WRONG: origin request isn't member.create request")
		return badMember()
	}

	accountTree, walletTree, memberTree := childTrees(children)

	balance := accountBalance(accountTree.SideEffect.Activation)
	walletRef := walletRef(memberTree.SideEffect.Activation)
	accountRef := accountRef(walletTree.SideEffect.Activation)

	response := createResponse(contractResult)

	memberRef, err := insolar.NewIDFromString(response.Reference)
	if err != nil || memberRef == nil {
		log.Error("invalid member reference")
		return badMember()
	}

	return &observer.Member{
		MemberRef:        *memberRef,
		WalletRef:        *walletRef.GetLocal(),
		AccountRef:       *accountRef.GetLocal(),
		Balance:          balance,
		MigrationAddress: response.MigrationAddress,
		AccountState:     *accountTree.SideEffect.Activation.Request.GetLocal(),
		Status:           "SUCCESS",
	}
}

func childTrees(
	children []tree.Outgoing,
) (
	accountTree *tree.Structure,
	walletTree *tree.Structure,
	memberTree *tree.Structure,
) {
	for _, child := range children {
		if isNewAccount(child.Structure.Request) {
			accountTree = child.Structure
		}
		if isNewWallet(child.Structure.Request) {
			accountTree = child.Structure
		}
		if isNewMember(child.Structure.Request) {
			accountTree = child.Structure
		}
	}

	return accountTree, walletTree, memberTree
}

func badMember() *observer.Member {
	return &observer.Member{
		MemberRef:        gen.ID(), // FIXME: what to do with this? (it's a unique key probably)
		Balance:          "",
		MigrationAddress: "",
		AccountState:     insolar.ID{},
		Status:           "FAILED",
		WalletRef:        insolar.ID{},
		AccountRef:       insolar.ID{},
	}
}

// // TODO: bad naming
// func buildMember()

// func (c *MemberCollector) CollectOld(rec *observer.Record) *observer.Member {
// 	if rec == nil {
// 		return nil
// 	}
//
// 	act := observer.CastToActivate(rec)
// 	if act.IsActivate() {
// 		if act.Virtual.GetActivate().Image.Equal(*proxyAccount.PrototypeReference) {
// 			if act.ID.Pulse() == insolar.GenesisPulse.PulseNumber {
// 				balance := accountBalance(rec)
// 				return &observer.Member{
// 					MemberRef:    gen.ID(),
// 					Balance:      balance,
// 					AccountState: rec.ID,
// 					Status:       "INTERNAL",
// 				}
// 			}
// 		}
// 	}
//
// 	couple := c.collector.Collect(rec)
// 	if couple == nil {
// 		return nil
// 	}
//
// 	m, err := c.build(couple.Activate, couple.Result)
// 	if err != nil {
// 		log.Error(errors.Wrapf(err, "failed to build member"))
// 		return nil
// 	}
// 	return m
// }

func isMemberCreateRequest(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}
	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	switch args.Params.CallSite {
	case "member.create", "member.migrationCreate":
		return true
	}
	return false
}

func successResult(chain interface{}) bool {
	result := observer.CastToResult(chain)
	return result.IsSuccess()
}

func isNewAccount(request record.IncomingRequest) bool {
	if request.Method != "New" {
		return false
	}
	if request.Prototype == nil {
		return false
	}
	return request.Prototype.Equal(*proxyAccount.PrototypeReference)
}

func isNewWallet(request record.IncomingRequest) bool {
	if request.Method != "New" {
		return false
	}
	if request.Prototype == nil {
		return false
	}
	return request.Prototype.Equal(*proxyWallet.PrototypeReference)
}

func isNewMember(request record.IncomingRequest) bool {
	if request.Method != "New" {
		return false
	}
	if request.Prototype == nil {
		return false
	}
	return request.Prototype.Equal(*proxyMember.PrototypeReference)
}

func isAccountActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()
	return act.Image.Equal(*proxyAccount.PrototypeReference)
}

func isWalletActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()
	return act.Image.Equal(*proxyWallet.PrototypeReference)
}

func isMemberActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()
	return act.Image.Equal(*proxyMember.PrototypeReference)
}

// func (c *MemberCollector) build(act *observer.Activate, res *observer.Result) (*observer.Member, error) {
// 	if res == nil || act == nil {
// 		return nil, errors.New("trying to create member from noncomplete builder")
// 	}
//
// 	if res.Virtual.GetResult().Payload == nil {
// 		return nil, errors.New("member creation result payload is nil")
// 	}
// 	response := &member.MigrationCreateResponse{} // FIXME maybe it's a hack?
// 	res.ParseFirstPayloadValue(response)
//
// 	balance := accountBalance((*observer.Record)(act))
// 	ref, err := insolar.NewIDFromString(response.Reference)
// 	if err != nil || ref == nil {
// 		return nil, errors.New("invalid member reference")
// 	}
// 	return &observer.Member{
// 		MemberRef:        *ref,
// 		Balance:          balance,
// 		MigrationAddress: response.MigrationAddress,
// 		AccountState:     act.ID,
// 		Status:           "SUCCESS",
// 	}, nil
// }

func createResponse(result record.Result) member.MigrationCreateResponse {
	response := member.MigrationCreateResponse{}

	ParseFirstValueResult(&result, response)

	return response
}

func accountBalance(act *record.Activate) string {
	memory := act.Memory
	balance := ""

	if memory == nil {
		log.Warn(errors.New("account memory is nil"))
		return "0"
	}

	acc := account.Account{}
	if err := insolar.Deserialize(memory, &acc); err != nil {
		log.Error(errors.New("failed to deserialize account memory"))
	} else {
		balance = acc.Balance
	}
	return balance
}

func accountRef(act *record.Activate) insolar.Reference {
	memory := act.Memory

	if memory == nil {
		log.Warn(errors.New("wallet memory is nil"))
		return insolar.Reference{}
	}

	wlt := wallet.Wallet{}
	if err := insolar.Deserialize(memory, &wlt); err != nil {
		log.Error(errors.New("failed to deserialize wallet memory"))
		return insolar.Reference{}
	}

	walletRef, err := insolar.NewReferenceFromString(wlt.Accounts["XNS"])
	if err != nil {
		panic("SOMETHING WENT WRONG: can't create reference from string")
	}

	return *walletRef
}

func walletRef(act *record.Activate) insolar.Reference {
	memory := act.Memory

	if memory == nil {
		log.Warn(errors.New("failed to deserialize member memory"))
		return insolar.Reference{}
	}

	mbr := member.Member{}
	if err := insolar.Deserialize(memory, &mbr); err != nil {
		log.Error(errors.New("failed to deserialize member memory")) // TODO
		return insolar.Reference{}
	}

	return mbr.Wallet
}

func ParseResultPayload(res *record.Result) (foundation.Result, error) {
	var firstValue interface{}
	var contractErr *foundation.Error
	requestErr, err := foundation.UnmarshalMethodResult(res.Payload, &firstValue, &contractErr)

	if err != nil {
		return foundation.Result{}, errors.Wrap(err, "failed to unmarshal result payload")
	}

	result := foundation.Result{
		Error:   requestErr,
		Returns: []interface{}{firstValue, contractErr},
	}
	return result, nil
}

func ParseFirstValueResult(res *record.Result, v interface{}) {
	result, err := ParseResultPayload(res)
	if err != nil {
		return
	}
	returns := result.Returns
	data, err := json.Marshal(returns[0])
	if err != nil {
		log.Warn("failed to marshal Payload.Returns[0]")
		debug.PrintStack()
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		log.WithField("json", string(data)).Warn("failed to unmarshal Payload.Returns[0]")
		debug.PrintStack()
	}
}
