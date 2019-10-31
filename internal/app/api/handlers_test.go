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

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/models"
)

const (
	recordNum       = 198
	finishRecordNum = 256
	amount          = "1020"
	fee             = "178"
)

func requireEqualResponse(t *testing.T, resp *http.Response, received interface{}, expected interface{}) {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(bodyBytes, received)
	require.NoError(t, err)
	require.Equal(t, expected, received)
}

func transactionModel(txID []byte, pulseNum int64) *models.Transaction {
	return &models.Transaction{
		TransactionID:     txID,
		PulseRecord:       [2]int64{pulseNum, recordNum},
		StatusRegistered:  true,
		Amount:            amount,
		Fee:               fee,
		FinishPulseRecord: [2]int64{1, 2},
	}
}

func transactionResponse(txID string, pulseNum int64, ts float32) *observerapi.SchemasTransactionAbstract {
	return &observerapi.SchemasTransactionAbstract{
		Amount:      amount,
		Fee:         NullableString(fee),
		Index:       fmt.Sprintf("%d:%d", pulseNum, recordNum),
		PulseNumber: pulseNum,
		Status:      "pending",
		Timestamp:   ts,
		TxID:        txID,
	}
}

func TestTransaction_WrongFormat(t *testing.T) {
	txID := "123"
	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txID)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	received := &ErrorMessage{}
	expected := &ErrorMessage{Error: []string{"tx_id wrong format"}}
	requireEqualResponse(t, resp, received, expected)
}

func TestTransaction_NoContent(t *testing.T) {
	txID := gen.RecordReference().String()
	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txID)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	received := &ErrorMessage{}
	expected := &ErrorMessage{Error: []string{"empty tx_id"}}
	requireEqualResponse(t, resp, received, expected)
}

func TestTransaction_TypeMigration(t *testing.T) {
	txID := gen.RecordReference()
	pulseNumber := gen.PulseNumber()
	pntime, err := pulseNumber.AsApproximateTime()
	require.NoError(t, err)
	ts := pntime.Unix()

	fromMember := gen.Reference()
	toMember := gen.Reference()
	toDeposit := gen.Reference()

	transaction := transactionModel(txID.Bytes(), int64(pulseNumber))
	transaction.Type = models.TTypeMigration
	transaction.MemberFromReference = fromMember.Bytes()
	transaction.MemberToReference = toMember.Bytes()
	transaction.DepositToReference = toDeposit.Bytes()

	err = db.Insert(transaction)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txID.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	receivedTransaction := &SchemaMigration{}
	expectedTransaction := &SchemaMigration{
		SchemasTransactionAbstract: SchemasTransactionAbstract{
			Amount:      "10",
			Fee:         NullableString("1"),
			Index:       fmt.Sprintf("%d:%d", pulseNumber, recordNum),
			PulseNumber: int64(pulseNumber),
			Status:      "pending",
			Timestamp:   ts,
			TxID:        txID.String(),
		},
		ToMemberReference:   toMember.String(),
		FromMemberReference: fromMember.String(),
		ToDepositReference:  toDeposit.String(),
		Type:                string(models.TTypeMigration),
	}

	requireEqualResponse(t, resp, receivedTransaction, expectedTransaction)
}
