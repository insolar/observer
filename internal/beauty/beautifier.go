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
	"github.com/insolar/insolar/logicrunner/builtin/contract/wallet"
	"github.com/insolar/insolar/logicrunner/common"
	"github.com/insolar/observer/internal/ledger/store"
	"github.com/pkg/errors"
)

func NewBeautifier() *Beautifier {
	return &Beautifier{}
}

type HalfRequest struct {
	timestamp uint
	value     interface{}
}

type HalfResponse struct {
	timestamp uint
	value     interface{}
}

type Beautifier struct {
	Publisher     store.DBSetPublisher `inject:""`
	db            *pg.DB
	logger        insolar.Logger
	resultParts   map[insolar.ID]HalfResponse // half of requestID/response
	requestsParts map[insolar.ID]HalfRequest  // half of responseID/request
}

type InsDeposit struct {
	Id              uint `sql:",pk_id"`
	Timestamp       uint
	HoldReleaseDate uint
	Amount          string
	Bonus           string
	EthHash         string
	Status          string
	MemberID        uint
}

type InsFee struct {
	Id        uint `sql:",pk_id"`
	AmountMin uint
	AmountMax uint
	Fee       uint
	Status    string
}

type InsMember struct {
	Id               uint   `sql:",pk_id"`
	Reference        string `sql:",notnull"`
	Balance          string
	MigrationAddress string
}

type InsTransaction struct {
	tableName     struct{}            `sql:"transactions"`
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

type InsRecord struct {
	tableName struct{} `sql:"records"`
	Key       string
	Value     string
	Scope     uint
}

// Init initialize connection to db and subscribe for records
func (b *Beautifier) Init(ctx context.Context) error {
	b.Publisher.Subscribe(func(key store.Key, value []byte) {
		if key.Scope() != store.ScopeRecord {
			return
		}
		pn := insolar.NewPulseNumber(key.ID())
		k := append([]byte{byte(key.Scope())}, key.ID()...)
		b.ParseAndStore([]sequence.Item{{Key: k, Value: value}}, pn)
	})
	b.db = pg.Connect(&pg.Options{
		User:     "postgres",
		Password: "",
		Database: "postgres",
	})
	b.logger = inslogger.FromContext(ctx)
	return nil
}

func (b *Beautifier) Start(ctx context.Context) error {
	return nil
}

// Stop close connection to db
func (b *Beautifier) Stop(ctx context.Context) error {
	return b.db.Close()
}

// ParseAndStore consume array of records and pulse number parse them and save to db
func (b *Beautifier) ParseAndStore(records []sequence.Item, pulseNumber insolar.PulseNumber) {
	for i := 0; i < len(records); i++ {
		b.parse(records[i].Key, records[i].Value, pulseNumber)
	}
}

func (b *Beautifier) parse(key []byte, value []byte, pulseNumber insolar.PulseNumber) {

	switch result := b.build(key, value).(type) {
	case InsTransaction:
		err := b.storeTx(result)
		if err != nil {
			b.logger.Error(errors.Wrapf(err, "failed to save transaction"))
		}
	case InsMember:
		err := b.storeMember(result)
		if err != nil {
			b.logger.Error(errors.Wrapf(err, "failed to save member"))
		}
	case InsDeposit:
		err := b.storeDeposit(result)
		if err != nil {
			b.logger.Error(errors.Wrapf(err, "failed to save deposit"))
		}
	case InsFee:
		err := b.storeFee(result)
		if err != nil {
			b.logger.Error(errors.Wrapf(err, "failed to save fee"))
		}
	default:
		b.logger.Debug("not supported type")
	}
}

// model builder
func (b *Beautifier) build(key []byte, value []byte) interface{} {

	id := insolar.ID{}
	copy(id[:], key[1:])
	rec := record.Material{}
	err := rec.Unmarshal(value)
	if err != nil {
		b.logger.Error(errors.Wrapf(err, "failed to unmarshal record data"))
		return ""
	}
	switch rec.Virtual.Union.(type) {
	case *record.Virtual_IncomingRequest:
		in := rec.Virtual.GetIncomingRequest()
		if in.CallType != record.CTGenesis {
			var args []interface{}
			err = insolar.Deserialize(in.Arguments, &args)
			if err != nil {
				b.logger.Error(errors.Wrapf(err, "failed to deserialize arguments"))
				return nil
			}
			if in.Method == "Call" {
				request := member.Request{}
				var pulseTimeStamp int64
				var signature string
				var raw []byte
				if len(args) > 0 {
					if rawRequest, ok := args[0].([]byte); ok {
						err = signer.UnmarshalParams(rawRequest, &raw, &signature, &pulseTimeStamp)
						if err != nil {
							b.logger.Error(errors.Wrapf(err, "failed to unmarshal params"))
							return ""
						}
						err = json.Unmarshal(raw, &request)
					}
				}

				callParams := request.Params.CallParams.(map[string]interface{})
				if request.Params.CallSite == "member.transfer" {
					amount := callParams["amount"]
					toMemberReference := callParams["toMemberReference"]
					return InsTransaction{
						TxID:          id.String(),
						Status:        "PENDING",
						Amount:        amount.(string),
						ReferenceFrom: request.Params.Reference,
						ReferenceTo:   toMemberReference.(string),
						Pulse:         id.Pulse(),
						Timestamp:     pulseTimeStamp,
						Fee:           "99999",
					}
				}
				if request.Params.CallSite == "member.create" {
					b.logger.Info("Catch member create call")
				}
			}

		}
	case *record.Virtual_Result:
		res := rec.Virtual.GetResult()
		b.logger.Infof("res: %v %v %v", res.Request.String(), res.Object.String(), string(res.Payload))
	case *record.Virtual_Activate:
		act := rec.Virtual.GetActivate()
		w := wallet.Wallet{}
		serializer := common.NewCBORSerializer()
		switch {
		case serializer.Deserialize(act.Memory, &w) == nil && w.Balance != "":
			b.logger.Infof("act: (Wallet %v %v) %v %v %v", id.String(), w.Balance, act.Request.String(), act.Parent.String(), string(act.Memory))
		default:
			b.logger.Infof("act: %v %v %v", act.Request.String(), act.Parent.String(), string(act.Memory))
		}
	case *record.Virtual_Amend:
		amn := rec.Virtual.GetAmend()
		w := wallet.Wallet{}
		serializer := common.NewCBORSerializer()
		switch {
		case serializer.Deserialize(amn.Memory, &w) == nil && w.Balance != "":
			b.logger.Infof("amn: (Wallet %v %v) %v %v", id.String(), w.Balance, amn.Request.String(), amn.PrevState.String())
		default:
			b.logger.Infof("amn: %v %v", amn.Request.String(), amn.PrevState.String())
		}
	}
	return ""
}

func (b *Beautifier) storeTx(tx InsTransaction) error {
	// store to db
	_, err := b.db.Model(&tx).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}

func (b *Beautifier) storeMember(member InsMember) error {
	// store to db
	_, err := b.db.Model(&member).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}

func (b *Beautifier) storeDeposit(deposit InsDeposit) error {
	// store to db
	_, err := b.db.Model(&deposit).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}

func (b *Beautifier) storeFee(fee InsFee) error {
	// store to db
	_, err := b.db.Model(&fee).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}
