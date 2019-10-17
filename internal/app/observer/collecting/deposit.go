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
	"reflect"
	"strings"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/genesisrefs"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/deposit"
	proxyDeposit "github.com/insolar/insolar/logicrunner/builtin/proxy/deposit"
	"github.com/insolar/insolar/logicrunner/builtin/proxy/migrationdaemon"
	proxyDaemon "github.com/insolar/insolar/logicrunner/builtin/proxy/migrationdaemon"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
)

type DepositCollector struct {
	log       *logrus.Logger
	results   observer.ResultCollector
	activates observer.ActivateCollector
	halfChain observer.ChainCollector
	chains    observer.ChainCollector
}

func NewDepositCollector(log *logrus.Logger) *DepositCollector {
	results := NewResultCollector(isDepositMigrationCall, successResult)
	activates := NewActivateCollector(isDepositNew, isDepositActivate)
	resultRelation := &RelationDesc{
		Is:     isCoupledResult,
		Origin: coupledResultOrigin,
		Proper: isCoupledResult,
	}
	activateRelation := &RelationDesc{
		Is:     isCoupledActivate,
		Origin: coupledActivateOrigin,
		Proper: isCoupledActivate,
	}
	daemonCall := &RelationDesc{
		Is: isDaemonMigrationCall,
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.ID
		},
		Proper: isDaemonMigrationCall,
	}
	daemonRelation := &RelationDesc{
		Is: func(chain interface{}) bool {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return isDaemonMigrationCall(c.Parent)
		},
		Origin: func(chain interface{}) insolar.ID {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			request := observer.CastToRequest(c.Parent)
			return request.Reason()
		},
		Proper: func(chain interface{}) bool {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return isDaemonMigrationCall(c.Parent)
		},
	}
	return &DepositCollector{
		log:       log,
		results:   results,
		activates: activates,
		halfChain: NewChainCollector(daemonCall, activateRelation),
		chains:    NewChainCollector(resultRelation, daemonRelation),
	}
}

func (c *DepositCollector) Collect(rec *observer.Record) *observer.Deposit {
	if rec == nil {
		return nil
	}
	log := c.log

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

	res := c.results.Collect(rec)
	act := c.activates.Collect(rec)
	half := c.halfChain.Collect(rec)

	if act != nil {
		half = c.halfChain.Collect(act)
	}

	var chain *observer.Chain
	if res != nil {
		chain = c.chains.Collect(res)
	}

	if half != nil {
		chain = c.chains.Collect(half)
	}

	if chain == nil {
		return nil
	}

	coupleAct, coupleRes := c.unwrapDepositChain(chain)

	d, err := c.build(coupleAct, coupleRes)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build member"))
		return nil
	}
	return d
}

func isDepositMigrationCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "deposit.migration"
}

func isDaemonMigrationCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "DepositMigrationCall" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	return in.Prototype.Equal(*proxyDaemon.PrototypeReference)
}

func isDepositNew(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "New" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	return in.Prototype.Equal(*proxyDeposit.PrototypeReference)
}

func isDepositActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()
	return act.Image.Equal(*proxyDeposit.PrototypeReference)
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

	depositRef := act.Request()

	return &observer.Deposit{
		EthHash:         strings.ToLower(state.TxHash),
		Ref:             depositRef,
		Member:          *memberRef,
		Timestamp:       transferDate.Unix(),
		HoldReleaseDate: 0,
		Amount:          state.Amount,
		Balance:         state.Balance,
		DepositState:    act.ID,
	}, nil
}

func (c *DepositCollector) initialDepositState(act *record.Activate) *deposit.Deposit {
	log := c.log
	d := deposit.Deposit{}
	err := insolar.Deserialize(act.Memory, &d)
	if err != nil {
		log.Error(errors.New("failed to deserialize deposit contract state"))
	}
	return &d
}

func (c *DepositCollector) unwrapDepositChain(chain *observer.Chain) (*observer.Activate, *observer.Result) {
	log := c.log

	half := chain.Child.(*observer.Chain)
	coupledAct, ok := half.Child.(*observer.CoupledActivate)
	if !ok {
		log.Error(errors.Errorf("trying to use %s as *observer.Chain", reflect.TypeOf(chain.Child)))
		return nil, nil
	}
	if coupledAct.Activate == nil {
		log.Error(errors.New("invalid coupled activate chain, child is nil"))
		return nil, nil
	}
	actRecord := coupledAct.Activate

	coupledRes, ok := chain.Parent.(*observer.CoupledResult)
	if !ok {
		log.Error(errors.Errorf("trying to use %s as *observer.Chain", reflect.TypeOf(chain.Parent)))
		return nil, nil
	}
	if coupledRes.Result == nil {
		log.Error(errors.New("invalid coupled result chain, child is nil"))
		return nil, nil
	}
	resRecord := coupledRes.Result
	return actRecord, resRecord
}
