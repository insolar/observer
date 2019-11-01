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

	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/internal/app/observer/postgres"
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
			Index:       fmt.Sprintf("%d:198", pulseNumber),
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

func TestTransaction_ClosedLimitMultiple(t *testing.T) {
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

	// request two recent closed transactions using API
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

	resp, err := http.Get("http://" + apihost + "/coins")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	jsonResp := component.XnsCoinStats{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	expected := component.XnsCoinStats{
		Total:       totalr,
		Max:         maxr,
		Circulating: circr,
	}
	require.Equal(t, expected, jsonResp)

	resp, err = http.Get("http://" + apihost + "/coins/total")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, totalr, string(bodyBytes))

	resp, err = http.Get("http://" + apihost + "/coins/max")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, maxr, string(bodyBytes))

	resp, err = http.Get("http://" + apihost + "/coins/circulating")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, circr, string(bodyBytes))
}
