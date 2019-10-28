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

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/insolar/application/builtin/contract/member"
	proxyAccount "github.com/insolar/insolar/application/builtin/proxy/account"
	proxyCostCenter "github.com/insolar/insolar/application/builtin/proxy/costcenter"
	proxyMember "github.com/insolar/insolar/application/builtin/proxy/member"
	proxyWallet "github.com/insolar/insolar/application/builtin/proxy/wallet"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"
)

const (
	TransferMethodName     = "Transfer"
	StandardTransferMethod = "member.transfer"
)

type StandardTransferCollector struct {
	log *logrus.Logger

	fetcher store.RecordFetcher
	builder tree.Builder
}

func NewStandardTransferCollector(log *logrus.Logger, fetcher store.RecordFetcher, builder tree.Builder) *StandardTransferCollector {
	c := &StandardTransferCollector{
		log:     log,
		fetcher: fetcher,
		builder: builder,
	}
	return c
}

func (c *StandardTransferCollector) Collect(ctx context.Context, rec *observer.Record) *observer.Transfer {
	if rec == nil {
		return nil
	}

	result := observer.CastToResult(rec)
	if !result.IsResult() {
		return nil
	}

	req, err := c.fetcher.Request(ctx, result.Request())
	if err != nil {
		c.log.WithField("req", result.Request()).
			Error(errors.Wrapf(err, "result without request"))
		return nil
	}
	call, ok := c.isTransferCall(&req)
	if !ok {
		return nil
	}

	if !result.IsSuccess() {
		return c.makeFailedTransfer(call)
	}

	callTree, err := c.builder.Build(ctx, req.ID)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to build call tree call "))
		return c.build(call, result, nil, nil, nil, nil)
	}

	walletTransferStructure, err := c.find(callTree.Outgoings, c.isWalletTransfer)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to find wallet.Transfer call in outgouings of member.Call"))
		return c.build(call, result, nil, nil, nil, nil)
	}

	accountTransferStructure, err := c.find(walletTransferStructure.Outgoings, c.isAccountTransfer)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to find account.Transfer call in outgouings of wallet.Transfer"))
		return c.build(call, result, walletTransferStructure, nil, nil, nil)
	}

	calcFeeCallTree, err := c.find(accountTransferStructure.Outgoings, c.isCalcFee)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to find costcenter.CalcFee call in outgouings of account.Transfer"))
		return c.build(call, result, walletTransferStructure, accountTransferStructure, nil, nil)
	}

	getFeeMemberCallTree, err := c.find(accountTransferStructure.Outgoings, c.isGetFeeMember)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to find costcenter.GetFeeMember call in outgouings of account.Transfer"))
		return c.build(call, result, walletTransferStructure, accountTransferStructure, calcFeeCallTree, nil)
	}

	return c.build(
		call,
		result,
		walletTransferStructure,
		accountTransferStructure,
		calcFeeCallTree,
		getFeeMemberCallTree,
	)
}

func (c *StandardTransferCollector) find(outs []tree.Outgoing, predicate func(*record.IncomingRequest) bool) (*tree.Structure, error) {
	for _, req := range outs {
		if req.Structure == nil {
			continue
		}

		if predicate(&req.Structure.Request) {
			return req.Structure, nil
		}
	}
	return nil, errors.New("failed to find corresponding request in calls tree")
}

func (c *StandardTransferCollector) makeFailedTransfer(apiCall *observer.Request) *observer.Transfer {
	from, to, amount := c.parseCall(apiCall)
	return &observer.Transfer{
		TxID:      apiCall.ID,
		From:      from,
		To:        to,
		Amount:    amount,
		Status:    observer.Failed,
		Kind:      observer.Standard,
		Direction: observer.APICall,
	}
}

func (c *StandardTransferCollector) build(
	apiCall *observer.Request,
	result *observer.Result,
	wallet *tree.Structure, // nolint: unparam
	account *tree.Structure,
	calc *tree.Structure, // nolint: unparam
	getFeeMember *tree.Structure, // nolint: unparam
) *observer.Transfer {

	from, to, amount := c.parseCall(apiCall)
	resultValue := &member.TransferResponse{Fee: "0"}
	result.ParseFirstPayloadValue(resultValue)

	// TODO: fill details field from call tree info
	// costCenterRef := calc.Request.Object
	// costCenter := insolar.ID{}
	// if costCenterRef != nil {
	// 	costCenter = *costCenterRef.GetLocal()
	// }
	//
	// var serializedRef []byte
	// observer.ParseFirstValueResult(&getFeeMember.Result, &serializedRef)
	// feeMemberRef := insolar.NewReferenceFromBytes(serializedRef)
	// feeMember := *feeMemberRef.GetLocal()
	var detachRequest *insolar.ID
	if account != nil {
		detachRequest = &account.RequestID
	}
	return &observer.Transfer{
		TxID:          apiCall.ID,
		From:          from,
		To:            to,
		Amount:        amount,
		Fee:           resultValue.Fee,
		Status:        observer.Success,
		Kind:          observer.Standard,
		Direction:     observer.APICall,
		DetachRequest: detachRequest,
		// TransferRequestMember:  apiCall.ID,
		// TransferRequestWallet:  wallet.RequestID,
		// TransferRequestAccount: account.RequestID,
		// CalcFeeRequest:         calc.RequestID,
		// FeeMemberRequest:       getFeeMember.RequestID,
		// CostCenterRef:          costCenter,
		// FeeMemberRef:           feeMember,
	}
}

func (c *StandardTransferCollector) parseCall(apiCall *observer.Request) (*insolar.ID, *insolar.ID, string) {
	callArguments := apiCall.ParseMemberCallArguments()
	callParams := &TransferCallParams{}
	apiCall.ParseMemberContractCallParams(callParams)
	from, err := insolar.NewIDFromString(callArguments.Params.Reference)
	if err != nil {
		c.log.Error("invalid callArguments.Params.Reference")
	}
	to, err := insolar.NewIDFromString(callParams.ToMemberReference)
	if err != nil {
		c.log.Error("invalid callParams.ToMemberReference")
	}
	return from, to, callParams.Amount
}

func (c *StandardTransferCollector) isTransferCall(rec *record.Material) (*observer.Request, bool) {
	request := observer.CastToRequest((*observer.Record)(rec))

	if !request.IsIncoming() {
		return nil, false
	}

	if !request.IsMemberCall() {
		return nil, false
	}

	args := request.ParseMemberCallArguments()
	return request, args.Params.CallSite == StandardTransferMethod
}

func (c *StandardTransferCollector) isGetFeeMember(req *record.IncomingRequest) bool {
	if req.Method != "GetFeeMember" {
		return false
	}
	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyCostCenter.PrototypeReference)
}

func (c *StandardTransferCollector) isAccountTransfer(req *record.IncomingRequest) bool {
	if req.Method != TransferMethodName {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyAccount.PrototypeReference)
}

// nolint: unused
func (c *StandardTransferCollector) isMemberAccept(req *record.IncomingRequest) bool {
	if req.Method != "Accept" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyMember.PrototypeReference)
}

func (c *StandardTransferCollector) isCalcFee(req *record.IncomingRequest) bool {
	if req.Method != "CalcFee" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyCostCenter.PrototypeReference)
}

func (c *StandardTransferCollector) isWalletTransfer(req *record.IncomingRequest) bool {
	if req.Method != TransferMethodName {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyWallet.PrototypeReference)
}
