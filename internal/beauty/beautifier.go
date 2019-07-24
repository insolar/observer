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
	"encoding/json"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/ledger/heavy/sequence"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member/signer"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/ledger/store"
)

func NewBeautifier() *Beautifier {
	return &Beautifier{txs: make(chan *Transaction, 1)}
}

type HalfRequest struct {
	timestamp int64
	value     *record.IncomingRequest
}

type HalfResponse struct {
	timestamp int64
	value     *record.Result
}

type Beautifier struct {
	Publisher store.DBSetPublisher `inject:""`
	db        *pg.DB
	results   map[insolar.ID]HalfResponse
	requests  map[insolar.ID]HalfRequest
	txs       chan *Transaction
}

type Deposit struct {
	Id              uint `sql:",pk_id"`
	Timestamp       uint
	HoldReleaseDate uint
	Amount          string
	Bonus           string
	EthHash         string
	Status          string
	MemberID        uint
}

type Fee struct {
	Id        uint `sql:",pk_id"`
	AmountMin uint
	AmountMax uint
	Fee       uint
	Status    string
}

type Member struct {
	Id               uint   `sql:",pk_id"`
	Reference        string `sql:",notnull"`
	Balance          string
	MigrationAddress string
}

type Transaction struct {
	tableName struct{} `sql:"transactions"`

	Id            uint                `sql:",pk_id"`
	TxID          string              `sql:",notnull"`
	Amount        string              `sql:",notnull"`
	Fee           string              `sql:",notnull"`
	Timestamp     int64               `sql:",notnull"`
	Pulse         insolar.PulseNumber `sql:",notnull"`
	Status        string              `sql:",notnull"`
	ReferenceFrom string              `sql:",notnull"`
	ReferenceTo   string              `sql:",notnull"`
}

type Record struct {
	tableName struct{} `sql:"records"`

	Key   string
	Value string
	Scope uint
}

// Init initializes connection to db and subscribes beautifier on db updates.
func (b *Beautifier) Init(ctx context.Context) error {
	b.Publisher.Subscribe(func(key store.Key, value []byte) {
		if key.Scope() != store.ScopeRecord {
			return
		}
		pn := insolar.NewPulseNumber(key.ID())
		k := append([]byte{byte(key.Scope())}, key.ID()...)
		b.ParseAndStore([]sequence.Item{{Key: k, Value: value}}, pn)
	})
	// TODO: move connection params to config
	b.db = pg.Connect(&pg.Options{
		User:     "postgres",
		Password: "",
		Database: "postgres",
	})
	return nil
}

func (b *Beautifier) Start(ctx context.Context) error {
	go b.transactionSaver(b.txs)
	return nil
}

// Stop closes connection to db.
func (b *Beautifier) Stop(ctx context.Context) error {
	return b.db.Close()
}

// ParseAndStore consume array of records and pulse number parse them and save to db
func (b *Beautifier) ParseAndStore(records []sequence.Item, pulseNumber insolar.PulseNumber) {
	for i := 0; i < len(records); i++ {
		b.parse(records[i].Key, records[i].Value, pulseNumber)
	}
}

func (b *Beautifier) parse(key []byte, value []byte, pn insolar.PulseNumber) {
	logger := inslogger.FromContext(context.Background())

	id := insolar.ID{}
	copy(id[:], key[1:])
	rec := record.Material{}
	err := rec.Unmarshal(value)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to unmarshal record data"))
		return
	}
	switch rec.Virtual.Union.(type) {
	case *record.Virtual_IncomingRequest:
		in := rec.Virtual.GetIncomingRequest()
		if in.CallType != record.CTGenesis && in.Method == "Call" {
			request := b.parseCallArguments(in.Arguments)
			if request.Params.CallSite == "member.transfer" {
				amount, toMemberReference := b.parseCallParams(request)
				b.txs <- &Transaction{
					TxID:          id.String(),
					Status:        "PENDING",
					Amount:        amount,
					ReferenceFrom: request.Params.Reference,
					ReferenceTo:   toMemberReference,
					Pulse:         pn,
					Timestamp:     int64(pn),
					Fee:           "99999",
				}
			}
		}
	}
}

func (b *Beautifier) parseCallArguments(inArgs []byte) member.Request {
	logger := inslogger.FromContext(context.Background())
	var args []interface{}
	err := insolar.Deserialize(inArgs, &args)
	if err != nil {
		logger.Warn(errors.Wrapf(err, "failed to deserialize request arguments"))
		return member.Request{}
	}

	request := member.Request{}
	if len(args) > 0 {
		if rawRequest, ok := args[0].([]byte); ok {
			var (
				pulseTimeStamp int64
				signature      string
				raw            []byte
			)
			err = signer.UnmarshalParams(rawRequest, &raw, &signature, &pulseTimeStamp)
			if err != nil {
				logger.Warn(errors.Wrapf(err, "failed to unmarshal params"))
				return member.Request{}
			}
			err = json.Unmarshal(raw, &request)
			if err != nil {
				logger.Warn(errors.Wrapf(err, "failed to unmarshal json member request"))
				return member.Request{}
			}
		}
	}
	return request
}

func (b *Beautifier) parseCallParams(request member.Request) (string, string) {
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

func (b *Beautifier) storeTx(tx *Transaction) error {
	_, err := b.db.Model(tx).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}

func (b *Beautifier) storeMember(member *Member) error {
	_, err := b.db.Model(member).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}

func (b *Beautifier) storeDeposit(deposit *Deposit) error {
	_, err := b.db.Model(deposit).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}

func (b *Beautifier) storeFee(fee *Fee) error {
	_, err := b.db.Model(fee).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}

func (b *Beautifier) transactionSaver(txs chan *Transaction) {
	logger := inslogger.FromContext(context.Background())
	for tx := range txs {
		err := b.storeTx(tx)
		if err != nil {
			logger.Error(err)
		}
	}
}
