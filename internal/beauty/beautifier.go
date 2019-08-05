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
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member/signer"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/replica"

	log "github.com/sirupsen/logrus"
)

func NewBeautifier() *Beautifier {
	return &Beautifier{
		cfg:            configuration.Default(),
		requests:       make(map[insolar.ID]SuspendedRequest),
		results:        make(map[insolar.ID]UnrelatedResult),
		intentions:     make(map[insolar.ID]SuspendedIntention),
		activates:      make(map[insolar.ID]UnrelatedActivate),
		balanceUpdates: make(map[insolar.ID]BalanceUpdate),
		txs:            make(map[insolar.ID]*Transfer),
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
	Configurator   configuration.Configurator `inject:""`
	Publisher      replica.Publisher          `inject:""`
	cfg            *configuration.Configuration
	db             *pg.DB
	prevPulse      insolar.PulseNumber
	requests       map[insolar.ID]SuspendedRequest
	results        map[insolar.ID]UnrelatedResult
	intentions     map[insolar.ID]SuspendedIntention
	activates      map[insolar.ID]UnrelatedActivate
	balanceUpdates map[insolar.ID]BalanceUpdate
	txs            map[insolar.ID]*Transfer
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
	if b.Configurator != nil {
		b.cfg = b.Configurator.Actual()
	} else {
		b.cfg = configuration.Default()
	}
	if b.Publisher != nil {
		b.Publisher.Subscribe(func(recordNumber uint32, rec *record.Material) {
			pn := rec.ID.Pulse()
			b.process(pn, rec)
		})
	}
	opt, err := pg.ParseURL(b.cfg.DB.URL)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to parse cfg.DB.URL"))
		return nil
	}
	b.db = pg.Connect(opt)
	if b.cfg.DB.CreateTables {
		b.createTables()
	}
	return nil
}

func (b *Beautifier) Start(ctx context.Context) error {
	return nil
}

// Stop closes connection to db.
func (b *Beautifier) Stop(ctx context.Context) error {
	err := b.db.Close()
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to close db"))
	}
	return nil
}

func (b *Beautifier) createTables() {
	if err := b.db.CreateTable(&Transfer{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		log.Error(errors.Wrapf(err, "failed to create transactions table"))
	}
	if err := b.db.CreateTable(&Member{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		log.Error(errors.Wrapf(err, "failed to create members table"))
	}
	if err := b.db.CreateTable(&Deposit{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		log.Error(errors.Wrapf(err, "failed to create deposits table"))
	}
	if err := b.db.CreateTable(&Object{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		log.Error(errors.Wrapf(err, "failed to create objects table"))
	}
	if err := b.db.CreateTable(&Request{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		log.Error(errors.Wrapf(err, "failed to create requests table"))
	}
	if err := b.db.CreateTable(&Result{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		log.Error(errors.Wrapf(err, "failed to create results table"))
	}
}

func (b *Beautifier) process(pn insolar.PulseNumber, rec *record.Material) {
	if b.prevPulse == 0 {
		b.prevPulse = pn
	}
	if b.prevPulse != pn {
		b.flush(b.prevPulse)
		b.prevPulse = pn
	}

	id := rec.ID
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
	var args []interface{}
	err := insolar.Deserialize(inArgs, &args)
	if err != nil {
		log.Warn(errors.Wrapf(err, "failed to deserialize request arguments"))
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
				log.Warn(errors.Wrapf(err, "failed to unmarshal params"))
				return member.Request{}
			}
			err = json.Unmarshal(raw, &request)
			if err != nil {
				log.Warn(errors.Wrapf(err, "failed to unmarshal json member request"))
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

func parsePayload(payload []byte) []interface{} {
	rets := []interface{}{}
	err := insolar.Deserialize(payload, &rets)
	if err != nil {
		log.Warnf("failed to parse payload as two interfaces")
		return []interface{}{}
	}
	return rets
}

func (b *Beautifier) flush(pn insolar.PulseNumber) {
	log.WithField("pulse", pn).Debugf("flushing beautified values")

	b.insertValues()
	b.updateValues()
}

func (b *Beautifier) insertValues() {
	tx, err := b.db.Begin()
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to create db transaction"))
		return
	}
	defer func() {
		err := tx.Commit()
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to commit db transaction"))
		}
	}()

	for _, transfer := range b.txs {
		err := storeTransfer(tx, transfer)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to save transfer"))
			return
		}
		if transfer.Status != PENDING {
			delete(b.txs, transfer.requestID)
			delete(b.requests, transfer.requestID)
			delete(b.results, transfer.requestID)
		}
	}

	for _, m := range b.members {
		err := storeMember(tx, m)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to save member"))
			return
		}
		// if m.Status != PENDING && m.Balance != "" {
		if m.Balance != "" {
			delete(b.txs, m.requestID)
			delete(b.requests, m.requestID)
			delete(b.results, m.requestID)
		}
	}

	for id, d := range b.deposits {
		err := storeDeposit(tx, d)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to save deposit"))
			return
		}
		delete(b.deposits, id)
	}

	for _, req := range b.rawRequests {
		err := storeRequest(tx, req)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to save request"))
			return
		}
		delete(b.rawResults, req.requestID)
	}

	for _, res := range b.rawResults {
		err := storeResult(tx, res)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to save result"))
			return
		}
		delete(b.rawResults, res.requestID)
	}

	for _, obj := range b.rawObjects {
		err := storeObject(tx, obj)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to save object"))
			return
		}
		delete(b.rawResults, obj.requestID)
	}
}

func (b *Beautifier) updateValues() {
	tx, err := b.db.Begin()
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to create db transaction"))
		return
	}
	defer func() {
		err := tx.Commit()
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to commit db transaction"))
		}
	}()

	// TODO: 1. Try to apply upd in-memory 2. Check update opportunity in DB and if it has then apply upd or defer it
	for id, upd := range b.balanceUpdates {
		err := updateBalance(tx, upd.id, upd.prevState, upd.balance)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to update balance"))
			return
		}
		delete(b.balanceUpdates, id)
	}

	for id, upd := range b.depositUpdates {
		err := updateDeposit(tx, upd.id, upd.amount, upd.withdrawn, upd.status, upd.prevState)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to update deposit"))
			return
		}
		delete(b.depositUpdates, id)
	}
}
