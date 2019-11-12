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

package models

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/insolar/insolar/pulse"
)

type Member struct {
	tableName struct{} `sql:"members"` //nolint: unused,structcheck

	Reference        []byte `sql:"member_ref"`
	WalletReference  []byte `sql:"wallet_ref"`
	AccountReference []byte `sql:"account_ref"`
	AccountState     []byte `sql:"account_state"`
	MigrationAddress string `sql:"migration_address"`
	Balance          string `sql:"balance"`
	Status           string `sql:"status"`
}

type Deposit struct {
	tableName struct{} `sql:"deposits"` //nolint: unused,structcheck

	Reference       []byte `sql:"deposit_ref"`
	MemberReference []byte `sql:"member_ref"`
	EtheriumHash    string `sql:"eth_hash"`
	State           []byte `sql:"deposit_state"`
	HoldReleaseDate int64  `sql:"hold_release_date"`
	Amount          string `sql:"amount"`
	Balance         string `sql:"balance"`
	TransferDate    int64  `sql:"transfer_date"` // TODO: Do we really need it?
	DepositNumber   int64  `sql:"deposit_number"`
	Vesting         int64  `sql:"vesting"`
	VestingStep     int64  `sql:"vesting_step"`
}

type TransactionStatus string

const (
	TStatusUnknown    TransactionStatus = "unknown"
	TStatusRegistered TransactionStatus = "registered"
	TStatusSent       TransactionStatus = "sent"
	TStatusReceived   TransactionStatus = "received"
	TStatusFailed     TransactionStatus = "failed"
)

type TransactionType string

const (
	TTypeUnknown   TransactionType = "unknown"
	TTypeTransfer  TransactionType = "transfer"
	TTypeMigration TransactionType = "migration"
	TTypeRelease   TransactionType = "release"
)

type TxIndexType int

const (
	TxIndexTypePulseRecord       TxIndexType = 1
	TxIndexTypeFinishPulseRecord TxIndexType = 2
)

type Transaction struct {
	tableName struct{} `sql:"simple_transactions"` //nolint: unused,structcheck

	// Indexes.
	ID            int64  `sql:"id"`
	TransactionID []byte `sql:"tx_id"`

	// Request registered.
	StatusRegistered     bool            `sql:"status_registered"`
	Type                 TransactionType `sql:"type"`
	PulseRecord          [2]int64        `sql:"pulse_record" pg:",array"`
	MemberFromReference  []byte          `sql:"member_from_ref"`
	MemberToReference    []byte          `sql:"member_to_ref"`
	DepositToReference   []byte          `sql:"deposit_to_ref"`
	DepositFromReference []byte          `sql:"deposit_from_ref"`
	Amount               string          `sql:"amount"`
	Fee                  string          `sql:"fee"`

	// Result received.
	StatusSent bool `sql:"status_sent"`

	// Saga result received.
	StatusFinished    bool     `sql:"status_finished"`
	FinishSuccess     bool     `sql:"finish_success"`
	FinishPulseRecord [2]int64 `sql:"finish_pulse_record" pg:",array"`
}

type MigrationAddress struct {
	tableName struct{} `sql:"migration_addresses"` //nolint: unused,structcheck

	ID        int64  `sql:"id,notnull"`
	Addr      string `sql:"addr,notnull"`
	Timestamp int64  `sql:"timestamp,notnull"`
	Wasted    bool   `sql:"wasted,notnull"`
}

type Notification struct {
	tableName struct{} `sql:"notifications"` //nolint: unused,structcheck

	Message string    `sql:"message,notnull"`
	Start   time.Time `sql:"start,notnull"`
	Stop    time.Time `sql:"stop,notnull"`
}

type fieldCache struct {
	sync.Mutex
	cache map[reflect.Type][]string
}

var fieldsCache = fieldCache{
	cache: make(map[reflect.Type][]string),
}

func getFields(tType reflect.Type) []string {
	fieldsCache.Lock()
	defer fieldsCache.Unlock()

	if fields, ok := fieldsCache.cache[tType]; ok {
		return append(fields[:0:0], fields...)
	}
	fieldsCache.cache[tType] = getFieldList(tType)
	fields := fieldsCache.cache[tType]
	return append(fields[:0:0], fields...)
}

func (t Transaction) Fields() []string {
	tType := reflect.TypeOf(t)
	return getFields(tType)
}

func (m Member) Fields() []string {
	tType := reflect.TypeOf(m)
	return getFields(tType)
}

func (deposit Deposit) Fields() []string {
	tType := reflect.TypeOf(deposit)
	return getFields(tType)
}

func (t Transaction) QuotedFields() []string {
	fields := t.Fields()
	for i := range fields {
		fields[i] = fmt.Sprintf("'%s'", fields[i])
	}
	return fields
}

func (ma MigrationAddress) Fields() []string {
	tType := reflect.TypeOf(ma)
	return getFields(tType)
}

func getFieldList(t reflect.Type) []string {
	var fieldList []string

	for i := 0; i < t.NumField(); i++ {
		// ignore tableName
		if t.Field(i).Name == "tableName" {
			continue
		}
		tag := t.Field(i).Tag.Get("sql")
		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}
		fieldList = append(fieldList, tag)
	}

	return fieldList
}

func (t *Transaction) Status() TransactionStatus {
	registered := t.StatusRegistered
	sent := t.StatusRegistered && t.StatusSent
	finished := t.StatusRegistered && t.StatusFinished

	if finished {
		if t.FinishSuccess {
			return TStatusReceived
		}
		return TStatusFailed
	}
	if sent {
		return TStatusSent
	}
	if registered {
		return TStatusRegistered
	}

	return TStatusUnknown
}

func (t *Transaction) PulseNumber() int64 {
	return t.PulseRecord[0]
}

func (t *Transaction) RecordNumber() int64 {
	return t.PulseRecord[1]
}

func (t *Transaction) Index(indexType TxIndexType) string {
	var result string
	switch indexType {
	case TxIndexTypeFinishPulseRecord:
		result = fmt.Sprintf("%d:%d", t.FinishPulseRecord[0], t.FinishPulseRecord[1])
	default: // TxIndexTypePulseRecord
		result = fmt.Sprintf("%d:%d", t.PulseRecord[0], t.PulseRecord[1])
	}
	return result
}

func (t *Transaction) Timestamp() int64 {
	p := t.PulseNumber()
	pulseTime, err := pulse.Number(p).AsApproximateTime()
	if err != nil {
		return 0
	}
	return pulseTime.Unix()
}

func (deposit *Deposit) ReleaseAmount(currentTime int64) int64 {
	amount, _ := strconv.ParseInt(deposit.Amount, 10, 64)

	if deposit.HoldReleaseDate == 0 {
		return amount
	}

	if currentTime <= deposit.HoldReleaseDate {
		return 0
	}

	if currentTime >= (deposit.Vesting + deposit.HoldReleaseDate) {
		return amount
	}

	currentStep := (currentTime - deposit.HoldReleaseDate) / deposit.VestingStep
	stepValue := float64(deposit.VestingStep) / float64(deposit.Vesting)
	releasedCoef := float64(currentStep) * stepValue
	amountFloat := big.NewFloat(float64(amount))
	releaseAmount := new(big.Float).Mul(big.NewFloat(releasedCoef), amountFloat)
	res, _ := releaseAmount.Int64()

	return res
}

func (deposit *Deposit) Status(currentTime int64) string {
	if deposit.HoldReleaseDate == 0 {
		return "AVAILABLE"
	}
	if currentTime <= deposit.HoldReleaseDate {
		return "LOCKED"
	}
	return "AVAILABLE"
}
