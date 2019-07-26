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

package beauty

import (
	"context"
	"errors"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
)

type Transaction struct {
	tableName struct{} `sql:"transactions"`

	Id            uint                `sql:",pk_id"`
	TxID          string              `sql:",notnull"`
	Amount        string              `sql:",notnull"`
	Fee           string              `sql:",notnull"`
	Timestamp     int64               `sql:",notnull"`
	Pulse         insolar.PulseNumber `sql:",notnull"`
	Status        string              `sql:",notnull"`
	ReferenceTo   string              `sql:",notnull"`
	ReferenceFrom string              `sql:",notnull"`
}

func (b Beautifier) processTransferCall(pn insolar.PulseNumber, id insolar.ID, in *record.IncomingRequest, request member.Request) {
	amount, toMemberReference := b.parseTransferCallParams(request)
	status := "PENDING"
	fee := "0"
	if result, ok := b.results[id]; ok {
		status, fee = txStatus(result.value.Payload)
	} else {
		b.requests[id] = SuspendedRequest{timestamp: time.Now().Unix(), value: in}
	}
	b.txs[id] = &Transaction{
		TxID:          id.String(),
		Status:        status,
		Amount:        amount,
		ReferenceTo:   request.Params.Reference,
		ReferenceFrom: toMemberReference,
		Pulse:         pn,
		Timestamp:     int64(pn),
		Fee:           fee,
	}
}

func (b *Beautifier) processTransferResult(pn insolar.PulseNumber, rec *insolar.ID, res *record.Result) {
	logger := inslogger.FromContext(context.Background())
	tx, ok := b.txs[*rec]
	if !ok {
		logger.Error(errors.New("failed to get cached transaction"))
		return
	}
	status, fee := txStatus(res.Payload)
	tx.Status = status
	tx.Fee = fee
}

func (b *Beautifier) parseTransferCallParams(request member.Request) (string, string) {
	var (
		logger = inslogger.FromContext(context.Background())
		amount = ""
		to     = ""
	)
	callParams, ok := request.Params.CallParams.(map[string]interface{})
	if !ok {
		logger.Warnf("failed to cast CallParams to map[string]interface{}")
		return "", ""
	}
	if a, ok := callParams["amount"]; ok {
		if amount, ok = a.(string); !ok {
			logger.Warnf(`failed to cast CallParams["amount"] to string`)
		}
	} else {
		logger.Warnf(`failed to get CallParams["amount"]`)
	}
	if t, ok := callParams["toMemberReference"]; ok {
		if to, ok = t.(string); !ok {
			logger.Warnf(`failed to cast CallParams["toMemberReference"] to string`)
		}
	} else {
		logger.Warnf(`failed to get CallParams["toMemberReference"]`)
	}
	return amount, to
}

func txStatus(payload []byte) (string, string) {
	rets := parsePayload(payload)
	if len(rets) < 2 {
		return "NOT_ENOUGH_PAYLOAD_PARAMS", ""
	}
	if retError, ok := rets[1].(error); ok {
		if retError != nil {
			return "CANCELED", ""
		}
	}
	params, ok := rets[0].(map[string]interface{})
	if !ok {
		return "FIRST_PARAM_NOT_MAP", ""
	}
	feeInterface, ok := params["fee"]
	if !ok {
		return "FEE_PARAM_NOT_EXIST", ""
	}
	fee, ok := feeInterface.(string)
	if !ok {
		return "FEE_NOT_STRING", ""
	}
	return "SUCCESS", fee
}

func (b *Beautifier) storeTx(tx *Transaction) error {
	_, err := b.db.Model(tx).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}
