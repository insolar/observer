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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/instrumentation/inslogger"
	apiconfiguration "github.com/insolar/observer/configuration/api"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/observability"
)

const (
	recordNum                  = 198
	finishRecordNum            = 256
	amount                     = "1020"
	fee                        = "178"
	currentTime                = int64(1606435200)
	notExistedMigrationAddress = "0x35567Abc4Fa54fe30d200F76A4868A70383e7938"
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
			ID:        32000 + int64(i),
			Addr:      fmt.Sprintf("migration_addr_%v", i),
			Timestamp: time.Now().Unix(),
			Wasted:    wasted[i],
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
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=100&index=" + received[1]["index"])
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
			ID:        31000 + int64(i),
			Addr:      fmt.Sprintf("migration_addr_%v", i),
			Timestamp: time.Now().Unix(),
			Wasted:    wasted[i],
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

func TestTransaction_NotRegistered(t *testing.T) {
	defer truncateDB(t)

	txID := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	transaction := transactionModel(txID.Bytes(), int64(pulseNumber))
	transaction.StatusRegistered = false

	err := db.Insert(transaction)
	require.NoError(t, err)

	txIDStr := url.QueryEscape(txID.String())
	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txIDStr)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestTransaction_ClosedNotRegistered(t *testing.T) {
	defer truncateDB(t)

	transaction := transactionModel(gen.RecordReference().Bytes(), int64(gen.PulseNumber()))
	transaction.StatusRegistered = false
	transaction.StatusFinished = true

	err := db.Insert(transaction)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/transactions/closed?limit=10")
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

	txIDStr := url.QueryEscape(txID.String())
	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txIDStr)
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

func insertMember(t *testing.T, reference insolar.Reference, walletReference, accountReference *insolar.Reference, balance, publicKey string) {
	member := &observer.Member{
		MemberRef: reference,
		Balance:   balance,
		PublicKey: publicKey,
	}
	if walletReference != nil {
		member.WalletRef = *walletReference
	}
	if accountReference != nil {
		member.AccountRef = *accountReference
	}
	repo := postgres.NewMemberStorage(observability.Make(context.Background()), db)
	err := repo.Insert(member)
	require.NoError(t, err)
}

func insertDeposit(
	t *testing.T, reference insolar.Reference, memberReference insolar.Reference, amount, balance, etheriumHash string,
	depositNumber int64, status models.DepositStatus,
) {
	deposit := models.Deposit{
		Reference:       reference.Bytes(),
		MemberReference: memberReference.Bytes(),
		Amount:          amount,
		Balance:         balance,
		EtheriumHash:    etheriumHash,
		State:           gen.RecordReference().GetLocal().Bytes(),
		Timestamp:       currentTime - 10,
		DepositNumber:   &depositNumber,
		InnerStatus:     status,
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

func TestTransactionsSearch_NotRegistered(t *testing.T) {
	defer truncateDB(t)

	pulseNumber := gen.PulseNumber()

	txIDFirst := gen.RecordReference()
	transaction := transactionModel(txIDFirst.Bytes(), int64(pulseNumber))
	transaction.StatusRegistered = false
	err := db.Insert(transaction)
	require.NoError(t, err)

	txIDSecond := gen.RecordReference()
	insertTransaction(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235)

	resp, err := http.Get("http://" + apihost + "/api/transactions?" + "limit=10")
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

	insertMember(t, member1, nil, nil, balance1, randomString())
	insertMember(t, member2, nil, nil, balance2, randomString())

	member1Str := url.QueryEscape(member1.String())
	resp, err := http.Get("http://" + apihost + "/api/member/" + member1Str + "/balance")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberBalanceYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, balance1, received.Balance)
}

func TestObserverServer_SupplyStats(t *testing.T) {
	total := "1111111111111"
	totalr := "111.1111111111"

	coins := models.SupplyStats{
		Created: time.Now(),
		Total:   total,
	}

	err := db.Insert(&coins)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/stats/supply/total")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, totalr, string(bodyBytes))
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

func TestMember_NoContent_MigrationAddress(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/member/" + notExistedMigrationAddress)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMember(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "1010101"

	deposite1 := gen.Reference()
	deposite2 := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, "")
	insertDeposit(t, deposite2, member, "2000", "2000", "eth_hash_2", 2, models.DepositStatusConfirmed)
	insertDeposit(t, deposite1, member, "10000", "1000", "eth_hash_1", 1, models.DepositStatusConfirmed)

	memberStr := url.QueryEscape(member.String())
	resp, err := http.Get("http://" + apihost + "/api/member/" + memberStr)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		Reference:        member.String(),
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "0",
				AvailableAmount:  "1000",
				DepositReference: deposite1.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  0,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
			{
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  0,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMemberByPublicKey(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberStr := member.String()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "1010101"
	publicKey := randomString()

	deposite1 := gen.Reference()
	deposite2 := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, publicKey)
	insertDeposit(t, deposite2, member, "2000", "2000", "eth_hash_2", 2, models.DepositStatusConfirmed)
	insertDeposit(t, deposite1, member, "10000", "1000", "eth_hash_1", 1, models.DepositStatusConfirmed)

	pkStr := url.QueryEscape(publicKey)
	resp, err := http.Get("http://" + apihost + "/api/member/byPublicKey?publicKey=" + pkStr)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		Reference:        memberStr,
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				MemberReference:  &memberStr,
				AmountOnHold:     "0",
				AvailableAmount:  "1000",
				DepositReference: deposite1.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  0,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
			{
				MemberReference:  &memberStr,
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  0,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMemberByPublicKeyWrapped(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberStr := member.String()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "1010101"
	publicKey := randomString()

	deposite1 := gen.Reference()
	deposite2 := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, publicKey)
	insertDeposit(t, deposite2, member, "2000", "2000", "eth_hash_2", 2, models.DepositStatusConfirmed)
	insertDeposit(t, deposite1, member, "10000", "1000", "eth_hash_1", 1, models.DepositStatusConfirmed)

	pkStr := url.QueryEscape("-----BEGIN RSA PUBLIC KEY-----\n" + publicKey + "\n-----END RSA PUBLIC KEY-----")
	resp, err := http.Get("http://" + apihost + "/api/member/byPublicKey?publicKey=" + pkStr)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		Reference:        memberStr,
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				MemberReference:  &memberStr,
				AmountOnHold:     "0",
				AvailableAmount:  "1000",
				DepositReference: deposite1.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  0,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
			{
				MemberReference:  &memberStr,
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  0,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMemberByPublicKey_NoContent(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/member/byPublicKey?publicKey=" + randomString())
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMember_UnconfirmedDeposit(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "1010101"

	deposite1 := gen.Reference()
	deposite2 := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, "")
	insertDeposit(t, deposite2, member, "2000", "2000", "eth_hash_2", 1, models.DepositStatusConfirmed)
	insertDeposit(t, deposite1, member, "10000", "1000", "eth_hash_1", 2, models.DepositStatusCreated)

	memberStr := url.QueryEscape(member.String())
	resp, err := http.Get("http://" + apihost + "/api/member/" + memberStr)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		Reference:        member.String(),
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  0,
				Index:            1,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_MigrationAddress(t *testing.T) {
	defer truncateDB(t)

	memberRef := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "1010101"
	migrationAddress := "0xF4e1507486dFE411785B00d7D00A1f1a484f00E6"

	deposite1 := gen.Reference()
	deposite2 := gen.Reference()
	member := models.Member{
		Reference:        memberRef.Bytes(),
		Balance:          balance,
		WalletReference:  memberWalletReference.Bytes(),
		AccountReference: memberAccountReference.Bytes(),
		MigrationAddress: migrationAddress,
	}
	err := db.Insert(&member)
	require.NoError(t, err)

	insertDeposit(t, deposite2, memberRef, "2000", "2000", "eth_hash_2", 2, models.DepositStatusConfirmed)
	insertDeposit(t, deposite1, memberRef, "10000", "1000", "eth_hash_1", 1, models.DepositStatusConfirmed)

	resp, err := http.Get("http://" + apihost + "/api/member/" + migrationAddress)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		Reference:        memberRef.String(),
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "0",
				AvailableAmount:  "1000",
				DepositReference: deposite1.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  0,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
				MemberReference:  NullableString(memberRef.String()),
			},
			{
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  0,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   0,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
				MemberReference:  NullableString(memberRef.String()),
			},
		},
		MigrationAddress: NullableString(migrationAddress),
	}
	require.Equal(t, expected, received)
}

func TestMember_WithoutDeposit(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "989898989"

	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, "")

	resp, err := http.Get("http://" + apihost + "/api/member/" + member.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := ResponsesMemberYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	expected := ResponsesMemberYaml{
		Reference:        member.String(),
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

	deposite := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, "")

	deposit := models.Deposit{
		Reference:       deposite.Bytes(),
		MemberReference: member.Bytes(),
		Amount:          "500000000",
		Balance:         balance,
		EtheriumHash:    "eth_hash_1",
		HoldReleaseDate: currentTime,
		Vesting:         1826,
		VestingStep:     10,
		State:           gen.RecordReference().GetLocal().Bytes(),
		Timestamp:       currentTime - 10,
		DepositNumber:   newInt(100),
		InnerStatus:     models.DepositStatusConfirmed,
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
		Reference:        member.String(),
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "500000000",
				AvailableAmount:  "0",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime,
				Index:            100,
				ReleasedAmount:   "0",
				ReleaseEndDate:   currentTime + deposit.Vesting,
				Status:           "LOCKED",
				Timestamp:        currentTime - 10,
				NextRelease: &SchemaNextRelease{
					Amount:    "11539",
					Timestamp: currentTime,
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
	balance := "500000000"

	deposite := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, "")

	deposit := models.Deposit{
		Reference:       deposite.Bytes(),
		MemberReference: member.Bytes(),
		Amount:          balance,
		Balance:         balance,
		EtheriumHash:    "eth_hash_1",
		HoldReleaseDate: currentTime,
		Vesting:         1000,
		VestingStep:     10,
		State:           gen.RecordReference().GetLocal().Bytes(),
		Timestamp:       currentTime - 10,
		DepositNumber:   newInt(200),
		InnerStatus:     models.DepositStatusConfirmed,
	}
	err := db.Insert(&deposit)
	require.NoError(t, err)

	err = pStorage.Insert(&observer.Pulse{
		Number: 60199947,
	})
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
		Reference:        member.String(),
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "499977756",
				AvailableAmount:  "22244",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime,
				Index:            200,
				ReleasedAmount:   "22244",
				ReleaseEndDate:   currentTime + deposit.Vesting,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
				NextRelease: &SchemaNextRelease{
					Amount:    "22738",
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
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, "")

	deposit := models.Deposit{
		Reference:       deposite.Bytes(),
		MemberReference: member.Bytes(),
		Amount:          balance,
		Balance:         balance,
		EtheriumHash:    "eth_hash_1",
		HoldReleaseDate: currentTime,
		Vesting:         1000,
		VestingStep:     10,
		State:           gen.RecordReference().GetLocal().Bytes(),
		Timestamp:       currentTime - 10,
		DepositNumber:   newInt(300),
		InnerStatus:     models.DepositStatusConfirmed,
	}
	err := db.Insert(&deposit)
	require.NoError(t, err)

	err = pStorage.Insert(&observer.Pulse{
		Number: 60200937,
	})
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
		Reference:        member.String(),
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
				Index:            300,
				ReleasedAmount:   "5000",
				ReleaseEndDate:   currentTime + deposit.Vesting,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
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
	amount := "500000000"
	balance := "499995000"

	deposite := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, "")

	deposit := models.Deposit{
		Reference:       deposite.Bytes(),
		MemberReference: member.Bytes(),
		Amount:          amount,
		Balance:         balance,
		EtheriumHash:    "eth_hash_1",
		HoldReleaseDate: currentTime,
		Vesting:         18260,
		VestingStep:     10,
		State:           gen.RecordReference().GetLocal().Bytes(),
		Timestamp:       currentTime - 10,
		DepositNumber:   newInt(500),
		InnerStatus:     models.DepositStatusConfirmed,
	}
	err := db.Insert(&deposit)
	require.NoError(t, err)

	err = pStorage.Insert(&observer.Pulse{
		Number: 60200047,
	})
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
		Reference:        member.String(),
		AccountReference: memberAccountReference.String(),
		Balance:          balance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "499986154",
				AvailableAmount:  "8846",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime,
				Index:            500,
				ReleasedAmount:   "13846",
				ReleaseEndDate:   currentTime + deposit.Vesting,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
				NextRelease: &SchemaNextRelease{
					Amount:    "1185",
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
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := []SchemasTransactionAbstract{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 0)
}

func TestMemberTransaction_Empty(t *testing.T) {
	member := gen.Reference()
	insertMember(t, member, nil, nil, "10000", "")
	resp, err := http.Get("http://" + apihost + fmt.Sprintf("/api/member/%s/transactions?limit=15", member.String()))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := []SchemasTransactionAbstract{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 0)
}

func TestMemberTransactions(t *testing.T) {
	defer truncateDB(t)

	member1 := gen.Reference()
	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	member2 := gen.Reference()
	txIDThird := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	insertMember(t, member1, nil, nil, "10000", randomString())
	insertTransactionForMembers(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234, member1, member2)
	insertTransactionForMembers(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235, member2, member1)
	insertMember(t, member2, nil, nil, "20000", randomString())
	insertTransactionForMembers(t, txIDThird.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1236, member2, member2)

	member1Str := url.QueryEscape(member1.String())
	resp, err := http.Get(
		"http://" + apihost + "/api/member/" + member1Str + "/transactions?" +
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

func TestMemberTransactions_NotRegistered(t *testing.T) {
	defer truncateDB(t)

	member1 := gen.Reference()
	pulseNumber := gen.PulseNumber()

	txIDFirst := gen.RecordReference()
	transaction := transactionModel(txIDFirst.Bytes(), int64(pulseNumber))
	transaction.StatusRegistered = false
	transaction.MemberToReference = member1.Bytes()
	err := db.Insert(transaction)
	require.NoError(t, err)

	txIDSecond := gen.RecordReference()

	insertMember(t, member1, nil, nil, "10000", "")
	insertTransactionForMembers(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235, gen.Reference(), member1)

	member1Str := url.QueryEscape(member1.String())
	resp, err := http.Get("http://" + apihost + "/api/member/" + member1Str + "/transactions?" + "limit=10")
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

func TestMemberTransactions_Direction(t *testing.T) {
	defer truncateDB(t)

	member1 := gen.Reference()
	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	member2 := gen.Reference()
	txIDThird := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	insertMember(t, member1, nil, nil, "10000", randomString())
	insertTransactionForMembers(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234, member1, member2)
	insertTransactionForMembers(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235, member2, member1)
	insertMember(t, member2, nil, nil, "20000", randomString())
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

	insertMember(t, member1, nil, nil, "10000", randomString())
	insertTransactionForMembers(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234, member1, member2)
	insertTransactionForMembers(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235, member2, member1)
	insertMember(t, member2, nil, nil, "20000", randomString())
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
	t.Run("ok", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/fee/123")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		received := ResponsesFeeYaml{}
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, testFee.String(), received.Fee)
	})

	t.Run("uuid", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/fee/31f277c7-67f8-45b5-ae26-ff127d62a9ba")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		received := ResponsesInvalidAmountYaml{}
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, []string{"invalid amount"}, received.Error)
	})

	t.Run("negative", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/fee/-1")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		received := ResponsesInvalidAmountYaml{}
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, []string{"negative amount"}, received.Error)
	})
}

func TestObserverServer_NetworkStats(t *testing.T) {
	stats := models.NetworkStats{
		Created:           time.Now(),
		PulseNumber:       123,
		TotalTransactions: 23,
		MonthTransactions: 10,
		TotalAccounts:     3,
		Nodes:             11,
		CurrentTPS:        45,
		MaxTPS:            1498,
	}

	repo := postgres.NewNetworkStatsRepository(db)
	err := repo.InsertStats(stats)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/stats/network")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	jsonResp := ResponsesNetworkStatsYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	expected := ResponsesNetworkStatsYaml{
		Accounts:              3,
		CurrentTPS:            45,
		LastMonthTransactions: 10,
		MaxTPS:                1498,
		Nodes:                 11,
		TotalTransactions:     23,
	}
	require.Equal(t, expected, jsonResp)
}

func TestObserverServer_MarketStats(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/stats/market")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	jsonResp := ResponsesMarketStatsYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	expected := ResponsesMarketStatsYaml{
		Price: "0.05",
	}
	require.Equal(t, expected, jsonResp)
}

func TestObserverServer_Notifications(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/notification")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	err = db.Insert(&models.Notification{
		Message: "old",
		Start:   time.Now().Add(-10 * time.Hour),
		Stop:    time.Now().Add(-9 * time.Hour),
	})
	require.NoError(t, err)

	resp, err = http.Get("http://" + apihost + "/api/notification")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	err = db.Insert(&models.Notification{
		Message: "in the future",
		Start:   time.Now().Add(20 * time.Hour),
		Stop:    time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	resp, err = http.Get("http://" + apihost + "/api/notification")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	err = db.Insert(&models.Notification{
		Message: "show now",
		Start:   time.Now().Add(-3 * time.Hour),
		Stop:    time.Now().Add(3 * time.Hour),
	})
	require.NoError(t, err)

	resp, err = http.Get("http://" + apihost + "/api/notification")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	jsonResp := ResponsesNotificationInfoYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	expected := ResponsesNotificationInfoYaml{
		Notification: "show now",
	}
	require.Equal(t, expected, jsonResp)
}

func TestIsMigrationAddressFailed(t *testing.T) {
	address := "0x012345678901234567890123456789qwertyuiop"
	migrationAddress := models.MigrationAddress{
		ID:        32000 + int64(0),
		Addr:      address,
		Timestamp: time.Now().Unix(),
		Wasted:    true,
	}

	err := db.Insert(&migrationAddress)
	require.NoError(t, err)

	type testCase struct {
		address string
		result  bool
	}

	var testCases []testCase
	testCases = append(
		testCases,
		testCase{address: notExistedMigrationAddress, result: false},
		testCase{address: address, result: true},
	)

	for _, test := range testCases {
		resp, err := http.Get("http://" + apihost + "/admin/isMigrationAddress/" + test.address)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		jsonResp := ResponsesIsMigrationAddressYaml{}
		err = json.Unmarshal(bodyBytes, &jsonResp)
		require.NoError(t, err)
		expected := ResponsesIsMigrationAddressYaml{
			IsMigrationAddress: test.result,
		}
		require.Equal(t, expected, jsonResp)
	}

}

func TestObserverServer_CMC_Price(t *testing.T) {
	// first interval
	statsTime := time.Date(2020, 1, 3, 6, 0, 0, 0, time.UTC)
	err := db.Insert(&models.CoinMarketCapStats{
		Price:                100,
		PercentChange24Hours: 1,
		Rank:                 2,
		MarketCap:            3,
		Volume24Hours:        4,
		CirculatingSupply:    5,
		Created:              statsTime,
	})
	require.NoError(t, err)

	statsTime = time.Date(2020, 1, 3, 7, 0, 0, 0, time.UTC)
	err = db.Insert(&models.CoinMarketCapStats{
		Price:                200,
		PercentChange24Hours: 11,
		Rank:                 22,
		MarketCap:            33,
		Volume24Hours:        44,
		CirculatingSupply:    55,
		Created:              statsTime,
	})
	require.NoError(t, err)

	// second interval
	statsTime = time.Date(2020, 1, 3, 14, 0, 0, 0, time.UTC)
	err = db.Insert(&models.CoinMarketCapStats{
		Price:                300,
		PercentChange24Hours: 111,
		Rank:                 222,
		MarketCap:            333,
		Volume24Hours:        444,
		CirculatingSupply:    555,
		Created:              statsTime,
	})
	require.NoError(t, err)

	// third interval
	statsTime = time.Date(2020, 1, 3, 23, 0, 0, 0, time.UTC)
	err = db.Insert(&models.CoinMarketCapStats{
		Price:                400,
		PercentChange24Hours: 1111,
		Rank:                 2222,
		MarketCap:            3333,
		Volume24Hours:        4444,
		CirculatingSupply:    5555,
		Created:              statsTime,
	})
	require.NoError(t, err)

	logger := inslogger.FromContext(context.Background())
	observerAPI := NewObserverServer(db, logger, pStorage, apiconfiguration.Configuration{
		FeeAmount:   testFee,
		Price:       testPrice,
		PriceOrigin: "coin_market_cap",
	})

	e := echo.New()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	mockCtx := e.NewContext(req, res)

	err = observerAPI.MarketStats(mockCtx)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, res.Code)

	bodyBytes, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	received := ResponsesMarketStatsYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	require.Equal(t, "400", received.Price)
	require.Equal(t, "3333", *received.MarketCap)
	require.Equal(t, "2222", *received.Rank)
	require.Equal(t, "5555", *received.CirculatingSupply)
	require.Equal(t, "1111", *received.DailyChange)
	require.Equal(t, "4444", *received.Volume)

	points := *received.PriceHistory
	require.Equal(t, 3, len(points))

	require.Equal(t,
		time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC).Unix(),
		points[0].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 3, 8, 0, 0, 0, time.UTC).Unix(),
		points[1].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 3, 16, 0, 0, 0, time.UTC).Unix(),
		points[2].Timestamp)

	require.Equal(t, "150", points[0].Price)
	require.Equal(t, "300", points[1].Price)
	require.Equal(t, "400", points[2].Price)
}

func TestObserverServer_Binance_Price(t *testing.T) {
	// first interval
	statsTime := time.Date(2020, 1, 3, 6, 0, 0, 0, time.UTC)
	err := db.Insert(&models.BinanceStats{
		SymbolPriceUSD:     100,
		Symbol:             "1",
		SymbolPriceBTC:     "2",
		BTCPriceUSD:        "3",
		PriceChangePercent: "4",
		Created:            statsTime,
	})
	require.NoError(t, err)

	statsTime = time.Date(2020, 1, 3, 7, 0, 0, 0, time.UTC)
	err = db.Insert(&models.BinanceStats{
		SymbolPriceUSD:     200,
		Symbol:             "11",
		SymbolPriceBTC:     "22",
		BTCPriceUSD:        "33",
		PriceChangePercent: "44",
		Created:            statsTime,
	})
	require.NoError(t, err)

	// second interval
	statsTime = time.Date(2020, 1, 3, 14, 0, 0, 0, time.UTC)
	err = db.Insert(&models.BinanceStats{
		SymbolPriceUSD:     300,
		Symbol:             "111",
		SymbolPriceBTC:     "222",
		BTCPriceUSD:        "333",
		PriceChangePercent: "444",
		Created:            statsTime,
	})
	require.NoError(t, err)

	// third interval
	statsTime = time.Date(2020, 1, 3, 23, 0, 0, 0, time.UTC)
	err = db.Insert(&models.BinanceStats{
		SymbolPriceUSD:     400,
		Symbol:             "1111",
		SymbolPriceBTC:     "2222",
		BTCPriceUSD:        "3333",
		PriceChangePercent: "4444",
		Created:            statsTime,
	})
	require.NoError(t, err)

	logger := inslogger.FromContext(context.Background())
	observerAPI := NewObserverServer(db, logger, pStorage, apiconfiguration.Configuration{
		FeeAmount:   testFee,
		Price:       testPrice,
		PriceOrigin: "binance",
	})

	e := echo.New()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	mockCtx := e.NewContext(req, res)

	err = observerAPI.MarketStats(mockCtx)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, res.Code)

	bodyBytes, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	received := ResponsesMarketStatsYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	require.Equal(t, "400", received.Price)
	require.Equal(t, "4444", *received.DailyChange)

	points := *received.PriceHistory
	require.Equal(t, 3, len(points))

	require.Equal(t,
		time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC).Unix(),
		points[0].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 3, 8, 0, 0, 0, time.UTC).Unix(),
		points[1].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 3, 16, 0, 0, 0, time.UTC).Unix(),
		points[2].Timestamp)

	require.Equal(t, "150", points[0].Price)
	require.Equal(t, "300", points[1].Price)
	require.Equal(t, "400", points[2].Price)
}

func newInt(val int64) *int64 {
	return &val
}

func randomString() string {
	id, _ := uuid.NewRandom()
	return id.String()
}
