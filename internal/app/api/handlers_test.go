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
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
)

const (
	recordNum       = 198
	finishRecordNum = 256
	amount          = "1020"
	fee             = "178"
	currentTime     = int64(1606435200)
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

func TestMigrationAddresses_WrongArguments(t *testing.T) {
	// if `limit` is not a number, API returns `bad request`
	resp, err := http.Get("http://" + apihost + "/admin/migration/addresses?limit=LOL")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is zero, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=0")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is negative, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=-10")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is > 1000, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=1001")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `index` is not a number, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=100&index=LOL")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestMigrationAddresses_HappyPath(t *testing.T) {
	defer truncateDB(t)

	// Make sure /admin/migration/addresses returns non-assigned migration addresses
	// sorted by ID with provided `limit` and `index` arguments.

	// insert migration addresses
	var err error
	wasted := []bool{false, false, true, false, true}
	for i := 0; i < len(wasted); i++ {
		migrationAddress := models.MigrationAddress{
			ID: 32000 + int64(i),
			Addr: fmt.Sprintf("migration_addr_%v", i),
			Timestamp: time.Now().Unix(),
			Wasted: wasted[i],
		}

		err = db.Insert(&migrationAddress)
		require.NoError(t, err)
	}

	// request two oldest non-assigned migration addresses
	resp, err := http.Get("http://" + apihost + "/admin/migration/addresses?limit=2")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received []map[string]string
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, 2, len(received))
	require.Equal(t, "32000", received[0]["index"])
	require.Equal(t, "migration_addr_0", received[0]["address"])
	require.Equal(t, "32001", received[1]["index"])
	require.Equal(t, "migration_addr_1", received[1]["address"])

	// request the rest of non-assigned migration addresses
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=100&index="+received[1]["index"])
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, 1, len(received))
	require.Equal(t, "32003", received[0]["index"])
	require.Equal(t, "migration_addr_3", received[0]["address"])
}

func TestMigrationAddressesCount(t *testing.T) {
	defer truncateDB(t)

	// Make sure /admin/migration/addresses/count returns the total number
	// of non-assigned migration addresses.

	// insert migration addresses
	var err error
	wasted := []bool{true, false, true, false, true}
	expectedCount := 0
	for i := 0; i < len(wasted); i++ {
		migrationAddress := models.MigrationAddress{
			ID: 31000 + int64(i),
			Addr: fmt.Sprintf("migration_addr_%v", i),
			Timestamp: time.Now().Unix(),
			Wasted: wasted[i],
		}

		if !wasted[i] {
			expectedCount++
		}

		err = db.Insert(&migrationAddress)
		require.NoError(t, err)
	}

	resp, err := http.Get("http://" + apihost + "/admin/migration/addresses/count")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received map[string]int
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, expectedCount, received["count"])
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

	// if `order` is not "chronological" or "reverse", API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/api/transactions/closed?limit=100&order=LOL")
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
		resp, err := http.Get("http://" + apihost + "/api/transactions/closed?index=1%3A3003&order=reverse&limit=1")
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
		resp, err := http.Get("http://" + apihost + "/api/transactions/closed?index=1%3A3002&order=chronological&limit=1000")
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

	// Request both transactions using `reverse` order
	{
		resp, err := http.Get("http://" + apihost + "/api/transactions/closed?index=1%3A3004&order=reverse&limit=2")
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

	// Request both transactions using `chronological` order, with a large `limit`
	{
		resp, err := http.Get("http://" + apihost + "/api/transactions/closed?index=1%3A3001&order=chronological&limit=1000")
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

func insertTransactionForMembers(
	t *testing.T, transactionID []byte, pulse int64, finishPulse int64, sequence int64,
	memberFromReference, memberToReference insolar.Reference,
) {
	transaction := models.Transaction{
		TransactionID:       transactionID,
		PulseRecord:         [2]int64{pulse, sequence},
		StatusRegistered:    true,
		Amount:              "10",
		Fee:                 "1",
		FinishPulseRecord:   [2]int64{finishPulse, sequence},
		Type:                models.TTypeTransfer,
		MemberFromReference: memberFromReference.Bytes(),
		MemberToReference:   memberToReference.Bytes(),
	}
	err := db.Insert(&transaction)
	require.NoError(t, err)
}

func insertMember(t *testing.T, reference insolar.Reference, walletReference, accountReference *insolar.Reference, balance string) {
	member := models.Member{
		Reference: reference.Bytes(),
		Balance:   balance,
	}
	if walletReference != nil {
		member.WalletReference = walletReference.Bytes()
	}
	if accountReference != nil {
		member.AccountReference = accountReference.Bytes()
	}
	err := db.Insert(&member)
	require.NoError(t, err)
}

func insertDeposit(
	t *testing.T, reference insolar.Reference, memberReference insolar.Reference, amount, balance, etheriumHash string,
) {
	deposit := models.Deposit{
		Reference:       reference.Bytes(),
		MemberReference: memberReference.Bytes(),
		Amount:          amount,
		Balance:         balance,
		EtheriumHash:    etheriumHash,
	}
	err := db.Insert(&deposit)
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
			"order=reverse")
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

func TestTransactionsSearch_OrderChronological(t *testing.T) {
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
			"order=chronological")
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
			"order=reverse")
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
			"order=some_not_valid_order")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	expected := ErrorMessage{
		Error: []string{
			"Query parameter 'value' should be txID, fromMemberReference, toMemberReference or pulseNumber.",
			"Query parameter 'status' should be 'registered', 'sent', 'received' or 'failed'.",
			"Query parameter 'type' should be 'transfer', 'migration' or 'release'.",
			"Query parameter 'index' should have the '<pulse_number>:<sequence_number>' format.",
			"Query parameter 'order' should be 'reverse' or 'chronological'.",
		},
	}
	received := ErrorMessage{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, expected, received)
}

func TestMemberBalance_WrongFormat(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/member/" + "not_valid_ref" + "/balance")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	received := &ErrorMessage{}
	expected := &ErrorMessage{Error: []string{"reference wrong format"}}
	requireEqualResponse(t, resp, received, expected)
}

func TestMemberBalance_NoContent(t *testing.T) {
	ref := gen.Reference().String()
	resp, err := http.Get("http://" + apihost + "/api/member/" + ref + "/balance")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMemberBalance(t *testing.T) {
	defer truncateDB(t)
	member1 := gen.Reference()
	balance1 := "1234567"
	member2 := gen.Reference()
	balance2 := "567890"

	insertMember(t, member1, nil, nil, balance1)
	insertMember(t, member2, nil, nil, balance2)

	resp, err := http.Get("http://" + apihost + "/api/member/" + member1.String() + "/balance")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberBalanceYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, balance1, received.Balance)
}

func TestObserverServer_Coins(t *testing.T) {
	total := "1111111111111"
	totalr := "111.1111111111"
	max := "2222222222222"
	maxr := "222.2222222222"
	circ := "33333333333333"
	circr := "3333.3333333333"

	coins := postgres.StatsModel{
		Created:     time.Time{},
		Total:       total,
		Max:         max,
		Circulating: circ,
	}

	err := db.Insert(&coins)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/coins")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	jsonResp := component.XnsCoinStats{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	expected := component.XnsCoinStats{
		Total:       total,
		Max:         max,
		Circulating: circ,
	}
	require.Equal(t, expected, jsonResp)

	resp, err = http.Get("http://" + apihost + "/api/coins/total")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, totalr, string(bodyBytes))

	resp, err = http.Get("http://" + apihost + "/api/coins/max")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, maxr, string(bodyBytes))

	resp, err = http.Get("http://" + apihost + "/api/coins/circulating")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, circr, string(bodyBytes))
}

func TestMember_WrongFormat(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/member/" + "not_valid_ref")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	received := &ErrorMessage{}
	expected := &ErrorMessage{Error: []string{"reference wrong format"}}
	requireEqualResponse(t, resp, received, expected)
}

func TestMember_NoContent(t *testing.T) {
	ref := gen.Reference().String()
	resp, err := http.Get("http://" + apihost + "/api/member/" + ref)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMember(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "1010101"

	deposite := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance)
	insertDeposit(t, deposite, member, "10000", "1000", "eth_hash_1")

	resp, err := http.Get("http://" + apihost + "/api/member/" + member.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "0",
				AvailableAmount:  "1000",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  0,
				Index:            0,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        0,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_WithoutDeposit(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "989898989"

	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance)

	resp, err := http.Get("http://" + apihost + "/api/member/" + member.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
	}
	require.Equal(t, expected, received)
}

func TestMember_Hold(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "5000"
	clock.nowTime = currentTime

	deposite := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance)

	deposit := models.Deposit{
		Reference:       deposite.Bytes(),
		MemberReference: member.Bytes(),
		Amount:          "5000",
		Balance:         balance,
		EtheriumHash:    "eth_hash_1",
		HoldReleaseDate: currentTime,
		Vesting:         1000,
		VestingStep:     10,
	}
	err := db.Insert(&deposit)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/member/" + member.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "5000",
				AvailableAmount:  "0",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime,
				Index:            0,
				ReleasedAmount:   "0",
				ReleaseEndDate:   currentTime + deposit.Vesting,
				Status:           "LOCKED",
				Timestamp:        0,
				NextRelease: &SchemaNextRelease{
					Amount:    "50",
					Timestamp: currentTime + deposit.VestingStep,
				},
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_Vesting(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "5000"

	deposite := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance)

	deposit := models.Deposit{
		Reference:       deposite.Bytes(),
		MemberReference: member.Bytes(),
		Amount:          balance,
		Balance:         balance,
		EtheriumHash:    "eth_hash_1",
		HoldReleaseDate: currentTime,
		Vesting:         1000,
		VestingStep:     10,
	}
	err := db.Insert(&deposit)
	require.NoError(t, err)

	clock.nowTime = currentTime + deposit.VestingStep + 1

	resp, err := http.Get("http://" + apihost + "/api/member/" + member.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "4950",
				AvailableAmount:  "50",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime,
				Index:            0,
				ReleasedAmount:   "50",
				ReleaseEndDate:   currentTime + deposit.Vesting,
				Status:           "AVAILABLE",
				Timestamp:        0,
				NextRelease: &SchemaNextRelease{
					Amount:    "50",
					Timestamp: currentTime + 2*deposit.VestingStep,
				},
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_VestingAll(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "5000"

	deposite := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance)

	deposit := models.Deposit{
		Reference:       deposite.Bytes(),
		MemberReference: member.Bytes(),
		Amount:          balance,
		Balance:         balance,
		EtheriumHash:    "eth_hash_1",
		HoldReleaseDate: currentTime,
		Vesting:         1000,
		VestingStep:     10,
	}
	err := db.Insert(&deposit)
	require.NoError(t, err)

	clock.nowTime = currentTime + deposit.Vesting + 1

	resp, err := http.Get("http://" + apihost + "/api/member/" + member.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "0",
				AvailableAmount:  "5000",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime,
				Index:            0,
				ReleasedAmount:   "5000",
				ReleaseEndDate:   currentTime + deposit.Vesting,
				Status:           "AVAILABLE",
				Timestamp:        0,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_VestingAndSpent(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	amount := "5000"
	balance := "4500"

	deposite := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance)

	deposit := models.Deposit{
		Reference:       deposite.Bytes(),
		MemberReference: member.Bytes(),
		Amount:          amount,
		Balance:         balance,
		EtheriumHash:    "eth_hash_1",
		HoldReleaseDate: currentTime,
		Vesting:         1000,
		VestingStep:     10,
	}
	err := db.Insert(&deposit)
	require.NoError(t, err)

	clock.nowTime = currentTime + deposit.VestingStep*11 + 1

	resp, err := http.Get("http://" + apihost + "/api/member/" + member.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "4450",
				AvailableAmount:  "50",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime,
				Index:            0,
				ReleasedAmount:   "550",
				ReleaseEndDate:   currentTime + deposit.Vesting,
				Status:           "AVAILABLE",
				Timestamp:        0,
				NextRelease: &SchemaNextRelease{
					Amount:    "50",
					Timestamp: currentTime + 12*deposit.VestingStep,
				},
			},
		},
	}

	require.Equal(t, expected, received)
}

func TestMemberTransaction_WrongFormat(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/member/not_valid_ref/transactions?limit=15")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	received := &ErrorMessage{}
	expected := &ErrorMessage{Error: []string{"reference wrong format"}}
	requireEqualResponse(t, resp, received, expected)
}

func TestMemberTransaction_NoContent(t *testing.T) {
	member := gen.Reference()
	resp, err := http.Get("http://" + apihost + fmt.Sprintf("/api/member/%s/transactions?limit=15", member.String()))
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMemberTransaction_Empty(t *testing.T) {
	member := gen.Reference()
	insertMember(t, member, nil, nil, "10000")
	resp, err := http.Get("http://" + apihost + fmt.Sprintf("/api/member/%s/transactions?limit=15", member.String()))
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMemberTransactions(t *testing.T) {
	defer truncateDB(t)

	member1 := gen.Reference()
	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	member2 := gen.Reference()
	txIDThird := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	insertMember(t, member1, nil, nil, "10000")
	insertTransactionForMembers(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234, member1, member2)
	insertTransactionForMembers(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235, member2, member1)
	insertMember(t, member2, nil, nil, "20000")
	insertTransactionForMembers(t, txIDThird.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1236, member2, member2)

	resp, err := http.Get(
		"http://" + apihost + "/api/member/" + member1.String() + "/transactions?" +
			"limit=3&" +
			"&status=registered&" +
			"type=transfer&" +
			"index=" + pulseNumber.String() + "%3A1237&" +
			"order=reverse")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := []SchemasTransactionAbstract{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 2)
	require.Equal(t, txIDSecond.String(), received[0].TxID)
	require.Equal(t, txIDFirst.String(), received[1].TxID)
}

func TestMemberTransactions_Direction(t *testing.T) {
	defer truncateDB(t)

	member1 := gen.Reference()
	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	member2 := gen.Reference()
	txIDThird := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	insertMember(t, member1, nil, nil, "10000")
	insertTransactionForMembers(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234, member1, member2)
	insertTransactionForMembers(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235, member2, member1)
	insertMember(t, member2, nil, nil, "20000")
	insertTransactionForMembers(t, txIDThird.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1236, member2, member2)

	resp, err := http.Get(
		"http://" + apihost + "/api/member/" + member1.String() + "/transactions?" +
			"limit=3&" +
			"&status=registered&" +
			"type=transfer&" +
			"direction=incoming&" +
			"index=" + pulseNumber.String() + "%3A1237&" +
			"order=reverse")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := []SchemasTransactionAbstract{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 1)
	require.Equal(t, txIDSecond.String(), received[0].TxID)
}

func TestMemberTransactions_OrderChronological(t *testing.T) {
	defer truncateDB(t)
	member1 := gen.Reference()
	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	member2 := gen.Reference()
	txIDThird := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	insertMember(t, member1, nil, nil, "10000")
	insertTransactionForMembers(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234, member1, member2)
	insertTransactionForMembers(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235, member2, member1)
	insertMember(t, member2, nil, nil, "20000")
	insertTransactionForMembers(t, txIDThird.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1236, member2, member2)

	resp, err := http.Get(
		"http://" + apihost + "/api/member/" + member1.String() + "/transactions?" +
			"limit=3&" +
			"value=" + pulseNumber.String() +
			"&status=registered&" +
			"type=transfer&" +
			"index=" + pulseNumber.String() + "%3A1233&" +
			"order=chronological")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := []SchemasTransactionAbstract{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 2)
	require.Equal(t, txIDFirst.String(), received[0].TxID)
	require.Equal(t, txIDSecond.String(), received[1].TxID)
}

func TestMemberTransactions_WrongEverything(t *testing.T) {
	member := gen.Reference()
	resp, err := http.Get(
		"http://" + apihost + "/api/member/" + member.String() + "/transactions?" +
			"limit=15&" +
			"status=some_not_valid_status&" +
			"type=some_not_valid_type&" +
			"index=some_not_valid_index&" +
			"direction=some_not_valid_direction&" +
			"order=some_not_valid_order")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	expected := ErrorMessage{
		Error: []string{
			"Query parameter 'direction' should be 'incoming', 'outgoing' or 'all'.",
			"Query parameter 'status' should be 'registered', 'sent', 'received' or 'failed'.",
			"Query parameter 'type' should be 'transfer', 'migration' or 'release'.",
			"Query parameter 'index' should have the '<pulse_number>:<sequence_number>' format.",
			"Query parameter 'order' should be 'reverse' or 'chronological'.",
		},
	}
	received := ErrorMessage{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, expected, received)
}

func TestFee(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/fee/123")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	received := ResponsesFeeYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, testFee.String(), received.Fee)
}
