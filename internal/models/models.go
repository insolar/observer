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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/insolar/insolar/pulse"
)

type JSONMap map[string]interface{}

func (m *JSONMap) Scan(b interface{}) error {
	if b == nil {
		*m = nil
		return nil
	}
	return json.Unmarshal(b.([]byte), m)
}

func (m JSONMap) Value() (driver.Value, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

type Member struct {
	tableName struct{} `sql:"members"` //nolint: unused,structcheck

	Reference        []byte `sql:"member_ref"`
	WalletReference  []byte `sql:"wallet_ref"`
	AccountReference []byte `sql:"account_ref"`
	AccountState     []byte `sql:"account_state"`
	MigrationAddress string `sql:"migration_address"`
	Balance          string `sql:"balance"`
	Status           string `sql:"status"`
	PublicKey        string `sql:"public_key"`
}

type DepositStatus string

const (
	DepositStatusCreated   DepositStatus = "created"
	DepositStatusConfirmed DepositStatus = "confirmed"
)

type Deposit struct {
	tableName struct{} `sql:"deposits"` //nolint: unused,structcheck

	Reference       []byte `sql:"deposit_ref"`
	MemberReference []byte `sql:"member_ref"`
	EtheriumHash    string `sql:"eth_hash"`
	State           []byte `sql:"deposit_state"`
	HoldReleaseDate int64  `sql:"hold_release_date"`
	Amount          string `sql:"amount"`
	Balance         string `sql:"balance"`
	Timestamp       int64  `sql:"transfer_date"`
	DepositNumber   *int64 `sql:"deposit_number"`
	Vesting         int64  `sql:"vesting"`
	VestingStep     int64  `sql:"vesting_step"`

	InnerStatus DepositStatus `sql:"status"`
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

	CallParams JSONMap `sql:"call_params"`

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

type Pulse struct {
	tableName struct{} `sql:"pulses"` //nolint: unused,structcheck

	Pulse     uint32 `sql:"pulse,notnull"`
	PulseDate int64  `sql:"pulse_date"`
	Entropy   []byte `sql:"entropy"`
	Nodes     uint32 `sql:"nodes"`
}

type NetworkStats struct {
	tableName struct{} `sql:"network_stats"` //nolint: unused,structcheck

	Created           time.Time `sql:"created,pk,default:now(),notnull"`
	PulseNumber       int       `sql:"pulse_number,notnull"`
	TotalTransactions int       `sql:"total_transactions,notnull"`
	MonthTransactions int       `sql:"month_transactions,notnull"`
	TotalAccounts     int       `sql:"total_accounts,notnull"`
	Nodes             int       `sql:"nodes,notnull"`
	CurrentTPS        int       `sql:"current_tps,notnull"`
	MaxTPS            int       `sql:"max_tps,notnull"`
}

type SupplyStats struct {
	tableName struct{} `sql:"supply_stats"` //nolint: unused,structcheck

	Created time.Time `sql:"created,pk,default:now(),notnull"`
	Total   string    `sql:"total"`
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

func (d Deposit) Fields() []string {
	tType := reflect.TypeOf(d)
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

func (d *Deposit) ReleaseAmount(balance, amount *big.Int, currentTime int64) (amountOnHold *big.Int, releaseAmount *big.Int) {
	if d.HoldReleaseDate == 0 {
		return big.NewInt(0), amount
	}

	if currentTime <= d.HoldReleaseDate {
		return amount, big.NewInt(0)
	}

	if currentTime >= (d.Vesting + d.HoldReleaseDate) {
		return big.NewInt(0), amount
	}

	currentStep := (currentTime - d.HoldReleaseDate) / d.VestingStep
	steps := d.Vesting / d.VestingStep
	releaseAmount = big.NewInt(0).Quo(
		big.NewInt(0).Mul(amount, big.NewInt(currentStep)),
		big.NewInt(steps),
	)

	amountOnHold = big.NewInt(0).Sub(amount, releaseAmount)

	// if amountOnHold greater then balance,
	// then it should be balance
	if amountOnHold.Cmp(balance) == 1 {
		amountOnHold = balance
	}

	// if releaseAmount greater then balance,
	// then it should be balance
	if releaseAmount.Cmp(balance) == 1 {
		releaseAmount = balance
	}

	return amountOnHold, releaseAmount
}

func (d *Deposit) Status(currentTime int64) string {
	if d.HoldReleaseDate == 0 {
		return "AVAILABLE"
	}
	if currentTime < d.HoldReleaseDate {
		return "LOCKED"
	}
	return "AVAILABLE"
}

func (s *SupplyStats) TotalInXNS() string {
	return convertCoinsToXNS(s.Total)
}

// convertCoinsToXNS places decimal point correctly into string to convert
// from coins to XNS
func convertCoinsToXNS(str string) string {
	l := len(str)

	switch {
	case l == 0:
		return ""
	case l <= 10:
		return fmt.Sprintf("0.%010s", str)
	default:
		return str[:l-10] + "." + str[l-10:]
	}
}
