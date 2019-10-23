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

	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"
)

const (
	MemberCall             = "Call"
	WithdrawTransferMethod = "deposit.transfer"
)

type WithdrawTransferCollector struct {
	log     *logrus.Logger
	fetcher store.RecordFetcher
	builder tree.Builder
}

func NewWithdrawTransferCollector(log *logrus.Logger, fetcher store.RecordFetcher, builder tree.Builder) *WithdrawTransferCollector {
	c := &WithdrawTransferCollector{
		log:     log,
		fetcher: fetcher,
		builder: builder,
	}
	return c
}

func (c *WithdrawTransferCollector) Collect(ctx context.Context, rec *observer.Record) *observer.Transfer {
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
		c.makeFailedTransfer(call)
		return nil
	}

	callTree, err := c.builder.Build(ctx, result.Request())
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to build call tree call "))
		return c.build(call, result, nil)
	}

	depositTransferStructure, err := c.find(callTree.Outgoings, c.isDepositTransfer)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to find deposit.Transfer call in outgouings of member.Call"))
		return c.build(call, result, nil)
	}

	return c.build(call, result, depositTransferStructure)
}

func (c *WithdrawTransferCollector) find(outs []tree.Outgoing, predicate func(*record.IncomingRequest) bool) (*tree.Structure, error) {
	for _, req := range outs {
		if predicate(&req.Structure.Request) {
			return req.Structure, nil
		}
	}
	return nil, errors.New("failed to find corresponding request in calls tree")
}

func (c *WithdrawTransferCollector) makeFailedTransfer(apiCall *observer.Request) *observer.Transfer {
	from, amount, ethHash := c.parseCall(apiCall)
	return &observer.Transfer{
		TxID:      apiCall.ID,
		Amount:    amount,
		From:      from,
		To:        from,
		EthHash:   ethHash,
		Status:    observer.Failed,
		Kind:      observer.Withdraw,
		Direction: observer.APICall,
	}
}

func (c *WithdrawTransferCollector) build(
	apiCall *observer.Request,
	result *observer.Result,
	depositTransfer *tree.Structure,
) *observer.Transfer {
	from, amount, ethHash := c.parseCall(apiCall)
	resultValue := &member.TransferResponse{Fee: "0"}
	result.ParseFirstPayloadValue(resultValue)

	var detachRequest *insolar.ID
	if depositTransfer != nil {
		detachRequest = &depositTransfer.RequestID
	}
	return &observer.Transfer{
		TxID:          apiCall.ID,
		Amount:        amount,
		From:          from,
		To:            from,
		Fee:           resultValue.Fee,
		EthHash:       ethHash,
		Status:        observer.Success,
		Kind:          observer.Withdraw,
		Direction:     observer.APICall,
		DetachRequest: detachRequest,
	}
}

func (c *WithdrawTransferCollector) parseCall(apiCall *observer.Request) (*insolar.ID, string, string) {
	callArguments := apiCall.ParseMemberCallArguments()
	callParams := &TransferCallParams{}
	apiCall.ParseMemberContractCallParams(callParams)
	memberFrom, err := insolar.NewIDFromString(callArguments.Params.Reference)
	if err != nil {
		c.log.Warn("invalid callArguments.Params.Reference")
	}
	return memberFrom, callParams.Amount, callParams.EthTxHash
}

func (c *WithdrawTransferCollector) isTransferCall(rec *record.Material) (*observer.Request, bool) {
	request := observer.CastToRequest((*observer.Record)(rec))
	if !request.IsIncoming() {
		return nil, false
	}

	if !request.IsMemberCall() {
		return nil, false
	}

	args := request.ParseMemberCallArguments()
	return request, args.Params.CallSite == WithdrawTransferMethod
}

func (c *WithdrawTransferCollector) isDepositTransfer(req *record.IncomingRequest) bool {
	if req.Method != "Transfer" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyDeposit.PrototypeReference)
}

type TransferCallParams struct {
	Amount            string `json:"amount"`
	ToMemberReference string `json:"toMemberReference"`
	EthTxHash         string `json:"ethTxHash"`
}
