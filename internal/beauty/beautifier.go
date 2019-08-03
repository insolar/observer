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
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/ledger/store"
)

func NewBeautifier() *Beautifier {
	return &Beautifier{
		requests:       make(map[insolar.ID]SuspendedRequest),
		results:        make(map[insolar.ID]UnrelatedResult),
		intentions:     make(map[insolar.ID]SuspendedIntention),
		activates:      make(map[insolar.ID]UnrelatedActivate),
		balanceUpdates: make(map[insolar.ID]BalanceUpdate),
		txs:            make(map[insolar.ID]*Transaction),
		members:        make(map[insolar.ID]*Member),
		deposits:       make(map[insolar.ID]*Deposit),
		depositUpdates: make(map[insolar.ID]DepositUpdate),
		rawObjects:     make(map[insolar.ID]*Object),
		rawResults:     make(map[insolar.ID]*Result),
		rawRequests:    make(map[insolar.ID]*Request),
	}
}

type SuspendedRequest struct {
	timestamp int64
	value     *record.IncomingRequest
}

type UnrelatedResult struct {
	timestamp int64
	value     *record.Result
}

type SuspendedIntention struct {
	timestamp int64
	value     *record.IncomingRequest
}

type UnrelatedActivate struct {
	timestamp int64
	id        insolar.ID
	value     *record.Activate
}

type BalanceUpdate struct {
	timestamp int64
	id        insolar.ID
	prevState string
	balance   string
}

type DepositUpdate struct {
	id        insolar.ID
	amount    string
	withdrawn string
	status    string
	prevState string
}

type Beautifier struct {
	Publisher      store.DBSetPublisher `inject:""`
	db             *pg.DB
	prevPulse      insolar.PulseNumber
	requests       map[insolar.ID]SuspendedRequest
	results        map[insolar.ID]UnrelatedResult
	intentions     map[insolar.ID]SuspendedIntention
	activates      map[insolar.ID]UnrelatedActivate
	balanceUpdates map[insolar.ID]BalanceUpdate
	txs            map[insolar.ID]*Transaction
	members        map[insolar.ID]*Member
	deposits       map[insolar.ID]*Deposit
	depositUpdates map[insolar.ID]DepositUpdate
	rawObjects     map[insolar.ID]*Object
	rawResults     map[insolar.ID]*Result
	rawRequests    map[insolar.ID]*Request
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
	if err := b.db.CreateTable(&Deposit{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
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
		b.prevPulse = pn
	}

	id := parseID(key)
	rec := parseRecord(value)
	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_IncomingRequest:
		in := v.IncomingRequest
		b.parseRequest(id, in)
		b.processRequest(pn, id, in)
	case *record.Virtual_Result:
		res := v.Result
		b.parseResult(id, res)
		b.processResult(pn, res)
	case *record.Virtual_Activate:
		act := v.Activate
		b.parseActivate(id, act)
		b.processActivate(pn, id, act)
	case *record.Virtual_Amend:
		amd := v.Amend
		b.parseAmend(id, amd)
		b.processAmend(id, amd)
	case *record.Virtual_Deactivate:
		b.parseDeactivate(id, v.Deactivate)
	}
}

func (b *Beautifier) processRequest(pn insolar.PulseNumber, id insolar.ID, in *record.IncomingRequest) {
	switch in.Method {
	case "Call":
		b.processCallRequest(pn, id, in)
	case "New":
		b.processNewRequest(pn, id, in)
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

func (b *Beautifier) processResult(pn insolar.PulseNumber, res *record.Result) {
	rec := *res.Request.Record()
	if req, ok := b.requests[rec]; ok {
		in := req.value
		request := b.parseMemberCallArguments(in.Arguments)
		switch request.Params.CallSite {
		case "member.transfer":
			b.processTransferResult(pn, rec, res)
		case "member.create":
			b.processMemberCreateResult(rec, res)
		case "member.migrationCreate":
			b.processMemberCreateResult(rec, res)
		}
	} else {
		b.results[rec] = UnrelatedResult{timestamp: time.Now().Unix(), value: res}
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

func (b *Beautifier) processActivate(pn insolar.PulseNumber, id insolar.ID, act *record.Activate) {
	rec := *act.Request.Record()
	if req, ok := b.intentions[rec]; ok {
		switch {
		case isWalletActivate(act):
			b.processWalletActivate(id, req.value, act)
		case isDepositActivate(act):
			b.processDepositActivate(pn, id, act)
		}
	} else {
		b.activates[rec] = UnrelatedActivate{timestamp: time.Now().Unix(), id: id, value: act}
	}
}

func (b *Beautifier) processNewRequest(pn insolar.PulseNumber, id insolar.ID, in *record.IncomingRequest) {
	switch {
	case isNewWallet(in):
		b.processNewWallet(pn, id, in)
	}
}

func (b *Beautifier) processAmend(id insolar.ID, amd *record.Amend) {
	switch {
	case isWalletAmend(amd):
		b.processWalletAmend(id, amd)
	case isDepositAmend(amd):
		b.processDepositAmend(id, amd)
	}
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
			continue
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
			continue
		}
		if a.Status != PENDING && a.Balance != "" {
			delete(b.txs, a.requestID)
			delete(b.requests, a.requestID)
			delete(b.results, a.requestID)
		}
	}

	for id, d := range b.deposits {
		err := b.storeDeposit(d)
		if err != nil {
			logger.Error(err)
			continue
		}
		delete(b.deposits, id)
	}

	for _, req := range b.rawRequests {
		err := b.storeRequest(req)
		if err != nil {
			logger.Error(err)
			continue
		}
		delete(b.rawResults, req.requestID)
	}

	for _, res := range b.rawResults {
		err := b.storeResult(res)
		if err != nil {
			logger.Error(err)
			continue
		}
		delete(b.rawResults, res.requestID)
	}

	for _, obj := range b.rawObjects {
		err := b.storeObject(obj)
		if err != nil {
			logger.Error(err)
			continue
		}
		delete(b.rawResults, obj.requestID)
	}

	for id, upd := range b.balanceUpdates {
		err := b.updateBalance(upd.id, upd.prevState, upd.balance)
		if err != nil {
			logger.Error(err)
			continue
		}
		delete(b.balanceUpdates, id)
	}

	for id, upd := range b.depositUpdates {
		err := b.updateDeposit(upd.id, upd.amount, upd.withdrawn, upd.status, upd.prevState)
		if err != nil {
			logger.Error(err)
			continue
		}
		delete(b.depositUpdates, id)
	}
}
