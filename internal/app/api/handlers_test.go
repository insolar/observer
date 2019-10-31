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

func transactionResponse(txID string, pulseNum int64, ts int64) *SchemasTransactionAbstract {
	return &SchemasTransactionAbstract{
		Amount:      amount,
		Fee:         NullableString(fee),
		Index:       fmt.Sprintf("%d:%d", pulseNum, recordNum),
		PulseNumber: pulseNum,
		Status:      "registered",
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
}

func TestTransaction_ClosedBadRequest(t *testing.T) {
	// if `limit` is not specified, API returns `bad request`
	resp, err := http.Get("http://" + apihost + "/api/transactions/closed")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is not a number, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/api/transactions/closed?limit=LOL")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is zero, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/api/transactions/closed?limit=0")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is negative, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/api/transactions/closed?limit=-10")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is > 1000, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/api/transactions/closed?limit=1001")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `direction` is not "before" or "after", API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/api/transactions/closed?limit=100&direction=LOL")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `index` is in a wrong format, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/api/transactions/closed?limit=100&index=LOL")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTransactions_ClosedLimitSingle(t *testing.T) {
	defer truncateDB(t)

	// insert a single closed transaction
	var err error
	txID := gen.RecordReference()
	pulseNumber := gen.PulseNumber()
	pntime, err := pulseNumber.AsApproximateTime()
	require.NoError(t, err)

	fromMember := gen.Reference()
	toMember := gen.Reference()
	toDeposit := gen.Reference()

	transaction := models.Transaction{
		TransactionID:     txID.Bytes(),
		PulseRecord:       [2]int64{int64(pulseNumber), 198},
		StatusRegistered:  true,
		Amount:            "10",
		Fee:               "1",
		FinishPulseRecord: [2]int64{1, 3001}, // keep this key unique between tests!
		Type:              models.TTypeMigration,

		MemberFromReference: fromMember.Bytes(),
		MemberToReference:   toMember.Bytes(),
		DepositToReference:  toDeposit.Bytes(),
		StatusFinished:      true,
		FinishSuccess:       true,
	}

	err = db.Insert(&transaction)
	require.NoError(t, err)

	// request one recent closed transaction using API
	resp, err := http.Get("http://" + apihost + "/api/transactions/closed?limit=1")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	expectedTransaction := SchemaMigration{
		SchemasTransactionAbstract: SchemasTransactionAbstract{
			Amount:      "10",
			Fee:         NullableString("1"),
			Index:       "1:3001", // == FinishPulseRecord
			PulseNumber: int64(pulseNumber),
			Status:      string(models.TStatusReceived),
			Timestamp:   pntime.Unix(),
			TxID:        txID.String(),
		},
		ToMemberReference:   toMember.String(),
		FromMemberReference: fromMember.String(),
		ToDepositReference:  toDeposit.String(),
		Type:                string(models.TTypeMigration),
	}

	var received []SchemaMigration
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 1)
	require.Equal(t, expectedTransaction, received[0])
}

func TestTransactions_ClosedLimitMultiple(t *testing.T) {
	defer truncateDB(t)

	var err error

	// insert two finished transactions, one with finishSuccess, second with !finishSuccess
	finishSuccessValues := []bool{true, false}
	for i := 0; i < 2; i++ {
		txID := gen.RecordReference()
		pulseNumber := gen.PulseNumber()

		fromMember := gen.Reference()
		toMember := gen.Reference()
		toDeposit := gen.Reference()

		transaction := models.Transaction{
			TransactionID:     txID.Bytes(),
			PulseRecord:       [2]int64{int64(pulseNumber), 198 + int64(i)},
			StatusRegistered:  true,
			Amount:            "10",
			Fee:               "1",
			FinishPulseRecord: [2]int64{1, 3002 + int64(i)}, // keep this key unique between tests!
			Type:              models.TTypeMigration,

			MemberFromReference: fromMember.Bytes(),
			MemberToReference:   toMember.Bytes(),
			DepositToReference:  toDeposit.Bytes(),
			StatusFinished:      true,
			FinishSuccess:       finishSuccessValues[i],
		}

		err = db.Insert(&transaction)
		require.NoError(t, err)
	}

	// Here is the order of two transactions in the database:
	// Finish pulse | Status
	// -------------+-----------
	//       1:3003 | failed
	//       1:3002 | received

	// request two recent closed transactions using API
	{
		resp, err := http.Get("http://" + apihost + "/api/transactions/closed?limit=2")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received []SchemaMigration
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, received, 2)
		// the latest transaction comes first in JSON, thus it will be `failed`
		// and the second (older) transaction in JSON will be `received`
		require.Equal(t, string(models.TStatusFailed), received[0].Status)
		require.Equal(t, string(models.TStatusReceived), received[1].Status)
	}

	// Request second (older) transaction using a cursor
	{
		resp, err := http.Get("http://" + apihost + "/api/transactions/closed?index=1%3A3003&direction=before&limit=1")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received []SchemaMigration
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, received, 1)
		require.Equal(t, string(models.TStatusReceived), received[0].Status)
	}

	// Request first (newer) transaction using a cursor, with a large `limit`
	{
		resp, err := http.Get("http://" + apihost + "/api/transactions/closed?index=1%3A3002&direction=after&limit=1000")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received []SchemaMigration
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, received, 1)
		require.Equal(t, string(models.TStatusFailed), received[0].Status)
	}

	// Request both transactions using `before` condition
	{
		resp, err := http.Get("http://" + apihost + "/api/transactions/closed?index=1%3A3004&direction=before&limit=2")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received []SchemaMigration
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, received, 2)
		require.Equal(t, string(models.TStatusFailed), received[0].Status)
		require.Equal(t, string(models.TStatusReceived), received[1].Status)
	}

	// Request both transactions using `after` condition, with a large `limit`
	{
		resp, err := http.Get("http://" + apihost + "/api/transactions/closed?index=1%3A3001&direction=after&limit=1000")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var received []SchemaMigration
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Len(t, received, 2)
		require.Equal(t, string(models.TStatusReceived), received[0].Status)
		require.Equal(t, string(models.TStatusFailed), received[1].Status)
	}
}

func TestTransaction_TypeMigration(t *testing.T) {
	defer truncateDB(t)

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
		SchemasTransactionAbstract: *transactionResponse(txID.String(), int64(pulseNumber), ts),
		ToMemberReference:          toMember.String(),
		FromMemberReference:        fromMember.String(),
		ToDepositReference:         toDeposit.String(),
		Type:                       string(models.TTypeMigration),
	}

	requireEqualResponse(t, resp, receivedTransaction, expectedTransaction)
}

func TestTransaction_TypeTransfer(t *testing.T) {
	defer truncateDB(t)

	txID := gen.RecordReference()
	pulseNumber := gen.PulseNumber()
	pntime, err := pulseNumber.AsApproximateTime()
	require.NoError(t, err)
	ts := pntime.Unix()

	fromMember := gen.Reference()
	toMember := gen.Reference()

	transaction := transactionModel(txID.Bytes(), int64(pulseNumber))
	transaction.Type = models.TTypeTransfer
	transaction.MemberFromReference = fromMember.Bytes()
	transaction.MemberToReference = toMember.Bytes()

	err = db.Insert(transaction)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txID.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	receivedTransaction := &SchemaMigration{}
	expectedTransaction := &SchemaMigration{
		SchemasTransactionAbstract: *transactionResponse(txID.String(), int64(pulseNumber), ts),
		ToMemberReference:          toMember.String(),
		FromMemberReference:        fromMember.String(),
		Type:                       string(models.TTypeTransfer),
	}

	requireEqualResponse(t, resp, receivedTransaction, expectedTransaction)
}

func TestTransaction_TypeRelease(t *testing.T) {
	defer truncateDB(t)

	txID := gen.RecordReference()
	pulseNumber := gen.PulseNumber()
	pntime, err := pulseNumber.AsApproximateTime()
	require.NoError(t, err)
	ts := pntime.Unix()

	toMember := gen.Reference()
	fromDeposit := gen.Reference()

	transaction := transactionModel(txID.Bytes(), int64(pulseNumber))
	transaction.Type = models.TTypeRelease
	transaction.MemberToReference = toMember.Bytes()
	transaction.DepositFromReference = fromDeposit.Bytes()

	err = db.Insert(transaction)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txID.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	receivedTransaction := &SchemaRelease{}
	expectedTransaction := &SchemaRelease{
		SchemasTransactionAbstract: *transactionResponse(txID.String(), int64(pulseNumber), ts),
		ToMemberReference:          toMember.String(),
		FromDepositReference:       fromDeposit.String(),
		Type:                       string(models.TTypeRelease),
	}

	requireEqualResponse(t, resp, receivedTransaction, expectedTransaction)
}

func TestTransactionsSearch_WrongFormat(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/transactions")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTransactionsSearch_NoContent(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/transactions?limit=15&status=failed")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func insertTransaction(t *testing.T, transactionID []byte, pulse int64, finishPulse int64, sequence int64) {
	transaction := models.Transaction{
		TransactionID:     transactionID,
		PulseRecord:       [2]int64{pulse, sequence},
		StatusRegistered:  true,
		Amount:            "10",
		Fee:               "1",
		FinishPulseRecord: [2]int64{finishPulse, sequence},
		Type:              models.TTypeMigration,
	}
	err := db.Insert(&transaction)
	require.NoError(t, err)
}

func TestTransactionsSearch(t *testing.T) {
	defer truncateDB(t)

	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	txIDThird := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	insertTransaction(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234)
	insertTransaction(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235)
	insertTransaction(t, txIDThird.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1236)

	resp, err := http.Get(
		"http://" + apihost + "/api/transactions?" +
			"limit=3&" +
			"value=" + pulseNumber.String() +
			"&status=registered&" +
			"type=migration&" +
			"index=" + pulseNumber.String() + "%3A1237&" +
			"direction=before")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := []SchemasTransactionAbstract{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 3)
	require.Equal(t, txIDThird.String(), received[0].TxID)
	require.Equal(t, txIDSecond.String(), received[1].TxID)
	require.Equal(t, txIDFirst.String(), received[2].TxID)
}

func TestTransactionsSearch_DirectionAfter(t *testing.T) {
	defer truncateDB(t)
	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	txIDThird := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	insertTransaction(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234)
	insertTransaction(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235)
	insertTransaction(t, txIDThird.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1236)

	resp, err := http.Get(
		"http://" + apihost + "/api/transactions?" +
			"limit=3&" +
			"value=" + pulseNumber.String() +
			"&status=registered&" +
			"type=migration&" +
			"index=" + pulseNumber.String() + "%3A1233&" +
			"direction=after")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := []SchemasTransactionAbstract{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 3)
	require.Equal(t, txIDFirst.String(), received[0].TxID)
	require.Equal(t, txIDSecond.String(), received[1].TxID)
	require.Equal(t, txIDThird.String(), received[2].TxID)
}

func TestTransactionsSearch_ValueTx(t *testing.T) {
	defer truncateDB(t)
	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	txIDThird := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	insertTransaction(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234)
	insertTransaction(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235)
	insertTransaction(t, txIDThird.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1236)

	resp, err := http.Get(
		"http://" + apihost + "/api/transactions?" +
			"limit=15&" +
			"value=" + txIDFirst.String() +
			"&status=registered&" +
			"type=migration&" +
			"index=" + pulseNumber.String() + "%3A1237&" +
			"direction=before")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := []SchemasTransactionAbstract{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 1)
	require.Equal(t, txIDFirst.String(), received[0].TxID)
}

func TestTransactionsSearch_WrongEverything(t *testing.T) {
	resp, err := http.Get(
		"http://" + apihost + "/api/transactions?" +
			"limit=15&" +
			"value=some_not_valid_value&" +
			"status=some_not_valid_status&" +
			"type=some_not_valid_type&" +
			"index=some_not_valid_index&" +
			"direction=some_not_valid_direction")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	expected := ErrorMessage{
		Error: []string{
			"Query parameter 'value' should be txID, fromMemberReference, toMemberReference or pulseNumber.",
			"Query parameter 'status' should be 'registered', 'sent', 'received' or 'failed'.",
			"Query parameter 'type' should be 'transfer', 'migration' or 'after'.",
			"Query parameter 'index' should have the '<pulse_number>:<sequence_number>' format.",
			"Query parameter 'direction' should be 'before' or 'after'.",
		},
	}
	received := ErrorMessage{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, expected, received)
}
