// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/google/uuid"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/mainnet/application/appfoundation"
	"github.com/insolar/mainnet/application/builtin/contract/member"
	proxyDeposit "github.com/insolar/mainnet/application/builtin/proxy/deposit"
	proxyMember "github.com/insolar/mainnet/application/builtin/proxy/member"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/models"
)

func TestTxRegisterCollector_Collect(t *testing.T) {
	ctx := context.Background()
	c := NewTxRegisterCollector(inslogger.FromContext(ctx))

	t.Run("transfer happy path", func(t *testing.T) {
		txID := *insolar.NewRecordReference(gen.ID())
		memberFrom := gen.Reference()
		memberTo := gen.Reference()
		expectedTx := observer.TxRegister{
			TransactionID:       txID,
			Type:                models.TTypeTransfer,
			PulseNumber:         int64(txID.GetLocal().Pulse()),
			RecordNumber:        int64(rand.Int31()),
			MemberFromReference: memberFrom.Bytes(),
			MemberToReference:   memberTo.Bytes(),
			DepositToReference:  nil,
			Amount:              "123",
		}
		request := member.Request{
			Params: member.Params{
				Reference: memberFrom.String(),
				CallSite:  callSiteTransfer,
				CallParams: map[string]interface{}{
					paramToMemberRef: memberTo.String(),
					paramAmount:      expectedTx.Amount,
				},
			},
		}
		encodedRequest, err := json.Marshal(&request)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)

		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.IncomingRequest{
					Method:     methodCall,
					APINode:    gen.Reference(),
					ReturnMode: record.ReturnResult,
					Arguments:  arguments,
					Prototype:  proxyMember.PrototypeReference,
				}),
				ID: *txID.GetLocal(),
			},
			RecordNumber: uint32(expectedTx.RecordNumber),
		}
		tx := c.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("transfer with bad MemberToReference", func(t *testing.T) {
		txID := *insolar.NewRecordReference(gen.ID())
		memberFrom := gen.Reference()
		memberTo := uuid.New()
		expectedTx := observer.TxRegister{
			TransactionID:       txID,
			Type:                models.TTypeTransfer,
			PulseNumber:         int64(txID.GetLocal().Pulse()),
			RecordNumber:        int64(rand.Int31()),
			MemberFromReference: memberFrom.Bytes(),
			DepositToReference:  nil,
			Amount:              "123",
		}
		request := member.Request{
			Params: member.Params{
				Reference: memberFrom.String(),
				CallSite:  callSiteTransfer,
				CallParams: map[string]interface{}{
					paramToMemberRef: memberTo.String(),
					paramAmount:      expectedTx.Amount,
				},
			},
		}
		encodedRequest, err := json.Marshal(&request)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)

		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.IncomingRequest{
					Method:     methodCall,
					APINode:    gen.Reference(),
					ReturnMode: record.ReturnResult,
					Arguments:  arguments,
					Prototype:  proxyMember.PrototypeReference,
				}),
				ID: *txID.GetLocal(),
			},
			RecordNumber: uint32(expectedTx.RecordNumber),
		}
		tx := c.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("migration with transaction type happy path", func(t *testing.T) {
		txID := *insolar.NewRecordReference(gen.ID())
		memberFrom := gen.Reference()
		memberTo := gen.Reference()
		depositTo := gen.Reference()
		expectedTx := observer.TxRegister{
			TransactionID:       txID,
			Type:                models.TTypeMigration,
			PulseNumber:         int64(txID.GetLocal().Pulse()),
			RecordNumber:        int64(rand.Int31()),
			MemberFromReference: memberFrom.Bytes(),
			MemberToReference:   memberTo.Bytes(),
			DepositToReference:  depositTo.Bytes(),
			Amount:              "123",
		}

		arguments, err := insolar.Serialize([]interface{}{
			expectedTx.Amount,
			depositTo,
			memberFrom,
			txID,
			memberTo,
			"migration",
		})
		require.NoError(t, err)
		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.IncomingRequest{
					Method:     methodTransferToDeposit,
					ReturnMode: record.ReturnResult,
					Arguments:  arguments,
					Caller:     gen.Reference(),
					Prototype:  proxyDeposit.PrototypeReference,
				}),
				ID: *txID.GetLocal(),
			},
			RecordNumber: uint32(expectedTx.RecordNumber),
		}
		tx := c.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("migration without transaction type happy path", func(t *testing.T) {
		txID := *insolar.NewRecordReference(gen.ID())
		memberFrom := gen.Reference()
		memberTo := gen.Reference()
		depositTo := gen.Reference()
		expectedTx := observer.TxRegister{
			TransactionID:       txID,
			Type:                models.TTypeMigration,
			PulseNumber:         int64(txID.GetLocal().Pulse()),
			RecordNumber:        int64(rand.Int31()),
			MemberFromReference: memberFrom.Bytes(),
			MemberToReference:   memberTo.Bytes(),
			DepositToReference:  depositTo.Bytes(),
			Amount:              "123",
		}

		arguments, err := insolar.Serialize([]interface{}{
			expectedTx.Amount,
			depositTo,
			memberFrom,
			txID,
			memberTo,
		})
		require.NoError(t, err)
		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.IncomingRequest{
					Method:     methodTransferToDeposit,
					ReturnMode: record.ReturnResult,
					Arguments:  arguments,
					Caller:     gen.Reference(),
					Prototype:  proxyDeposit.PrototypeReference,
				}),
				ID: *txID.GetLocal(),
			},
			RecordNumber: uint32(expectedTx.RecordNumber),
		}
		tx := c.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("allocation deposit to deposit happy path", func(t *testing.T) {
		txID := *insolar.NewRecordReference(gen.ID())
		memberFrom := gen.Reference()
		memberTo := gen.Reference()
		depositTo := gen.Reference()
		expectedTx := observer.TxRegister{
			TransactionID:       txID,
			Type:                models.TTypeAllocation,
			PulseNumber:         int64(txID.GetLocal().Pulse()),
			RecordNumber:        int64(rand.Int31()),
			MemberFromReference: memberFrom.Bytes(),
			MemberToReference:   memberTo.Bytes(),
			DepositToReference:  depositTo.Bytes(),
			Amount:              "123",
		}

		arguments, err := insolar.Serialize([]interface{}{
			expectedTx.Amount,
			depositTo,
			memberFrom,
			txID,
			memberTo,
			"allocation",
		})
		require.NoError(t, err)
		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.IncomingRequest{
					Method:     methodTransferToDeposit,
					ReturnMode: record.ReturnResult,
					Arguments:  arguments,
					Caller:     gen.Reference(),
					Prototype:  proxyDeposit.PrototypeReference,
				}),
				ID: *txID.GetLocal(),
			},
			RecordNumber: uint32(expectedTx.RecordNumber),
		}
		tx := c.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("release happy path", func(t *testing.T) {
		txID := *insolar.NewRecordReference(gen.ID())
		memberTo := gen.Reference()
		expectedTx := observer.TxRegister{
			TransactionID:       txID,
			Type:                models.TTypeRelease,
			PulseNumber:         int64(txID.GetLocal().Pulse()),
			RecordNumber:        int64(rand.Int31()),
			MemberFromReference: nil,
			MemberToReference:   memberTo.Bytes(),
			DepositToReference:  nil,
			Amount:              "123",
		}

		request := member.Request{
			Params: member.Params{
				Reference: memberTo.String(),
				CallSite:  callSiteRelease,
				CallParams: map[string]interface{}{
					paramAmount: expectedTx.Amount,
				},
			},
		}
		encodedRequest, err := json.Marshal(&request)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)

		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.IncomingRequest{
					Method:     methodCall,
					APINode:    gen.Reference(),
					ReturnMode: record.ReturnResult,
					Arguments:  arguments,
					Prototype:  proxyMember.PrototypeReference,
				}),
				ID: *txID.GetLocal(),
			},
			RecordNumber: uint32(expectedTx.RecordNumber),
		}
		tx := c.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("allocation happy path", func(t *testing.T) {
		txID := *insolar.NewRecordReference(gen.ID())
		memberTo := gen.Reference()
		memberFrom := gen.Reference()
		expectedTx := observer.TxRegister{
			TransactionID:       txID,
			Type:                models.TTypeAllocation,
			PulseNumber:         int64(txID.GetLocal().Pulse()),
			RecordNumber:        int64(rand.Int31()),
			MemberFromReference: memberFrom.Bytes(),
			MemberToReference:   memberTo.Bytes(),
			DepositToReference:  nil,
			Amount:              "123",
		}

		request := member.Request{
			Params: member.Params{
				Reference: memberFrom.String(),
				CallSite:  callSiteAllocation,
				CallParams: map[string]interface{}{
					paramToMemberRef: memberTo.String(),
					"toDepositName":  "genesis_deposit2",
					paramAmount:      expectedTx.Amount,
				},
			},
		}
		encodedRequest, err := json.Marshal(&request)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)

		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.IncomingRequest{
					Method:     methodCall,
					APINode:    gen.Reference(),
					ReturnMode: record.ReturnResult,
					Arguments:  arguments,
					Prototype:  proxyMember.PrototypeReference,
				}),
				ID: *txID.GetLocal(),
			},
			RecordNumber: uint32(expectedTx.RecordNumber),
		}
		tx := c.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		require.Equal(t, &expectedTx, tx)
	})

	t.Run("allocation, error wrong memberTo", func(t *testing.T) {
		txID := *insolar.NewRecordReference(gen.ID())
		memberTo := uuid.New()
		memberFrom := gen.Reference()

		request := member.Request{
			Params: member.Params{
				Reference: memberFrom.String(),
				CallSite:  callSiteAllocation,
				CallParams: map[string]interface{}{
					paramToMemberRef: memberTo,
					"toDepositName":  "genesis_deposit2",
					paramAmount:      "123",
				},
			},
		}
		encodedRequest, err := json.Marshal(&request)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)

		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.IncomingRequest{
					Method:     methodCall,
					APINode:    gen.Reference(),
					ReturnMode: record.ReturnResult,
					Arguments:  arguments,
					Prototype:  proxyMember.PrototypeReference,
				}),
				ID: *txID.GetLocal(),
			},
			RecordNumber: uint32(123),
		}
		tx := c.Collect(ctx, rec)
		require.Nil(t, tx)
	})
}

func TestTxDepositTransferCollector_Collect(t *testing.T) {
	ctx := context.Background()
	log := inslogger.FromContext(ctx)

	t.Run("deposit.transfer happy path", func(t *testing.T) {
		collector := NewTxDepositTransferCollector(log)

		txID := *insolar.NewRecordReference(gen.ID())
		memberTo := gen.Reference()
		depositFrom := insolar.NewReference(gen.ID())

		expectedTx := observer.TxDepositTransferUpdate{
			TransactionID:        txID,
			DepositFromReference: depositFrom.Bytes(),
		}

		arguments, err := insolar.Serialize([]interface{}{
			"100",
			memberTo,
			txID,
		})
		require.NoError(t, err)
		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.IncomingRequest{
					Method:     methodTransfer,
					ReturnMode: record.ReturnResult,
					Arguments:  arguments,
					Caller:     gen.Reference(),
					Prototype:  proxyDeposit.PrototypeReference,
				}),
				ID:       *txID.GetLocal(),
				ObjectID: *depositFrom.GetLocal(),
			},
			RecordNumber: uint32(rand.Int31()),
		}
		tx := collector.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &expectedTx, tx)
	})
}

func TestTxResultCollector_Collect(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	log := inslogger.FromContext(ctx)
	var (
		fetcher   *store.RecordFetcherMock
		collector *TxResultCollector
	)
	setup := func() {
		fetcher = store.NewRecordFetcherMock(mc)
		collector = NewTxResultCollector(log, fetcher)
	}

	t.Run("transfer happy path", func(t *testing.T) {
		setup()
		defer mc.Finish()

		memberRequest := member.Request{Params: member.Params{CallSite: callSiteTransfer}}
		encodedRequest, err := json.Marshal(&memberRequest)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnResult,
				Method:     methodCall,
				Arguments:  arguments,
				APINode:    gen.Reference(),
			}),
		}
		txID := *insolar.NewRecordReference(gen.ID())
		resultPayload, err := insolar.Serialize(&foundation.Result{
			Returns: []interface{}{member.TransferResponse{
				Fee: "123",
			}, nil},
		})
		require.NoError(t, err)
		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.Result{
					Request: txID,
					Payload: resultPayload,
				}),
			},
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *txID.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &observer.TxResult{TransactionID: txID, Fee: "123"}, tx)
	})

	t.Run("migration happy path", func(t *testing.T) {
		setup()
		defer mc.Finish()

		txID := *insolar.NewRecordReference(gen.ID())
		arguments, err := insolar.Serialize([]interface{}{nil, nil, nil, &txID, nil, nil})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnResult,
				Method:     methodTransferToDeposit,
				Arguments:  arguments,
				Prototype:  proxyDeposit.PrototypeReference,
			}),
		}
		require.NoError(t, err)
		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.Result{
					Request: txID,
				}),
			},
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *txID.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &observer.TxResult{TransactionID: txID, Fee: "0"}, tx)
	})

	t.Run("release happy path", func(t *testing.T) {
		setup()
		defer mc.Finish()

		memberRequest := member.Request{Params: member.Params{CallSite: callSiteRelease}}
		encodedRequest, err := json.Marshal(&memberRequest)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnResult,
				Method:     methodCall,
				Arguments:  arguments,
				APINode:    gen.Reference(),
			}),
		}
		txID := *insolar.NewRecordReference(gen.ID())
		require.NoError(t, err)
		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.Result{
					Request: txID,
				}),
			},
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *txID.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &observer.TxResult{TransactionID: txID, Fee: "0"}, tx)
	})

	t.Run("allocation happy path", func(t *testing.T) {
		setup()
		defer mc.Finish()

		memberRequest := member.Request{Params: member.Params{CallSite: callSiteAllocation}}
		encodedRequest, err := json.Marshal(&memberRequest)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnResult,
				Method:     methodCall,
				Arguments:  arguments,
				APINode:    gen.Reference(),
			}),
		}
		txID := *insolar.NewRecordReference(gen.ID())
		require.NoError(t, err)
		rec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.Result{
					Request: txID,
				}),
			},
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *txID.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		require.Equal(t, &observer.TxResult{TransactionID: txID, Fee: "0"}, tx)
	})

	t.Run("transfer with error", func(t *testing.T) {
		setup()
		defer mc.Finish()

		memberRequest := member.Request{Params: member.Params{CallSite: callSiteTransfer}}
		encodedRequest, err := json.Marshal(&memberRequest)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnResult,
				Method:     methodCall,
				Arguments:  arguments,
				APINode:    gen.Reference(),
			}),
		}
		txID := *insolar.NewRecordReference(gen.ID())
		resultID := gen.ID()
		resultPayload, err := insolar.Serialize(&foundation.Result{
			Returns: []interface{}{member.TransferResponse{
				Fee: "123",
			}, errors.New("failed")},
		})
		require.NoError(t, err)
		rec := exporter.Record{
			RecordNumber: 11,
			Record: record.Material{
				ID: resultID,
				Virtual: record.Wrap(&record.Result{
					Request: txID,
					Payload: resultPayload,
				}),
			},
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *txID.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &observer.TxResult{TransactionID: txID, Fee: "123", Failed: &observer.TxFailed{
			FinishPulseNumber:  int64(rec.Record.ID.Pulse()),
			FinishRecordNumber: int64(rec.RecordNumber),
		}}, tx)
	})
}

func TestTxSagaResultCollector_Collect(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	log := inslogger.FromContext(ctx)
	var (
		fetcher   *store.RecordFetcherMock
		collector *TxSagaResultCollector
	)
	setup := func() {
		fetcher = store.NewRecordFetcherMock(mc)
		collector = NewTxSagaResultCollector(log, fetcher)
	}

	t.Run("saga success", func(t *testing.T) {
		setup()
		defer mc.Finish()

		txID := *insolar.NewRecordReference(gen.ID())
		arguments, err := insolar.Serialize([]interface{}{
			appfoundation.SagaAcceptInfo{Request: txID},
		})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnSaga,
				Method:     methodAccept,
				Arguments:  arguments,
				APINode:    gen.Reference(),
			}),
		}
		requestRef := gen.Reference()

		resultPayload, err := insolar.Serialize(&foundation.Result{
			Returns: []interface{}{nil},
		})
		require.NoError(t, err)
		resultRec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.Result{
					Request: requestRef,
					Payload: resultPayload,
				}),
				ID: gen.ID(),
			},
			RecordNumber: rand.Uint32(),
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *requestRef.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, resultRec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		expectedTx := observer.TxSagaResult{
			TransactionID:      txID,
			FinishSuccess:      true,
			FinishPulseNumber:  int64(resultRec.Record.ID.Pulse()),
			FinishRecordNumber: int64(resultRec.RecordNumber),
		}
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("saga fail", func(t *testing.T) {
		setup()
		defer mc.Finish()

		txID := *insolar.NewRecordReference(gen.ID())
		arguments, err := insolar.Serialize([]interface{}{
			appfoundation.SagaAcceptInfo{Request: txID},
		})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnSaga,
				Method:     methodAccept,
				Arguments:  arguments,
				APINode:    gen.Reference(),
			}),
		}
		requestRef := gen.Reference()

		resultPayload, err := insolar.Serialize(&foundation.Result{
			Returns: []interface{}{&foundation.Error{S: "test error"}},
		})
		require.NoError(t, err)
		resultRec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.Result{
					Request: requestRef,
					Payload: resultPayload,
				}),
				ID: gen.ID(),
			},
			RecordNumber: rand.Uint32(),
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *requestRef.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, resultRec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		expectedTx := observer.TxSagaResult{
			TransactionID:      txID,
			FinishSuccess:      false,
			FinishPulseNumber:  int64(resultRec.Record.ID.Pulse()),
			FinishRecordNumber: int64(resultRec.RecordNumber),
		}
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("call success does not produce tx", func(t *testing.T) {
		setup()
		defer mc.Finish()

		memberRequest := member.Request{Params: member.Params{CallSite: callSiteTransfer}}
		encodedRequest, err := json.Marshal(&memberRequest)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnResult,
				Method:     methodCall,
				Arguments:  arguments,
				APINode:    gen.Reference(),
				Prototype:  proxyMember.PrototypeReference,
			}),
		}
		txID := *insolar.NewRecordReference(gen.ID())

		resultPayload, err := insolar.Serialize(&foundation.Result{
			Returns: []interface{}{nil, nil},
		})
		require.NoError(t, err)
		resultRec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.Result{
					Request: txID,
					Payload: resultPayload,
				}),
				ID: gen.ID(),
			},
			RecordNumber: rand.Uint32(),
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *txID.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, resultRec)
		require.Nil(t, tx)
	})

	t.Run("call fail", func(t *testing.T) {
		setup()
		defer mc.Finish()

		memberRequest := member.Request{Params: member.Params{CallSite: callSiteTransfer}}
		encodedRequest, err := json.Marshal(&memberRequest)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnResult,
				Method:     methodCall,
				Arguments:  arguments,
				APINode:    gen.Reference(),
			}),
		}
		txID := *insolar.NewRecordReference(gen.ID())

		resultPayload, err := insolar.Serialize(&foundation.Result{
			Returns: []interface{}{nil, &foundation.Error{S: "test error"}},
		})
		require.NoError(t, err)
		resultRec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.Result{
					Request: txID,
					Payload: resultPayload,
				}),
				ID: gen.ID(),
			},
			RecordNumber: rand.Uint32(),
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *txID.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, resultRec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		expectedTx := observer.TxSagaResult{
			TransactionID:      txID,
			FinishSuccess:      false,
			FinishPulseNumber:  int64(resultRec.Record.ID.Pulse()),
			FinishRecordNumber: int64(resultRec.RecordNumber),
		}
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("call fail allocation", func(t *testing.T) {
		setup()
		defer mc.Finish()

		memberRequest := member.Request{Params: member.Params{CallSite: callSiteAllocation}}
		encodedRequest, err := json.Marshal(&memberRequest)
		require.NoError(t, err)
		signedRequest, err := insolar.Serialize([]interface{}{encodedRequest, nil, nil})
		require.NoError(t, err)
		arguments, err := insolar.Serialize([]interface{}{&signedRequest})
		require.NoError(t, err)
		request := record.Material{
			Virtual: record.Wrap(&record.IncomingRequest{
				ReturnMode: record.ReturnResult,
				Method:     methodCall,
				Arguments:  arguments,
				APINode:    gen.Reference(),
			}),
		}
		txID := *insolar.NewRecordReference(gen.ID())

		resultPayload, err := insolar.Serialize(&foundation.Result{
			Returns: []interface{}{nil, &foundation.Error{S: "test error"}},
		})
		require.NoError(t, err)
		resultRec := exporter.Record{
			Record: record.Material{
				Virtual: record.Wrap(&record.Result{
					Request: txID,
					Payload: resultPayload,
				}),
				ID: gen.ID(),
			},
			RecordNumber: rand.Uint32(),
		}

		fetcher.RequestMock.Inspect(func(_ context.Context, reqID insolar.ID) {
			require.Equal(t, *txID.GetLocal(), reqID)
		}).Return(request, nil)

		tx := collector.Collect(ctx, resultRec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		expectedTx := observer.TxSagaResult{
			TransactionID:      txID,
			FinishSuccess:      false,
			FinishPulseNumber:  int64(resultRec.Record.ID.Pulse()),
			FinishRecordNumber: int64(resultRec.RecordNumber),
		}
		assert.Equal(t, &expectedTx, tx)
	})
}

func TestCborSerializationDeserialization(t *testing.T) {
	var amount, txType string

	arguments, err := insolar.Serialize([]interface{}{"100"})
	require.NoError(t, err)
	err = insolar.Deserialize(arguments, []interface{}{&amount, &txType})
	require.NoError(t, err)
	require.Equal(t, "", txType)

	arguments, err = insolar.Serialize([]interface{}{"100", "migration"})
	require.NoError(t, err)
	err = insolar.Deserialize(arguments, []interface{}{&amount, &txType})
	require.NoError(t, err)
	require.Equal(t, "migration", txType)
}
