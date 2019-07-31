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
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/ledger/heavy/sequence"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member/signer"
	"github.com/insolar/observer/internal/ledger/store"
	"github.com/pkg/errors"
)

func NewBeautifier() *Beautifier {
	return &Beautifier{
		requests:    make(map[insolar.ID]SuspendedRequest),
		results:     make(map[insolar.ID]HeadlessResult),
		txs:         make(map[insolar.ID]*Transaction),
		members:     make(map[insolar.ID]*Member),
		rawObjects:  make(map[insolar.ID]*Object),
		rawResults:  make(map[insolar.ID]*Result),
		rawRequests: make(map[insolar.ID]*Request),
	}
}

type SuspendedRequest struct {
	timestamp int64
	value     *record.IncomingRequest
}

type HeadlessResult struct {
	timestamp int64
	value     *record.Result
}

type Beautifier struct {
	Publisher   store.DBSetPublisher `inject:""`
	db          *pg.DB
	prevPulse   insolar.PulseNumber
	requests    map[insolar.ID]SuspendedRequest
	results     map[insolar.ID]HeadlessResult
	txs         map[insolar.ID]*Transaction
	members     map[insolar.ID]*Member
	rawObjects  map[insolar.ID]*Object
	rawResults  map[insolar.ID]*Result
	rawRequests map[insolar.ID]*Request
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
	if err := b.db.CreateTable(&Transaction{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		return err
	}
	if err := b.db.CreateTable(&Member{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		return err
	}
	if err := b.db.CreateTable(&Object{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		return err
	}
	if err := b.db.CreateTable(&Request{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		return err
	}
	if err := b.db.CreateTable(&Result{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		return err
	}
	return nil
}

func (b *Beautifier) Start(ctx context.Context) error {
	return nil
}

// Stop closes connection to db.
func (b *Beautifier) Stop(ctx context.Context) error {
	return b.db.Close()
}

// ParseAndStore consume array of records and pulse number parse them and save to db
func (b *Beautifier) ParseAndStore(records []sequence.Item, pulseNumber insolar.PulseNumber) {
	for i := 0; i < len(records); i++ {
		b.process(records[i].Key, records[i].Value, pulseNumber)
	}
}

func (b *Beautifier) process(key []byte, value []byte, pn insolar.PulseNumber) {
	if b.prevPulse != pn {
		b.flush(b.prevPulse)
	}

	id := parseID(key)
	rec := parseRecord(value)
	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_IncomingRequest:
		in := rec.Virtual.GetIncomingRequest()
		b.parseRequest(id, v.IncomingRequest)
		if in.CallType != record.CTGenesis && in.Method == "Call" {
			b.processCallRequest(pn, id, in)
		}
	case *record.Virtual_Result:
		res := rec.Virtual.GetResult()
		b.parseResult(id, v.Result)
		if rec := res.Request.Record(); rec != nil {
			b.processResult(pn, rec, res)
		}
	case *record.Virtual_Activate:
		b.parseActivate(id, v.Activate)
	case *record.Virtual_Amend:
		b.parseAmend(id, v.Amend)
	case *record.Virtual_Deactivate:
		b.parseDeactivate(id, v.Deactivate)
	}
}

func (b *Beautifier) processCallRequest(pn insolar.PulseNumber, id insolar.ID, in *record.IncomingRequest) {
	request := b.parseMemberCallArguments(in.Arguments)
	switch request.Params.CallSite {
	case "member.transfer":
		b.processTransferCall(pn, id, in, request)
	case "member.create":
		b.processMemberCreate(pn, id, in, request)
	case "member.migrationCreate":
		b.processMemberCreate(pn, id, in, request)
	}
}

func (b *Beautifier) processResult(pn insolar.PulseNumber, rec *insolar.ID, res *record.Result) {
	if req, ok := b.requests[*rec]; ok {
		in := req.value
		request := b.parseMemberCallArguments(in.Arguments)
		switch request.Params.CallSite {
		case "member.transfer":
			b.processTransferResult(pn, rec, res)
		case "member.create":
			b.processMemberCreateResult(pn, rec, res)
		case "member.migrationCreate":
			b.processMemberCreateResult(pn, rec, res)
		}
	} else {
		b.results[*rec] = HeadlessResult{timestamp: time.Now().Unix(), value: res}
	}
}

func (b *Beautifier) parseMemberCallArguments(inArgs []byte) member.Request {
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

func parseID(fullKey []byte) insolar.ID {
	id := insolar.ID{}
	copy(id[:], fullKey[1:])
	return id
}

func parseRecord(value []byte) record.Material {
	logger := inslogger.FromContext(context.Background())
	rec := record.Material{}
	err := rec.Unmarshal(value)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to unmarshal record data"))
		return record.Material{}
	}
	return rec
}

func parsePayload(payload []byte) []interface{} {
	logger := inslogger.FromContext(context.Background())
	rets := []interface{}{}
	err := insolar.Deserialize(payload, &rets)
	if err != nil {
		logger.Warnf("failed to parse payload as two interfaces")
		return []interface{}{}
	}
	return rets
}

func (b *Beautifier) flush(pn insolar.PulseNumber) {
	logger := inslogger.FromContext(context.Background())

	// TODO: make flush under single db transaction

	for _, tx := range b.txs {
		err := b.storeTx(tx)
		if err != nil {
			logger.Error(err)
		}
		if tx.Status != PENDING {
			delete(b.txs, tx.requestID)
			delete(b.requests, tx.requestID)
			delete(b.results, tx.requestID)
		}
	}

	for _, a := range b.members {
		err := b.storeMember(a)
		if err != nil {
			logger.Error(err)
		}
		if a.Status != PENDING {
			delete(b.txs, a.requestID)
			delete(b.requests, a.requestID)
			delete(b.results, a.requestID)
		}
	}

	for _, req := range b.rawRequests {
		err := b.storeRequest(req)
		if err != nil {
			logger.Error(err)
		}
		delete(b.rawResults, req.requestID)
	}

	for _, res := range b.rawResults {
		err := b.storeResult(res)
		if err != nil {
			logger.Error(err)
		}
		delete(b.rawResults, res.requestID)
	}

	for _, obj := range b.rawObjects {
		err := b.storeObject(obj)
		if err != nil {
			logger.Error(err)
		}
		delete(b.rawResults, obj.requestID)
	}
}
