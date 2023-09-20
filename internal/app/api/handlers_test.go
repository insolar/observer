package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/secrets"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/mainnet/application/appfoundation"
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

func insertBurnedBalance(t *testing.T, balance string) {
	burnedBalance := models.BurnedBalance{
		Balance:      balance,
		AccountState: gen.Reference().Bytes(),
	}
	err := db.Insert(&burnedBalance)
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
				HoldReleaseDate:  currentTime - 10,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
			{
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  currentTime - 10,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_MigrationAdmin_WithBurnedBalance(t *testing.T) {
	defer truncateDB(t)

	member := appfoundation.GetMigrationAdminMember()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "1010101"
	burnedBalance := "20202020202"

	deposite1 := gen.Reference()
	deposite2 := gen.Reference()
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, "")
	insertBurnedBalance(t, burnedBalance)
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
		BurnedBalance:    &burnedBalance,
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "0",
				AvailableAmount:  "1000",
				DepositReference: deposite1.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime - 10,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
			{
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  currentTime - 10,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_MigrationAdmin_WithoutBurnedBalance(t *testing.T) {
	defer truncateDB(t)

	member := appfoundation.GetMigrationAdminMember()
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
		BurnedBalance:    NullableString("0"),
		WalletReference:  memberWalletReference.String(),
		Deposits: &[]SchemaDeposit{
			{
				AmountOnHold:     "0",
				AvailableAmount:  "1000",
				DepositReference: deposite1.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime - 10,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
			{
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  currentTime - 10,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   currentTime - 10,
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
	privateKey, err := secrets.GeneratePrivateKeyEthereum()
	publicKeyPEM, err := secrets.ExportPublicKeyPEM(secrets.ExtractPublicKey(privateKey))
	publicKey := string(publicKeyPEM)

	deposite1 := gen.Reference()
	deposite2 := gen.Reference()
	canonicakPK, err := foundation.ExtractCanonicalPublicKey(publicKey)
	require.NoError(t, err)
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, canonicakPK)
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
				HoldReleaseDate:  currentTime - 10,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
			{
				MemberReference:  &memberStr,
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  currentTime - 10,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMemberByPublicKeyDifferentPEM(t *testing.T) {
	defer truncateDB(t)

	member := gen.Reference()
	memberStr := member.String()
	memberWalletReference := gen.Reference()
	memberAccountReference := gen.Reference()
	balance := "1010101"
	publicKey := "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEwDcgWZ1SbG+nbiXZkmYUZEfk2nkk\n1PEmEWoj4g6DLEkdaQVorOkqlloEz1zXclQaAE1S8i3F7OFNrNxLkm34ow==\n-----END PUBLIC KEY-----\n"

	deposite1 := gen.Reference()
	deposite2 := gen.Reference()
	canonicakPK, err := foundation.ExtractCanonicalPublicKey(publicKey)
	require.NoError(t, err)
	insertMember(t, member, &memberWalletReference, &memberAccountReference, balance, canonicakPK)
	insertDeposit(t, deposite2, member, "2000", "2000", "eth_hash_2", 2, models.DepositStatusConfirmed)
	insertDeposit(t, deposite1, member, "10000", "1000", "eth_hash_1", 1, models.DepositStatusConfirmed)

	publicKeyDifferentPEM := "-----BEGIN PUBLIC KEY-----\nThisIsNewField:testvalue\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEwDcgWZ1SbG+nbiXZkmYUZEfk2nkk\n1PEmEWoj4g6DLEkdaQVorOkqlloEz1zXclQaAE1S8i3F7OFNrNxLkm34ow==\n-----END PUBLIC KEY-----\n"
	pkStr := url.QueryEscape(publicKeyDifferentPEM)
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
				HoldReleaseDate:  currentTime - 10,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
			{
				MemberReference:  &memberStr,
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  currentTime - 10,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMemberByPublicKey_NoContent(t *testing.T) {
	privateKey, err := secrets.GeneratePrivateKeyEthereum()
	publicKey, err := secrets.ExportPublicKeyPEM(secrets.ExtractPublicKey(privateKey))
	pkStr := url.QueryEscape(string(publicKey))
	resp, err := http.Get("http://" + apihost + "/api/member/byPublicKey?publicKey=" + pkStr)
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
				HoldReleaseDate:  currentTime - 10,
				Index:            1,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   currentTime - 10,
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
				HoldReleaseDate:  currentTime - 10,
				Index:            1,
				ReleasedAmount:   "10000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
				MemberReference:  NullableString(memberRef.String()),
			},
			{
				AmountOnHold:     "0",
				AvailableAmount:  "2000",
				DepositReference: deposite2.String(),
				EthTxHash:        "eth_hash_2",
				HoldReleaseDate:  currentTime - 10,
				Index:            2,
				ReleasedAmount:   "2000",
				ReleaseEndDate:   currentTime - 10,
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

func TestMember_NoHold(t *testing.T) {
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
				AmountOnHold:     "0",
				AvailableAmount:  "500000000",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime - 10,
				Index:            100,
				ReleasedAmount:   "500000000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_NoHold_When_Balance_Smaller_than_Amount(t *testing.T) {
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
				AmountOnHold:     "0",
				AvailableAmount:  "5000",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime - 10,
				Index:            100,
				ReleasedAmount:   "500000000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_Vesting_AllFromTheStart(t *testing.T) {
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
				AmountOnHold:     "0",
				AvailableAmount:  "500000000",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime - 10,
				Index:            200,
				ReleasedAmount:   "500000000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
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
				HoldReleaseDate:  currentTime - 10,
				Index:            300,
				ReleasedAmount:   "5000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
			},
		},
	}
	require.Equal(t, expected, received)
}

func TestMember_VestingAllFromTheStartAndSpent(t *testing.T) {
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
				AmountOnHold:     "0",
				AvailableAmount:  "499995000",
				DepositReference: deposite.String(),
				EthTxHash:        "eth_hash_1",
				HoldReleaseDate:  currentTime - 10,
				Index:            500,
				ReleasedAmount:   "500000000",
				ReleaseEndDate:   currentTime - 10,
				Status:           "AVAILABLE",
				Timestamp:        currentTime - 10,
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

func newInt(val int64) *int64 {
	return &val
}

func randomString() string {
	id, _ := uuid.NewRandom()
	return id.String()
}

func TestPulseNumber(t *testing.T) {
	err := pStorage.Insert(&observer.Pulse{
		Number: 60199947,
	})
	require.NoError(t, err)
	err = pStorage.Insert(&observer.Pulse{
		Number: 60199957,
	})
	require.NoError(t, err)
	err = pStorage.Insert(&observer.Pulse{
		Number: 60199937,
	})
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/pulse/number")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received map[string]int64
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, int64(60199957), received["pulseNumber"])
}

func TestTransactionsByPulseNumberRange_WrongFormat(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/transactions/inPulseNumberRange")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTransactionsByPulseNumberRange_NoContent(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/transactions/inPulseNumberRange?limit=15&fromPulseNumber=0&toPulseNumber=1")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestTransactionsByPulseNumberRange_InvalidRange(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/transactions/inPulseNumberRange?limit=15&fromPulseNumber=10&toPulseNumber=1")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	received := struct {
		Error []string `json:"error"`
	}{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	require.Equal(t, []string{"Invalid input range: fromPulseNumber must chronologically precede toPulseNumber"}, received.Error)
}

func TestTransactionsByPulseNumberRange(t *testing.T) {
	defer truncateDB(t)

	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	txIDThird := gen.RecordReference()
	txIDFourth := gen.RecordReference()
	pulseNumber := gen.PulseNumber()

	insertTransaction(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234)
	insertTransaction(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235)
	insertTransaction(t, txIDThird.Bytes(), int64(pulseNumber)+20, int64(pulseNumber)+30, 1236)
	insertTransaction(t, txIDFourth.Bytes(), int64(pulseNumber)+30, int64(pulseNumber)+40, 1237)

	fromPulseNumber := pulseNumber.String()
	toPulseNumber := strconv.Itoa(int(pulseNumber) + 20)
	resp, err := http.Get(
		"http://" + apihost + "/api/transactions/inPulseNumberRange?" +
			"limit=10" +
			"&fromPulseNumber=" + fromPulseNumber +
			"&toPulseNumber=" + toPulseNumber +
			"&index=" + pulseNumber.String() + "%3A1234")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	received := []SchemasTransactionAbstract{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 2)
	require.Equal(t, txIDSecond.String(), received[0].TxID)
	require.Equal(t, txIDThird.String(), received[1].TxID)
}

func TestTransactionsByPulseNumberRange_WithMemberReference(t *testing.T) {
	defer truncateDB(t)

	txIDFirst := gen.RecordReference()
	txIDSecond := gen.RecordReference()
	txIDThird := gen.RecordReference()
	pulseNumber := gen.PulseNumber()
	member1 := gen.Reference()
	member2 := gen.Reference()

	insertMember(t, member1, nil, nil, "10000", randomString())
	insertTransactionForMembers(t, txIDFirst.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1234, member1, member2)
	insertTransactionForMembers(t, txIDSecond.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1235, member2, member1)
	insertMember(t, member2, nil, nil, "10000", randomString())
	insertTransactionForMembers(t, txIDThird.Bytes(), int64(pulseNumber), int64(pulseNumber)+10, 1236, member2, member2)

	fromPulseNumber := pulseNumber.String()
	toPulseNumber := strconv.Itoa(int(pulseNumber) + 20)
	resp, err := http.Get(
		"http://" + apihost + "/api/transactions/inPulseNumberRange?" +
			"limit=10" +
			"&memberReference=" + member1.String() +
			"&fromPulseNumber=" + fromPulseNumber +
			"&toPulseNumber=" + toPulseNumber)
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

func TestTransactionsByPulseNumberRange_WrongEverything(t *testing.T) {
	resp, err := http.Get(
		"http://" + apihost + "/api/transactions/inPulseNumberRange?" +
			"limit=15&" +
			"memberReference=some_not_valid_reference&" +
			"fromPulseNumber=1&" +
			"toPulseNumber=2")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	expected := ErrorMessage{
		Error: []string{
			"reference wrong format",
		},
	}
	received := ErrorMessage{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, expected, received)
}

func createPulse(pulseNumber uint32) (*observer.Pulse, error) {
	pulse := observer.Pulse{
		Number: insolar.PulseNumber(pulseNumber),
	}
	pTime, err := pulse.Number.AsApproximateTime()
	if err != nil {
		return nil, err
	}
	pulse.Timestamp = pTime.UnixNano()
	return &pulse, err
}

func TestPulseRange_WrongFormat(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/pulse/range")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPulseRange_NoContent(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/pulse/range?fromTimestamp=0&toTimestamp=1&limit=10")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestPulseRange_InvalidRange(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/pulse/range?fromTimestamp=10&toTimestamp=1&limit=10")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	received := struct {
		Error []string `json:"error"`
	}{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	require.Equal(t, []string{"Invalid input range: fromTimestamp must chronologically precede toTimestamp"}, received.Error)
}

func TestPulseRange(t *testing.T) {
	firstPulse, err := createPulse(60209957)
	require.NoError(t, err)
	secondPulse, err := createPulse(60209967)
	require.NoError(t, err)
	thirdPulse, err := createPulse(60209977)
	require.NoError(t, err)

	err = pStorage.Insert(firstPulse)
	require.NoError(t, err)
	err = pStorage.Insert(secondPulse)
	require.NoError(t, err)
	err = pStorage.Insert(thirdPulse)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/pulse/range?limit=10" +
		"&fromTimestamp=" + strconv.FormatInt(firstPulse.Timestamp/time.Second.Nanoseconds(), 10) +
		"&toTimestamp=" + strconv.FormatInt(secondPulse.Timestamp/time.Second.Nanoseconds(), 10))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received []int64
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 2)
	require.Equal(t, int64(firstPulse.Number), received[0])
	require.Equal(t, int64(secondPulse.Number), received[1])
}

func TestPulseRange_Limit(t *testing.T) {
	firstPulse, err := createPulse(60209957)
	require.NoError(t, err)
	secondPulse, err := createPulse(60209967)
	require.NoError(t, err)
	thirdPulse, err := createPulse(60209977)
	require.NoError(t, err)

	err = pStorage.Insert(firstPulse)
	require.NoError(t, err)
	err = pStorage.Insert(secondPulse)
	require.NoError(t, err)
	err = pStorage.Insert(thirdPulse)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/pulse/range?limit=1" +
		"&fromTimestamp=" + strconv.FormatInt(firstPulse.Timestamp/time.Second.Nanoseconds(), 10) +
		"&toTimestamp=" + strconv.FormatInt(thirdPulse.Timestamp/time.Second.Nanoseconds()+20, 10))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received []int64
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 1)
	require.Equal(t, int64(firstPulse.Number), received[0])
}

func TestPulseRange_PulseNumber(t *testing.T) {
	firstPulse, err := createPulse(60209957)
	require.NoError(t, err)
	secondPulse, err := createPulse(60209967)
	require.NoError(t, err)
	thirdPulse, err := createPulse(60209977)
	require.NoError(t, err)

	err = pStorage.Insert(firstPulse)
	require.NoError(t, err)
	err = pStorage.Insert(secondPulse)
	require.NoError(t, err)
	err = pStorage.Insert(thirdPulse)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/pulse/range?limit=10" +
		"&pulseNumber=" + firstPulse.Number.String() +
		"&fromTimestamp=" + strconv.FormatInt(firstPulse.Timestamp/time.Second.Nanoseconds(), 10) +
		"&toTimestamp=" + strconv.FormatInt(thirdPulse.Timestamp/time.Second.Nanoseconds()+20, 10))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received []int64
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Len(t, received, 2)
	require.Equal(t, int64(secondPulse.Number), received[0])
	require.Equal(t, int64(thirdPulse.Number), received[1])
}
