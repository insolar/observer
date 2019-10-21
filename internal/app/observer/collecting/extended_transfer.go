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

type ExtendedTransferCollector struct {
	log *logrus.Logger

	fetcher store.RecordFetcher
	builder tree.Builder
}

func NewExtendedTransferCollector(log *logrus.Logger, fetcher store.RecordFetcher, builder tree.Builder) *ExtendedTransferCollector {
	c := &ExtendedTransferCollector{
		log:     log,
		fetcher: fetcher,
		builder: builder,
	}
	return c
}

func (c *ExtendedTransferCollector) Collect(rec *observer.Record) *observer.ExtendedTransfer {
	if rec == nil {
		return nil
	}

	res := observer.CastToResult(rec)
	if !res.IsResult() {
		return nil
	}

	req, err := c.fetcher.Request(context.Background(), res.Request())
	if err != nil {
		panic("result without request")
	}
	call, ok := c.isTransferCall(&req)
	if !ok {
		return nil
	}

	if !res.IsSuccess() {
		return c.makeFailedTransfer(&req)
	}

	root := call
	result := res
	callTree, err := c.builder.Build(context.Background(), req.ID)
	if err != nil {
		return c.makeFailedTransfer(&req)
	}

	walletTransferStructure, err := c.find(callTree.Outgoings, c.isWalletTransfer)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to find wallet.Transfer call in outgouings of member.Call"))
		return c.makeFailedTransfer(&req)
	}

	accountTransferStructure, err := c.find(walletTransferStructure.Outgoings, c.isAccountTransfer)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to find account.Transfer call in outgouings of wallet.Transfer"))
		return c.makeFailedTransfer(&req)
	}

	calcFeeCallTree, err := c.find(accountTransferStructure.Outgoings, c.isCalcFee)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to find costcenter.CalcFee call in outgouings of account.Transfer"))
		return c.makeFailedTransfer(&req)
	}

	getFeeMemberCallTree, err := c.find(accountTransferStructure.Outgoings, c.isGetFeeMember)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to find costcenter.GetFeeMember call in outgouings of account.Transfer"))
		return c.makeFailedTransfer(&req)
	}

	memberAcceptCallTree, err := c.find(accountTransferStructure.Outgoings, c.isMemberAccept)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to find member.Accept call in outgouings of account.Transfer"))
		return c.makeFailedTransfer(&req)
	}

	transfer, err := c.build(
		root,
		result,
		walletTransferStructure,
		accountTransferStructure,
		calcFeeCallTree,
		getFeeMemberCallTree,
		memberAcceptCallTree,
	)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to build transfer"))
		return nil
	}
	return transfer
}

func (c *ExtendedTransferCollector) find(outs []tree.Outgoing, predicate func(*record.IncomingRequest) bool) (*tree.Structure, error) {
	for _, req := range outs {
		if predicate(&req.Structure.Request) {
			return req.Structure, nil
		}
	}
	return nil, errors.New("failed to find corresponding request in calls tree")
}

func (c *ExtendedTransferCollector) makeFailedTransfer(rec *record.Material) *observer.ExtendedTransfer {
	requestTime, err := rec.ID.Pulse().AsApproximateTime()
	transferTime := int64(0)
	if err == nil {
		transferTime = requestTime.Unix()
	}
	return &observer.ExtendedTransfer{
		DepositTransfer: observer.DepositTransfer{
			Transfer: observer.Transfer{
				TxID:      rec.ID,
				From:      insolar.ID{},
				To:        insolar.ID{},
				Amount:    "",
				Fee:       "",
				Timestamp: transferTime,
				Pulse:     rec.ID.Pulse(),
				Status:    "FAILED",
			},
			EthHash: "",
		},
		TransferRequestMember:  insolar.ID{},
		TransferRequestWallet:  insolar.ID{},
		TransferRequestAccount: insolar.ID{},
		AcceptRequestMember:    insolar.ID{},
		AcceptRequestWallet:    insolar.ID{},
		AcceptRequestAccount:   insolar.ID{},
		CalcFeeRequest:         insolar.ID{},
		FeeMemberRequest:       insolar.ID{},
		CostCenterRef:          insolar.ID{},
		FeeMemberRef:           insolar.ID{},
	}
}

func (c *ExtendedTransferCollector) build(
	apiCall *observer.Request,
	result *observer.Result,
	wallet *tree.Structure,
	account *tree.Structure,
	calc *tree.Structure,
	getFeeMember *tree.Structure,
	accept *tree.Structure,
) (*observer.ExtendedTransfer, error) {

	callArguments := apiCall.ParseMemberCallArguments()
	pn := apiCall.ID.Pulse()
	callParams := &transferCallParams{}
	apiCall.ParseMemberContractCallParams(callParams)
	resultValue := &member.TransferResponse{Fee: "0"}
	result.ParseFirstPayloadValue(resultValue)
	memberFrom, err := insolar.NewIDFromString(callArguments.Params.Reference)
	if err != nil {
		return nil, errors.New("invalid fromMemberReference")
	}
	memberTo := memberFrom
	if callArguments.Params.CallSite == "member.transfer" {
		memberTo, err = insolar.NewIDFromString(callParams.ToMemberReference)
		if err != nil {
			return nil, errors.New("invalid toMemberReference")
		}
	}

	transferDate, err := pn.AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert transfer pulse to time")
	}

	costCenterRef := calc.Request.Object
	costCenter := insolar.ID{}
	if costCenterRef != nil {
		costCenter = *costCenterRef.GetLocal()
	}

	feeMember := insolar.ID{}
	var serializedRef []byte
	observer.ParseFirstValueResult(&getFeeMember.Result, &serializedRef)
	feeMemberRef := insolar.NewReferenceFromBytes(serializedRef)
	feeMember = *feeMemberRef.GetLocal()
	return &observer.ExtendedTransfer{
		DepositTransfer: observer.DepositTransfer{
			Transfer: observer.Transfer{
				TxID:      apiCall.ID,
				From:      *memberFrom,
				To:        *memberTo,
				Amount:    callParams.Amount,
				Fee:       resultValue.Fee,
				Timestamp: transferDate.Unix(),
				Pulse:     pn,
				Status:    "SUCCESS",
			},
			EthHash: "",
		},
		TransferRequestMember:  apiCall.ID,
		TransferRequestWallet:  wallet.RequestID,
		TransferRequestAccount: account.RequestID,
		AcceptRequestMember:    accept.RequestID,
		CalcFeeRequest:         calc.RequestID,
		FeeMemberRequest:       getFeeMember.RequestID,
		CostCenterRef:          costCenter,
		FeeMemberRef:           feeMember,
	}, nil
}

func (c *ExtendedTransferCollector) isTransferCall(rec *record.Material) (*observer.Request, bool) {
	request := observer.CastToRequest((*observer.Record)(rec))

	if !request.IsIncoming() {
		return nil, false
	}

	if !request.IsMemberCall() {
		return nil, false
	}

	args := request.ParseMemberCallArguments()
	return request, args.Params.CallSite == "member.transfer"
}

func (c *ExtendedTransferCollector) isGetFeeMember(req *record.IncomingRequest) bool {
	if req.Method != "GetFeeMember" {
		return false
	}
	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyCostCenter.PrototypeReference)
}

func (c *ExtendedTransferCollector) isAccountTransfer(req *record.IncomingRequest) bool {
	if req.Method != "Transfer" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyAccount.PrototypeReference)
}

func (c *ExtendedTransferCollector) isMemberAccept(req *record.IncomingRequest) bool {
	if req.Method != "Accept" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyMember.PrototypeReference)
}

func (c *ExtendedTransferCollector) isCalcFee(req *record.IncomingRequest) bool {
	if req.Method != "CalcFee" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyCostCenter.PrototypeReference)
}

func (c *ExtendedTransferCollector) isWalletTransfer(req *record.IncomingRequest) bool {
	if req.Method != "Transfer" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyWallet.PrototypeReference)
}
