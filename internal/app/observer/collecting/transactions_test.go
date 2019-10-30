package collecting

import (
	"context"
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/models"
)

func TestTxRegisterCollector_Collect(t *testing.T) {
	ctx := context.Background()
	c := NewTxRegisterCollector()

	t.Run("transfer happy path", func(t *testing.T) {
		txID := gen.ID()
		memberFrom := gen.Reference()
		memberTo := gen.Reference()
		expectedTx := observer.TxRegister{
			TransactionID:        insolar.NewReference(txID).Bytes(),
			Type:                 models.TTypeTransfer,
			PulseNumber:          int64(txID.Pulse()),
			RecordNumber:         int64(rand.Int31()),
			MemberFromReference:  memberFrom.Bytes(),
			MemberToReference:    memberTo.Bytes(),
			DepositToReference:   nil,
			DepositFromReference: nil,
			Amount:               "123",
		}
		request := member.Request{
			Params: member.Params{
				Reference: memberFrom.String(),
				CallSite:  txTransfer,
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
				}),
				ID: txID,
			},
			RecordNumber: uint32(expectedTx.RecordNumber),
		}
		tx := c.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &expectedTx, tx)
	})

	t.Run("migration happy path", func(t *testing.T) {
		txID := gen.Reference()
		memberFrom := gen.Reference()
		memberTo := gen.Reference()
		depositTo := gen.Reference()
		expectedTx := observer.TxRegister{
			TransactionID:        txID.Bytes(),
			Type:                 models.TTypeMigration,
			PulseNumber:          int64(txID.GetLocal().Pulse()),
			RecordNumber:         int64(rand.Int31()),
			MemberFromReference:  memberFrom.Bytes(),
			MemberToReference:    memberTo.Bytes(),
			DepositToReference:   depositTo.Bytes(),
			DepositFromReference: nil,
			Amount:               "123",
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
		txID := gen.Reference()
		memberTo := gen.Reference()
		depositFrom := insolar.NewReference(gen.ID())
		expectedTx := observer.TxRegister{
			TransactionID:        txID.Bytes(),
			Type:                 models.TTypeRelease,
			PulseNumber:          int64(txID.GetLocal().Pulse()),
			RecordNumber:         int64(rand.Int31()),
			MemberFromReference:  nil,
			MemberToReference:    memberTo.Bytes(),
			DepositFromReference: depositFrom.Bytes(),
			DepositToReference:   nil,
			Amount:               "123",
		}

		arguments, err := insolar.Serialize([]interface{}{
			expectedTx.Amount,
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
				}),
				ID:       *txID.GetLocal(),
				ObjectID: *depositFrom.GetLocal(),
			},
			RecordNumber: uint32(expectedTx.RecordNumber),
		}
		tx := c.Collect(ctx, rec)
		require.NotNil(t, tx)
		require.NoError(t, tx.Validate())
		assert.Equal(t, &expectedTx, tx)
	})
}
