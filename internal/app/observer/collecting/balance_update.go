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
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/pkg/panic"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BalanceUpdateCollector struct {
	log *logrus.Logger
}

func NewBalanceUpdateCollector(log *logrus.Logger) *BalanceUpdateCollector {
	return &BalanceUpdateCollector{
		log: log,
	}
}

type Balance struct {
	foundation.BaseContract
	Balance  uint64
	GroupRef insolar.Reference
}

func (c *BalanceUpdateCollector) Collect(rec *observer.Record) *observer.BalanceUpdate {
	defer panic.Catch("balance_update_collector")

	if rec == nil {
		return nil
	}

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return nil
	}
	if !isBalanceAmend(v.Amend) {
		return nil
	}

	balance, err := balanceUpdate(rec)

	if err != nil {
		logrus.Info(err.Error())
		return nil
	}

	return &observer.BalanceUpdate{
		Balance:  balance.Balance,
		GroupRef: balance.GroupRef,
	}
}

func isBalanceAmend(amd *record.Amend) bool {
	prototypeRef, _ := insolar.NewReferenceFromString("0111A7rSyB9B9zk2FHqBzD15g7DnfVY3kbDkTRoJHiHm")
	return amd.Image.Equal(*prototypeRef)
}

func balanceUpdate(act *observer.Record) (*Balance, error) {
	var memory []byte
	switch v := act.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
	default:
		log.Error(errors.New("invalid record to get balance memory"))
	}

	if memory == nil {
		log.Warn(errors.New("balance memory is nil"))
		return nil, errors.New("invalid record to get balance memory")
	}

	var balance Balance

	err := insolar.Deserialize(memory, &balance)
	if err != nil {
		log.Error(errors.New("failed to deserialize balance memory"))
	}

	return &balance, nil
}
