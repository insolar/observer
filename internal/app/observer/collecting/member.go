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
	"github.com/insolar/insolar/application/builtin/contract/member/signer"
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
}

func NewMemberCollector(
	fetcher store.RecordFetcher,
	builder tree.Builder,
) *MemberCollector {
	return &MemberCollector{
		fetcher: fetcher,
		builder: builder,
	}
}

func (c *MemberCollector) Collect(ctx context.Context, rec *observer.Record) *observer.Member {
	if rec == nil {
		return nil
	}

	// TODO: check - Are we really need int?
	act := observer.CastToActivate(rec)
	if act.IsActivate() {
		if act.Virtual.GetActivate().Image.Equal(*proxyAccount.PrototypeReference) {
			if act.ID.Pulse() == insolar.GenesisPulse.PulseNumber {
				balance := accountBalance(record.Unwrap(&act.Virtual).(*record.Activate))
				return &observer.Member{
					MemberRef:    gen.ID(),
					Balance:      balance,
					AccountState: rec.ID,
					Status:       "INTERNAL",
					// TODO: Some fields
				}
			}
		}
	}

	result := observer.CastToResult(rec) // TODO: still observer.Result
	if !result.IsResult() {
		return nil
	}

	requestID := *result.Virtual.GetResult().Request.GetLocal()

	// Fetch root request.
	originRequest, err := c.fetcher.Request(ctx, requestID)
	if err != nil {
		if errors.Cause(err) != store.ErrNotFound { // TODO: What type of error here? Should we log it or return nil only?
			panic("SOMETHING WENT WRONG: can't fetch request for result")
			return nil
		}
	}

	if !isMemberCreateRequest(originRequest) {
		// TODO
		return nil
	}

	if !result.IsSuccess() { // TODO: still observer.Result
		// TODO
		return badMember()
	}

	// Build contract tree.
	memberContractTree, err := c.builder.Build(ctx, originRequest.ID)
	if err != nil {
		panic("SOMETHING WENT WRONG: can't build tree")
		return badMember()
	}

	// callRequest := memberContractTree.Request
	children := memberContractTree.Outgoings
	// sideEffect := memberContractTree.SideEffect // TODO: check is empty (because member.create doesn't has side effect)
	contractResult := memberContractTree.Result // TODO: check equality with result above (maybe)

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
			walletTree = child.Structure
		}
		if isNewMember(child.Structure.Request) {
			memberTree = child.Structure
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

func isMemberCreateRequest(materialRequest record.Material) bool {
	incoming, ok := record.Unwrap(&materialRequest.Virtual).(*record.IncomingRequest)
	if !ok {
		return false
	}

	if incoming.Method != "Call" {
		return false
	}

	args := incoming.Arguments

	reqParams := ParseMemberCallArguments(args)
	switch reqParams.Params.CallSite {
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

func createResponse(result record.Result) member.MigrationCreateResponse {
	response := &member.MigrationCreateResponse{}

	ParseFirstValueResult(&result, response)

	return *response
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

func ParseMemberCallArguments(rawArguments []byte) member.Request {
	var args []interface{}

	err := insolar.Deserialize(rawArguments, &args)
	if err != nil {
		log.Warn(errors.Wrapf(err, "failed to deserialize request arguments"))
		return member.Request{}
	}

	request := member.Request{}
	if len(args) > 0 {
		if rawRequest, ok := args[0].([]byte); ok {
			var (
				pulseTimeStamp int64
				signature      string
				raw            []byte
			)
			err = signer.UnmarshalParams(rawRequest, &raw, &signature, &pulseTimeStamp)
			if err != nil {
				log.Warn(errors.Wrapf(err, "failed to unmarshal params"))
				return member.Request{}
			}
			err = json.Unmarshal(raw, &request)
			if err != nil {
				log.Warn(errors.Wrapf(err, "failed to unmarshal json member request"))
				return member.Request{}
			}
		}
	}
	return request
}
