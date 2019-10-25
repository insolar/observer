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
	"encoding/base64"

	"github.com/insolar/insolar/application"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"

	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	proxyMigrationAdmin "github.com/insolar/insolar/application/builtin/proxy/migrationadmin"
	proxyDaemon "github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	"github.com/insolar/insolar/application/genesisrefs"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"
)

const (
	MigrationTransferMethod = "deposit.migration"
)

type MigrationTransferCollector struct {
	log     *logrus.Logger
	fetcher store.RecordFetcher
	builder tree.Builder
}

func NewMigrationTransferCollector(log *logrus.Logger, fetcher store.RecordFetcher, builder tree.Builder) *MigrationTransferCollector {
	c := &MigrationTransferCollector{
		log:     log,
		fetcher: fetcher,
		builder: builder,
	}
	return c
}

func (c *MigrationTransferCollector) Collect(ctx context.Context, rec *observer.Record) *observer.Transfer {
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

	callTree, err := c.builder.Build(ctx, result.Request())
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to build call tree call "))
		// TODO: if we can't to get nested calls we can't decide Is it MigrationTransfer or just one of confirmation.
		return nil
		// return c.build(call, result, nil, nil, nil, nil)
	}

	depositMigrationCallStructure, err := c.find(callTree.Outgoings, c.isDepositMigrationCall)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to find migrationdaemon.DepositMigration call in outgouings of member.Call"))
		// TODO: if we can't to get nested calls we can't decide Is it MigrationTransfer or just one of confirmation.
		return nil
		// return c.build(call, result, nil, nil, nil, nil)
	}

	getMemberStructure, err := c.find(depositMigrationCallStructure.Outgoings, c.isGetMemberByMigrationAddress)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to find migrationAdminContract.GetMemberByMigrationAddress call"+
				" in outgouings of migrationdaemon.DepositMigration"))
		// TODO: if we can't to get nested calls we can't decide Is it MigrationTransfer or just one of confirmation.
		return nil
		// return c.build(call, result, depositMigrationCallStructure, nil, nil, nil)
	}

	_, err = c.find(depositMigrationCallStructure.Outgoings, c.isDepositNew)
	if err == nil { // It is just first confirmation but not transfer.
		return nil
	}

	depositConfirmStructure, err := c.find(depositMigrationCallStructure.Outgoings, c.isDepositConfirm)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to find deposit.Confirm call in outgouings of migrationdaemon.DepositMigration"))
		// TODO: if we can't to get nested calls we can't decide Is it MigrationTransfer or just one of confirmation.
		return nil
		// return c.build(call, result, depositMigrationCallStructure, getMemberStructure, nil, nil)
	}

	transferToDepositStructure, err := c.find(depositConfirmStructure.Outgoings, c.isTransferToDeposit)
	if err != nil {
		c.log.WithField("api_call", req.ID).
			Error(errors.Wrapf(err, "failed to find deposit.TransferToDeposit call in outgouings of deposit.Confirm"))
		// TODO: if we can't to get nested calls we can't decide Is it MigrationTransfer or just one of confirmation.
		return nil
		// return c.build(call, result, depositMigrationCallStructure, getMemberStructure, depositConfirmStructure, nil)
	}

	return c.build(call, result, depositMigrationCallStructure, getMemberStructure, depositConfirmStructure, transferToDepositStructure)
}

func (c *MigrationTransferCollector) find(outs []tree.Outgoing, predicate func(*record.IncomingRequest) bool) (*tree.Structure, error) {
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

func (c *MigrationTransferCollector) makeFailedTransfer(apiCall *observer.Request) *observer.Transfer {
	memberFrom := genesisrefs.GenesisRef(application.GenesisNameMigrationAdminMember)
	amount, txHash := c.parseCall(apiCall)
	return &observer.Transfer{
		TxID:      apiCall.ID,
		From:      memberFrom.GetLocal(),
		Amount:    amount,
		Fee:       "0",
		EthHash:   txHash,
		Status:    observer.Failed,
		Kind:      observer.Migration,
		Direction: observer.APICall,
	}
}

func (c *MigrationTransferCollector) build(
	apiCall *observer.Request,
	result *observer.Result, // nolint: unparam
	depositMigration *tree.Structure, // nolint: unparam
	getMember *tree.Structure,
	depositConfirm *tree.Structure, // nolint: unparam
	transferToDeposit *tree.Structure,
) *observer.Transfer {
	var (
		memberTo *insolar.ID
	)
	amount, txHash := c.parseCall(apiCall)
	memberFrom := genesisrefs.GenesisRef(application.GenesisNameMigrationAdminMember)
	if getMember != nil {
		var refTo string
		observer.ParseFirstValueResult(&getMember.Result, &refTo)
		buf, err := base64.StdEncoding.DecodeString(refTo)
		if err != nil {
			c.log.Error(errors.Wrapf(err, "failed to deserialize memberTo reference from base64"))
		}
		ref := insolar.NewReferenceFromBytes(buf)
		if ref != nil {
			memberTo = ref.GetLocal()
			if memberTo == nil {
				c.log.Error(errors.Wrapf(err, "failed to deserialize memberTo reference from result record"))
			}
		}
	}
	var detachRequest *insolar.ID
	if transferToDeposit != nil {
		detachRequest = &transferToDeposit.RequestID
	}
	return &observer.Transfer{
		TxID:          apiCall.ID,
		From:          memberFrom.GetLocal(),
		To:            memberTo,
		Amount:        amount,
		Fee:           "0",
		EthHash:       txHash,
		Status:        observer.Success,
		Kind:          observer.Migration,
		Direction:     observer.APICall,
		DetachRequest: detachRequest,
	}
}

func (c *MigrationTransferCollector) parseCall(apiCall *observer.Request) (string, string) {
	callParams := &TransferCallParams{}
	apiCall.ParseMemberContractCallParams(callParams)
	return callParams.Amount, callParams.EthTxHash
}

func (c *MigrationTransferCollector) isTransferCall(rec *record.Material) (*observer.Request, bool) {
	request := observer.CastToRequest((*observer.Record)(rec))

	if !request.IsIncoming() {
		return nil, false
	}

	if !request.IsMemberCall() {
		return nil, false
	}

	args := request.ParseMemberCallArguments()
	return request, args.Params.CallSite == MigrationTransferMethod
}

func (c *MigrationTransferCollector) isGetMemberByMigrationAddress(req *record.IncomingRequest) bool {
	if req.Method != "GetMemberByMigrationAddress" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyMigrationAdmin.PrototypeReference)
}

func (c *MigrationTransferCollector) isDepositMigrationCall(req *record.IncomingRequest) bool {
	if req.Method != "DepositMigrationCall" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyDaemon.PrototypeReference)
}

func (c *MigrationTransferCollector) isDepositNew(req *record.IncomingRequest) bool {
	if req.Method != "New" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyDeposit.PrototypeReference)
}

func (c *MigrationTransferCollector) isDepositConfirm(req *record.IncomingRequest) bool {
	if req.Method != "Confirm" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyDeposit.PrototypeReference)
}

func (c *MigrationTransferCollector) isTransferToDeposit(req *record.IncomingRequest) bool {
	if req.Method != "TransferToDeposit" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyDeposit.PrototypeReference)
}
