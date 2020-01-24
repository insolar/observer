//
// Copyright 2020 Insolar Technologies GmbH
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

	"github.com/google/uuid"
	"github.com/insolar/insolar/application/appfoundation"
	"github.com/insolar/insolar/application/builtin/contract/member"
	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	proxyMember "github.com/insolar/insolar/application/builtin/proxy/member"
	"github.com/insolar/insolar/application/genesisrefs"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/models"
)

const (
	callSiteTransfer = "member.transfer"
	callSiteRelease  = "deposit.transfer"
)

const (
	methodCall              = "Call"
	methodTransferToDeposit = "TransferToDeposit"
	methodTransfer          = "Transfer"
	methodAccept            = "Accept"
)

const (
	paramAmount      = "amount"
	paramToMemberRef = "toMemberReference"
)

type TxRegisterCollector struct {
	log insolar.Logger
}

func NewTxRegisterCollector(log insolar.Logger) *TxRegisterCollector {
	return &TxRegisterCollector{
		log: log,
	}
}

func (c *TxRegisterCollector) Collect(ctx context.Context, rec exporter.Record) *observer.TxRegister {
	log := c.log.WithFields(
		map[string]interface{}{
			"collector":          "TxRegisterCollector",
			"record_id":          rec.Record.ID.DebugString(),
			"collect_process_id": uuid.New(),
		})

	log.Debug("received record")
	defer log.Debug("record processed")

	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		log.Debug("skipped (not IncomingRequest)")
		return nil
	}

	log.Debug("parsing method ", request.Method)
	var tx *observer.TxRegister
	switch request.Method {
	case methodCall:
		tx = c.fromCall(log, rec)
	case methodTransferToDeposit:
		tx = c.fromMigration(log, rec)
	default:
		return nil
	}
	if tx == nil {
		return nil
	}
	if err := tx.Validate(); err != nil {
		log.Error(errors.Wrap(err, "invalid transaction received"))
		return nil
	}
	return tx
}

func (c *TxRegisterCollector) fromCall(log insolar.Logger, rec exporter.Record) *observer.TxRegister {
	txID := *insolar.NewRecordReference(rec.Record.ID)
	log = log.WithField("tx_id", txID.GetLocal().DebugString())

	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		log.Debug("skipped (not IncomingRequest)")
		return nil
	}

	// Skip non-member objects.
	if request.Prototype != nil && !request.Prototype.Equal(*proxyMember.PrototypeReference) {
		log.Debugf("skipped (not member object)")
		return nil
	}

	if request.Method != methodCall {
		log.Debug("skipped (not Call method)")
		return nil
	}

	// Skip internal calls.
	if request.APINode.IsEmpty() {
		log.Debug("skipped (APINode is empty)")
		return nil
	}

	// Skip saga.
	if request.IsDetachedCall() {
		log.Debug("skipped (saga)")
		return nil
	}

	args, callParams, err := parseExternalArguments(request.Arguments)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse arguments"))
		return nil
	}

	var res *observer.TxRegister
	switch {
	case args.Params.CallSite == callSiteTransfer:
		memberFrom, err := insolar.NewObjectReferenceFromString(args.Params.Reference)
		if err != nil {
			log.Error(errors.Wrap(err, "failed to parse from reference"))
			return nil
		}

		amount, ok := callParams[paramAmount].(string)
		if !ok {
			log.Errorf("not found %s in transaction callParams", paramAmount)
			return nil
		}

		res = &observer.TxRegister{
			Type:                models.TTypeTransfer,
			TransactionID:       txID,
			PulseNumber:         int64(rec.Record.ID.Pulse()),
			RecordNumber:        int64(rec.RecordNumber),
			Amount:              amount,
			MemberFromReference: memberFrom.Bytes(),
		}

		toMemberStr, ok := callParams[paramToMemberRef].(string)
		if !ok {
			log.Errorf("not found %s in transaction callParams", paramToMemberRef)
			return nil
		}

		memberTo, err := insolar.NewObjectReferenceFromString(toMemberStr)
		if err != nil {
			log.Error(errors.Wrap(err, "failed to parse to reference"))
		} else {
			res.MemberToReference = memberTo.Bytes()
		}
	case args.Params.CallSite == callSiteRelease:
		memberTo, err := insolar.NewObjectReferenceFromString(args.Params.Reference)
		if err != nil {
			log.Error(errors.Wrap(err, "failed to parse from reference"))
			return nil
		}

		amount, ok := callParams[paramAmount].(string)
		if !ok {
			log.Errorf("not found %s in transaction callParams", paramAmount)
			return nil
		}

		res = &observer.TxRegister{
			Type:              models.TTypeRelease,
			TransactionID:     txID,
			PulseNumber:       int64(rec.Record.ID.Pulse()),
			RecordNumber:      int64(rec.RecordNumber),
			Amount:            amount,
			MemberToReference: memberTo.Bytes(),
		}
	default:
		log.Debug("skipped (request callSite is not parsable)")
		return nil
	}

	log.Debug("created TxRegister")
	return res
}

func (c *TxRegisterCollector) fromMigration(log insolar.Logger, rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		log.Debug("skipped (not IncomingRequest)")
		return nil
	}

	// Skip non-deposit objects.
	if request.Prototype == nil || !request.Prototype.Equal(*proxyDeposit.PrototypeReference) {
		log.Debug("skipped (not deposit object)")
		return nil
	}

	if request.Method != methodTransferToDeposit {
		log.Debug("skipped (not TransferToDeposit method)")
		return nil
	}

	// Skip external calls.
	if request.Caller.IsEmpty() {
		log.Debug("skipped (Caller is empty)")
		return nil
	}

	var (
		amount                                string
		txID, toDeposit, fromMember, toMember insolar.Reference
	)
	err := insolar.Deserialize(request.Arguments, []interface{}{&amount, &toDeposit, &fromMember, &txID, &toMember})
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse arguments"))
		return nil
	}

	// Ensure txID is record reference so other collectors can match it.
	txID = *insolar.NewRecordReference(*txID.GetLocal())

	log = log.WithField("tx_id", txID.GetLocal().DebugString())
	log.Debug("created TxRegister")
	return &observer.TxRegister{
		Type:                models.TTypeMigration,
		TransactionID:       txID,
		PulseNumber:         int64(rec.Record.ID.Pulse()),
		RecordNumber:        int64(rec.RecordNumber),
		MemberFromReference: fromMember.Bytes(),
		MemberToReference:   toMember.Bytes(),
		DepositToReference:  toDeposit.Bytes(),
		Amount:              amount,
	}
}

type TxDepositTransferCollector struct {
	log insolar.Logger
}

func NewTxDepositTransferCollector(log insolar.Logger) *TxDepositTransferCollector {
	return &TxDepositTransferCollector{
		log: log,
	}
}

func (c *TxDepositTransferCollector) Collect(ctx context.Context, rec exporter.Record) *observer.TxDepositTransferUpdate {
	log := c.log.WithFields(
		map[string]interface{}{
			"collector":          "TxDepositTransferCollector",
			"record_id":          rec.Record.ID.DebugString(),
			"collect_process_id": uuid.New(),
		})

	log.Debug("received record")
	defer log.Debug("record processed")

	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		log.Debug("skipped (not IncomingRequest)")
		return nil
	}

	// Skip non-deposit objects.
	if request.Prototype == nil || !request.Prototype.Equal(*proxyDeposit.PrototypeReference) {
		log.Debug("skipped (not deposit object)")
		return nil
	}

	if request.Method != methodTransfer {
		log.Debug("skipped (not a Transfer)")
		return nil
	}

	// Skip external calls.
	if request.Caller.IsEmpty() {
		log.Debug("skipped (Caller is empty)")
		return nil
	}

	var (
		amount         string
		txID, toMember insolar.Reference
	)
	err := insolar.Deserialize(request.Arguments, []interface{}{&amount, &toMember, &txID})
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse arguments"))
		return nil
	}

	// Ensure txID is record reference so other collectors can match it.
	txID = *insolar.NewRecordReference(*txID.GetLocal())

	log = log.WithField("tx_id", txID.GetLocal().DebugString())
	log.Debug("created TxRegister")
	return &observer.TxDepositTransferUpdate{
		TransactionID:        txID,
		DepositFromReference: insolar.NewReference(rec.Record.ObjectID).Bytes(),
	}
}

func parseExternalArguments(in []byte) (member.Request, map[string]interface{}, error) {
	if in == nil {
		return member.Request{}, nil, nil
	}
	var signedRequest []byte
	err := insolar.Deserialize(in, []interface{}{&signedRequest})
	if err != nil {
		return member.Request{}, nil, err
	}

	if len(signedRequest) == 0 {
		return member.Request{}, nil, errors.New("failed to parse signed request")
	}
	request := member.Request{}
	{
		var encodedRequest []byte
		// IMPORTANT: argument number should match serialization. This is why we use nil as second and third values.
		err = insolar.Deserialize(signedRequest, []interface{}{&encodedRequest, nil, nil})
		if err != nil {
			return member.Request{}, nil, errors.Wrapf(err, "failed to unmarshal params")
		}
		err = json.Unmarshal(encodedRequest, &request)
		if err != nil {
			return member.Request{}, nil, errors.Wrapf(err, "failed to unmarshal json member request")
		}
	}

	if request.Params.CallParams == nil {
		return request, nil, nil
	}

	callParams, ok := request.Params.CallParams.(map[string]interface{})
	if !ok {
		return member.Request{}, nil, errors.New("failed to decode CallParams")
	}
	return request, callParams, nil
}

type TxResultCollector struct {
	fetcher store.RecordFetcher
	log     insolar.Logger
}

func NewTxResultCollector(log insolar.Logger, fetcher store.RecordFetcher) *TxResultCollector {
	return &TxResultCollector{
		fetcher: fetcher,
		log:     log,
	}
}

func (c *TxResultCollector) Collect(ctx context.Context, rec exporter.Record) *observer.TxResult {
	log := c.log.WithFields(
		map[string]interface{}{
			"collector":          "TxResultCollector",
			"record_id":          rec.Record.ID.DebugString(),
			"collect_process_id": uuid.New(),
		})
	log.Debug("received record")
	defer log.Debug("record processed")

	result, ok := record.Unwrap(&rec.Record.Virtual).(*record.Result)
	if !ok {
		log.Debug("skipped (not Result)")
		return nil
	}

	// Ensure txID is record reference so other collectors can match it.
	txID := *insolar.NewRecordReference(*result.Request.GetLocal())
	log = log.WithField("tx_id", txID.GetLocal().DebugString())

	requestRecord, err := c.fetcher.Request(ctx, *txID.GetLocal())
	if err != nil {
		log.Error(errors.Wrapf(
			err,
			"failed to fetch request with id %s",
			txID.GetLocal().DebugString()),
		)
		return nil
	}

	request, ok := record.Unwrap(&requestRecord.Virtual).(*record.IncomingRequest)
	if !ok {
		log.Debug("skipped (matching request is not IncomingRequest)")
		return nil
	}

	if request.Method == methodTransferToDeposit {
		return c.fromMigration(log, *request)
	}

	if request.Method != methodCall {
		log.Debug("skipped (method is not Call)")
		return nil
	}
	// Skip non-API requests.
	if request.APINode.IsEmpty() {
		log.Debug("skipped (APINode is empty)")
		return nil
	}
	// API calls never have prototype.
	if request.Prototype != nil {
		return nil
	}

	// Skip saga.
	if request.IsDetachedCall() {
		log.Debug("skipped (request is saga)")
		return nil
	}
	args, _, err := parseExternalArguments(request.Arguments)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse request arguments"))
		return nil
	}

	// Migration and release don't have fees.
	if args.Params.CallSite == callSiteRelease {
		tx := &observer.TxResult{
			TransactionID: txID,
			Fee:           "0",
		}
		if err = tx.Validate(); err != nil {
			log.Error(errors.Wrap(err, "failed to validate transaction"))
			return nil
		}
		return tx
	}

	// Processing transfer between members. Its the only transfer that has fee.
	if args.Params.CallSite != callSiteTransfer {
		log.Debug("skipped (callSite is not Transfer)")
		return nil
	}
	response := member.TransferResponse{}
	err = insolar.Deserialize(result.Payload, &foundation.Result{
		Returns: []interface{}{&response, nil},
	})
	if err != nil {
		log.Error(errors.Wrap(err, "failed to deserialize method result"))
		return nil
	}

	var tx *observer.TxResult
	resultWrapper := observer.Result(rec.Record)
	if !resultWrapper.IsSuccess(log) {
		// failed tx
		// we need to write finish pulse because there could be no saga
		log.Debug("created failed TxResult")
		tx = &observer.TxResult{
			TransactionID: txID,
			Fee:           response.Fee,
			Failed: &observer.TxFailed{
				FinishPulseNumber:  int64(rec.Record.ID.Pulse()),
				FinishRecordNumber: int64(rec.RecordNumber),
			},
		}
	} else {
		log.Debug("created TxResult")
		tx = &observer.TxResult{
			TransactionID: txID,
			Fee:           response.Fee,
		}
	}

	if err = tx.Validate(); err != nil {
		log.Error(errors.Wrap(err, "failed to validate transaction"))
		return nil
	}
	return tx
}

func (c *TxResultCollector) fromMigration(
	log insolar.Logger,
	request record.IncomingRequest,
) *observer.TxResult {
	// Skip API requests.
	if !request.APINode.IsEmpty() {
		log.Debug("skipped (APINode is empty)")
		return nil
	}

	// Skip saga.
	if request.IsDetachedCall() {
		log.Debug("skipped (request is saga)")
		return nil
	}

	if !request.Prototype.Equal(*proxyDeposit.PrototypeReference) {
		log.Debugf("skipped (not deposit object)")
		return nil
	}

	var txID insolar.Reference
	err := insolar.Deserialize(request.Arguments, []interface{}{nil, nil, nil, &txID, nil})
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse arguments"))
		return nil
	}

	return &observer.TxResult{
		TransactionID: txID,
		Fee:           "0",
	}
}

type TxSagaResultCollector struct {
	fetcher store.RecordFetcher
	log     insolar.Logger
}

func NewTxSagaResultCollector(log insolar.Logger, fetcher store.RecordFetcher) *TxSagaResultCollector {
	return &TxSagaResultCollector{
		fetcher: fetcher,
		log:     log,
	}
}

func (c *TxSagaResultCollector) Collect(ctx context.Context, rec exporter.Record) *observer.TxSagaResult {
	log := c.log.WithFields(
		map[string]interface{}{
			"collector":          "TxSagaResultCollector",
			"record_id":          rec.Record.ID.DebugString(),
			"collect_process_id": uuid.New(),
		})
	log.Debug("received record")
	defer log.Debug("record processed")

	result := rec.Record.Virtual.GetResult()
	if result == nil {
		return nil
	}

	if rec.Record.ObjectID == *genesisrefs.ContractFeeMember.GetLocal() {
		log.Debug("skipped (fee member object)")
		return nil
	}

	log = log.WithField("request_id", result.Request.GetLocal().DebugString())

	requestRecord, err := c.fetcher.Request(ctx, *result.Request.GetLocal())
	if err != nil {
		log.Error(errors.Wrapf(
			err,
			"failed to fetch request with id %s",
			result.Request.GetLocal().DebugString()),
		)
		panic("failed to find request")
	}

	request, ok := record.Unwrap(&requestRecord.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	log.Debug("parsing method ", request.Method)
	var tx *observer.TxSagaResult
	switch request.Method {
	case methodAccept:
		tx = c.fromAccept(log, rec, *request, *result)
	case methodCall:
		tx = c.fromCall(log, rec, *request, *result)
	}
	if tx != nil {
		if err := tx.Validate(); err != nil {
			log.Error(errors.Wrap(err, "failed to validate transaction"))
			return nil
		}
	}
	return tx
}

func (c *TxSagaResultCollector) fromAccept(
	log insolar.Logger,
	resultRec exporter.Record,
	request record.IncomingRequest,
	result record.Result,
) *observer.TxSagaResult {
	// Skip non-saga.
	if !request.IsDetachedCall() {
		log.Debug("skipped (request is not saga)")
		return nil
	}

	var acceptArgs appfoundation.SagaAcceptInfo
	err := insolar.Deserialize(request.Arguments, []interface{}{&acceptArgs})
	if err != nil {
		log.Error(errors.Wrap(err, "failed to deserialize method arguments"))
		return nil
	}
	// Ensure txID is record reference so other collectors can match it.
	txID := *insolar.NewRecordReference(*acceptArgs.Request.GetLocal())
	log = log.WithField("tx_id", txID.GetLocal().DebugString())

	response := foundation.Result{}
	err = insolar.Deserialize(result.Payload, &response)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to deserialize method result"))
		return nil
	}

	if len(response.Returns) < 1 {
		log.Error(errors.Wrap(err, "unexpected number of Accept method returned parameters"))
		return nil
	}

	// The first return parameter of Accept method is error, so we check if its not nil.
	if response.Error != nil || response.Returns[0] != nil {
		log.Error("saga resulted with error")
		log.Debug("created failed TxSagaResult")
		return &observer.TxSagaResult{
			TransactionID:      txID,
			FinishSuccess:      false,
			FinishPulseNumber:  int64(resultRec.Record.ID.Pulse()),
			FinishRecordNumber: int64(resultRec.RecordNumber),
		}
	}

	log.Debug("created success TxSagaResult")
	return &observer.TxSagaResult{
		TransactionID:      txID,
		FinishSuccess:      true,
		FinishPulseNumber:  int64(resultRec.Record.ID.Pulse()),
		FinishRecordNumber: int64(resultRec.RecordNumber),
	}
}

func (c *TxSagaResultCollector) fromCall(
	log insolar.Logger,
	resultRec exporter.Record,
	request record.IncomingRequest,
	result record.Result,
) *observer.TxSagaResult {

	// Ensure txID is record reference so other collectors can match it.
	txID := *insolar.NewRecordReference(*result.Request.GetLocal())
	log = log.WithField("tx_id", txID.GetLocal().DebugString())

	// Skip non-API requests.
	if request.APINode.IsEmpty() {
		log.Debug("skipped (request APINode is empty)")
		return nil
	}
	// Skip saga.
	if request.IsDetachedCall() {
		log.Debug("skipped (request is saga)")
		return nil
	}
	args, _, err := parseExternalArguments(request.Arguments)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse request arguments"))
		return nil
	}

	isTransfer := args.Params.CallSite == callSiteTransfer
	isRelease := args.Params.CallSite == callSiteRelease
	if !isTransfer && !isRelease {
		log.Debug("skipped (request callSite is not parsable)")
		return nil
	}

	// API calls never have prototype.
	if request.Prototype != nil {
		return nil
	}

	var response foundation.Result
	err = insolar.Deserialize(result.Payload, &response)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to deserialize method result"))
		return nil
	}
	if len(response.Returns) < 2 {
		log.Error(errors.Wrap(err, "unexpected number of Call method returned parameters"))
		return nil
	}

	// The second return parameter of Call method is error, so we check if its not nil.
	if response.Error != nil || response.Returns[1] != nil {
		log.Debug("created failed TxSagaResult")
		return &observer.TxSagaResult{
			TransactionID:      txID,
			FinishSuccess:      false,
			FinishPulseNumber:  int64(resultRec.Record.ID.Pulse()),
			FinishRecordNumber: int64(resultRec.RecordNumber),
		}
	}

	// Successful call does not produce transactions since it will be produced by saga call. It avoids double insert
	// on conflict.
	return nil
}
