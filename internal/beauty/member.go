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
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Member struct {
	tableName struct{} `sql:"members"`

	MemberRef        string `sql:",pk"`
	Balance          string `sql:",notnull"`
	MigrationAddress string
	WalletState      string `sql:",notnull"`
	Status           string

	requestID insolar.ID
}

func (b *Beautifier) processMemberCreate(pn insolar.PulseNumber, id insolar.ID, in *record.IncomingRequest, request member.Request) {
	status := PENDING
	migrationAddress := ""
	if result, ok := b.results[id]; ok {
		status, migrationAddress = memberStatus(result.value.Payload)
	} else {
		b.requests[id] = SuspendedRequest{timestamp: time.Now().Unix(), value: in}
	}
	if _, ok := b.members[id]; !ok {
		b.members[id] = &Member{
			MemberRef:        id.String(),
			Balance:          "",
			MigrationAddress: migrationAddress,
			Status:           status,
			requestID:        id,
		}
	}
}

func (b *Beautifier) processMemberCreateResult(rec insolar.ID, res *record.Result) {
	member, ok := b.members[rec]
	if !ok {
		log.Error(errors.New("failed to get cached transaction"))
		return
	}
	status, migrationAddress := memberStatus(res.Payload)
	member.Status = status
	member.MigrationAddress = migrationAddress
}

func memberStatus(payload []byte) (string, string) {
	rets := parsePayload(payload)
	if len(rets) < 2 {
		return "NOT_ENOUGH_PAYLOAD_PARAMS", ""
	}
	if retError, ok := rets[1].(error); ok {
		if retError != nil {
			return CANCELED, ""
		}
	}
	params, ok := rets[0].(map[string]interface{})
	if !ok {
		return "FIRST_PARAM_NOT_MAP", ""
	}
	migrationAddressInterface, ok := params["migrationAddress"]
	if !ok {
		return SUCCESS, ""
	}
	migrationAddress, ok := migrationAddressInterface.(string)
	if !ok {
		return "MIGRATION_ADDRESS_NOT_STRING", ""
	}
	return SUCCESS, migrationAddress
}

func (b *Beautifier) processNewWallet(pn insolar.PulseNumber, id insolar.ID, in *record.IncomingRequest) {
	status := PENDING
	migrationAddress := ""
	balance := ""
	walletState := ""
	if act, ok := b.activates[id]; !ok {
		b.intentions[id] = SuspendedIntention{timestamp: time.Now().Unix(), value: in}
	} else {
		walletState = act.id.String()
		balance = initialBalance(act.value)
	}
	origin := *in.Reason.Record()
	if res, ok := b.results[origin]; !ok {
		b.intentions[id] = SuspendedIntention{timestamp: time.Now().Unix(), value: in}
	} else {
		status, migrationAddress = memberStatus(res.value.Payload)
	}
	if _, ok := b.members[origin]; !ok {
		b.members[origin] = &Member{
			MemberRef:        origin.String(),
			Balance:          balance,
			MigrationAddress: migrationAddress,
			WalletState:      walletState,
			Status:           status,
			requestID:        origin,
		}
	}
}

func (b *Beautifier) processWalletActivate(id insolar.ID, direct *record.IncomingRequest, act *record.Activate) {
	origin := *direct.Reason.Record()
	member, ok := b.members[origin]
	if !ok {
		log.Error(errors.New("failed to get cached transaction"))
		return
	}
	balance := initialBalance(act)
	member.WalletState = id.String()
	member.Balance = balance
}

func (b *Beautifier) processWalletAmend(id insolar.ID, amd *record.Amend) {
	balance := walletBalance(amd)
	b.balanceUpdates[id] = BalanceUpdate{
		timestamp: time.Now().Unix(),
		id:        id,
		prevState: amd.PrevState.String(),
		balance:   balance,
	}
}

func storeMember(tx *pg.Tx, member *Member) error {
	_, err := tx.Model(member).OnConflict("(id) DO UPDATE").Insert()
	if err != nil {
		return err
	}
	return nil
}

func updateBalance(tx *pg.Tx, id insolar.ID, prevState, balance string) error {
	res, err := tx.Model(&Member{}).
		Set("balance=?,wallet_state=?", balance, id.String()).
		Where("wallet_state=?", prevState).
		Update()
	if err != nil {
		return errors.Wrapf(err, "failed to update member balance by amend record")

	}
	if res.RowsAffected() != 1 {
		return errors.Errorf("failed to update member balance by amend record res=%v", res)
	}
	return nil
}
