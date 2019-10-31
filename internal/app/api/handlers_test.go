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

func TestTransaction_WrongFormat(t *testing.T) {
	txID := "123"
	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txID)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
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

	// if limit is > 1000, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/api/transactions/closed?limit=1001")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTransaction_ClosedLimitSingle(t *testing.T) {
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
		FinishPulseRecord: [2]int64{1, 2},
		Type:              models.TTypeMigration,

		MemberFromReference: fromMember.Bytes(),
		MemberToReference:   toMember.Bytes(),
		DepositToReference:  toDeposit.Bytes(),
		StatusFinished: true,
		FinishSuccess: true,
	}

	err = db.Insert(&transaction)
	require.NoError(t, err)

	// request closed transactions using API
	resp, err := http.Get("http://" + apihost + "/api/transactions/closed?limit=1")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	expectedTransaction := SchemaMigration{
		SchemasTransactionAbstract: SchemasTransactionAbstract{
			Amount:      "10",
			Fee:         NullableString("1"),
			Index:       fmt.Sprintf("%d:198", pulseNumber),
			PulseNumber: int64(pulseNumber),
			Status:      "received",
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
	require.Equal(t, received[0], expectedTransaction)
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

	transaction := models.Transaction{
		TransactionID:     txID.Bytes(),
		PulseRecord:       [2]int64{int64(pulseNumber), 198},
		StatusRegistered:  true,
		Amount:            "10",
		Fee:               "1",
		FinishPulseRecord: [2]int64{1, 2},
		Type:              models.TTypeMigration,

		MemberFromReference: fromMember.Bytes(),
		MemberToReference:   toMember.Bytes(),
		DepositToReference:  toDeposit.Bytes(),
	}

	err = db.Insert(&transaction)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txID.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	receivedTransaction := &SchemaMigration{}
	expectedTransaction := &SchemaMigration{
		SchemasTransactionAbstract: SchemasTransactionAbstract{
			Amount:      "10",
			Fee:         NullableString("1"),
			Index:       fmt.Sprintf("%d:198", pulseNumber),
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

	err = json.Unmarshal(bodyBytes, receivedTransaction)
	require.NoError(t, err)
	require.Equal(t, expectedTransaction, receivedTransaction)
}
