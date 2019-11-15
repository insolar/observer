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
	"fmt"
	"runtime/debug"

	"github.com/insolar/insolar/application/builtin/contract/account"
	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/application/builtin/contract/pkshard"
	"github.com/insolar/insolar/application/builtin/contract/wallet"
	proxyAccount "github.com/insolar/insolar/application/builtin/proxy/account"
	proxyMember "github.com/insolar/insolar/application/builtin/proxy/member"
	proxyWallet "github.com/insolar/insolar/application/builtin/proxy/wallet"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"
)

const (
	MethodNew = "New"
)

type MemberCollector struct {
	log *logrus.Logger

	fetcher store.RecordFetcher
	builder tree.Builder
}

func NewMemberCollector(
	log *logrus.Logger,
	fetcher store.RecordFetcher,
	builder tree.Builder,
) *MemberCollector {
	return &MemberCollector{
		log:     log,
		fetcher: fetcher,
		builder: builder,
	}
}

func (c *MemberCollector) Collect(ctx context.Context, rec *observer.Record) []*observer.Member {
	if rec == nil {
		return nil
	}

	// genesis member records
	if rec.ID.Pulse() == insolar.GenesisPulse.PulseNumber && isPKShardActivate(rec) {
		return c.processGenesisRecord(ctx, rec)
	}

	result := observer.CastToResult(rec) // TODO: still observer.Result
	if result == nil {
		return nil
	}

	if !result.IsSuccess() { // TODO: still observer.Result
		// TODO: what should we do with bad result records?
		return nil
	}

	requestID := result.Request()
	if requestID.IsEmpty() {
		panic(fmt.Sprintf("recordID %s: empty requestID from result", rec.ID.String()))
	}

	// Fetch root request.
	originRequest, err := c.fetcher.Request(ctx, requestID)
	if err != nil {
		panic(errors.Wrapf(err, "recordID %s: failed to fetch request", rec.ID.String()))
	}

	if !c.isMemberCreateRequest(originRequest) {
		return nil
	}

	// Build contract tree.
	memberContractTree, err := c.builder.Build(ctx, originRequest.ID)
	if err != nil {
		panic(errors.Wrapf(
			err,
			"recordID %s: failed to build contract tree for member", originRequest.ID.String(),
		))
	}

	children := memberContractTree.Outgoings
	contractResult := memberContractTree.Result

	accountTree, walletTree, memberTree := childTrees(children)

	if accountTree == nil || walletTree == nil || memberTree == nil {
		c.log.Warnf(
			"recordID %s: no children found for member creation, request: %s, result: %s",
			originRequest.ID.String(), memberContractTree.Request.String(), contractResult.String())
		return nil
	}

	balance := c.accountBalance(accountTree.SideEffect.Activation)
	walletRef := c.walletRef(memberTree.SideEffect.Activation)
	accountRef := c.accountRef(walletTree.SideEffect.Activation)

	response := c.createResponse(contractResult)

	memberRef, err := insolar.NewReferenceFromString(response.Reference)
	if err != nil || memberRef == nil {
		panic("invalid member reference")
	}

	return []*observer.Member{{
		MemberRef:        *memberRef,
		WalletRef:        walletRef,
		AccountRef:       accountRef,
		Balance:          balance,
		MigrationAddress: response.MigrationAddress,
		AccountState:     accountTree.SideEffect.ID,
		Status:           "SUCCESS",
	}}
}

func (c *MemberCollector) processGenesisRecord(ctx context.Context, rec *observer.Record) []*observer.Member {
	var (
		memberState      *member.Member
		walletState      *wallet.Wallet
		accountRefString string
		activateID       insolar.ID
		balance          string
	)
	activate := rec.Virtual.GetActivate()
	shard := c.initialPKShard(activate)
	var (
		members []*observer.Member
	)
	for _, memberRefStr := range shard.Map {
		memberRef, err := insolar.NewReferenceFromString(memberRefStr)
		if err != nil {
			c.log.WithField("member_ref_str", memberRefStr).
				Errorf("failed to build reference from string")
			continue
		}
		memberActivate, err := c.fetcher.SideEffect(ctx, *memberRef.GetLocal())
		if err != nil {
			c.log.WithField("member_ref", memberRef).
				Error("failed to find member activate record")
			continue
		}
		activate := memberActivate.Virtual.GetActivate()
		memberState = c.initialMemberState(activate)
		walletActivate, err := c.fetcher.SideEffect(ctx, *memberState.Wallet.GetLocal())
		if err != nil {
			c.log.WithField("wallet_ref", memberState.Wallet).
				Warnf("failed to find wallet activate record")
			continue
		}
		activate = walletActivate.Virtual.GetActivate()
		walletState = c.initialWalletState(activate)

		for _, value := range walletState.Accounts {
			accountRefString = value
			break
		}
		accountRef, err := insolar.NewReferenceFromString(accountRefString)
		if err != nil {
			c.log.WithField("account_ref_str", accountRefString).
				Warnf("failed to build reference from string")
			continue
		}
		if accountRef != nil {
			accountActivate, err := c.fetcher.SideEffect(ctx, *accountRef.GetLocal())
			if err != nil {
				c.log.WithField("account_ref", accountRef).
					Error("failed to find account activate record")
				continue
			}
			activateID = accountActivate.ID
			activate = accountActivate.Virtual.GetActivate()
			balance = c.accountBalance(activate)
		}

		members = append(members, &observer.Member{
			MemberRef:    *memberRef,
			WalletRef:    memberState.Wallet,
			AccountRef:   *accountRef,
			Balance:      balance,
			AccountState: activateID,
			Status:       "INTERNAL",
		})
	}
	return members
}

func (c *MemberCollector) isMemberCreateRequest(materialRequest record.Material) bool {
	incoming := materialRequest.Virtual.GetIncomingRequest()
	if incoming == nil {
		return false
	}

	if incoming.Method != "Call" {
		return false
	}

	args := incoming.Arguments

	reqParams := c.ParseMemberCallArguments(args)
	switch reqParams.Params.CallSite {
	case "member.create", "member.migrationCreate":
		return true
	}
	return false
}

func (c *MemberCollector) createResponse(result record.Result) member.MigrationCreateResponse {
	response := &member.MigrationCreateResponse{}

	c.ParseFirstValueResult(&result, response)

	return *response
}

func (c *MemberCollector) accountBalance(act *record.Activate) string {
	memory := act.Memory
	balance := ""

	if memory == nil {
		c.log.Warn(errors.New("account memory is nil"))
		return "0"
	}

	acc := account.Account{}
	if err := insolar.Deserialize(memory, &acc); err != nil {
		c.log.Error(errors.New("failed to deserialize account memory"))
	} else {
		balance = acc.Balance
	}
	return balance
}

func (c *MemberCollector) accountRef(act *record.Activate) insolar.Reference {
	memory := act.Memory

	if memory == nil {
		c.log.Warn(errors.New("wallet memory is nil"))
		return insolar.Reference{}
	}

	wlt := wallet.Wallet{}
	if err := insolar.Deserialize(memory, &wlt); err != nil {
		c.log.Error(errors.New("failed to deserialize wallet memory"))
		return insolar.Reference{}
	}

	walletRef, err := insolar.NewReferenceFromString(wlt.Accounts["XNS"])
	if err != nil {
		panic("SOMETHING WENT WRONG: can't create reference from string")
	}

	return *walletRef
}

func (c *MemberCollector) walletRef(act *record.Activate) insolar.Reference {
	memory := act.Memory

	if memory == nil {
		c.log.Warn(errors.New("failed to deserialize member memory"))
		return insolar.Reference{}
	}

	mbr := member.Member{}
	if err := insolar.Deserialize(memory, &mbr); err != nil {
		c.log.Error(errors.New("failed to deserialize member memory")) // TODO
		return insolar.Reference{}
	}

	return mbr.Wallet
}

func (c *MemberCollector) ParseResultPayload(res *record.Result) (foundation.Result, error) {
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

func (c *MemberCollector) ParseFirstValueResult(res *record.Result, v interface{}) {
	result, err := c.ParseResultPayload(res)
	if err != nil {
		return
	}
	returns := result.Returns
	data, err := json.Marshal(returns[0])
	if err != nil {
		c.log.Warn("failed to marshal Payload.Returns[0]")
		debug.PrintStack()
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		c.log.Warnf("failed to unmarshal Payload.Returns[0]: %v", string(data))
		debug.PrintStack()
	}
}

func (c *MemberCollector) ParseMemberCallArguments(rawArguments []byte) member.Request {
	var args []interface{}

	err := insolar.Deserialize(rawArguments, &args)
	if err != nil {
		c.log.Warn(errors.Wrapf(err, "failed to deserialize request arguments"))
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
			err = insolar.Deserialize(rawRequest, []interface{}{&raw, &signature, &pulseTimeStamp})
			if err != nil {
				c.log.Warn(errors.Wrapf(err, "failed to unmarshal params"))
				return member.Request{}
			}
			err = json.Unmarshal(raw, &request)
			if err != nil {
				c.log.Warn(errors.Wrapf(err, "failed to unmarshal json member request"))
				return member.Request{}
			}
		}
	}
	return request
}

func childTrees(
	children []tree.Outgoing,
) (
	accountTree *tree.Structure,
	walletTree *tree.Structure,
	memberTree *tree.Structure,
) {
	for _, child := range children {
		request := child.Structure.Request

		switch {
		case isNewAccount(request):
			accountTree = child.Structure
		case isNewWallet(request):
			walletTree = child.Structure
		case isNewMember(request):
			memberTree = child.Structure
		}
	}

	return accountTree, walletTree, memberTree
}

func isNewAccount(request record.IncomingRequest) bool {
	if request.Method != MethodNew {
		return false
	}
	if request.Prototype == nil {
		return false
	}
	return request.Prototype.Equal(*proxyAccount.PrototypeReference)
}

func isNewWallet(request record.IncomingRequest) bool {
	if request.Method != MethodNew {
		return false
	}
	if request.Prototype == nil {
		return false
	}
	return request.Prototype.Equal(*proxyWallet.PrototypeReference)
}

func isNewMember(request record.IncomingRequest) bool {
	if request.Method != MethodNew {
		return false
	}
	if request.Prototype == nil {
		return false
	}
	return request.Prototype.Equal(*proxyMember.PrototypeReference)
}

func (c *MemberCollector) initialPKShard(act *record.Activate) *pkshard.PKShard {
	shard := pkshard.PKShard{}
	err := insolar.Deserialize(act.Memory, &shard)
	if err != nil {
		c.log.Error(errors.New("failed to deserialize pkshard contract state"))
	}
	return &shard
}

func (c *MemberCollector) initialMemberState(act *record.Activate) *member.Member {
	m := member.Member{}
	err := insolar.Deserialize(act.Memory, &m)
	if err != nil {
		c.log.Error(errors.New("failed to deserialize member contract state"))
	}
	return &m
}

func (c *MemberCollector) initialWalletState(act *record.Activate) *wallet.Wallet {
	w := wallet.Wallet{}
	err := insolar.Deserialize(act.Memory, &w)
	if err != nil {
		c.log.Error(errors.New("failed to deserialize wallet contract state"))
	}
	return &w
}
