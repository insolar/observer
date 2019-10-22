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

package collecting

import (
	"context"
	"github.com/insolar/insolar/application/genesisrefs"
	"github.com/insolar/insolar/log"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"
	"strings"

	"github.com/insolar/insolar/application/builtin/contract/deposit"
	"github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
)

const (
	CallSite = "deposit.migration"
)

type DepositCollector struct {
	log       *logrus.Logger
	results   observer.ResultCollector
	activates observer.ActivateCollector
	fetcher   store.RecordFetcher
	builder   tree.Builder
}

func NewDepositCollector(log *logrus.Logger, fetcher store.RecordFetcher) *DepositCollector {
	return &DepositCollector{
		log: log,
		fetcher: fetcher,
		builder: tree.NewBuilder(fetcher),
	}
}

func (c *DepositCollector) SetFetcher(fetcher store.RecordFetcher) {
	c.fetcher = fetcher
}

func (c *DepositCollector) SetBuilder(builder tree.Builder) {
	c.builder = builder
}

func (c *DepositCollector) Collect(rec *observer.Record) *observer.Deposit {
	if rec == nil {
		return nil
	}

	// genesis admin deposit record
	if rec.ID.Pulse() == insolar.GenesisPulse.PulseNumber && isDepositActivate(rec) {
		timeActivate, err := rec.ID.Pulse().AsApproximateTime()
		if err != nil {
			log.Errorf("wrong timestamp in genesis deposit record: %+v", rec)
			return nil
		}
		activate := rec.Virtual.GetActivate()
		state := c.initialDepositState(activate)
		return &observer.Deposit{
			EthHash:         strings.ToLower(state.TxHash),
			Ref:             *genesisrefs.ContractMigrationDeposit.GetLocal(),
			Member:          *genesisrefs.ContractMigrationAdminMember.GetLocal(),
			Timestamp:       timeActivate.Unix(),
			HoldReleaseDate: 0,
			Amount:          state.Amount,
			Balance:         state.Balance,
			DepositState:    rec.ID,
		}
	}

	res := observer.CastToResult(rec)
	if !res.IsResult() || !res.IsSuccess() {
		return nil
	}

	ctx := context.Background()

	req, err := c.fetcher.Request(ctx, res.Request())
	if err != nil {
		c.log.WithField("req", res.Request()).Error(errors.Wrapf(err, "result without request"))
		return nil
	}

	_, ok := c.isDepositCall(&req)
	if !ok {
		return nil
	}

	callTree, err := c.builder.Build(ctx, req.ID)
	if err != nil {
		return nil
	}

	var activate = &observer.Activate{}

	for _, o := range callTree.Outgoings {
		if o.Structure.SideEffect.Activation.Image.Equal(*migrationdaemon.PrototypeReference) {
			activate = observer.CastToActivate(o.Structure.SideEffect.Activation)
		}
	}

	d, err := c.build(activate, res)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to build member"))
		return nil
	}
	return d
}

func (c *DepositCollector) isDepositCall(rec *record.Material) (*observer.Request, bool) {

	request := observer.CastToRequest((*observer.Record)(rec))

	if !request.IsIncoming() || !request.IsMemberCall() {
		return nil, false
	}

	args := request.ParseMemberCallArguments()
	return request, args.Params.CallSite == CallSite
}

func isDepositActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()
	return act.Image.Equal(*migrationdaemon.PrototypeReference)
}


func (c *DepositCollector) build(act *observer.Activate, res *observer.Result) (*observer.Deposit, error) {
	callResult := &migrationdaemon.DepositMigrationResult{}
	res.ParseFirstPayloadValue(callResult)
	if !res.IsSuccess() {
		return nil, errors.New("invalid create deposit result payload")
	}
	activate := act.Virtual.GetActivate()
	state := c.initialDepositState(activate)
	transferDate, err := act.ID.Pulse().AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert deposit create pulse (%d) to time", act.ID.Pulse())
	}

	memberRef, err := insolar.NewIDFromString(callResult.Reference)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make memberRef from base58 string")
	}

	return &observer.Deposit{
		EthHash:         strings.ToLower(state.TxHash), // from activate
		Ref:             act.Request(),                 // from activate
		Member:          *memberRef,                    // from result
		Timestamp:       transferDate.Unix(),
		HoldReleaseDate: 0,
		Amount:          state.Amount,  // from activate
		Balance:         state.Balance, // from activate
		DepositState:    act.ID,        // from activate
	}, nil
}

func (c *DepositCollector) initialDepositState(act *record.Activate) *deposit.Deposit {
	d := deposit.Deposit{}
	err := insolar.Deserialize(act.Memory, &d)
	if err != nil {
		c.log.Error(errors.New("failed to deserialize deposit contract state"))
	}
	return &d
}
