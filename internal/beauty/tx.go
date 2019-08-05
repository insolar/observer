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
	"errors"
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	log "github.com/sirupsen/logrus"
)

type Transfer struct {
	tableName struct{} `sql:"transactions"`

	ID            uint                `sql:",pk_id"`
	TxID          string              `sql:",notnull"`
	Amount        string              `sql:",notnull"`
	Fee           string              `sql:",notnull"`
	TransferDate  int64               `sql:",notnull"`
	PulseNum      insolar.PulseNumber `sql:",notnull"`
	Status        string              `sql:",notnull"`
	MemberFromRef string              `sql:",notnull"`
	MemberToRef   string              `sql:",notnull"`
	WalletFromRef string              `sql:",notnull"`
	WalletToRef   string              `sql:",notnull"`
	EthHash       string              `sql:",notnull"`

	requestID insolar.ID
}

func (b Beautifier) processTransferCall(pn insolar.PulseNumber, id insolar.ID, in *record.IncomingRequest, request member.Request) {
	callParams := b.parseTransferCallParams(request)
	r := txResult{status: PENDING, fee: "0"}
	if result, ok := b.results[id]; ok {
		r = txStatus(result.value.Payload)
	} else {
		b.requests[id] = SuspendedRequest{timestamp: time.Now().Unix(), value: in}
	}
	b.txs[id] = &Transfer{
		TxID:          id.String(),
		Status:        r.status,
		Amount:        callParams.amount,
		MemberToRef:   request.Params.Reference,
		MemberFromRef: callParams.toMemberReference,
		PulseNum:      pn,
		TransferDate:  int64(pn),
		Fee:           r.fee,
		WalletFromRef: "TODO",
		WalletToRef:   "TODO",
		EthHash:       "TODO",
		requestID:     id,
	}
}

func (b *Beautifier) processTransferResult(pn insolar.PulseNumber, rec insolar.ID, res *record.Result) {
	tx, ok := b.txs[rec]
	if !ok {
		log.Error(errors.New("failed to get cached transaction"))
		return
	}
	result := txStatus(res.Payload)
	tx.Status = result.status
	tx.Fee = result.fee
}

type transferCallParams struct {
	amount            string
	toMemberReference string
}

func (b *Beautifier) parseTransferCallParams(request member.Request) transferCallParams {
	var (
		amount = ""
		to     = ""
	)
	callParams, ok := request.Params.CallParams.(map[string]interface{})
	if !ok {
		log.Warnf("failed to cast CallParams to map[string]interface{}")
		return transferCallParams{}
	}
	if a, ok := callParams["amount"]; ok {
		if amount, ok = a.(string); !ok {
			log.Warnf(`failed to cast CallParams["amount"] to string`)
		}
	} else {
		log.Warnf(`failed to get CallParams["amount"]`)
	}
	if t, ok := callParams["toMemberReference"]; ok {
		if to, ok = t.(string); !ok {
			log.Warnf(`failed to cast CallParams["toMemberReference"] to string`)
		}
	} else {
		log.Warnf(`failed to get CallParams["toMemberReference"]`)
	}
	return transferCallParams{
		amount:            amount,
		toMemberReference: to,
	}
}

type txResult struct {
	status string
	fee    string
}

func txStatus(payload []byte) txResult {
	rets := parsePayload(payload)
	if len(rets) < 2 {
		return txResult{status: "NOT_ENOUGH_PAYLOAD_PARAMS", fee: ""}
	}
	if retError, ok := rets[1].(error); ok {
		if retError != nil {
			return txResult{status: CANCELED, fee: ""}
		}
	}
	params, ok := rets[0].(map[string]interface{})
	if !ok {
		return txResult{status: "FIRST_PARAM_NOT_MAP", fee: ""}
	}
	feeInterface, ok := params["fee"]
	if !ok {
		return txResult{status: "FEE_PARAM_NOT_EXIST", fee: ""}
	}
	fee, ok := feeInterface.(string)
	if !ok {
		return txResult{status: "FEE_NOT_STRING", fee: ""}
	}
	return txResult{status: SUCCESS, fee: fee}
}

func storeTransfer(tx *pg.Tx, transaction *Transfer) error {
	_, err := tx.Model(transaction).OnConflict("(id) DO UPDATE").Insert()
	if err != nil {
		return err
	}
	return nil
}
