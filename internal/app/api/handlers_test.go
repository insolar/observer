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
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/insolar/observer/internal/app/api/observerapi"
	"github.com/insolar/observer/internal/models"
)

func TestTransaction_Invalid(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/transaction/123")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTransaction_NoContent(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/transaction/" + gen.ID().String())
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestTransaction_SingleRecord(t *testing.T) {
	txID := gen.RecordReference()
	time := float32(1572428401)
	pulseNumber := int64(26193138)

	transaction := models.Transaction{
		TransactionID:     txID.Bytes(),
		PulseRecord:       [2]int64{pulseNumber, 198},
		StatusRegistered:  true,
		Amount:            "10",
		Fee:               "1",
		FinishPulseRecord: [2]int64{1, 2},
		Type:              models.TTypeMigration,
	}

	err := db.Insert(&transaction) // AALEKSEEV TODO <--- example
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/transaction/" + txID.String())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	receivedTransaction := &observerapi.SchemasTransactionAbstract{}
	expectedTransaction := &observerapi.SchemasTransactionAbstract{
		Amount:      "10",
		Fee:         NullableString("1"),
		Index:       fmt.Sprintf("%d:198", pulseNumber),
		PulseNumber: pulseNumber,
		Status:      "pending",
		Timestamp:   time,
		TxID:        txID.String(),
		Type:        string(models.TTypeMigration),
	}

	err = json.Unmarshal(bodyBytes, receivedTransaction)
	require.NoError(t, err)
	require.Equal(t, expectedTransaction, receivedTransaction)
}
