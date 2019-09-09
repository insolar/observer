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
	"strings"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/deposit"
	proxyDeposit "github.com/insolar/insolar/logicrunner/builtin/proxy/deposit"
	"github.com/insolar/insolar/logicrunner/builtin/proxy/migrationdaemon"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/v2/internal/app/observer"
)

type DepositCollector struct {
	log       *logrus.Logger
	collector *BoundCollector
}

func NewDepositCollector(log *logrus.Logger) *DepositCollector {
	collector := NewBoundCollector(isDepositMigrationCall, successResult, isDepositNew, isDepositActivate)
	return &DepositCollector{
		log:       log,
		collector: collector,
	}
}

func (c *DepositCollector) Collect(rec *observer.Record) *observer.Deposit {
	if rec == nil {
		return nil
	}
	log := c.log

	couple := c.collector.Collect(rec)
	if couple == nil {
		return nil
	}

	d, err := c.build(couple.Activate, couple.Result)
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

	memberRef, err := insolar.NewReferenceFromBase58(callResult.Reference)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make memberRef from base58 string")
	}

	depositRef := insolar.NewReference(act.Request())

	return &observer.Deposit{
		EthHash:         strings.ToLower(state.TxHash),
		Ref:             *depositRef,
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
