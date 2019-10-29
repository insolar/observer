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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/insolar/observer/internal/app/api/observerapi"
	"github.com/insolar/observer/internal/models"
	"github.com/stretchr/testify/require"
)

func TestTransaction_NoContent(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/transaction/123")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestTransaction_SingleRecord(t *testing.T) {
	txID := "123"

	transaction := models.Transaction{
		TransactionID:    []byte(txID),
		PulseNumber:      1,
		StatusRegistered: true,
		Amount:           "10",
		Fee:              "1",
	}

	err := db.Insert(&transaction)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	receivedTransaction := &observerapi.SchemasTransactionAbstract{}
	expectedTransaction := &observerapi.SchemasTransactionAbstract{
		Amount:      "10",
		Fee:         "1",
		Index:       0,
		PulseNumber: 1,
		Status:      "pending",
		Timestamp:   0,
		TxID:        txID,
		Type:        "unknown",
	}

	err = json.Unmarshal(bodyBytes, receivedTransaction)
	require.NoError(t, err)
	require.Equal(t, expectedTransaction, receivedTransaction)
}
