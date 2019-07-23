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
	"github.com/insolar/insolar/logicrunner/common"
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
	db       *pg.DB
	logger   insolar.Logger
	requests map[uint]HalfRequest  // half of request/response
	results  map[uint]HalfResponse // half of response/request
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
	tableName     struct{} `sql:"transactions"`
	Id            uint     `sql:",pk_id"`
	TxID          string   `sql:",notnull"`
	Amount        string   `sql:",notnull"`
	Fee           string   `sql:",notnull"`
	Timestamp     uint     `sql:",notnull"`
	Pulse         uint     `sql:",notnull"`
	Status        string   `sql:",notnull"`
	ReferenceFrom string   `sql:",notnull"`
	ReferenceTo   string   `sql:",notnull"`
}

func (b *Beautifier) Init(ctx context.Context) error {
	b.db = pg.Connect(&pg.Options{
		User:     "postgres",
		Password: "",
		Database: "postgres",
	})
	// Conf from env
	//	Addr:     os.Getenv("DB_HOST")+":"+os.Getenv("DB_PORT"),
	//	User:     os.Getenv("DB_USER"),
	//	Password: os.Getenv("DB_PASS"),
	//	Database: os.Getenv("DB_NAME"),
	b.logger = inslogger.FromContext(ctx)
	return nil
}

func (b *Beautifier) Start(ctx context.Context) error {
	// WorkFlow
	// Start from previous work
	// Take chunk of raw data and insert in db (it can be tx or account creation)
	// save done work in db
	return nil
}

func (b *Beautifier) Stop(ctx context.Context) error {
	return b.db.Close()
}

func (b *Beautifier) ParseAndStore(records []sequence.Item, pulseNumber insolar.PulseNumber) {
	for i := len(records); i <= len(records); i++ {
		b.parse(records[i], pulseNumber)
	}
}

func (b *Beautifier) parse(record sequence.Item, pulseNumber insolar.PulseNumber) {

	switch v := b.build(record.Key, record.Value).(type) {
	case InsTransaction:
		err := b.storeTx(v)
		if err != nil {
			b.logger.Error(errors.Wrapf(err, "failed to save transaction"))
		}
	case InsMember:
		err := b.storeMember(v)
		if err != nil {
			b.logger.Error(errors.Wrapf(err, "failed to save member"))
		}
	case InsDeposit:
		err := b.storeDeposit(v)
		if err != nil {
			b.logger.Error(errors.Wrapf(err, "failed to save deposit"))
		}
	case InsFee:
		err := b.storeFee(v)
		if err != nil {
			b.logger.Error(errors.Wrapf(err, "failed to save fee"))
		}
	default:
		b.logger.Debug("not supported type")
	}
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

func (b *Beautifier) build(key []byte, value []byte) interface{} {
	var (
		err error
		// key   insolar.ID
		// value []byte
	)

	id := insolar.ID{}
	copy(id[:], key[1:])
	// log.Infof("pulse: %v", id.Pulse())
	if key[0] == 2 {
		rec := record.Material{}
		err = rec.Unmarshal(value)
		if err != nil {
			b.logger.Error(errors.Wrapf(err, "failed to unmarshal record data"))
			return nil
		}
		switch rec.Virtual.Union.(type) {
		case *record.Virtual_IncomingRequest:
			in := rec.Virtual.GetIncomingRequest()
			if in.CallType != record.CTGenesis {
				args := []interface{}{}
				insolar.Deserialize(in.Arguments, &args)
				request := member.Request{}
				if len(args) > 0 {
					if rawRequest, ok := args[0].([]byte); ok {
						var signature string
						var pulseTimeStamp int64
						var raw []byte
						signer.UnmarshalParams(rawRequest, &raw, &signature, &pulseTimeStamp)
						// logger.Infof("RAW: %v %v ", signature, string(rawRequest))
						err = json.Unmarshal(raw, &request)
					}
				}
				if in.Method == "Transfer" {
					ref := insolar.Reference{}.FromSlice(args[1].([]byte))
					b.logger.Infof("TRANSFER amount: %v toMember: %v", args[0], ref.String())
				}
				callParams, _ := request.Params.CallParams.(map[string]interface{})
				b.logger.Infof("in %v %v %v toMember: %v", in.Method, id.String(), request.Params.CallSite, callParams["toMemberReference"])
			}
		// case *record.Virtual_OutgoingRequest:
		// 	out := rec.Virtual.GetOutgoingRequest()
		// 	logger.Infof("out %v %v %v", id.Pulse(), out.Method, string(out.Arguments))
		// 	logger.Infof("out method:%v type:%v args:%v", out.Method, out.CallType.String(), string(out.Arguments))
		// case *record.Virtual_Type: // TODO: узнать про Type
		// 	t := rec.Virtual.GetType()
		// 	logger.Infof("type: %v", t)
		// logger.Infof("Rec type: %v", reflect.TypeOf(rec.Virtual.Union).String())
		// switch rec.Virtual.Union.(type) {
		// case *record.Virtual_IncomingRequest:
		// 	in := rec.Virtual.GetIncomingRequest()
		// 	logger.Infof("in: %v", in)
		// }
		case *record.Virtual_Result:
			res := rec.Virtual.GetResult()
			// res.
			// serializer := common.NewCBORSerializer()
			// rets := []interface{}{}
			// if err := serializer.Deserialize(res.Payload, &rets); err == nil && len(rets) > 0 {
			// 	msg, ok := rets[0].(string)
			// 	if ok && msg == "" {
			// 		logger.Infof("Success transfer if it was transfer %v", res.Request.String())
			// 	}
			// }
			b.logger.Infof("res: %v %v %v", res.Request.String(), res.Object.String(), string(res.Payload))
		case *record.Virtual_Activate:
			act := rec.Virtual.GetActivate()
			// m := member.Member{}
			w := wallet.Wallet{}
			serializer := common.NewCBORSerializer()
			switch {
			// case serializer.Deserialize(act.Memory, &m) == nil:
			// 	logger.Infof("act: (Member %v %v) %v %v %v", id.String(), m.Name, act.Request.String(), act.Parent.String(), string(act.Memory))
			case serializer.Deserialize(act.Memory, &w) == nil && w.Balance != "":
				b.logger.Infof("act: (Wallet %v %v) %v %v %v", id.String(), w.Balance, act.Request.String(), act.Parent.String(), string(act.Memory))
			default:
				b.logger.Infof("act: %v %v %v", act.Request.String(), act.Parent.String(), string(act.Memory))
			}
		case *record.Virtual_Amend:
			amn := rec.Virtual.GetAmend()
			// m := member.Member{}
			w := wallet.Wallet{}
			serializer := common.NewCBORSerializer()
			switch {
			// case serializer.Deserialize(amn.Memory, &m) == nil:
			// 	logger.Infof("amn: (Member %v %v %v) %v %v %v", id.String(), m.Name, m.PublicKey, amn.Request.String(), amn.PrevState.String(), string(amn.Memory))
			case serializer.Deserialize(amn.Memory, &w) == nil && w.Balance != "":
				b.logger.Infof("amn: (Wallet %v %v) %v %v %v", id.String(), w.Balance, amn.Request.String(), amn.PrevState.String(), string(amn.Memory))
			default:
				b.logger.Infof("amn: %v %v %v", amn.Request.String(), amn.PrevState.String(), string(amn.Memory))
			}
		}
	}
}
