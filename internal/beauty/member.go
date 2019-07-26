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

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"golang.org/x/net/context"
)

type Account struct {
	tableName struct{} `sql:"accounts"`

	Id               uint   `sql:",pk_id"`
	Reference        string `sql:",notnull"`
	Status           string `sql:",notnull"`
	Balance          string `sql:",notnull"`
	MigrationAddress string
}

func (b *Beautifier) processMemberCreate(pn insolar.PulseNumber, id insolar.ID, in *record.IncomingRequest, request member.Request) {
	status := "PENDING"
	mirationAddress := ""
	if result, ok := b.results[id]; ok {
		status, mirationAddress = accountStatus(result.value.Payload)
	} else {
		b.requests[id] = SuspendedRequest{timestamp: time.Now().Unix(), value: in}
	}
	b.accounts[id] = &Account{
		Reference:        id.String(),
		Status:           status,
		Balance:          "0",
		MigrationAddress: mirationAddress,
	}
}

func (b *Beautifier) processMemberCreateResult(pn insolar.PulseNumber, rec *insolar.ID, res *record.Result) {
	logger := inslogger.FromContext(context.Background())
	account, ok := b.accounts[*rec]
	if !ok {
		logger.Error(errors.New("failed to get cached transaction"))
		return
	}
	status, mirationAddress := accountStatus(res.Payload)
	account.Status = status
	account.MigrationAddress = mirationAddress
}

func accountStatus(payload []byte) (string, string) {
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
	migrationAddressInterface, ok := params["migrationAddress"]
	if !ok {
		return "SUCCESS", ""
	}
	migrationAddress, ok := migrationAddressInterface.(string)
	if !ok {
		return "MIGRATION_ADDRESS_NOT_STRING", ""
	}
	return "SUCCESS", migrationAddress
}

func (b *Beautifier) storeAccount(account *Account) error {
	_, err := b.db.Model(account).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}
